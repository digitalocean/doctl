package commands

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/agents"
	"github.com/spf13/cobra"
)

const defaultHarnessURL = "http://127.0.0.1:18080"

// Agents creates the top-level hosted agents command group.
func Agents() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "agents",
			Short: "Work with DigitalOcean hosted coding agents",
			Long:  "The `doctl agents` commands manage hosted coding agent sessions (local PoC uses harness-api).",
		},
	}

	deploy := CmdBuilder(cmd, RunAgentsDeploy, "deploy", "Deploy a hosted coding agent session", "", Writer)
	AddStringFlag(deploy, doctl.ArgHostedAgent, "", "claude-code", "Agent to run (claude-code, opencode)", requiredOpt())
	AddStringFlag(deploy, doctl.ArgHostedAgentRepo, "", "", "Repository: local path or github.com/owner/repo")
	AddStringFlag(deploy, doctl.ArgHarnessURL, "", defaultHarnessURL, "Harness API base URL")
	AddStringFlag(deploy, doctl.ArgAgentsGitHubToken, "", "", "GitHub PAT (or set GITHUB_TOKEN)")
	AddStringFlag(deploy, doctl.ArgAgentsGitHubOwner, "", "gane5hvarma", "GitHub owner for pull requests")
	AddStringFlag(deploy, doctl.ArgAgentsOpenRouterAPIKey, "", "", "OpenRouter API key for free models (or OPENROUTER_API_KEY)")
	AddStringFlag(deploy, doctl.ArgAgentsOpenCodeAPIKey, "", "", "OpenCode Zen/Go subscription key (or OPENCODE_API_KEY)")
	AddStringFlag(deploy, doctl.ArgAgentsOpenCodeModel, "", "", "OpenCode model provider/model (or OPENCODE_MODEL), e.g. openrouter/openrouter/free")
	AddStringFlag(deploy, doctl.ArgAgentsAnthropicAPIKey, "", "", "Anthropic API key (or ANTHROPIC_API_KEY)")

	cmd.AddCommand(agentsAuthCmd())

	chat := CmdBuilder(cmd, RunAgentsChat, "chat <session-id>", "Interactive chat with a session", "", Writer)
	AddStringFlag(chat, doctl.ArgHarnessURL, "", defaultHarnessURL, "Harness API base URL")

	return cmd
}

func agentsCredentialsFromConfig(c *CmdConfig) agents.CredentialRequest {
	ghToken, _ := c.Doit.GetString(c.NS, doctl.ArgAgentsGitHubToken)
	if ghToken == "" {
		ghToken = os.Getenv("GITHUB_TOKEN")
	}
	ghOwner, _ := c.Doit.GetString(c.NS, doctl.ArgAgentsGitHubOwner)
	orKey, _ := c.Doit.GetString(c.NS, doctl.ArgAgentsOpenRouterAPIKey)
	if orKey == "" {
		orKey = os.Getenv("OPENROUTER_API_KEY")
	}
	ocKey, _ := c.Doit.GetString(c.NS, doctl.ArgAgentsOpenCodeAPIKey)
	if ocKey == "" {
		ocKey = os.Getenv("OPENCODE_API_KEY")
	}
	anthropic, _ := c.Doit.GetString(c.NS, doctl.ArgAgentsAnthropicAPIKey)
	if anthropic == "" {
		anthropic = os.Getenv("ANTHROPIC_API_KEY")
	}
	ocModel, _ := c.Doit.GetString(c.NS, doctl.ArgAgentsOpenCodeModel)
	if ocModel == "" {
		ocModel = os.Getenv("OPENCODE_MODEL")
	}
	doToken, _ := c.Doit.GetString("", doctl.ArgAccessToken)
	if doToken == "" {
		doToken = os.Getenv("DIGITALOCEAN_ACCESS_TOKEN")
	}
	if doToken == "" {
		doToken = os.Getenv("ENG_GRADIENT_MODEL_ACCESS_KEY")
	}
	return agents.CredentialRequest{
		GitHubToken:             ghToken,
		GitHubOwner:             ghOwner,
		OpenCodeAPIKey:          ocKey,
		OpenRouterAPIKey:        orKey,
		OpenCodeModel:           ocModel,
		DigitalOceanAccessToken: doToken,
		AnthropicAPIKey:         anthropic,
	}
}

func deploySpinnerMessages(agentName string) []string {
	label := strings.TrimSpace(agentName)
	switch strings.ToLower(label) {
	case "opencode", "open-code":
		label = "OpenCode"
	case "claude-code", "claude_code", "claude":
		label = "Claude Code"
	default:
		if label == "" {
			label = "agent"
		}
	}
	return []string{
		"Allocating sandbox...",
		"Preparing workspace...",
		fmt.Sprintf("Starting %s...", label),
	}
}

func validateDeployCredentials(agentName string, creds agents.CredentialRequest) error {
	switch strings.ToLower(strings.TrimSpace(agentName)) {
	case "opencode", "open-code":
		if creds.OpenRouterAPIKey == "" && creds.OpenCodeAPIKey == "" && creds.AnthropicAPIKey == "" &&
			creds.DigitalOceanAccessToken == "" {
			return fmt.Errorf("opencode agent requires an LLM credential: --access-token (DO Model Access Key), --openrouter-api-key, --opencode-api-key, or --anthropic-api-key")
		}
	}
	return nil
}

func harnessClient(c *CmdConfig) *agents.Client {
	url := defaultHarnessURL
	if v := os.Getenv("HARNESS_URL"); v != "" {
		url = v
	}
	if v, err := c.Doit.GetString(c.NS, doctl.ArgHarnessURL); err == nil && v != "" {
		url = v
	}
	return agents.NewClient(url)
}

func RunAgentsDeploy(c *CmdConfig) error {
	agentName, err := c.Doit.GetString(c.NS, doctl.ArgHostedAgent)
	if err != nil {
		return err
	}
	kind, err := agents.AgentKindFromCLI(agentName)
	if err != nil {
		return err
	}
	repo, _ := c.Doit.GetString(c.NS, doctl.ArgHostedAgentRepo)
	client := harnessClient(c)
	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("harness-api unreachable: %w", err)
	}

	spin := deploySpinnerMessages(agentName)
	for _, msg := range spin {
		fmt.Fprintf(c.Out, "\r⠋ %s", msg)
		time.Sleep(350 * time.Millisecond)
	}

	start := time.Now()
	sess, err := client.CreateSession(ctx, kind, repo)
	if err != nil {
		return err
	}
	creds := agentsCredentialsFromConfig(c)
	if err := validateDeployCredentials(agentName, creds); err != nil {
		return err
	}
	if creds.HasAny() {
		if err := client.SetCredentials(ctx, sess.SessionID, creds); err != nil {
			return fmt.Errorf("set session credentials: %w", err)
		}
		fmt.Fprintf(c.Out, "\r⠋ Credentials registered for session\n")
	}
	readyTimeout := 30 * time.Second
	if strings.Contains(repo, "github.com") || (strings.Count(repo, "/") == 1 && repo != "") {
		readyTimeout = 3 * time.Minute
	}
	readyCtx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()
	_, err = client.WaitForReady(readyCtx, sess.SessionID, 200*time.Millisecond)
	if err != nil {
		return err
	}
	elapsed := time.Since(start).Seconds()
	fmt.Fprintf(c.Out, "\r✓ Session created: %s\n", sess.SessionID)
	fmt.Fprintf(c.Out, "  Status: ready (%.1fs)\n", elapsed)
	return nil
}

func agentsAuthCmd() *Command {
	auth := &Command{
		Command: &cobra.Command{
			Use:   "auth",
			Short: "Link external providers to a session",
		},
	}
	github := CmdBuilder(auth, RunAgentsAuthGitHub, "github", "Authorize GitHub for a session", "", Writer)
	AddStringFlag(github, doctl.ArgHostedAgentSession, "", "", "Session ID", requiredOpt())
	AddStringFlag(github, doctl.ArgHarnessURL, "", defaultHarnessURL, "Harness API base URL")
	return auth
}

func RunAgentsAuthGitHub(c *CmdConfig) error {
	sessionID, err := c.Doit.GetString(c.NS, doctl.ArgHostedAgentSession)
	if err != nil {
		return err
	}
	client := harnessClient(c)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Fprintln(c.Out, "⠋ Initiating GitHub OAuth flow...")
	resp, err := client.StartOAuthFlow(ctx, sessionID)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.Out, "✓ Opening browser to authorize DigitalOcean Agents...")
	fmt.Fprintf(c.Out, "\n  → %s\n", resp.AuthorizeURL)
	fmt.Fprintln(c.Out, "    (paste this URL if your browser didn't open)\n")

	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", resp.AuthorizeURL).Start()
	}

	fmt.Fprintln(c.Out, "⠙ Waiting for authorization...")
	authorized, err := waitGitHubAuth(ctx, client, sessionID)
	if err != nil {
		return err
	}
	if !authorized {
		return fmt.Errorf("timed out waiting for GitHub authorization")
	}
	login := githubDisplayLogin(ctx, client, sessionID)
	fmt.Fprintf(c.Out, "✓ GitHub authorized as %s\n", login)
	fmt.Fprintf(c.Out, "✓ Token stored server-side (scope: session %s)\n", sessionID)
	return nil
}

func waitGitHubAuth(ctx context.Context, client *agents.Client, sessionID string) (bool, error) {
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	resp, err := client.StreamSession(streamCtx, sessionID, "", false)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	done := make(chan bool, 1)
	go func() {
		dec := agents.NewStreamDecoder(resp.Body)
		for {
			ev, err := dec.Next()
			if err == io.EOF {
				done <- false
				return
			}
			if err != nil {
				done <- false
				return
			}
			if ev.SessionUpdated != nil {
				if ev.SessionUpdated.SessionDelta.ProviderAuth != nil {
					if ev.SessionUpdated.SessionDelta.ProviderAuth["github"] == agents.ProviderAuthAuthorized {
						done <- true
						return
					}
				}
			}
		}
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case ok := <-done:
			return ok, nil
		case <-ticker.C:
			sess, err := client.GetSession(ctx, sessionID)
			if err == nil && sess.ProviderAuth != nil && sess.ProviderAuth["github"] == agents.ProviderAuthAuthorized {
				return true, nil
			}
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
}

func RunAgentsChat(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return fmt.Errorf("session id is required")
	}
	sessionID := c.Args[0]
	client := harnessClient(c)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess, err := client.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	agentLabel := strings.TrimPrefix(sess.AgentKind, "AGENT_KIND_")
	agentLabel = strings.ReplaceAll(strings.ToLower(agentLabel), "_", "-")

	fmt.Fprintf(c.Out, "Connected to %s (%s)\n", sessionID, agentLabel)
	printGitHubBanner(c.Out, sess, githubDisplayLogin(ctx, client, sessionID))

	var (
		mu          sync.Mutex
		pendingHITL *agents.HITLRequest
	)

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	resp, err := client.StreamSession(streamCtx, sessionID, "", false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	go func() {
		dec := agents.NewStreamDecoder(resp.Body)
		for {
			ev, err := dec.Next()
			if err == io.EOF {
				return
			}
			if err != nil {
				fmt.Fprintf(c.Out, "\nstream error: %v\n", err)
				return
			}
			renderEvent(c.Out, ev, &mu, &pendingHITL)
		}
	}()

	sc := bufio.NewScanner(os.Stdin)
	fmt.Fprint(c.Out, "\nType a message or '/help' for commands. Ctrl+D to exit.\n\n> ")
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			fmt.Fprint(c.Out, "> ")
			continue
		}
		switch line {
		case "a", "r", "d":
			mu.Lock()
			req := pendingHITL
			mu.Unlock()
			if req == nil {
				fmt.Fprintln(c.Out, "No pending HITL request.")
				fmt.Fprint(c.Out, "> ")
				continue
			}
			outcome := agents.HITLOutcomeApprove
			if line == "r" {
				outcome = agents.HITLOutcomeReject
			} else if line == "d" {
				outcome = agents.HITLOutcomeDefer
			}
			if err := client.ResolveHITL(ctx, sessionID, req.RequestID, outcome); err != nil {
				fmt.Fprintf(c.Out, "HITL error: %v\n", err)
			}
			mu.Lock()
			pendingHITL = nil
			mu.Unlock()
			fmt.Fprint(c.Out, "> ")
			continue
		case "/help":
			fmt.Fprintln(c.Out, "Commands: type a message to send; a/r/d approve/reject/defer HITL; Ctrl+D to exit.")
			fmt.Fprint(c.Out, "> ")
			continue
		}
		if _, err := client.SendInput(ctx, sessionID, line); err != nil {
			fmt.Fprintf(c.Out, "send error: %v\n", err)
		}
		fmt.Fprint(c.Out, "> ")
	}
	fmt.Fprintln(c.Out, "\nDetached.")
	return nil
}

func githubDisplayLogin(ctx context.Context, client *agents.Client, sessionID string) string {
	meta, err := client.GetSessionMeta(ctx, sessionID)
	if err == nil && meta.GitHubLogin != "" {
		return meta.GitHubLogin
	}
	if err == nil && meta.GitHubOwner != "" {
		return meta.GitHubOwner
	}
	return ""
}

func printGitHubBanner(out io.Writer, sess *agents.Session, githubLogin string) {
	if sess.ProviderAuth == nil {
		return
	}
	if sess.ProviderAuth["github"] != agents.ProviderAuthAuthorized {
		return
	}
	if githubLogin != "" {
		fmt.Fprintf(out, "GitHub: authorized as %s\n", githubLogin)
		return
	}
	fmt.Fprintln(out, "GitHub: token configured")
}

func renderEvent(out io.Writer, ev *agents.Event, mu *sync.Mutex, pending **agents.HITLRequest) {
	switch {
	case ev.RunStarted != nil:
		if preview := strings.TrimSpace(ev.RunStarted.UserInputPreview); preview != "" {
			fmt.Fprintf(out, "\n--- run started ---\n> %s\n", preview)
		} else {
			fmt.Fprintln(out, "\n--- run started ---")
		}
		flushWriter(out)
	case ev.TokenChunk != nil:
		fmt.Fprint(out, ev.TokenChunk.Text)
		flushWriter(out)
	case ev.ToolCallStarted != nil:
		fmt.Fprintf(out, "\n⏵ %s ...\n", ev.ToolCallStarted.ToolName)
		flushWriter(out)
	case ev.ToolCallCompleted != nil:
		if s := strings.TrimSpace(ev.ToolCallCompleted.Summary); s != "" {
			fmt.Fprintf(out, "✓ %s (%dms)\n", s, ev.ToolCallCompleted.DurationMs.Int64())
		} else {
			fmt.Fprintf(out, "✓ tool done (%dms)\n", ev.ToolCallCompleted.DurationMs.Int64())
		}
		flushWriter(out)
	case ev.RunCompleted != nil:
		fmt.Fprintln(out, "\n--- run completed ---")
		flushWriter(out)
	case ev.HITLRequested != nil && ev.HITLRequested.Request.RequestID != "":
		req := ev.HITLRequested.Request
		mu.Lock()
		*pending = &req
		mu.Unlock()
		fmt.Fprintf(out, "\n[HITL] Action requires approval:\n")
		printHITLBlock(out, &req)
		fmt.Fprint(out, "  [a]pprove  [r]eject  [d]efer\n\n> ")
	case ev.SessionUpdated != nil:
		// GitHub auth banner is shown at connect; skip replayed SessionUpdated noise.
	case ev.RunFailed != nil:
		fmt.Fprintf(out, "\nRun failed: %s\n", ev.RunFailed.Message)
		flushWriter(out)
	}
}

func flushWriter(out io.Writer) {
	if f, ok := out.(interface{ Flush() error }); ok {
		_ = f.Flush()
	}
}

func printHITLBlock(out io.Writer, req *agents.HITLRequest) {
	switch req.Action {
	case "HITL_ACTION_BASH":
		if cmd, ok := req.Details["command"].(string); ok {
			fmt.Fprintf(out, "  bash: %s\n", cmd)
		}
	case "HITL_ACTION_GITHUB_CREATE_PR":
		fmt.Fprintf(out, "  github.create_pr\n")
		if t, ok := req.Details["title"].(string); ok {
			fmt.Fprintf(out, "    title:   %q\n", t)
		}
		if b, ok := req.Details["branch"].(string); ok {
			fmt.Fprintf(out, "    branch:  %s → main\n", b)
		}
		if r, ok := req.Details["repo"].(string); ok {
			fmt.Fprintf(out, "    repo:    %s\n", r)
		}
	default:
		fmt.Fprintf(out, "  action: %s\n", req.Action)
	}
	if req.Workdir != "" {
		fmt.Fprintf(out, "  Workdir: %s\n", req.Workdir)
	}
}
