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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	yaml "gopkg.in/yaml.v2"
)

func TestResolveHarnessAgent(t *testing.T) {
	t.Run("opencode aliases", func(t *testing.T) {
		for _, in := range []string{"opencode", "open-code", "OPENCODE"} {
			got, err := resolveHarnessAgent(in)
			require.NoError(t, err)
			assert.Equal(t, "opencode", got)
		}
	})

	// codex is the Codex CLI supervised by DigitalOcean; codex-agentapi is
	// OpenAI's sandbox-provider adapter. Mapping codex onto codex-agentapi
	// silently hands the caller the wrong one, so pin both directions.
	t.Run("codex maps to the Codex CLI", func(t *testing.T) {
		got, err := resolveHarnessAgent("codex")
		require.NoError(t, err)
		assert.Equal(t, codexAgentName, got)
		assert.NotEqual(t, openAIAgentsAdapter, got)
	})

	t.Run("codex-agentapi aliases", func(t *testing.T) {
		for _, in := range []string{"codex-agentapi", "openai-codex", "CODEX-AGENTAPI"} {
			got, err := resolveHarnessAgent(in)
			require.NoError(t, err)
			assert.Equal(t, openAIAgentsAdapter, got)
		}
	})

	t.Run("unknown harness", func(t *testing.T) {
		_, err := resolveHarnessAgent("not-a-harness")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
		assert.Contains(t, err.Error(), "codex-agentapi")
	})
}

func TestBuildHarnessManifest(t *testing.T) {
	t.Run("opencode with repo and prompt", func(t *testing.T) {
		raw, err := buildHarnessManifest(harnessManifestOpts{
			harness: "opencode",
			repo:    "https://github.com/katanemo/plano",
			prompt:  "Review README",
			name:    "demo",
		})
		require.NoError(t, err)

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(raw, &doc))
		assert.Equal(t, "opencode", doc["agent"])
		assert.Equal(t, "demo", doc["name"])
		repos, ok := doc["repos"].([]any)
		require.True(t, ok)
		require.Len(t, repos, 1)
		assert.Equal(t, "katanemo/plano", repos[0])
		assert.NotContains(t, doc, "config")
	})

	t.Run("owner/repo shorthand", func(t *testing.T) {
		raw, err := buildHarnessManifest(harnessManifestOpts{harness: "opencode", repo: "katanemo/plano"})
		require.NoError(t, err)
		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(raw, &doc))
		repos, ok := doc["repos"].([]any)
		require.True(t, ok)
		assert.Equal(t, "katanemo/plano", repos[0])
	})

	t.Run("codex builds the Codex CLI manifest", func(t *testing.T) {
		raw, err := buildHarnessManifest(harnessManifestOpts{harness: "codex", prompt: "hello world"})
		require.NoError(t, err)

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(raw, &doc))
		assert.Equal(t, codexAgentName, doc["agent"])
		secrets, ok := doc["secrets"].(map[any]any)
		require.True(t, ok)
		assert.Equal(t, "${OPENAI_API_KEY}", secrets["OPENAI_API_KEY"])

		// config is codex-agentapi's OpenAI create-session body and the
		// CODEX_* env is how that adapter finds the OpenAI-side environment.
		// Neither belongs here, and the prompt goes out post-ready via
		// sendInitialPrompt rather than riding in config.input.
		assert.NotContains(t, doc, "config")
		assert.NotContains(t, doc, "env")
		assert.NotContains(t, string(raw), "hello world")
	})

	t.Run("codex-agentapi includes openai config and env", func(t *testing.T) {
		raw, err := buildHarnessManifest(harnessManifestOpts{harness: "codex-agentapi", prompt: "hello world"})
		require.NoError(t, err)

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(raw, &doc))
		assert.Equal(t, openAIAgentsAdapter, doc["agent"])
		env, ok := doc["env"].(map[any]any)
		require.True(t, ok)
		assert.Equal(t, "${ENV_ID}", env["CODEX_ENVIRONMENT_ID"])
		assert.Equal(t, "${REMOTE_URL}", env["CODEX_REMOTE_URL"])
		_, hasKeyInEnv := env["CODEX_API_KEY"]
		assert.False(t, hasKeyInEnv)
		secrets, ok := doc["secrets"].(map[any]any)
		require.True(t, ok)
		assert.Equal(t, "${OPENAI_API_KEY}", secrets["CODEX_API_KEY"])

		cfg, ok := doc["config"].(map[any]any)
		require.True(t, ok)
		input, ok := cfg["input"].([]any)
		require.True(t, ok)
		require.Len(t, input, 1)
	})

	t.Run("claude-code references ANTHROPIC_API_KEY", func(t *testing.T) {
		raw, err := buildHarnessManifest(harnessManifestOpts{harness: "claude-code"})
		require.NoError(t, err)

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(raw, &doc))
		assert.Equal(t, "claude-code", doc["agent"])
		_, hasEnv := doc["env"]
		assert.False(t, hasEnv)
		secrets, ok := doc["secrets"].(map[any]any)
		require.True(t, ok)
		assert.Equal(t, "${ANTHROPIC_API_KEY}", secrets["ANTHROPIC_API_KEY"])
		assert.NotContains(t, doc, "config")
	})

	t.Run("opencode has no injected key requirement", func(t *testing.T) {
		raw, err := buildHarnessManifest(harnessManifestOpts{harness: "opencode"})
		require.NoError(t, err)

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(raw, &doc))
		assert.NotContains(t, doc, "env")
	})
}

func TestManifestIncludesPrompt(t *testing.T) {
	assert.False(t, manifestIncludesPrompt([]byte("agent: opencode\n"), "hello"))
	assert.True(t, manifestIncludesPrompt([]byte("text: hello\n"), "hello"))
}

func TestWaitForSessionReady(t *testing.T) {
	prev := sessionReadyPollInterval
	sessionReadyPollInterval = time.Millisecond
	t.Cleanup(func() { sessionReadyPollInterval = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			GetSession("sess_1").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_1",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		var out bytes.Buffer
		prog := newCreationProgress(&out)
		sess, err := waitForSessionReady(context.Background(), config.HostedAgents(), "sess_1", prog)
		require.NoError(t, err)
		require.NotNil(t, sess)
		got := out.String()
		assert.Contains(t, got, "Agent is ready")
		assert.NotContains(t, got, "SESSION_STATUS_")
		assert.NotContains(t, got, "session_id=")
	})
}

func TestWaitForSessionReady_ProvisioningHints(t *testing.T) {
	prevPoll := sessionReadyPollInterval
	prevHint := creationHintInterval
	prevClock := creationClock
	sessionReadyPollInterval = time.Millisecond
	creationHintInterval = time.Millisecond
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	creationClock = func() time.Time { return now }
	t.Cleanup(func() {
		sessionReadyPollInterval = prevPoll
		creationHintInterval = prevHint
		creationClock = prevClock
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		calls := 0
		tm.hostedAgents.EXPECT().
			GetSession("sess_hint").
			DoAndReturn(func(id string) (*do.HostedAgentSession, error) {
				calls++
				now = now.Add(2 * time.Millisecond)
				status := godo.HostedAgentSessionStatusProvisioning
				if calls >= 3 {
					status = godo.HostedAgentSessionStatusReady
				}
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_hint",
						Status:    status,
					},
				}, nil
			}).AnyTimes()

		var out bytes.Buffer
		prog := newCreationProgress(&out)
		sess, err := waitForSessionReady(context.Background(), config.HostedAgents(), "sess_hint", prog)
		require.NoError(t, err)
		require.NotNil(t, sess)
		got := out.String()
		assert.Contains(t, got, "Waiting for agent")
		assert.Contains(t, got, "Agent is ready")
		assert.NotContains(t, got, "SESSION_STATUS_")
	})
}

// A bare sandbox runs no agent, so every line of the wait must stop naming one.
// "Starting agent runtime…" is the one that matters most: coding-base ships no
// runtime to start, so on a slow provision it reads as a stalled step rather
// than as a stage that does not apply here. The noun comes off the session's
// AgentKind, not off this invocation's flags, so it is also right when the
// caller never saw the create — `port-forward` into someone else's sandbox.
func TestWaitForSessionReady_BareSandboxSaysSandboxNotAgent(t *testing.T) {
	prevPoll := sessionReadyPollInterval
	prevHint := creationHintInterval
	prevClock := creationClock
	sessionReadyPollInterval = time.Millisecond
	creationHintInterval = time.Millisecond
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	creationClock = func() time.Time { return now }
	t.Cleanup(func() {
		sessionReadyPollInterval = prevPoll
		creationHintInterval = prevHint
		creationClock = prevClock
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		calls := 0
		tm.hostedAgents.EXPECT().
			GetSession("sess_bare").
			DoAndReturn(func(id string) (*do.HostedAgentSession, error) {
				calls++
				now = now.Add(2 * time.Millisecond)
				status := godo.HostedAgentSessionStatusProvisioning
				if calls >= 4 {
					status = godo.HostedAgentSessionStatusReady
				}
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_bare",
						AgentKind: godo.HostedAgentKindNone,
						Status:    status,
					},
				}, nil
			}).AnyTimes()

		var out bytes.Buffer
		prog := newCreationProgress(&out)
		sess, err := waitForSessionReady(context.Background(), config.HostedAgents(), "sess_bare", prog)
		require.NoError(t, err)
		require.NotNil(t, sess)
		got := out.String()
		assert.Contains(t, got, "Sandbox is ready")
		assert.NotContains(t, strings.ToLower(got), "agent")
	})
}

func TestWaitForSessionReady_BitsReadyWhileProvisioning(t *testing.T) {
	prevPoll := sessionReadyPollInterval
	prevHint := creationHintInterval
	prevClock := creationClock
	sessionReadyPollInterval = time.Millisecond
	creationHintInterval = time.Millisecond
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	creationClock = func() time.Time { return now }
	t.Cleanup(func() {
		sessionReadyPollInterval = prevPoll
		creationHintInterval = prevHint
		creationClock = prevClock
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		calls := 0
		tm.hostedAgents.EXPECT().
			GetSession("sess_bits").
			DoAndReturn(func(id string) (*do.HostedAgentSession, error) {
				calls++
				now = now.Add(2 * time.Millisecond)
				sess := &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_bits",
						Status:    godo.HostedAgentSessionStatusProvisioning,
					},
				}
				if calls >= 2 {
					sess.OpenAIEnvironmentID = "env_abc"
				}
				if calls >= 4 {
					sess.Status = godo.HostedAgentSessionStatusReady
				}
				return sess, nil
			}).AnyTimes()

		var out bytes.Buffer
		prog := newCreationProgress(&out)
		sess, err := waitForSessionReady(context.Background(), config.HostedAgents(), "sess_bits", prog)
		require.NoError(t, err)
		require.NotNil(t, sess)
		got := out.String()
		assert.Contains(t, got, "Environment ready · waiting for agent")
		assert.Contains(t, got, "Agent is ready")
		assert.NotContains(t, got, "openai_environment_id=")
	})
}

func TestRunAgentsCreate_Harness(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			DoAndReturn(func(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				assert.Contains(t, string(manifest), "agent: opencode")
				assert.Contains(t, string(manifest), "repos:")
				assert.Contains(t, string(manifest), "katanemo/plano")
				assert.Contains(t, string(manifest), "name: demo")
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_run_1",
						Name:      "demo",
						Status:    godo.HostedAgentSessionStatusProvisioning,
					},
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetSession("sess_run_1").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_run_1",
					Name:      "demo",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentRepo, "https://github.com/katanemo/plano")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerPrompt, "Review README")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")

		tm.hostedAgents.EXPECT().
			StartProviderAuth("github").
			Return(&godo.HostedAgentProviderAuthStart{Provider: "github", Status: "success"}, nil)

		// buildHarnessManifest embeds the prompt only for Codex, so an
		// opencode session has to be told once it is ready.
		tm.hostedAgents.EXPECT().
			SendInput("sess_run_1", &godo.HostedAgentSendInputRequest{Text: "Review README"}).
			Return(nil, nil)

		require.NoError(t, RunAgentsCreate(config))
		out := buf.String()
		assert.Contains(t, out, "Creating agent session")
		assert.Contains(t, out, "GitHub already connected")
		assert.Contains(t, out, "Validating configuration")
		assert.Contains(t, out, "Creating hosted session")
		assert.Contains(t, out, "Session created")
		assert.Contains(t, out, "Agent is ready")
		assert.NotContains(t, out, "SESSION_STATUS_")
		assert.Contains(t, out, "doctl harness-runtime launch demo")
		assert.NotContains(t, out, "katanemo/plano", "repo is hidden by default; only shown with -v/--verbose")
	})
}

// TestRunAgentsCreate_ClaudeCodeMissingAnthropicKey ensures a missing inference
// key is caught locally (no CreateSessionFromManifest/GetSession
// expectations set below — gomock fails the test if either is called) rather
// than letting the harness fail once it's already hosted.
func TestRunAgentsCreate_ClaudeCodeMissingAnthropicKey(t *testing.T) {
	_ = os.Unsetenv(anthropicAPIKeyEnv)
	prevInteractive := Interactive
	Interactive = false
	t.Cleanup(func() {
		Interactive = prevInteractive
		_ = os.Unsetenv(anthropicAPIKeyEnv)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "claude-code")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo-claude")

		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), anthropicAPIKeyEnv)
	})
}

// --secret ANTHROPIC_API_KEY=… must satisfy claude-code without the process
// environment. Design partners hit this with NAME=-: --dry-run passed (no
// key check) and the real create then demanded the env var.
func TestRunAgentsCreate_ClaudeCodeSecretSatisfiesKey(t *testing.T) {
	_ = os.Unsetenv(anthropicAPIKeyEnv)
	prevInteractive := Interactive
	Interactive = false
	t.Cleanup(func() {
		Interactive = prevInteractive
		_ = os.Unsetenv(anthropicAPIKeyEnv)
	})

	orig := validateAnthropicAPIKey
	t.Cleanup(func() { validateAnthropicAPIKey = orig })
	var gotKey string
	validateAnthropicAPIKey = func(ctx context.Context, apiKey string) error {
		gotKey = apiKey
		return nil
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			DoAndReturn(func(manifest []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				assert.Contains(t, string(manifest), "sk-ant-from-secret")
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_secret_key",
						Name:      "demo-claude",
						Status:    godo.HostedAgentSessionStatusProvisioning,
					},
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetSession("sess_secret_key").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_secret_key",
					Name:      "demo-claude",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "claude-code")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo-claude")
		config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"ANTHROPIC_API_KEY=sk-ant-from-secret"})

		require.NoError(t, RunAgentsCreate(config))
		assert.Equal(t, "sk-ant-from-secret", gotKey)
	})
}

// secrets: OPENAI_API_KEY: ${OPENAI_API_KEY} must expand from the environment
// on the real create path (not only when --secret supplies a concrete value).
func TestRunAgentsCreate_SecretPlaceholderExpandsFromEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-from-env")
	prevInteractive := Interactive
	Interactive = false
	t.Cleanup(func() { Interactive = prevInteractive })

	spec := t.TempDir() + "/agents.yaml"
	require.NoError(t, os.WriteFile(spec, []byte(`name: syed-verify
agent: opencode
secrets:
  OPENAI_API_KEY: ${OPENAI_API_KEY}
`), 0o600))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			DoAndReturn(func(manifest []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				assert.Contains(t, string(manifest), "sk-from-env")
				assert.NotContains(t, string(manifest), "${OPENAI_API_KEY}")
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_syed",
						Name:      "syed-verify",
						Status:    godo.HostedAgentSessionStatusProvisioning,
					},
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetSession("sess_syed").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_syed",
					Name:      "syed-verify",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, spec)
		require.NoError(t, RunAgentsCreate(config))
	})
}

// TestPrepareClaudeCodeStart_ValidatesKey confirms the key resolved via
// ensureEnvVar is the one handed to validateAnthropicAPIKey, and that
// non-claude-code manifests skip validation entirely (no network call).
func TestPrepareClaudeCodeStart_ValidatesKey(t *testing.T) {
	t.Setenv(anthropicAPIKeyEnv, "sk-ant-test")
	orig := validateAnthropicAPIKey
	t.Cleanup(func() { validateAnthropicAPIKey = orig })

	var gotKey string
	calls := 0
	validateAnthropicAPIKey = func(ctx context.Context, apiKey string) error {
		calls++
		gotKey = apiKey
		return nil
	}

	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "claude-code"})
	require.NoError(t, err)
	require.NoError(t, prepareClaudeCodeStart(context.Background(), raw, nil))
	assert.Equal(t, 1, calls)
	assert.Equal(t, "sk-ant-test", gotKey)

	// A non-claude-code manifest must not trigger validation.
	raw, err = buildHarnessManifest(harnessManifestOpts{harness: "opencode"})
	require.NoError(t, err)
	require.NoError(t, prepareClaudeCodeStart(context.Background(), raw, nil))
	assert.Equal(t, 1, calls, "opencode manifest should not call validateAnthropicAPIKey")
}

// TestRunAgentsCreate_ClaudeCodeBogusAnthropicKey ensures a present-but-invalid
// ANTHROPIC_API_KEY is caught locally too — not just a missing one. No
// CreateSessionFromManifest expectation is set below, so gomock fails the
// test if the bogus key is not rejected before that call.
func TestRunAgentsCreate_ClaudeCodeBogusAnthropicKey(t *testing.T) {
	t.Setenv(anthropicAPIKeyEnv, "sk-ant-bogus")
	orig := validateAnthropicAPIKey
	t.Cleanup(func() { validateAnthropicAPIKey = orig })
	validateAnthropicAPIKey = func(ctx context.Context, apiKey string) error {
		return fmt.Errorf("%s was rejected by Anthropic (HTTP 401): invalid x-api-key", anthropicAPIKeyEnv)
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "claude-code")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo-claude")

		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rejected by Anthropic")
	})
}

func TestRunAgentsCreate_HarnessVerboseShowsRepoAndID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			DoAndReturn(func(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_run_2",
						Name:      "demo-verbose",
						Status:    godo.HostedAgentSessionStatusProvisioning,
					},
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetSession("sess_run_2").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_run_2",
					Name:      "demo-verbose",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		prevVerbose := Verbose
		Verbose = true
		defer func() { Verbose = prevVerbose }()

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentRepo, "https://github.com/katanemo/plano")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo-verbose")

		tm.hostedAgents.EXPECT().
			StartProviderAuth("github").
			Return(&godo.HostedAgentProviderAuthStart{Provider: "github", Status: "success"}, nil)

		require.NoError(t, RunAgentsCreate(config))
		out := buf.String()
		assert.Contains(t, out, "katanemo/plano")
		assert.Contains(t, out, "sess_run_2")
	})
}

func TestPrintAttachBanner(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var out bytes.Buffer
	printAttachBanner(&out, &do.HostedAgentSession{
		HostedAgentSession: &godo.HostedAgentSession{
			SessionID: "sess_attach_1",
			Name:      "smoke-test",
			AgentKind: godo.HostedAgentKindOpenCode,
		},
	}, "")

	assert.Equal(t, "\n● Connected  OpenCode · smoke-test\n"+
		"  Enter send · Option/Alt + Enter newline · Ctrl + D detach · /help for more\n"+
		"  y/a approve · n/r reject · d defer\n",
		out.String())
}

// TestPrintAttachBannerStaysCompact pins the banner's height. It used to be a
// 15-line bordered card; the whole point of the compact header is that
// attaching costs a few lines, so a regression that reintroduces a multi-line
// block should fail here rather than in review.
func TestPrintAttachBannerStaysCompact(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var out bytes.Buffer
	printAttachBanner(&out, &do.HostedAgentSession{
		HostedAgentSession: &godo.HostedAgentSession{
			SessionID: "sess_attach_1",
			Name:      "smoke-test",
			AgentKind: godo.HostedAgentKindOpenCode,
		},
	}, "")

	body := strings.TrimPrefix(out.String(), "\n")
	assert.Len(t, strings.Split(strings.TrimRight(body, "\n"), "\n"), 3)
	assert.NotContains(t, out.String(), "╭", "the attach banner must not be a bordered card")
}

// The approval keys are the reason the banner is worth more than one line, so
// pin that all three outcomes are offered and that they carry their colors.
func TestPrintAttachBannerShowsColoredApprovalKeys(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = true
	t.Cleanup(func() { stylingEnabled = prev })

	var out bytes.Buffer
	printAttachBanner(&out, &do.HostedAgentSession{
		HostedAgentSession: &godo.HostedAgentSession{
			SessionID: "sess_attach_1",
			Name:      "smoke-test",
			AgentKind: godo.HostedAgentKindOpenCode,
		},
	}, "")

	got := out.String()
	for _, want := range []string{"y/a", "approve", "n/r", "reject", "d", "defer"} {
		assert.Contains(t, got, want)
	}
	assert.Contains(t, got, colorize("y/a", colSuccess), "approve reads green")
	assert.Contains(t, got, colorize("n/r", colError), "reject reads red")
	assert.Contains(t, got, colorize("d", colWarning), "defer reads amber")
}

// TestPrintAttachBannerBridgeNote pins that a bridge note earns its own
// indented line instead of being dropped with the old card.
func TestPrintAttachBannerBridgeNote(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var out bytes.Buffer
	printAttachBanner(&out, &do.HostedAgentSession{
		HostedAgentSession: &godo.HostedAgentSession{
			SessionID: "sess_attach_1",
			Name:      "smoke-test",
			AgentKind: godo.HostedAgentKindOpenAICodex,
		},
	}, "OpenAI · conv_123")

	got := out.String()
	assert.Contains(t, got, "● Connected  Codex · smoke-test\n")
	assert.Contains(t, got, "  Bridge OpenAI · conv_123\n")
}

func TestPrintDetachNotice(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var buf bytes.Buffer
	printDetachNotice(&buf, "my-session")
	out := buf.String()
	assert.Contains(t, out, "Disconnected from session locally")
	assert.Contains(t, out, "still active in the cloud")
}

func TestMaybeOfferGitHubAuth_AlreadyConnected(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			StartProviderAuth("github").
			Return(&godo.HostedAgentProviderAuthStart{Provider: "github", Status: "success"}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, maybeOfferGitHubAuth(config))
		assert.Contains(t, buf.String(), "GitHub already connected")
	})
}

func TestMaybeOfferGitHubAuth_SkipWhenDeclined(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			StartProviderAuth("github").
			Return(&godo.HostedAgentProviderAuthStart{
				Provider:   "github",
				Status:     "pending",
				ConnectURL: "https://example.com/connect",
				PollURL:    "https://example.com/poll",
			}, nil)

		prevAsk := askConnectGitHub
		t.Cleanup(func() { askConnectGitHub = prevAsk })
		askConnectGitHub = func() (bool, error) { return false, nil }

		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, maybeOfferGitHubAuth(config))
		assert.Contains(t, buf.String(), "Skipping GitHub connect")
	})
}

func TestMaybeOfferGitHubAuth_ConnectsWhenAccepted(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			StartProviderAuth("github").
			Return(&godo.HostedAgentProviderAuthStart{
				Provider:   "github",
				Status:     "pending",
				ConnectURL: "https://example.com/connect",
				PollURL:    "https://example.com/poll",
			}, nil)
		tm.hostedAgents.EXPECT().
			PollProviderAuth("github", "https://example.com/poll").
			Return(&godo.HostedAgentProviderAuthPoll{Provider: "github", Status: "success"}, nil)

		prevAsk := askConnectGitHub
		prevInterval := agentsAuthPollInterval
		t.Cleanup(func() {
			askConnectGitHub = prevAsk
			agentsAuthPollInterval = prevInterval
		})
		askConnectGitHub = func() (bool, error) { return true, nil }
		agentsAuthPollInterval = time.Millisecond

		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, maybeOfferGitHubAuth(config))
		assert.Contains(t, buf.String(), "github connected successfully")
	})
}

func TestRunAgentsCreate_RequiresHarnessOrSpec(t *testing.T) {
	t.Chdir(t.TempDir()) // no agents.yaml to discover
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no manifest given")
	})
}

func TestRunAgentsCreate_RejectsHarnessAndSpecTogether(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, "agent.yaml")
		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "mutually exclusive"))
	})
}

func TestRunAgentsCreate_FromConfig(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
				Name:     "demo",
				ConfigID: "cfg_abc123",
			}).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_cfg_1",
					Name:      "demo",
					ConfigID:  "cfg_abc123",
					Status:    godo.HostedAgentSessionStatusProvisioning,
				},
			}, nil)
		tm.hostedAgents.EXPECT().
			GetSession("sess_cfg_1").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_cfg_1",
					Name:      "demo",
					ConfigID:  "cfg_abc123",
					Status:    godo.HostedAgentSessionStatusReady,
					AgentKind: godo.HostedAgentKindOpenCode,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")

		require.NoError(t, RunAgentsCreate(config))
		got := buf.String()
		assert.Contains(t, got, "Creating hosted session from config")
		assert.Contains(t, got, "Agent is ready")
		assert.Contains(t, got, "cfg_abc123", "the implicit/source config must be visible on the ready card")
		assert.Contains(t, got, "doctl harness-runtime launch demo")
		assert.NotContains(t, got, autoCreatedConfigNotice,
			"--from-config must not claim a new Agent Config was created")
		assert.NotContains(t, got, "create --from-config cfg_abc123 --name <session>",
			"--from-config ready card should not advertise reuse of a just-created config")
	})
}

func TestRunAgentsCreate_FromConfigID_RequiresName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--name is required")
	})
}

func TestRunAgentsCreate_FromConfigID_RejectsRepo(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentRepo, "org/repo")
		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--gh-repo cannot be used with --from-config")
	})
}

func TestPrintRunReadySummary_ShowsPerPhaseTiming(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var buf bytes.Buffer
	printRunReadySummary(&buf, runReadySummary{
		Session: &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
			Name:      "demo",
			AgentKind: godo.HostedAgentKindOpenCode,
			ConfigID:  "cfg_from_spec",
		}},
		Prompt: "classify this",
		Timings: creationTimings{
			Create: time.Second,
			Ready:  24 * time.Second,
			Prompt: time.Second,
		},
		AutoCreatedConfig: true,
	})
	got := buf.String()
	assert.Contains(t, got, "Timing")
	assert.Contains(t, got, "1s")
	assert.Contains(t, got, "24s")
	assert.Contains(t, got, "26s")
	assert.Contains(t, got, "cfg_from_spec")
	assert.Contains(t, got, autoCreatedConfigNotice)
	assert.Contains(t, got, "doctl harness-runtime create --from-config cfg_from_spec --name <session>")
}

func TestPrintRunReadySummary_AutoCreatedConfigNotice(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	t.Run("shows notice and reuse step when config was auto-created", func(t *testing.T) {
		var buf bytes.Buffer
		printRunReadySummary(&buf, runReadySummary{
			Session: &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
				Name:      "demo",
				AgentKind: godo.HostedAgentKindOpenCode,
				ConfigID:  "cfg_auto",
			}},
			AutoCreatedConfig: true,
		})
		got := buf.String()
		assert.Contains(t, got, autoCreatedConfigNotice)
		assert.Contains(t, got, "reuse")
		assert.Contains(t, got, "create --from-config cfg_auto --name <session>")
	})

	t.Run("omits notice when session came from an existing config", func(t *testing.T) {
		var buf bytes.Buffer
		printRunReadySummary(&buf, runReadySummary{
			Session: &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
				Name:      "demo",
				AgentKind: godo.HostedAgentKindOpenCode,
				ConfigID:  "cfg_existing",
			}},
			AutoCreatedConfig: false,
		})
		got := buf.String()
		assert.Contains(t, got, "cfg_existing", "Config ID still shown as identity")
		assert.NotContains(t, got, autoCreatedConfigNotice)
		assert.NotContains(t, got, "reuse")
	})
}

func TestReadySummaryFor_AutoCreatedConfig(t *testing.T) {
	sess := &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
		Name:     "demo",
		ConfigID: "cfg_new",
	}}

	fromHarness := readySummaryFor(&agentCreationSource{harness: "opencode"}, sess)
	assert.True(t, fromHarness.AutoCreatedConfig)

	fromConfig := readySummaryFor(&agentCreationSource{configID: "cfg_existing"}, sess)
	assert.False(t, fromConfig.AutoCreatedConfig)

	noConfig := readySummaryFor(&agentCreationSource{harness: "opencode"},
		&do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{Name: "demo"}})
	assert.False(t, noConfig.AutoCreatedConfig)
}

func TestBuildHarnessManifestPermissions(t *testing.T) {
	t.Setenv(openAIAPIKeyEnv, "sk-test")

	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "codex", permission: "ask"})
	require.NoError(t, err)
	assert.Contains(t, string(raw), "permissions:")
	assert.Contains(t, string(raw), "default: ask")

	raw, err = buildHarnessManifest(harnessManifestOpts{harness: "codex"})
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "permissions:",
		"an unset permission must leave the block out rather than guess")
}

// A bare sandbox's manifest is just `agent: none`: no secrets slot (nothing in
// the guest reads a model key on its own), no config, no framework. The customer
// adds their own secrets and egress by writing a manifest.
func TestBuildHarnessManifestBareSandbox(t *testing.T) {
	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "none", name: "eval-runner"})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	assert.Equal(t, bareSandboxAgent, doc["agent"])
	assert.Equal(t, "eval-runner", doc["name"])
	assert.NotContains(t, doc, "secrets")
	assert.NotContains(t, doc, "config")
	assert.NotContains(t, doc, "env")
	// No template either: the server defaults a bare sandbox to coding-base, and
	// naming it here would pin doctl to a server-side default it does not own.
	assert.NotContains(t, doc, "template")
}

func TestResolveHarnessAgentAcceptsNone(t *testing.T) {
	agent, err := resolveHarnessAgent("none")
	require.NoError(t, err)
	assert.Equal(t, bareSandboxAgent, agent)

	agent, err = resolveHarnessAgent("NONE")
	require.NoError(t, err)
	assert.Equal(t, bareSandboxAgent, agent, "the flag is case-insensitive like every other value")

	// The list a user is shown when they get it wrong has to include the value
	// that now works, or they cannot discover it from the error.
	_, err = resolveHarnessAgent("bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "none")
}

func TestIsBareSandboxHarness(t *testing.T) {
	for _, h := range []string{"none", "None", " none ", "NONE"} {
		assert.Truef(t, isBareSandboxHarness(h), "%q should be a bare sandbox", h)
	}
	for _, h := range []string{"", "opencode", "codex", "claude-code", "codex-agentapi", "custom"} {
		assert.Falsef(t, isBareSandboxHarness(h), "%q should not be a bare sandbox", h)
	}
}

// bareSandbox has to see through a manifest, not just the flag: a manifest is
// the only way to set egress or a size, so it is the likelier way a bare
// sandbox gets created — and it is the path where launch would otherwise drop
// the user into a chat with nothing on the other end.
func TestAgentCreationSourceBareSandbox(t *testing.T) {
	cases := []struct {
		name string
		src  agentCreationSource
		want bool
	}{
		{"harness flag", agentCreationSource{harness: "none"}, true},
		{"flat manifest", agentCreationSource{manifest: []byte("agent: none\n")}, true},
		{"flat manifest, mixed case", agentCreationSource{manifest: []byte("agent: None\n")}, true},
		{"legacy envelope", agentCreationSource{manifest: []byte(
			"apiVersion: agents.digitalocean.com/v1alpha1\nkind: Agent\nspec:\n  runtime:\n    adapter: none\n")}, true},
		{"managed agent flag", agentCreationSource{harness: "opencode"}, false},
		{"managed agent manifest", agentCreationSource{manifest: []byte("agent: opencode\n")}, false},
		{"legacy envelope, managed agent", agentCreationSource{manifest: []byte(
			"apiVersion: agents.digitalocean.com/v1alpha1\nkind: Agent\nspec:\n  runtime:\n    adapter: codex\n")}, false},
		// An adapter that merely starts with "none" is a different adapter.
		{"prefix is not a match", agentCreationSource{manifest: []byte("agent: nonesuch\n")}, false},
		{"empty", agentCreationSource{}, false},
		// The server owns the error for a malformed manifest; guessing here
		// would turn it into a confusing local one.
		{"unparsable manifest", agentCreationSource{manifest: []byte("\tagent: none")}, false},
		// A config's agent lives server-side, so this path reports false and
		// lets the server answer.
		{"from config", agentCreationSource{configID: "cfg-1"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.src.bareSandbox())
		})
	}

	var nilSrc *agentCreationSource
	assert.False(t, nilSrc.bareSandbox())
}

// The ready card is the only place a `--harness none` user is told what to do
// next, so it must carry the commands that drive a sandbox and must NOT offer
// `launch`, which cannot work without an agent.
func TestPrintRunReadySummary_BareSandboxNextSteps(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var buf bytes.Buffer
	printRunReadySummary(&buf, runReadySummary{
		Session: &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
			Name:      "eval-runner",
			AgentKind: godo.HostedAgentKindNone,
		}},
		Harness: "none",
	})
	got := buf.String()
	for _, want := range []string{
		"exec eval-runner -- <command>",
		"upload eval-runner",
		"download eval-runner",
		"port-forward eval-runner",
	} {
		assert.Containsf(t, got, want, "bare-sandbox card should offer %q", want)
	}
	// Asserted against the flag constants, not literal strings: the card is the
	// only place these commands are spelled out for a customer to copy, and an
	// invented flag name produces a card that reads correct and does not run.
	// The first draft of this card said `upload <ref> <file> --path /workspace`,
	// which is not the command's shape at all.
	for _, want := range []string{
		"--" + doctl.ArgAgentLocalFile,
		"--" + doctl.ArgAgentWorkspacePath,
		"--" + doctl.ArgAgentSaveTo,
	} {
		assert.Containsf(t, got, want, "bare-sandbox card should use the real flag %q", want)
	}
	assert.NotContains(t, got, "launch eval-runner",
		"launch ends in a chat with the agent, and a bare sandbox has none")
}

// The managed-agent card is unchanged: launch is still the next step, and none
// of the bare-sandbox rows appear.
func TestPrintRunReadySummary_ManagedAgentStillOffersLaunch(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var buf bytes.Buffer
	printRunReadySummary(&buf, runReadySummary{
		Session: &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
			Name:      "demo",
			AgentKind: godo.HostedAgentKindOpenCode,
		}},
	})
	got := buf.String()
	assert.Contains(t, got, "launch demo")
	assert.NotContains(t, got, "port-forward demo")
}

// Each of these flags is read only by an agent, so on a bare sandbox they are
// refused rather than accepted and ignored. Accepting them silently is the bad
// outcome: --prompt would never be delivered, --gh-repo would set a guest env
// var with nothing to materialize it, and --permission would describe a policy
// enforced by an agent that is not running — all three looking like they worked.
func TestRunAgentsCreate_BareSandboxRejectsAgentOnlyFlags(t *testing.T) {
	cases := []struct {
		name    string
		flag    string
		value   any
		wantMsg string
	}{
		{"prompt", doctl.ArgAgentTriggerPrompt, "do the thing", "--prompt needs an agent to send it to"},
		{"repo", doctl.ArgAgentRepo, "org/repo", "--gh-repo is a hint for an agent to act on"},
		{"permission", doctl.ArgAgentPermission, "ask", "--permission policies the tools an agent may call"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgAgentHarness, "none")
				config.Doit.Set(config.NS, tc.flag, tc.value)
				err := RunAgentsCreate(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantMsg)
			})
		})
	}
}

// The create header prints before a session exists, so it cannot read AgentKind
// off the server the way the wait lines do — it has to come off the request.
func TestRunAgentsCreate_BareSandboxHeaderSaysSandbox(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "none")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "eval-01")

		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_bare",
					Name:      "eval-01",
					AgentKind: godo.HostedAgentKindNone,
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil).AnyTimes()
		tm.hostedAgents.EXPECT().
			GetSession(gomock.Any()).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_bare",
					Name:      "eval-01",
					AgentKind: godo.HostedAgentKindNone,
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil).AnyTimes()

		require.NoError(t, RunAgentsCreate(config))
		got := out.String()
		assert.Contains(t, got, "Creating sandbox session")
		assert.NotContains(t, got, "Creating agent session")
	})
}

// Refusing an explicit --permission is not enough on its own: --permission
// carries a cobra default, so the create path resolves it to "allow" before the
// bare-sandbox check ever runs. Asserting through resolveAgentCreationSource
// rather than buildHarnessManifest is deliberate — the unit test passes
// permission:"" by construction and cannot see this.
func TestResolveAgentCreationSource_BareSandboxWritesNoPermissions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "none")

		src, err := resolveAgentCreationSource(config)
		require.NoError(t, err)
		require.NotNil(t, src.manifest)

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(src.manifest, &doc))
		assert.Equal(t, bareSandboxAgent, doc["agent"])
		assert.NotContains(t, doc, "permissions",
			"a manifest declaring no agent must not carry an agent's tool policy")
	})
}

// The same default still lands for a managed agent: clearing it is scoped to the
// bare sandbox, not a change to what --harness codex writes.
func TestResolveAgentCreationSource_ManagedAgentKeepsPermissionDefault(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		t.Setenv(openAIAPIKeyEnv, "sk-test")
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "codex")

		src, err := resolveAgentCreationSource(config)
		require.NoError(t, err)
		require.NotNil(t, src.manifest)
		assert.Contains(t, string(src.manifest), "default: "+defaultHarnessPermission)
	})
}

// The same flags stay accepted for a managed agent — the rejection is scoped to
// the bare sandbox, not a new blanket restriction.
func TestRunAgentsCreate_ManagedAgentStillAcceptsPrompt(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerPrompt, "do the thing")
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)
		// --dry-run stops before any API call, so this asserts only that flag
		// resolution accepted the combination.
		require.NoError(t, RunAgentsCreate(config))
	})
}

func TestBuildHarnessManifestTemplate(t *testing.T) {
	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "none", template: "my-eval-image"})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	assert.Equal(t, "my-eval-image", doc["template"])

	// Omitted rather than emitted empty: the server derives the template from the
	// agent kind, and a blank `template` key is not the same as no key.
	raw, err = buildHarnessManifest(harnessManifestOpts{harness: "none"})
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "template")
}

func TestIsAgentlessTemplate(t *testing.T) {
	assert.True(t, isAgentlessTemplate("coding-base"))
	assert.True(t, isAgentlessTemplate("  Coding-Base  "),
		"the flag value is compared case-insensitively, like --harness")

	// Every template that carries an agent, and anything the team built itself:
	// doctl cannot know a custom template's base without a round trip, so it
	// passes through rather than guessing.
	for _, carriesAgent := range []string{
		"coding-codex", "coding-opencode", "coding-claude-code", "coding-hermes",
		"langgraph", "crewai", "my-own-template", "",
	} {
		assert.Falsef(t, isAgentlessTemplate(carriesAgent), "template %q", carriesAgent)
	}
}

// A managed harness on the one template known to ship no agent is refused. The
// pairing provisions and reaches READY, so nothing looks wrong until the first
// turn hangs — while the session row claims an agent kind that billing and every
// per-kind metric take at face value.
func TestRunAgentsCreate_ManagedHarnessRejectsAgentlessTemplate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "codex")
		config.Doit.Set(config.NS, doctl.ArgAgentTemplate, "coding-base")

		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ships no agent")
		// The remedy is dropping --harness, not writing --harness none: a lone
		// --template already means the bare sandbox.
		assert.Contains(t, err.Error(), "drop --harness")
	})
}

func TestResolveAgentCreationSource_TemplateReachesTheManifest(t *testing.T) {
	cases := []struct {
		name     string
		harness  string
		template string
	}{
		// The bare sandbox on a team's own image: the case --template exists for.
		{"bare sandbox, custom template", "none", "my-eval-image"},
		// coding-base is redundant here (it is already the default) but naming it
		// explicitly must not be an error — it is the honest way to write it down.
		{"bare sandbox, coding-base named explicitly", "none", "coding-base"},
		// A managed agent on a custom template is BYOC: the customer's tooling on
		// top of an agent base, which is what `template create` produces.
		{"managed agent, custom template", "codex", "my-codex-plus-tools"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				t.Setenv(openAIAPIKeyEnv, "sk-test")
				config.Doit.Set(config.NS, doctl.ArgAgentHarness, tc.harness)
				config.Doit.Set(config.NS, doctl.ArgAgentTemplate, tc.template)

				src, err := resolveAgentCreationSource(config)
				require.NoError(t, err)
				require.NotNil(t, src.manifest)

				var doc map[string]any
				require.NoError(t, yaml.Unmarshal(src.manifest, &doc))
				assert.Equal(t, tc.template, doc["template"])
			})
		})
	}
}

// A lone --template means --harness none, but not when a manifest is already in
// play: the discovered ./agents.yaml declares its own template, so implying a
// bare sandbox would silently ignore the file. cobra's mutual exclusion cannot
// catch this one — a discovered manifest sets no flag to be exclusive with.
func TestResolveAgentCreationSource_TemplateNeedsHarness(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "agents.yaml"),
			[]byte("agent: opencode\n"), 0o644))
		t.Chdir(dir)

		config.Doit.Set(config.NS, doctl.ArgAgentTemplate, "coding-base")

		_, err := resolveAgentCreationSource(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--template")
	})
}

// With nothing else supplying a manifest, --template is enough on its own: the
// only session that needs an image and no agent is a bare sandbox. The empty
// working directory is load-bearing — an agents.yaml here would be discovered
// and the case above would apply instead.
func TestResolveAgentCreationSource_TemplateAloneImpliesBareSandbox(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		t.Chdir(t.TempDir())
		config.Doit.Set(config.NS, doctl.ArgAgentTemplate, "coding-base")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "eval-01")

		src, err := resolveAgentCreationSource(config)
		require.NoError(t, err)
		require.NotNil(t, src.manifest)
		assert.True(t, src.bareSandbox())

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(src.manifest, &doc))
		assert.Equal(t, bareSandboxAgent, doc["agent"])
		assert.Equal(t, "coding-base", doc["template"])
		assert.NotContains(t, doc, "permissions",
			"an implied bare sandbox must not carry an agent's tool policy either")
	})
}

// The public spelling has to reach the manifest as the name the API knows, or
// a --dry-run promoted to a --spec file would create a session on a template
// the server has never heard of.
func TestResolveAgentCreationSource_SandboxAliasReachesManifestAsCodingBase(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		t.Chdir(t.TempDir())
		config.Doit.Set(config.NS, doctl.ArgAgentTemplate, "sandbox")

		src, err := resolveAgentCreationSource(config)
		require.NoError(t, err)
		assert.True(t, src.bareSandbox())

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(src.manifest, &doc))
		assert.Equal(t, baseTemplateCodingBase, doc["template"])
	})
}

// The alias resolves before the agentless check, so pairing it with a managed
// harness is refused the same way coding-base is — and the error quotes what
// the caller typed, not the platform name they have never seen.
func TestResolveAgentCreationSource_SandboxAliasIsStillAgentless(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		t.Chdir(t.TempDir())
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "codex")
		config.Doit.Set(config.NS, doctl.ArgAgentTemplate, "sandbox")

		_, err := resolveAgentCreationSource(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--template sandbox ships no agent")
		assert.NotContains(t, err.Error(), baseTemplateCodingBase)
	})
}

// A team template is passed through rather than matched against the platform
// catalogue, so the implication cannot be keyed on the name being coding-base.
func TestResolveAgentCreationSource_TeamTemplateAloneImpliesBareSandbox(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		t.Chdir(t.TempDir())
		config.Doit.Set(config.NS, doctl.ArgAgentTemplate, "my-team-image")

		src, err := resolveAgentCreationSource(config)
		require.NoError(t, err)
		assert.True(t, src.bareSandbox())
	})
}

// Someone who passed an agent-only flag did not mean a bare sandbox, so the
// implication is reported back as the choice it forced — naming the flag they
// typed. The explicit --harness none messages would cite a flag absent from the
// command line, which reads as doctl having invented one.
func TestResolveAgentCreationSource_ImpliedBareSandboxRejectsAgentOnlyFlags(t *testing.T) {
	cases := []struct {
		name  string
		flag  string
		value any
	}{
		{"prompt", doctl.ArgAgentTriggerPrompt, "do the thing"},
		{"repo", doctl.ArgAgentRepo, "org/repo"},
		{"permission", doctl.ArgAgentPermission, "ask"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				t.Chdir(t.TempDir())
				config.Doit.Set(config.NS, doctl.ArgAgentTemplate, "coding-base")
				config.Doit.Set(config.NS, tc.flag, tc.value)

				_, err := resolveAgentCreationSource(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "--"+tc.flag)
				assert.Contains(t, err.Error(), "name the agent with --"+doctl.ArgAgentHarness)
				assert.NotContains(t, err.Error(), "--harness none runs none",
					"the caller never wrote --harness none")
			})
		})
	}
}

// Naming a harness still wins over the implication: --template only means
// "no agent" when no agent was asked for.
func TestResolveAgentCreationSource_ExplicitHarnessBeatsTemplateImplication(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		t.Chdir(t.TempDir())
		t.Setenv(openAIAPIKeyEnv, "sk-test")
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "codex")
		config.Doit.Set(config.NS, doctl.ArgAgentTemplate, "coding-codex")

		src, err := resolveAgentCreationSource(config)
		require.NoError(t, err)
		assert.False(t, src.bareSandbox())
	})
}

// The manifest --harness none generates has to survive doctl's own client-side
// validation. It did not: `none` was missing from knownAgentAdapters, so create
// refused its own generated manifest with "not a known adapter" before any
// request left the machine. Neither buildHarnessManifest nor
// resolveAgentCreationSource can catch that — validation runs after both — so
// this asserts the generated bytes against the validator directly.
func TestValidateAgentManifest_BareSandboxGeneratedManifestIsValid(t *testing.T) {
	raw, err := buildHarnessManifest(harnessManifestOpts{harness: "none", name: "eval-runner"})
	require.NoError(t, err)

	out := validateAgentManifest(raw)
	assert.Empty(t, out.Errors, "doctl must accept the manifest it generated itself")
}

func TestValidateAdapter_None(t *testing.T) {
	var out agentManifestValidation
	validateAdapter("none", "agent", &out)
	assert.Empty(t, out.Errors)
	assert.Empty(t, out.Warnings,
		"none is supported, not merely declared, so it must not warn like openai-agents does")
}

// Every --harness shape has to survive --secret. The two manifest formats
// disagree about what a secrets block looks like — legacy spec.secrets is a list
// of slots, flat top-level secrets is a map keyed by name — and doctl generates
// flat, so appending the list form produced bytes the API refused with "cannot
// unmarshal array into Go struct field .secrets of type
// map[string]agentspec.flatSecret". It went unnoticed because it only bites when
// the generated manifest declares no secrets of its own, which is every harness
// except codex, claude-code and codex-agentapi.
func TestInjectManifestSecrets_GeneratedManifestsStayFlatShaped(t *testing.T) {
	for _, harness := range []string{"opencode", "codex", "claude-code", "none"} {
		t.Run(harness, func(t *testing.T) {
			t.Setenv(openAIAPIKeyEnv, "sk-test")
			t.Setenv(anthropicAPIKeyEnv, "sk-ant-test")

			raw, err := buildHarnessManifest(harnessManifestOpts{harness: harness})
			require.NoError(t, err)

			out, err := injectManifestSecrets(raw, map[string]string{"TOKEN": "abc"})
			require.NoError(t, err)

			var doc map[string]any
			require.NoError(t, yaml.Unmarshal(out, &doc))
			secrets, ok := yamlMap(doc["secrets"])
			require.Truef(t, ok, "secrets must be a mapping on a flat manifest, got %T:\n%s", doc["secrets"], out)
			assert.Equal(t, "abc", secrets["TOKEN"])
		})
	}
}

// The legacy envelope keeps the list form: it is a different grammar, not an
// older spelling of the same one, and rewriting it would change how the server
// parses the document.
func TestInjectManifestSecrets_LegacyEnvelopeKeepsTheListForm(t *testing.T) {
	in := []byte(`apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
metadata:
  name: demo
spec:
  runtime:
    adapter: opencode
`)
	out, err := injectManifestSecrets(in, map[string]string{"TOKEN": "abc"})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))
	spec, ok := yamlMap(doc["spec"])
	require.True(t, ok)
	_, isList := yamlList(spec["secrets"])
	assert.Truef(t, isList, "legacy spec.secrets must stay a list, got %T:\n%s", spec["secrets"], out)
}
