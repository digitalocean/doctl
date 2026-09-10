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

package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// There is no listing or read endpoint on the sessions API: the workspace is
// reachable through the transfer API (upload/download, which needs an exact
// path and a local file) or through a sandbox exec. So `ls` and `cat` are exec
// calls, which keeps them a single round trip and inherits exec's 1 MiB per
// stream output cap — enough to browse a tree or read source, not to move a
// build artifact, which is what `download` is for.
//
// Worth keeping as commands of their own rather than folding back into `exec`,
// even though each is one call: browsing and reading are the two things every
// caller needs, and going through here means a script says what it wants rather
// than which guest binary produces it. Arbitrary exec is expected to narrow,
// and when it does, or when the API grows a real listing endpoint, these two
// change underneath while the commands people typed stay the same.

// AgentFiles generates the `doctl harness-runtime files` subtree for browsing
// and reading a session's workspace. It is a group rather than two flat verbs
// because `ls` is already how the flat tree spells `list` (sessions), and one
// name cannot mean both.
func AgentFiles() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "files",
			Aliases: []string{"file", "fs"},
			Short:   "Browse and read a session workspace",
			Long:    agentsFilesRootHelpMD,
		},
	}

	ns := agentSubNS("agents.files")

	cmdLs := CmdBuilder(cmd, RunAgentsFilesLs, "ls <session> [path]",
		"List files in a session workspace",
		agentsFilesLsHelpMD,
		Writer, append(ns, aliasOpt("list"),
			displayerType(&displayers.HostedAgentWorkspaceEntry{}))...)
	AddBoolFlag(cmdLs, doctl.ArgAgentLsRecursive, "R", false, "List nested entries, not just the immediate children")
	AddIntFlag(cmdLs, doctl.ArgAgentExecTimeout, "", 0, "Maximum seconds the listing may run (0 uses the server default)")
	cmdLs.Example = `doctl harness-runtime files ls sess_abc123; doctl harness-runtime files ls my-session src; doctl harness-runtime files ls my-session src --recursive`

	cmdCat := CmdBuilder(cmd, RunAgentsFilesCat, "cat <session> <path>",
		"Print a file from a session workspace",
		agentsFilesCatHelpMD,
		Writer, append(ns,
			displayerType(&displayers.HostedAgentWorkspaceFile{}))...)
	AddIntFlag(cmdCat, doctl.ArgAgentExecTimeout, "", 0, "Maximum seconds the read may run (0 uses the server default)")
	cmdCat.Example = `doctl harness-runtime files cat sess_abc123 src/main.go`

	requireAgentSubcommand(cmd)
	return cmd
}

// workspaceListRootTarget is what `ls` lists when no path is given. The exec
// API resolves a relative path against the workspace root, so "." is the root
// without doctl having to know where the guest mounted it.
const workspaceListRootTarget = "."

// workspaceListFormat is find's -printf template: type, apparent size, mtime as
// epoch seconds, then the path. Tab-separated with the path last, so a name
// containing spaces survives the split. find is used rather than `ls -l`
// because its output is a format doctl chooses rather than one it has to
// reverse-engineer per coreutils version.
const workspaceListFormat = "%y\t%s\t%T@\t%p\n"

// RunAgentsFilesLs lists the entries of one directory in the session's
// workspace. Unlike exec, the result is parsed and rendered rather than passed
// through: the paths in the Path column are the ones to hand to a follow-up
// `ls`, `cat`, `download`, or `exec --workdir`.
func RunAgentsFilesLs(c *CmdConfig) error {
	// args[0] is the session; args[1], when given, is the directory to list.
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if len(c.Args) > 2 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	target := workspaceListRootTarget
	if len(c.Args) == 2 && c.Args[1] != "" {
		target = c.Args[1]
		// find reads a leading dash as one of its own options, and it has no
		// `--` to turn that off, so name the directory relative to the cwd
		// instead. The prefix shows up in the listed paths, where it is still
		// a path the other commands accept.
		if strings.HasPrefix(target, "-") {
			target = "./" + target
		}
	}

	recursive, err := c.Doit.GetBool(c.NS, doctl.ArgAgentLsRecursive)
	if err != nil {
		return err
	}
	timeout, err := c.Doit.GetInt(c.NS, doctl.ArgAgentExecTimeout)
	if err != nil {
		return err
	}

	// find always reports the path it was given before anything below it. That
	// row is dropped later for a directory, whose contents were what was asked
	// for, and kept for a file, which `ls` likewise answers with the file
	// itself. Asking find to drop it here instead (-mindepth 1) would answer a
	// file with nothing at all.
	argv := []string{"find", target}
	if !recursive {
		argv = append(argv, "-maxdepth", "1")
	}
	argv = append(argv, "-printf", workspaceListFormat)

	resp, err := execForWorkspaceFiles(c, sessionID, argv, timeout)
	if err != nil {
		return err
	}
	// A listing is structured output, not a byte stream to reproduce, so a
	// failure is reported as a doctl error instead of exec's exit-code
	// passthrough.
	if resp.ExitCode != 0 {
		return fmt.Errorf("listing %s: %s", target, workspaceFilesFailure(resp))
	}

	entries, err := parseWorkspaceListing(resp.Stdout, target)
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentWorkspaceEntry{Entries: entries})
}

// RunAgentsFilesCat prints one file from the session's workspace. The bytes
// are reproduced verbatim and the guest's exit code becomes doctl's, the same
// contract as exec, so `cat` composes in a pipeline.
func RunAgentsFilesCat(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if len(c.Args) > 2 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	path := c.Args[1]

	timeout, err := c.Doit.GetInt(c.NS, doctl.ArgAgentExecTimeout)
	if err != nil {
		return err
	}

	// `--` so a path that starts with a dash is read as a path and not as one
	// of cat's own flags.
	resp, err := execForWorkspaceFiles(c, sessionID, []string{"cat", "--", path}, timeout)
	if err != nil {
		return err
	}

	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentWorkspaceFile{
			Path:      path,
			Content:   resp.Stdout,
			SizeBytes: len(resp.Stdout),
		}); err != nil {
			return err
		}
		return exitWithGuestStatus(resp.ExitCode)
	}

	// Verbatim, and with no added newline: the file's bytes are the output
	// contract, so anything appended here would corrupt a piped payload.
	if _, err := io.WriteString(c.Out, resp.Stdout); err != nil {
		return err
	}
	if _, err := io.WriteString(execStderr, resp.Stderr); err != nil {
		return err
	}
	return exitWithGuestStatus(resp.ExitCode)
}

// execForWorkspaceFiles runs one read-only command in the session's sandbox.
// Retries are left in place, unlike `exec`: listing and reading a file are
// idempotent, so replaying one costs a round trip rather than a side effect.
func execForWorkspaceFiles(c *CmdConfig, sessionID string, argv []string, timeout int) (*godo.HostedAgentSandboxExecResponse, error) {
	// SIGTERM alongside SIGINT so Ctrl-C or a plain `kill` stops the wait
	// instead of hanging until the server's own timeout.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Workdir is left empty so the sandbox resolves relative paths against the
	// workspace root, matching `--workspace-path` on upload and download.
	return c.HostedAgents().ExecInSandbox(ctx, sessionID, &godo.HostedAgentSandboxExecRequest{
		Argv:           argv,
		TimeoutSeconds: int64(timeout),
	})
}

// workspaceFilesFailure is the guest's explanation for a failed read, e.g.
// "find: '/workspace/nope': No such file or directory". stdout is the fallback
// for a guest that reported the problem on the wrong stream, and the exit code
// for one that said nothing at all.
func workspaceFilesFailure(resp *godo.HostedAgentSandboxExecResponse) string {
	if msg := strings.TrimSpace(resp.Stderr); msg != "" {
		return msg
	}
	if msg := strings.TrimSpace(resp.Stdout); msg != "" {
		return msg
	}
	return fmt.Sprintf("exit status %d", resp.ExitCode)
}

// parseWorkspaceListing turns find's -printf output into displayable entries,
// sorted by path since find walks in directory order. target is the path find
// was pointed at, which it echoes back as a row of its own; see below for why
// that row survives for a file and not for a directory.
func parseWorkspaceListing(stdout, target string) ([]displayers.HostedAgentWorkspaceEntryItem, error) {
	// Every record ends in a newline, so output that does not was cut off at
	// the exec cap — a recursive listing of a repo with its dependencies
	// vendored in reaches a megabyte easily. Returning the entries that did
	// arrive would be worse than failing: they look like a whole listing, and
	// a script comparing them against the workspace would act on the gap.
	if stdout != "" && !strings.HasSuffix(stdout, "\n") {
		return nil, errors.New("the listing was truncated at the sandbox's 1 MiB output cap: list a narrower path, or drop --recursive")
	}

	// find echoes "./x" for children of ".", which parsing trims; the target
	// has to lose the same prefix for the comparison below to line up.
	self := strings.TrimPrefix(target, "./")

	entries := make([]displayers.HostedAgentWorkspaceEntryItem, 0)
	for _, line := range strings.Split(stdout, "\n") {
		if line == "" {
			continue
		}
		entry, err := parseWorkspaceListingLine(line)
		if err != nil {
			return nil, err
		}
		// Listing a directory means listing what is in it, so the directory's
		// own row is noise. Listing a file means the file, which is the row
		// find already returned and the answer `ls` gives for the same
		// argument, so that one stays.
		if entry.Path == self && entry.Type == workspaceEntryTypeDir {
			continue
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// parseWorkspaceListingLine reads one workspaceListFormat record. The path is
// taken as the rest of the line rather than as a field, so a name containing a
// tab does not shift the columns.
func parseWorkspaceListingLine(line string) (displayers.HostedAgentWorkspaceEntryItem, error) {
	fields := strings.SplitN(line, "\t", 4)
	if len(fields) != 4 {
		return displayers.HostedAgentWorkspaceEntryItem{}, fmt.Errorf("unexpected listing line from the sandbox: %q", line)
	}
	size, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return displayers.HostedAgentWorkspaceEntryItem{}, fmt.Errorf("unexpected size %q in listing line %q", fields[1], line)
	}
	modified, err := parseWorkspaceListingTime(fields[2])
	if err != nil {
		return displayers.HostedAgentWorkspaceEntryItem{}, fmt.Errorf("unexpected modification time %q in listing line %q", fields[2], line)
	}
	return displayers.HostedAgentWorkspaceEntryItem{
		Type:       workspaceEntryType(fields[0]),
		SizeBytes:  size,
		ModifiedAt: modified,
		// find echoes the directory it was given back as a prefix, so listing
		// the root yields "./opencode.log". The "./" is noise in a column meant
		// to be read and copied, and the path means the same without it.
		Path: strings.TrimPrefix(fields[3], "./"),
	}, nil
}

// parseWorkspaceListingTime reads find's %T@, epoch seconds with a fractional
// part. The halves are parsed as integers rather than as one float because
// float64 cannot hold an epoch to nanosecond precision: parsing "…06.588" as a
// float and scaling it back up reports .588000059, digits the guest never sent.
func parseWorkspaceListingTime(field string) (time.Time, error) {
	secsText, fracText, _ := strings.Cut(field, ".")
	secs, err := strconv.ParseInt(secsText, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	// Right-pad to nanoseconds so ".5" is half a second, and drop any digits
	// finer than that rather than failing over precision doctl cannot display.
	const nanoDigits = 9
	if len(fracText) > nanoDigits {
		fracText = fracText[:nanoDigits]
	}
	var nanos int64
	if fracText != "" {
		if nanos, err = strconv.ParseInt(fracText+strings.Repeat("0", nanoDigits-len(fracText)), 10, 64); err != nil {
			return time.Time{}, err
		}
	}
	return time.Unix(secs, nanos).UTC(), nil
}

// workspaceEntryTypeDir labels a directory, and marks the row a listing drops
// when the directory is the one being listed.
const workspaceEntryTypeDir = "dir"

// workspaceEntryType maps find's single-letter %y code to a label worth
// reading in a table. An unknown code is passed through rather than dropped,
// so a listing never hides an entry doctl did not recognize.
func workspaceEntryType(code string) string {
	switch code {
	case "f":
		return "file"
	case "d":
		return workspaceEntryTypeDir
	case "l":
		return "symlink"
	case "b", "c":
		return "device"
	case "p":
		return "fifo"
	case "s":
		return "socket"
	default:
		return code
	}
}
