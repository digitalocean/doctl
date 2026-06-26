/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// The `doctl agents` subcommand wraps the godo HostedAgents service, which
// in turn talks to the hosted-agents Harness API. All wire types and the SSE
// iterator live in godo (hosted_agents.go); this file handles CLI plumbing,
// argument parsing, and human-readable rendering of streamed events.
package commands

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Agents creates the `doctl agents` command tree.
func Agents() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "agents",
			Aliases: []string{"agent"},
			Short:   "Launch and manage hosted DigitalOcean agent sessions",
			Long: `The ` + "`" + `doctl agents` + "`" + ` commands manage hosted coding-agent sessions running in DigitalOcean sandboxes.

A session is one long-lived agent process (Claude Code, OpenCode, ...) running inside a workspace sandbox. doctl drives it: starting it from an agent spec, attaching an interactive TUI, listing existing sessions, resolving HITL approvals out of band, and tearing it down.`,
			GroupID: hostedAgentsGroup,
		},
	}

	cmdStart := CmdBuilder(cmd, RunAgentsStart, "start",
		"Start a new agent session",
		`Creates a new agent session from an agent manifest file and prints its session id and status.

The `+"`"+`--spec`+"`"+` flag is required and accepts a YAML manifest matching the `+"`"+`agents.digitalocean.com/v1alpha1`+"`"+` schema. The manifest is sent verbatim to the server, which owns parsing and validation.`,
		Writer, aliasOpt("deploy"),
		displayerType(&displayers.HostedAgentSession{}))
	AddStringFlag(cmdStart, doctl.ArgAgentSpec, "", "", `Path to an agent manifest in YAML or JSON. Set to "-" to read from stdin.`, requiredOpt())
	cmdStart.Example = `doctl agents start --spec agent-spec.yaml`

	cmdAttach := CmdBuilder(cmd, RunAgentsAttach, "attach <session-id>",
		"Attach to an agent session",
		`Opens an interactive line-mode TUI on an existing session. Streams events from the server and accepts typed input. A dropped SSE connection is reconnected automatically and the server replays any events missed during the gap.

When a HITL approval is pending, the prompt switches to `+"`"+`[y/n/d] > `+"`"+` and a single keystroke resolves it -- no Enter required in an interactive terminal: `+"`"+`y`+"`"+`/`+"`"+`a`+"`"+` approves, `+"`"+`n`+"`"+`/`+"`"+`r`+"`"+` rejects, `+"`"+`d`+"`"+` defers. Piped input (CI / scripts) must send the letter word (`+"`"+`yes`+"`"+`/`+"`"+`no`+"`"+`/`+"`"+`defer`+"`"+`) followed by a newline. The explicit `+"`"+`/a <request-id>`+"`"+`, `+"`"+`/r <request-id>`+"`"+`, `+"`"+`/d <request-id>`+"`"+` slash commands still work; type `+"`"+`/help`+"`"+` to see them. Ctrl-D detaches without destroying the session.`,
		Writer, aliasOpt("chat"))
	cmdAttach.Example = `doctl agents attach sess_abc123`

	cmdList := CmdBuilder(cmd, RunAgentsList, "list",
		"List agent sessions",
		`Lists agent sessions visible to the caller. Supports pagination and status filtering via `+"`"+`--page-size`+"`"+`, `+"`"+`--page-token`+"`"+`, and `+"`"+`--status`+"`"+`. When more pages exist, the next page token is printed after the table.`,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HostedAgentSession{}))
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of sessions to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	AddStringFlag(cmdList, doctl.ArgAgentStatus, "", "", "Filter by session status (e.g. SESSION_STATUS_READY, SESSION_STATUS_DESTROYED)")
	cmdList.Example = `doctl agents list --page-size 10 --status SESSION_STATUS_READY`

	CmdBuilder(cmd, RunAgentsShow, "show <session-id>",
		"Show a single agent session",
		"Prints details of one agent session.",
		Writer, aliasOpt("get"),
		displayerType(&displayers.HostedAgentSession{}))

	CmdBuilder(cmd, RunAgentsLogs, "logs <session-id>",
		"Replay the full event history for a session",
		"Replays the full server-side event history for a session, then exits.",
		Writer)

	CmdBuilder(cmd, RunAgentsApprove, "approve <session-id> <request-id> <approve|reject|defer>",
		"Resolve a pending HITL request out of band",
		"Approves, rejects, or defers a pending HITL request without attaching the interactive TUI. The resolution source is recorded as `RESOLUTION_SOURCE_OUT_OF_BAND`. Inside an attached session, the same outcomes are available as `/a`, `/r`, `/d` slash commands.",
		Writer)

	CmdBuilder(cmd, RunAgentsDestroy, "destroy <session-id>",
		"Destroy an agent session",
		"Tears down the workspace sandbox and removes the session.",
		Writer, aliasOpt("rm"))

	cmdUpload := CmdBuilder(cmd, RunAgentsUpload, "upload <session-id>",
		"Upload a file into a session workspace",
		`Streams a local file (or tar archive) into the session's sandbox workspace.

`+"`"+`--workspace-path`+"`"+` is resolved inside the workspace root (`+"`"+`/workspace`+"`"+`); a path that escapes the root is rejected by the server. Pass `+"`"+`--archive`+"`"+` when the local file is a tar that the server should extract at the destination. doctl computes the SHA-256 of the payload and forwards it so the guest can verify the upload. Files larger than 500 MiB are rejected by the server.`,
		Writer,
		displayerType(&displayers.HostedAgentWorkspaceUpload{}))
	AddStringFlag(cmdUpload, doctl.ArgAgentWorkspacePath, "", "", "Destination path inside the workspace root (/workspace)", requiredOpt())
	AddStringFlag(cmdUpload, doctl.ArgAgentLocalFile, "", "", "Path to the local file to upload", requiredOpt())
	AddBoolFlag(cmdUpload, doctl.ArgAgentArchive, "", false, "Treat the local file as a tar archive to extract at the destination")
	cmdUpload.Example = `doctl agents upload sess_abc123 --local-file ./main.go --workspace-path src/main.go`

	cmdDownload := CmdBuilder(cmd, RunAgentsDownload, "download <session-id>",
		"Download a file from a session workspace",
		`Streams a file (or tar archive) out of the session's sandbox workspace and writes it to a local destination.

`+"`"+`--workspace-path`+"`"+` is resolved inside the workspace root (`+"`"+`/workspace`+"`"+`). Pass `+"`"+`--archive`+"`"+` to tar-stream a directory. The download is chunked and the integrity SHA-256 arrives as an HTTP trailer after the body; doctl hashes the bytes as it writes them and verifies the trailer once the stream completes. A missing trailer or checksum mismatch means the transfer was truncated or corrupted, so the partial output is discarded and the command fails.`,
		Writer)
	AddStringFlag(cmdDownload, doctl.ArgAgentWorkspacePath, "", "", "Source path inside the workspace root (/workspace)", requiredOpt())
	AddStringFlag(cmdDownload, doctl.ArgAgentSaveTo, "", "", "Local file path to write the download to", requiredOpt())
	AddBoolFlag(cmdDownload, doctl.ArgAgentArchive, "", false, "Tar-stream the directory at the source path")
	cmdDownload.Example = `doctl agents download sess_abc123 --workspace-path src/main.go --save-to ./main.go`

	return cmd
}

// --- runners ----------------------------------------------------------------

// RunAgentsStart creates a new hosted agent session by uploading an agent
// manifest verbatim.
func RunAgentsStart(c *CmdConfig) error {
	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAgentSpec)
	if err != nil {
		return err
	}

	manifest, err := readManifest(os.Stdin, specPath)
	if err != nil {
		return err
	}

	sess, err := c.HostedAgents().CreateSessionFromManifest(manifest)
	if err != nil {
		if sessionLimitErr(err) {
			msg, _, _ := agentAPIError(err)
			return fmt.Errorf("%s. Free a slot by destroying one: run `doctl agents list` to find a session ID, then `doctl agents destroy SESSION_ID`", strings.TrimRight(msg, "."))
		}
		return err
	}
	return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}})
}

// readManifest returns the spec file as raw bytes. path "-" reads from stdin.
// The only client-side validation is "non-empty after trim" so a stray
// `--spec /dev/null` fails fast instead of hitting the server.
func readManifest(stdin io.Reader, path string) ([]byte, error) {
	var src io.Reader
	if path == "-" && stdin != nil {
		src = stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("opening manifest: %s does not exist", path)
			}
			return nil, fmt.Errorf("opening manifest: %w", err)
		}
		defer f.Close()
		src = f
	}

	raw, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("manifest is empty")
	}
	return raw, nil
}

// RunAgentsList lists hosted agent sessions visible to the caller.
func RunAgentsList(c *CmdConfig) error {
	opt, err := agentsListOptions(c)
	if err != nil {
		return err
	}
	sessions, nextPageToken, err := c.HostedAgents().ListSessions(opt)
	if err != nil {
		return err
	}
	if err := c.Display(&displayers.HostedAgentSession{Sessions: sessions}); err != nil {
		return err
	}
	if nextPageToken != "" {
		fmt.Fprintf(c.Out, "Next page token: %s\n", nextPageToken)
	}
	return nil
}

func agentsListOptions(c *CmdConfig) (*godo.HostedAgentSessionListOptions, error) {
	pageSize, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPageSize)
	if err != nil {
		return nil, err
	}
	pageToken, err := c.Doit.GetString(c.NS, doctl.ArgAgentPageToken)
	if err != nil {
		return nil, err
	}
	status, err := c.Doit.GetString(c.NS, doctl.ArgAgentStatus)
	if err != nil {
		return nil, err
	}
	if pageSize == 0 && pageToken == "" && status == "" {
		return nil, nil
	}
	opt := &godo.HostedAgentSessionListOptions{}
	if pageSize > 0 {
		opt.PageSize = pageSize
	}
	if pageToken != "" {
		opt.PageToken = pageToken
	}
	if status != "" {
		opt.Status = godo.HostedAgentSessionStatus(status)
	}
	return opt, nil
}

// RunAgentsShow prints one session.
func RunAgentsShow(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sess, err := c.HostedAgents().GetSession(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}})
}

// RunAgentsDestroy tears down a session.
func RunAgentsDestroy(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	if err := c.HostedAgents().DestroySession(c.Args[0]); err != nil {
		return err
	}
	notice("Session %s destroyed", c.Args[0])
	return nil
}

// RunAgentsUpload streams a local file (or tar archive) into a session's
// workspace sandbox. The SHA-256 of the payload is computed up front and
// forwarded so the guest can verify what it received.
func RunAgentsUpload(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sessionID := c.Args[0]

	workspacePath, err := c.Doit.GetString(c.NS, doctl.ArgAgentWorkspacePath)
	if err != nil {
		return err
	}
	localFile, err := c.Doit.GetString(c.NS, doctl.ArgAgentLocalFile)
	if err != nil {
		return err
	}
	isArchive, err := c.Doit.GetBool(c.NS, doctl.ArgAgentArchive)
	if err != nil {
		return err
	}

	f, err := os.Open(localFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("opening upload file: %s does not exist", localFile)
		}
		return fmt.Errorf("opening upload file: %w", err)
	}
	defer f.Close()

	// Hash the payload before sending so the X-Content-Sha256 header is set on
	// the request; rewind afterward so the same bytes stream as the body.
	sum, err := hashFile(f)
	if err != nil {
		return fmt.Errorf("hashing upload file: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewinding upload file: %w", err)
	}

	resp, err := c.HostedAgents().UploadWorkspace(sessionID, &godo.HostedAgentWorkspaceUploadRequest{
		Path:          workspacePath,
		IsArchive:     isArchive,
		ContentSHA256: sum,
		Body:          f,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentWorkspaceUpload{Uploads: []*godo.HostedAgentWorkspaceUploadResponse{resp}})
}

// hashFile returns the hex-encoded SHA-256 of r, reading it to EOF.
func hashFile(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// RunAgentsDownload streams a file (or tar archive) out of a session workspace.
// The godo download body hashes the stream and verifies the SHA-256 trailer
// only after the body is fully drained, so the integrity error surfaces at EOF
// (or on Close). The bytes are written to a temporary file first and only moved
// into place once the transfer verifies; a truncated or corrupted transfer is
// discarded.
func RunAgentsDownload(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sessionID := c.Args[0]

	workspacePath, err := c.Doit.GetString(c.NS, doctl.ArgAgentWorkspacePath)
	if err != nil {
		return err
	}
	saveTo, err := c.Doit.GetString(c.NS, doctl.ArgAgentSaveTo)
	if err != nil {
		return err
	}
	asArchive, err := c.Doit.GetBool(c.NS, doctl.ArgAgentArchive)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dl, err := c.HostedAgents().DownloadWorkspace(ctx, sessionID, &godo.HostedAgentWorkspaceDownloadRequest{
		Path:      workspacePath,
		AsArchive: asArchive,
	})
	if err != nil {
		return err
	}

	written, err := streamDownloadToFile(dl, saveTo)
	if err != nil {
		return err
	}

	notice("Downloaded %d bytes to %s", written, saveTo)
	return nil
}

// streamDownloadToFile copies the verified download body into saveTo. It writes
// to a sibling temp file and renames it into place only after the body reads to
// EOF and Close both succeed (which is where godo surfaces an invalid integrity
// trailer). On any failure the temp file is removed so no partial/corrupt
// output is left behind.
func streamDownloadToFile(dl *godo.HostedAgentWorkspaceDownload, saveTo string) (int64, error) {
	dir := filepath.Dir(saveTo)
	tmp, err := os.CreateTemp(dir, ".doctl-download-*")
	if err != nil {
		return 0, fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}

	// io.Copy drains the body to EOF, which triggers godo's trailer
	// verification; a missing or mismatched checksum is returned here.
	written, copyErr := io.Copy(tmp, dl.Body)
	closeBodyErr := dl.Body.Close()
	if copyErr != nil {
		cleanup()
		return 0, fmt.Errorf("downloading workspace file: %w", copyErr)
	}
	if closeBodyErr != nil {
		cleanup()
		return 0, fmt.Errorf("downloading workspace file: %w", closeBodyErr)
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return 0, fmt.Errorf("flushing download: %w", err)
	}
	if err := os.Rename(tmpName, saveTo); err != nil {
		os.Remove(tmpName)
		return 0, fmt.Errorf("saving download to %s: %w", saveTo, err)
	}
	return written, nil
}

// RunAgentsApprove resolves a pending HITL request out of band.
func RunAgentsApprove(c *CmdConfig) error {
	if len(c.Args) < 3 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if len(c.Args) > 3 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	sessionID, requestID := c.Args[0], c.Args[1]
	outcome, err := hitlOutcomeFor(c.Args[2])
	if err != nil {
		return err
	}
	if err := c.HostedAgents().ResolveHITL(sessionID, requestID, &godo.HostedAgentResolveHITLRequest{
		Outcome: outcome,
		Source:  godo.HostedAgentResolutionSourceOutOfBand,
	}); err != nil {
		return err
	}
	notice("HITL request %s resolved as %s", requestID, outcome)
	return nil
}

func hitlOutcomeFor(s string) (godo.HostedAgentHITLOutcome, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "approve":
		return godo.HostedAgentHITLOutcomeApprove, nil
	case "reject":
		return godo.HostedAgentHITLOutcomeReject, nil
	case "defer":
		return godo.HostedAgentHITLOutcomeDefer, nil
	default:
		return "", fmt.Errorf("unknown outcome %q; expected approve, reject, or defer", s)
	}
}

// RunAgentsLogs replays the full event history for a session, then exits.
func RunAgentsLogs(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := c.HostedAgents().StreamSession(ctx, c.Args[0], &godo.HostedAgentSessionStreamOptions{
		ReplayOnly: true,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	for stream.Next() {
		ev := stream.Current()
		// TOKEN_CHUNK events stream token-by-token without trailing newlines so
		// they render as one continuous line; printing a per-event header here
		// would land mid-line. Other event kinds are discrete and get a header.
		if ev.Kind != godo.HostedAgentEventKindTokenChunk {
			fmt.Fprintf(c.Out, "[%s] %s\n", ev.At.Time.UTC().Format("2006-01-02T15:04:05Z"), ev.Kind)
		}
		renderEvent(c.Out, ev)
	}
	return stream.Err()
}

// RunAgentsAttach opens the interactive TUI: one goroutine drains the SSE
// stream (with auto-reconnect), the main goroutine reads stdin.
func RunAgentsAttach(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sessionID := c.Args[0]
	svc := c.HostedAgents()

	sess, err := svc.GetSession(sessionID)
	if err != nil {
		if msg, terminal := classifyStreamError(err); terminal {
			return errors.New(strings.TrimSpace(msg))
		}
		return err
	}

	pending := &pendingHITL{}
	cursor := &eventCursor{}
	state := newAttachState(c.Out, pending)

	// All writes flow through the display so events don't clobber the user's
	// in-progress input once raw mode is on. Pass-through until raw=true.
	originalOut := c.Out
	c.Out = state.display
	defer func() { c.Out = originalOut }()

	fmt.Fprintf(c.Out, "Connected to %s (%s)\n", sessionID, sess.AgentKind)
	fmt.Fprintln(c.Out, "Type a message and press Enter to send. Ctrl-D to detach. HITL approvals are single-keystroke: y/a approve, n/r reject, d defer. Type `/help` for the full command list.")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thinking := newThinkingState(c.Out)
	defer thinking.stop()

	go streamWithReconnect(ctx, svc, sessionID, c.Out, pending, cursor, thinking)

	return runAttach(c, svc, sessionID, os.Stdin, state)
}

// runAttach dispatches to the raw-mode TTY loop or the legacy bufio line-mode
// loop based on whether stdin is an interactive terminal.
func runAttach(c *CmdConfig, svc do.HostedAgentsService, sessionID string, in io.Reader, state *attachState) error {
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return attachLoopTTY(c, svc, sessionID, f, state)
	}
	return attachLoop(c, svc, sessionID, in, state.pending)
}

// eventCursor holds the EventID of the latest event rendered. The stream
// goroutine writes; reconnect attempts read. Empty string means "no events
// rendered yet" and the server will start from the live tail.
type eventCursor struct {
	mu sync.Mutex
	id string
}

func (c *eventCursor) set(id string) {
	if id == "" {
		return
	}
	c.mu.Lock()
	c.id = id
	c.mu.Unlock()
}

func (c *eventCursor) get() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.id
}

// Backoff schedule for reconnects. Caps at maxReconnectBackoff and retries
// indefinitely; users break out via Ctrl-D (which cancels ctx) or Ctrl-C.
const (
	initialReconnectBackoff = 1 * time.Second
	maxReconnectBackoff     = 30 * time.Second
)

// thinkingState shows a spinner between RunStarted and the first real
// output. Animates above the prompt when out is a *promptDisplay; falls back
// to a one-shot "(thinking...)" print otherwise (pipes, line-mode).
type thinkingState struct {
	mu     sync.Mutex
	out    io.Writer
	active bool
	cancel context.CancelFunc
	done   chan struct{}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func newThinkingState(out io.Writer) *thinkingState {
	return &thinkingState{out: out}
}

func (s *thinkingState) start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true

	display, ok := s.out.(*promptDisplay)
	if !ok {
		fmt.Fprintln(s.out, "(thinking...)")
		s.mu.Unlock()
		return
	}
	// Reserve the spinner line and draw its first frame atomically so no
	// redraw or token can slip in between and shift the line the animator
	// (and stop) expect one row above the prompt.
	display.spinnerInit(spinnerFrames[0])
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.mu.Unlock()
	go s.animate(ctx, display, done)
}

func (s *thinkingState) stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	// Wait for the animator to exit before touching the line, so no stray
	// frame lands after we replace it.
	if done != nil {
		<-done
	}
	if display, ok := s.out.(*promptDisplay); ok {
		display.spinnerStop()
	}
}

func (s *thinkingState) animate(ctx context.Context, d *promptDisplay, done chan struct{}) {
	defer close(done)
	t := time.NewTicker(80 * time.Millisecond)
	defer t.Stop()
	ix := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ix = (ix + 1) % len(spinnerFrames)
			d.spinnerFrame(spinnerFrames[ix])
		}
	}
}

// streamWithReconnect drains the SSE iterator and reconnects on transient
// errors with bounded backoff, replaying from cursor.get(). Returns when ctx
// is cancelled or the server cleanly ends the stream.
func streamWithReconnect(
	ctx context.Context,
	svc do.HostedAgentsService,
	sessionID string,
	out io.Writer,
	pending *pendingHITL,
	cursor *eventCursor,
	thinking *thinkingState,
) {
	backoff := initialReconnectBackoff
	attempt := 0
	// Persisted across reconnects so a cursor-replayed segment is still
	// recognised as a repeat.
	dedup := &tokenDeduper{}

	for {
		if ctx.Err() != nil {
			return
		}

		opt := &godo.HostedAgentSessionStreamOptions{ReplayFrom: cursor.get()}
		stream, err := svc.StreamSession(ctx, sessionID, opt)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			thinking.stop()
			if msg, terminal := classifyStreamError(err); terminal {
				fmt.Fprintln(out, msg)
				return
			}
			attempt++
			fmt.Fprintf(out, "\n(reconnect attempt %d failed: %v; retrying in %s)\n", attempt, err, backoff)
			if !sleepCtx(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		if attempt > 0 {
			fmt.Fprintln(out, "(reconnected)")
		}
		attempt = 0
		backoff = initialReconnectBackoff

		drainStream(stream, out, pending, cursor, thinking, dedup)
		streamErr := stream.Err()
		stream.Close()

		if ctx.Err() != nil {
			return
		}
		if streamErr == nil {
			return
		}
		thinking.stop()
		if msg, terminal := classifyStreamError(streamErr); terminal {
			fmt.Fprintln(out, msg)
			return
		}
		fmt.Fprintf(out, "\n(stream dropped: %v; reconnecting in %s)\n", streamErr, backoff)
		if !sleepCtx(ctx, backoff) {
			return
		}
		backoff = nextBackoff(backoff)
	}
}

// classifyStreamError returns (user-facing message, terminal). Terminal
// errors stop the reconnect loop (auth, missing session, V0 single-connection
// rejection); status codes follow harness-api's apierr convention.
func classifyStreamError(err error) (string, bool) {
	var er *godo.ErrorResponse
	if !errors.As(err, &er) || er.Response == nil {
		return "", false
	}
	switch er.Response.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Sprintf("\nAuthentication failed: %s\nRun `doctl auth init` and try again.", er.Message), true
	case http.StatusForbidden:
		return fmt.Sprintf("\nAccess denied: %s", er.Message), true
	case http.StatusNotFound:
		return fmt.Sprintf("\nSession not found: %s", er.Message), true
	case http.StatusConflict:
		// V0 single-connection rejection. er.Message carries device + when.
		return fmt.Sprintf("\nSession already attached on another device: %s\nDetach there first, then re-run `doctl agents attach`.", er.Message), true
	}
	return "", false
}

// isRunTerminalErr reports the harness-api 409 returned when input is sent to
// a session whose run is already terminal; recovery is a fresh `agents start`.
func isRunTerminalErr(err error) bool {
	msg, status, ok := agentAPIError(err)
	return ok && status == http.StatusConflict &&
		strings.Contains(strings.ToLower(msg), "run is terminal")
}

// agentAPIError unwraps err to a *godo.ErrorResponse and returns its effective
// message and HTTP status. The reason lives in NestedError.Message when the
// top-level Message is empty (harness-api's {"error":{...}} envelope).
func agentAPIError(err error) (message string, status int, ok bool) {
	var er *godo.ErrorResponse
	if !errors.As(err, &er) || er.Response == nil {
		return "", 0, false
	}
	msg := er.Message
	if msg == "" && er.NestedError != nil {
		msg = er.NestedError.Message
	}
	return msg, er.Response.StatusCode, true
}

// sessionLimitErr reports the 409 from `agents start` when the team has hit its
// active-session cap (e.g. "team is at the limit of 4 active sessions").
func sessionLimitErr(err error) bool {
	msg, status, ok := agentAPIError(err)
	if !ok || status != http.StatusConflict {
		return false
	}
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "limit of") && strings.Contains(lower, "active sessions")
}

// drainStream consumes events until the iterator stops, rendering each one
// and updating the HITL pending map, the reconnect cursor, and the spinner.
func drainStream(stream *godo.HostedAgentSessionStream, out io.Writer, pending *pendingHITL, cursor *eventCursor, thinking *thinkingState, dedup *tokenDeduper) {
	for stream.Next() {
		ev := stream.Current()
		switch ev.Kind {
		case godo.HostedAgentEventKindHITLRequested:
			var p hitlRequestedPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil {
				pending.set(p.HitlID, hitlActionLabel(p.Payload))
			}
		case godo.HostedAgentEventKindHITLResolved:
			var p hitlResolvedPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil {
				pending.clearIf(p.HitlID)
			}
		}

		switch ev.Kind {
		case godo.HostedAgentEventKindRunStarted:
			dedup.reset()
			renderEvent(out, ev)
			thinking.start()
		case godo.HostedAgentEventKindTokenChunk:
			thinking.stop()
			var p tokenChunkPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil && dedup.allow(p.Text) {
				fmt.Fprint(out, p.Text)
			}
		default:
			thinking.stop()
			dedup.reset()
			renderEvent(out, ev)
		}

		// A finished run means the server has cancelled any still-pending tool
		// calls, so flush the local queue rather than leave stale entries.
		if ev.Kind == godo.HostedAgentEventKindRunCompleted || ev.Kind == godo.HostedAgentEventKindRunFailed {
			if n := pending.reset(); n > 0 {
				fmt.Fprintf(out, "(%d pending approval(s) cancelled — run ended)\n", n)
			}
		}

		cursor.set(ev.EventID)
	}
}

// minDedupeLen keeps short, legitimately-repeated output (e.g. "ok\n") from
// being swallowed; double-emitted reasoning blocks are always longer.
const minDedupeLen = 8

// tokenDeduper drops a run.token_delta that repeats the whole segment already
// printed in the current text run — some adapters stream the model's reasoning
// live and then re-send it as one consolidated delta.
type tokenDeduper struct {
	seg strings.Builder
}

func (d *tokenDeduper) allow(text string) bool {
	if text == "" {
		return true
	}
	if d.seg.Len() >= minDedupeLen && text == d.seg.String() {
		return false
	}
	d.seg.WriteString(text)
	return true
}

func (d *tokenDeduper) reset() {
	d.seg.Reset()
}

func nextBackoff(cur time.Duration) time.Duration {
	next := cur * 2
	if next > maxReconnectBackoff {
		return maxReconnectBackoff
	}
	return next
}

// sleepCtx waits for d or returns false immediately if ctx is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// pendingEntry is one HITL approval waiting on the user.
type pendingEntry struct {
	id     string
	action string // tool / kind hint shown by /pending; "" if unknown
}

// pendingHITL is a FIFO queue of HITL approvals. The head (entries[0]) is
// what single-keystroke y/n/d resolves; the rest are surfaced by /pending and
// resolvable by explicit id via /a /r /d.
type pendingHITL struct {
	mu      sync.Mutex
	entries []pendingEntry
}

// set enqueues id at the tail if not already queued. action is optional and
// used only for display; pass "" if unknown. Re-enqueueing the same id is a
// no-op so duplicate HITLRequested events (e.g. SSE replay) don't double-up.
func (p *pendingHITL) set(id string, action ...string) {
	if id == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.entries {
		if e.id == id {
			return
		}
	}
	a := ""
	if len(action) > 0 {
		a = action[0]
	}
	p.entries = append(p.entries, pendingEntry{id: id, action: a})
}

// get returns the head (oldest) id, or "" if the queue is empty.
func (p *pendingHITL) get() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.entries) == 0 {
		return ""
	}
	return p.entries[0].id
}

// clearIf removes id from anywhere in the queue. Used by both the optimistic
// post-resolve clear and the server's HITLResolved event, which may target
// a non-head entry (resolved by another device or by /a <id>).
func (p *pendingHITL) clearIf(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, e := range p.entries {
		if e.id == id {
			p.entries = append(p.entries[:i], p.entries[i+1:]...)
			return
		}
	}
}

// len returns the queue depth (used to flip the prompt to show "(N pending)").
func (p *pendingHITL) len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.entries)
}

// list returns a snapshot of all pending entries in arrival order.
func (p *pendingHITL) list() []pendingEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]pendingEntry, len(p.entries))
	copy(out, p.entries)
	return out
}

// reset drops every queued entry and returns how many were cleared.
func (p *pendingHITL) reset() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(p.entries)
	p.entries = nil
	return n
}

func attachLoop(c *CmdConfig, svc do.HostedAgentsService, sessionID string, in io.Reader, pending *pendingHITL) error {
	reader := bufio.NewReader(in)
	for {
		fmt.Fprint(c.Out, "\n", attachPrompt(pending))

		// HITL pending + TTY: single keystroke decides. Non-TTY falls back to
		// the bufio line-mode branch below.
		if id := pending.get(); id != "" {
			outcome, key, action := readHITLKeystroke(in)
			switch action {
			case hitlKeyResolve:
				fmt.Fprintf(c.Out, "%c\n", key)
				if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
					Outcome: outcome,
					Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
				}); err != nil {
					fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
				} else {
					// Clear client-side now; the server's HITLResolved event
					// will arrive over SSE and call clearIf again (idempotent).
					// Without this, the next loop iteration re-enters raw mode
					// and blocks on stdin until the SSE round-trip completes,
					// which the user sees as "had to press Enter after y".
					pending.clearIf(id)
				}
				continue
			case hitlKeyDetach:
				fmt.Fprintln(c.Out)
				return nil
			case hitlKeyIgnore:
				fmt.Fprintln(c.Out, "(press y, n, or d to resolve the pending approval, or Ctrl-D to detach)")
				continue
			case hitlKeyFallback:
				// Non-TTY — fall through to line mode.
			}
		}

		line, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(c.Out)
			return nil
		}
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Line-mode HITL shortcut (piped input only).
		if outcome, ok := hitlLetterShortcut(line); ok {
			if id := pending.get(); id != "" {
				if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
					Outcome: outcome,
					Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
				}); err != nil {
					fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
				} else {
					pending.clearIf(id)
				}
				continue
			}
		}
		if strings.HasPrefix(line, "/") {
			if err := handleAttachCommand(c, svc, sessionID, line, pending); err != nil {
				fmt.Fprintf(c.Out, "error: %v\n", err)
			}
			continue
		}
		resp, err := svc.SendInput(sessionID, &godo.HostedAgentSendInputRequest{Text: line})
		if err != nil {
			if isRunTerminalErr(err) {
				fmt.Fprintln(c.Out, "\nThis session's run has ended and can't accept new input.")
				fmt.Fprintln(c.Out, "Start a new session:  doctl agents start --spec <your-spec>.yaml")
				fmt.Fprintln(c.Out, "(detaching)")
				return nil
			}
			fmt.Fprintf(c.Out, "send failed: %v\n", err)
			continue
		}
		// Ack immediately; the first agent token can be tens of seconds away
		// and without this users re-submit, spawning a duplicate run.
		if resp != nil && resp.RunID != "" {
			fmt.Fprintf(c.Out, "(queued as %s; waiting for the agent...)\n", resp.RunID)
		} else {
			fmt.Fprintln(c.Out, "(queued; waiting for the agent...)")
		}
	}
}

// attachPrompt reflects the HITL queue depth so the user knows when more than
// one approval is waiting. The single-keystroke shortcut always targets the
// head; the count nudges the user to use /pending if they want to see the rest.
func attachPrompt(pending *pendingHITL) string {
	n := pending.len()
	switch {
	case n == 0:
		return "> "
	case n == 1:
		return "[y/n/d] > "
	default:
		return fmt.Sprintf("[y/n/d] (%d pending) > ", n)
	}
}

// attachState bundles the line buffer, pending HITL id, and the synchronized
// display that the SSE goroutine writes through.
type attachState struct {
	pending *pendingHITL
	display *promptDisplay
	mu      sync.Mutex // guards lineBuf
	lineBuf []byte
}

func newAttachState(out io.Writer, pending *pendingHITL) *attachState {
	s := &attachState{pending: pending}
	s.display = &promptDisplay{
		out:    out,
		prompt: func() string { return attachPrompt(pending) },
		lineBuf: func() string {
			s.mu.Lock()
			defer s.mu.Unlock()
			return string(s.lineBuf)
		},
	}
	return s
}

// promptDisplay serializes terminal writes between the input loop and the
// SSE goroutine. In raw mode it tracks whether the cursor sits on the prompt
// line or mid-stream so that streaming tokens don't get wiped, events drop
// to a fresh line, and HITL prompt flips render instantly. Pass-through
// otherwise.
type promptDisplay struct {
	mu      sync.Mutex
	out     io.Writer
	prompt  func() string
	lineBuf func() string
	raw     bool
	// midLine: cursor is at the end of a previous tokenless write. Next
	// Write must not clear-line, next echo must not paint to that line.
	midLine bool
}

func (p *promptDisplay) setRaw(on bool) {
	p.mu.Lock()
	p.raw = on
	p.mu.Unlock()
}

func (p *promptDisplay) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw {
		return p.out.Write(b)
	}
	if len(b) == 0 {
		return 0, nil
	}

	startsWithNL := b[0] == '\n'
	endsWithNL := b[len(b)-1] == '\n'

	if p.midLine {
		// Inject a separator only for discrete events that don't already
		// start with \n — so "(thinking...)" doesn't glue onto a token stream.
		if !startsWithNL && endsWithNL {
			fmt.Fprint(p.out, "\r\n")
		}
	} else {
		fmt.Fprint(p.out, "\r\x1b[K")
	}

	if _, err := io.WriteString(p.out, strings.ReplaceAll(string(b), "\n", "\r\n")); err != nil {
		return 0, err
	}

	if endsWithNL {
		fmt.Fprintf(p.out, "%s%s", p.prompt(), p.lineBuf())
		p.midLine = false
	} else {
		p.midLine = true
	}
	return len(b), nil
}

// echo writes a single keystroke for user feedback. Silent mid-stream so
// chars don't land on the agent's line; lineBuf still captures them and they
// reappear on the next prompt redraw.
func (p *promptDisplay) echo(b []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.raw && p.midLine {
		return
	}
	p.out.Write(b)
}

// redraw re-renders prompt + lineBuf. Flips "> " <-> "[y/n/d] > " the moment
// HITL state changes, no Enter needed.
func (p *promptDisplay) redraw() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw {
		return
	}
	if p.midLine {
		fmt.Fprint(p.out, "\r\n")
		p.midLine = false
	}
	fmt.Fprintf(p.out, "\r\x1b[K%s%s", p.prompt(), p.lineBuf())
}

// spinnerInit reserves the spinner's own line above the prompt and draws the
// first frame, then redraws the prompt below it — all in one locked write so
// the spinner row and the prompt row stay adjacent for spinnerFrame/spinnerStop.
func (p *promptDisplay) spinnerInit(frame string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw {
		return
	}
	if p.midLine {
		fmt.Fprint(p.out, "\r\n")
		p.midLine = false
	} else {
		fmt.Fprint(p.out, "\r\x1b[K")
	}
	fmt.Fprintf(p.out, "%s thinking...\r\n%s%s", frame, p.prompt(), p.lineBuf())
}

// spinnerFrame redraws the spinner line one row above the prompt.
// DECSC/DECRC (\x1b7 / \x1b8) save+restore the cursor so the prompt row
// below is preserved. No-op in non-raw or mid-stream state.
func (p *promptDisplay) spinnerFrame(frame string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw || p.midLine {
		return
	}
	fmt.Fprintf(p.out, "\x1b7\x1b[A\r\x1b[K%s thinking...\x1b8", frame)
}

// spinnerStop replaces the spinner frame with the plain "(thinking...)" text
// so a frozen braille glyph doesn't sit in scrollback.
func (p *promptDisplay) spinnerStop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw || p.midLine {
		return
	}
	fmt.Fprint(p.out, "\x1b7\x1b[A\r\x1b[K(thinking...)\x1b8")
}

// attachLoopTTY runs the raw-mode byte-by-byte input state machine. A 50ms
// ticker polls pending-HITL so a HITL event flips the prompt instantly,
// without needing the user to press Enter to "wake up" the loop.
func attachLoopTTY(c *CmdConfig, svc do.HostedAgentsService, sessionID string, f *os.File, state *attachState) error {
	fd := int(f.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Raw mode unavailable; fall back to bufio line mode.
		return attachLoop(c, svc, sessionID, f, state.pending)
	}
	defer term.Restore(fd, oldState)

	state.display.setRaw(true)
	defer state.display.setRaw(false)

	state.display.redraw()

	bytesCh := make(chan byte, 64)
	readErrCh := make(chan error, 1)
	go func() {
		var buf [1]byte
		for {
			n, err := f.Read(buf[:])
			if err != nil {
				readErrCh <- err
				return
			}
			if n == 1 {
				bytesCh <- buf[0]
			}
		}
	}()

	// 50ms ticker so HITL arrival reflows the prompt even when the user idles.
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	lastPending := state.pending.get()
	for {
		if cur := state.pending.get(); cur != lastPending {
			state.display.redraw()
			lastPending = cur
		}
		select {
		case b := <-bytesCh:
			stop, err := handleAttachByte(c, svc, sessionID, b, state)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		case <-ticker.C:
		case err := <-readErrCh:
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// handleAttachByte is the per-byte state machine. stop=true exits the loop
// (Ctrl-C, Ctrl-D on empty line).
func handleAttachByte(c *CmdConfig, svc do.HostedAgentsService, sessionID string, b byte, state *attachState) (stop bool, err error) {
	if id := state.pending.get(); id != "" {
		var outcome godo.HostedAgentHITLOutcome
		var matched bool
		switch b {
		case 'y', 'Y', 'a', 'A':
			outcome, matched = godo.HostedAgentHITLOutcomeApprove, true
		case 'n', 'N', 'r', 'R':
			outcome, matched = godo.HostedAgentHITLOutcomeReject, true
		case 'd', 'D':
			outcome, matched = godo.HostedAgentHITLOutcomeDefer, true
		case 0x03, 0x04: // Ctrl-C / Ctrl-D
			state.display.echo([]byte("\r\n"))
			return true, nil
		}
		if !matched {
			// Ignore non-HITL bytes while HITL pending.
			return false, nil
		}
		state.display.echo([]byte{b, '\r', '\n'})
		if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
			Outcome: outcome,
			Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
		}); err != nil {
			fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
		} else {
			state.pending.clearIf(id)
		}
		state.display.redraw()
		return false, nil
	}

	switch b {
	case 0x0d, 0x0a: // Enter
		state.mu.Lock()
		line := strings.TrimSpace(string(state.lineBuf))
		state.lineBuf = state.lineBuf[:0]
		state.mu.Unlock()
		state.display.echo([]byte("\r\n"))
		if line != "" {
			if detach := processAttachLine(c, svc, sessionID, line, state); detach {
				return true, nil
			}
		}
		state.display.redraw()
		return false, nil
	case 0x7f, 0x08: // Backspace / DEL
		state.mu.Lock()
		if len(state.lineBuf) > 0 {
			state.lineBuf = state.lineBuf[:len(state.lineBuf)-1]
			state.mu.Unlock()
			state.display.echo([]byte("\b \b"))
		} else {
			state.mu.Unlock()
		}
		return false, nil
	case 0x03: // Ctrl-C
		state.display.echo([]byte("\r\n"))
		return true, nil
	case 0x04: // Ctrl-D
		state.mu.Lock()
		empty := len(state.lineBuf) == 0
		state.mu.Unlock()
		if empty {
			state.display.echo([]byte("\r\n"))
			return true, nil
		}
		return false, nil
	default:
		// Printable ASCII only; escape sequences, UTF-8 multibyte, and arrow
		// keys are dropped (no history / cursor movement in V0).
		if b >= 0x20 && b < 0x7f {
			state.mu.Lock()
			state.lineBuf = append(state.lineBuf, b)
			state.mu.Unlock()
			state.display.echo([]byte{b})
		}
		return false, nil
	}
}

// processAttachLine dispatches an Enter-submitted line: HITL word shortcut,
// slash command, or SendInput. Returns detach=true when the session can no
// longer accept input (terminal run) and the loop should exit.
func processAttachLine(c *CmdConfig, svc do.HostedAgentsService, sessionID, line string, state *attachState) (detach bool) {
	if outcome, ok := hitlLetterShortcut(line); ok {
		if id := state.pending.get(); id != "" {
			if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
				Outcome: outcome,
				Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
			}); err != nil {
				fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
			} else {
				state.pending.clearIf(id)
			}
			return false
		}
	}
	if strings.HasPrefix(line, "/") {
		if err := handleAttachCommand(c, svc, sessionID, line, state.pending); err != nil {
			fmt.Fprintf(c.Out, "error: %v\n", err)
		}
		return false
	}
	resp, err := svc.SendInput(sessionID, &godo.HostedAgentSendInputRequest{Text: line})
	if err != nil {
		if isRunTerminalErr(err) {
			fmt.Fprintln(c.Out, "\nThis session's run has ended and can't accept new input.")
			fmt.Fprintln(c.Out, "Start a new session:  doctl agents start --spec <your-spec>.yaml")
			fmt.Fprintln(c.Out, "(detaching)")
			return true
		}
		fmt.Fprintf(c.Out, "send failed: %v\n", err)
		return false
	}
	if resp != nil && resp.RunID != "" {
		fmt.Fprintf(c.Out, "(queued as %s; waiting for the agent...)\n", resp.RunID)
	} else {
		fmt.Fprintln(c.Out, "(queued; waiting for the agent...)")
	}
	return false
}

// hitlLetterShortcut is the line-mode (piped / non-TTY) HITL path. Interactive
// terminals go through readHITLKeystroke instead. Returns (_, false) for any
// input that should fall through to slash-command / SendInput handling.
func hitlLetterShortcut(line string) (godo.HostedAgentHITLOutcome, bool) {
	switch strings.ToLower(line) {
	case "y", "yes", "a":
		return godo.HostedAgentHITLOutcomeApprove, true
	case "n", "no", "r":
		return godo.HostedAgentHITLOutcomeReject, true
	case "d", "defer":
		return godo.HostedAgentHITLOutcomeDefer, true
	}
	return "", false
}

type hitlKeyAction int

const (
	hitlKeyFallback hitlKeyAction = iota // not a TTY; caller should use bufio line mode
	hitlKeyResolve                       // y/n/d pressed; outcome is set
	hitlKeyDetach                        // Ctrl-C / Ctrl-D
	hitlKeyIgnore                        // any other key; byte was consumed
)

// readHITLKeystroke captures one y/n/d keystroke from an interactive terminal
// with no Enter. Returns hitlKeyFallback on non-TTY input so tests and pipes
// can use the bufio line-mode path.
func readHITLKeystroke(in io.Reader) (godo.HostedAgentHITLOutcome, byte, hitlKeyAction) {
	f, ok := in.(*os.File)
	if !ok {
		return "", 0, hitlKeyFallback
	}
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return "", 0, hitlKeyFallback
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", 0, hitlKeyFallback
	}
	defer term.Restore(fd, oldState)

	var buf [1]byte
	n, err := f.Read(buf[:])
	if err != nil || n != 1 {
		return "", 0, hitlKeyFallback
	}
	switch buf[0] {
	case 'y', 'Y', 'a', 'A':
		return godo.HostedAgentHITLOutcomeApprove, buf[0], hitlKeyResolve
	case 'n', 'N', 'r', 'R':
		return godo.HostedAgentHITLOutcomeReject, buf[0], hitlKeyResolve
	case 'd', 'D':
		return godo.HostedAgentHITLOutcomeDefer, buf[0], hitlKeyResolve
	case 0x03, 0x04: // Ctrl-C, Ctrl-D
		return "", buf[0], hitlKeyDetach
	}
	return "", buf[0], hitlKeyIgnore
}

// handleAttachCommand parses a slash command. `/a`, `/r`, `/d` accept either a
// bare request id or "implicit" (use the most recent pending one).
func handleAttachCommand(c *CmdConfig, svc do.HostedAgentsService, sessionID, line string, pending *pendingHITL) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}
	verb := parts[0]
	switch verb {
	case "/help":
		fmt.Fprintln(c.Out, "When a HITL approval is pending (single keystroke; no Enter needed in a TTY):")
		fmt.Fprintln(c.Out, "  y | a             approve the oldest pending request")
		fmt.Fprintln(c.Out, "  n | r             reject  the oldest pending request")
		fmt.Fprintln(c.Out, "  d                 defer   the oldest pending request")
		fmt.Fprintln(c.Out, "  (piped input: send `yes` / `no` / `defer` followed by a newline)")
		fmt.Fprintln(c.Out, "With an explicit request id (works on any queued request):")
		fmt.Fprintln(c.Out, "  /a [request-id]   approve (defaults to the oldest pending)")
		fmt.Fprintln(c.Out, "  /r [request-id]   reject")
		fmt.Fprintln(c.Out, "  /d [request-id]   defer")
		fmt.Fprintln(c.Out, "  /pending          list all HITL approvals waiting on you")
		return nil
	case "/pending":
		return listPendingHITLs(c, pending)
	case "/a", "/approve":
		return resolveFromAttach(svc, sessionID, parts, pending, godo.HostedAgentHITLOutcomeApprove)
	case "/r", "/reject":
		return resolveFromAttach(svc, sessionID, parts, pending, godo.HostedAgentHITLOutcomeReject)
	case "/d", "/defer":
		return resolveFromAttach(svc, sessionID, parts, pending, godo.HostedAgentHITLOutcomeDefer)
	default:
		return fmt.Errorf("unknown command %q (try /help)", verb)
	}
}

// listPendingHITLs renders the current HITL queue. The head is marked with
// "*" because that's what a single keystroke will resolve next.
func listPendingHITLs(c *CmdConfig, pending *pendingHITL) error {
	list := pending.list()
	if len(list) == 0 {
		fmt.Fprintln(c.Out, "(no HITL approvals pending)")
		return nil
	}
	fmt.Fprintf(c.Out, "%d HITL approval(s) pending (oldest first; * is next for single-keystroke y/n/d):\n", len(list))
	for i, e := range list {
		marker := " "
		if i == 0 {
			marker = "*"
		}
		if e.action != "" {
			fmt.Fprintf(c.Out, "  %s %s  (%s)\n", marker, e.id, e.action)
		} else {
			fmt.Fprintf(c.Out, "  %s %s\n", marker, e.id)
		}
	}
	return nil
}

func resolveFromAttach(svc do.HostedAgentsService, sessionID string, parts []string, pending *pendingHITL, outcome godo.HostedAgentHITLOutcome) error {
	id := ""
	if len(parts) >= 2 {
		id = parts[1]
	} else {
		id = pending.get()
	}
	if id == "" {
		return errors.New("no pending HITL request; provide a request id explicitly")
	}
	if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
		Outcome: outcome,
		Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
	}); err != nil {
		return err
	}
	// Optimistic local clear so the queue / prompt reflects the resolution
	// immediately, before the server's HITLResolved event comes back over SSE.
	pending.clearIf(id)
	return nil
}

// runSeparator visually divides agent output from the next user prompt.
const runSeparator = "────────────────────────────────────────"

// Payload shapes mirror the SPI data structs (spi/events.go) for the kinds
// doctl renders. godo leaves HostedAgentEvent.Payload as json.RawMessage.

type tokenChunkPayload struct {
	Text string `json:"text"`
}

type runStartedPayload struct {
	Agent string `json:"agent"`
}

type toolCallStartedPayload struct {
	Name string `json:"name"`
}

type toolCallCompletedPayload struct {
	OK         bool   `json:"ok"`
	DurationMS int64  `json:"duration_ms"`
	Summary    string `json:"summary,omitempty"`
}

type hitlRequestedPayload struct {
	HitlID  string         `json:"hitl_id"`
	Payload map[string]any `json:"payload"`
}

type hitlResolvedPayload struct {
	HitlID  string `json:"hitl_id"`
	Outcome int32  `json:"outcome"`
	Actor   string `json:"actor,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type runCompletedPayload struct {
	TotalTokensIn  int64 `json:"total_tokens_in"`
	TotalTokensOut int64 `json:"total_tokens_out"`
	RunCostMicros  int64 `json:"run_cost_micros"`
}

type runFailedPayload struct {
	Code    int32  `json:"code"`
	Message string `json:"message,omitempty"`
}

func renderEvent(w io.Writer, ev godo.HostedAgentEvent) {
	switch ev.Kind {
	case godo.HostedAgentEventKindTokenChunk:
		var p tokenChunkPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprint(w, p.Text)
		}
	case godo.HostedAgentEventKindRunStarted:
		var p runStartedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			if p.Agent != "" {
				fmt.Fprintf(w, "\n[run %s started (%s)]\n", ev.RunID, p.Agent)
			} else {
				fmt.Fprintf(w, "\n[run %s started]\n", ev.RunID)
			}
		}
	case godo.HostedAgentEventKindToolCallStarted:
		var p toolCallStartedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n> %s ...\n", p.Name)
		}
	case godo.HostedAgentEventKindToolCallCompleted:
		var p toolCallCompletedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			if p.Summary != "" {
				fmt.Fprintf(w, "  %s (%dms)\n", p.Summary, p.DurationMS)
			} else {
				fmt.Fprintf(w, "  ok (%dms)\n", p.DurationMS)
			}
		}
	case godo.HostedAgentEventKindHITLRequested:
		var p hitlRequestedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			renderHITLRequest(w, p.HitlID, p.Payload)
		}
	case godo.HostedAgentEventKindHITLResolved:
		var p hitlResolvedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n[HITL %s -> %s]\n", p.HitlID, hitlOutcomeLabel(p.Outcome))
		}
	case godo.HostedAgentEventKindRunCompleted:
		var p runCompletedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n[run done: %d in / %d out tokens, $%.4f]\n",
				p.TotalTokensIn, p.TotalTokensOut, float64(p.RunCostMicros)/1_000_000)
			fmt.Fprintln(w, runSeparator)
		}
	case godo.HostedAgentEventKindRunFailed:
		var p runFailedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			if p.Message != "" {
				fmt.Fprintf(w, "\n[run failed: %s (code %d)]\n", p.Message, p.Code)
			} else {
				fmt.Fprintf(w, "\n[run failed: code %d]\n", p.Code)
			}
			fmt.Fprintln(w, runSeparator)
		}
	case godo.HostedAgentEventKindSessionUpdated:
		fmt.Fprint(w, "\n[session updated]\n")
	}
}

// hitlOutcomeLabel maps the proto HITLOutcome int (on the SPI
// human_input_received event) to a human-readable verb.
func hitlOutcomeLabel(code int32) string {
	switch code {
	case 1:
		return "approve"
	case 2:
		return "reject"
	case 3:
		return "defer"
	default:
		return fmt.Sprintf("outcome %d", code)
	}
}

func renderHITLRequest(w io.Writer, hitlID string, payload map[string]any) {
	fmt.Fprintln(w, "\n\n[HITL] Action requires approval:")
	if len(payload) > 0 {
		if b, err := json.MarshalIndent(payload, "  ", "  "); err == nil {
			fmt.Fprintf(w, "  %s\n", b)
		}
	}
	fmt.Fprintf(w, "  hitl_id: %s\n", hitlID)
	fmt.Fprintln(w, "  Press y/n/d to approve, reject, or defer the oldest pending request (single keystroke; no Enter needed in a TTY).")
	fmt.Fprintf(w, "  Use `/a %s`, `/r %s`, `/d %s` to target this request explicitly; `/pending` lists all queued approvals.\n", hitlID, hitlID, hitlID)
}

// hitlActionLabel pulls the best human-readable label out of a HITLRequested
// payload map. The harness-api wire shape isn't strongly typed in our event
// payload (it's a generic JSON object), so we try a couple of plausible keys.
// Returns "" if no label is present so the caller can decide how to render.
func hitlActionLabel(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	for _, key := range []string{"action", "kind", "tool", "name"} {
		if v, ok := payload[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}
