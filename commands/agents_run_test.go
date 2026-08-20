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

	t.Run("codex maps to codex-agentapi", func(t *testing.T) {
		got, err := resolveHarnessAgent("codex")
		require.NoError(t, err)
		assert.Equal(t, openAIAgentsAdapter, got)
	})

	t.Run("unknown harness", func(t *testing.T) {
		_, err := resolveHarnessAgent("not-a-harness")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})
}

func TestBuildHarnessManifest(t *testing.T) {
	t.Run("opencode with repo and prompt", func(t *testing.T) {
		raw, err := buildHarnessManifest("opencode", "https://github.com/katanemo/plano", "Review README", "demo")
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
		raw, err := buildHarnessManifest("opencode", "katanemo/plano", "", "")
		require.NoError(t, err)
		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(raw, &doc))
		repos, ok := doc["repos"].([]any)
		require.True(t, ok)
		assert.Equal(t, "katanemo/plano", repos[0])
	})

	t.Run("codex includes openai config and env", func(t *testing.T) {
		raw, err := buildHarnessManifest("codex", "", "hello world", "")
		require.NoError(t, err)

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(raw, &doc))
		assert.Equal(t, openAIAgentsAdapter, doc["agent"])
		env, ok := doc["env"].(map[any]any)
		require.True(t, ok)
		assert.Equal(t, "${ENV_ID}", env["CODEX_ENVIRONMENT_ID"])
		assert.Equal(t, "${OPENAI_API_KEY}", env["CODEX_API_KEY"])

		cfg, ok := doc["config"].(map[any]any)
		require.True(t, ok)
		input, ok := cfg["input"].([]any)
		require.True(t, ok)
		require.Len(t, input, 1)
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

func TestRunAgentsRun_NoAttachHarness(t *testing.T) {
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
		config.Doit.Set(config.NS, doctl.ArgAgentNoAttach, true)

		tm.hostedAgents.EXPECT().
			StartProviderAuth("github").
			Return(&godo.HostedAgentProviderAuthStart{Provider: "github", Status: "success"}, nil)

		require.NoError(t, RunAgentsRun(config))
		out := buf.String()
		assert.Contains(t, out, "Launching agent session")
		assert.Contains(t, out, "GitHub already connected")
		assert.Contains(t, out, "Validating configuration")
		assert.Contains(t, out, "Creating hosted session")
		assert.Contains(t, out, "Session created")
		assert.Contains(t, out, "Agent is ready")
		assert.NotContains(t, out, "SESSION_STATUS_")
		assert.Contains(t, out, "doctl agent attach demo")
		assert.Contains(t, out, "katanemo/plano")
	})
}

func TestPrintAttachBanner(t *testing.T) {
	var out bytes.Buffer
	printAttachBanner(&out, &do.HostedAgentSession{
		HostedAgentSession: &godo.HostedAgentSession{
			SessionID: "sess_attach_1",
			Name:      "smoke-test",
			AgentKind: godo.HostedAgentKindOpenCode,
		},
	}, "")

	got := out.String()
	assert.Contains(t, got, "Connected")
	assert.Contains(t, got, "smoke-test")
	assert.Contains(t, got, "Quick help")
	assert.Contains(t, got, "Ctrl-D")
	assert.Contains(t, got, "session keeps running")
	assert.Contains(t, got, "doctl agent remove smoke-test")
}

func TestPrintDetachNotice(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var buf bytes.Buffer
	printDetachNotice(&buf, "my-session")
	out := buf.String()
	assert.Contains(t, out, "Disconnected")
	assert.Contains(t, out, "still running")
	assert.Contains(t, out, "doctl agent attach my-session")
	assert.Contains(t, out, "doctl agent remove my-session")
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

func TestRunAgentsRun_RequiresHarnessOrSpec(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunAgentsRun(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "one of --harness, --spec, or --config-id is required")
	})
}

func TestRunAgentsRun_RejectsHarnessAndSpecTogether(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, "agent.yaml")
		err := RunAgentsRun(config)
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "mutually exclusive"))
	})
}

func TestRunAgentsRun_FromConfigID_NoAttach(t *testing.T) {
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
		config.Doit.Set(config.NS, doctl.ArgAgentConfigID, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentNoAttach, true)

		require.NoError(t, RunAgentsRun(config))
		got := buf.String()
		assert.Contains(t, got, "Creating hosted session from config")
		assert.Contains(t, got, "Agent is ready")
		assert.Contains(t, got, "doctl agent attach demo")
	})
}

func TestRunAgentsRun_FromConfigID_RequiresName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentConfigID, "cfg_abc123")
		err := RunAgentsRun(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--name is required")
	})
}

func TestRunAgentsRun_FromConfigID_RejectsRepo(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentConfigID, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentRepo, "org/repo")
		err := RunAgentsRun(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--gh-repo cannot be used with --config-id")
	})
}
