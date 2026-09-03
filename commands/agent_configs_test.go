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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAgentConfigsCommand(t *testing.T) {
	cmd := AgentConfigs()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "generate", "create", "list", "get", "delete", "list-sessions", "start-session")
}

func TestAgentConfigCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dir := t.TempDir()
		specPath := filepath.Join(dir, "agent.yaml")
		require.NoError(t, os.WriteFile(specPath, []byte("agent: opencode\n"), 0o600))

		tm.hostedAgents.EXPECT().
			CreateAgentConfig(gomock.Any()).
			DoAndReturn(func(req *godo.HostedAgentConfigCreateRequest) (*godo.HostedAgentConfig, error) {
				assert.Equal(t, "my-config", req.Name)
				assert.Contains(t, req.ManifestYAML, "agent: opencode")
				return &godo.HostedAgentConfig{ID: "cfg_1", Name: "my-config"}, nil
			})

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-config")
		require.NoError(t, RunAgentsConfigCreate(config))
	})
}

// The point of --secret on config create is that a checked-in manifest can
// declare which credentials it needs while the value arrives from a secret
// store at create time. The value is captured server-side once, which is why
// --from-config later needs no --secret of its own.
func TestAgentConfigCreate_SecretInjectedFromFlag(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dir := t.TempDir()
		specPath := filepath.Join(dir, "agent.yaml")
		require.NoError(t, os.WriteFile(specPath, []byte("agent: claude-code\nsecrets:\n  - name: ANTHROPIC_API_KEY\n    source: tenantSecret\n    value: placeholder\n"), 0o600))
		keyFile := filepath.Join(dir, "anthropic.key")
		require.NoError(t, os.WriteFile(keyFile, []byte("sk-ant-from-file\n"), 0o600))

		tm.hostedAgents.EXPECT().
			CreateAgentConfig(gomock.Any()).
			DoAndReturn(func(req *godo.HostedAgentConfigCreateRequest) (*godo.HostedAgentConfig, error) {
				assert.Contains(t, req.ManifestYAML, "sk-ant-from-file")
				assert.NotContains(t, req.ManifestYAML, "placeholder", "the flag overrides a value already in the file")
				return &godo.HostedAgentConfig{ID: "cfg_secret", Name: "reviewer"}, nil
			})

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentName, "reviewer")
		config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"ANTHROPIC_API_KEY=@" + keyFile})

		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, RunAgentsConfigCreate(config))
		assert.NotContains(t, buf.String(), "sk-ant-from-file", "the card must never echo a secret value")
	})
}

// A --harness claude-code --dry-run manifest keeps ${ANTHROPIC_API_KEY} in env.
// --secret must satisfy that reference on config create; requiring the env var
// made NAME=- look documented-but-broken.
func TestAgentConfigCreate_SecretSatisfiesClaudeEnvRef(t *testing.T) {
	_ = os.Unsetenv(anthropicAPIKeyEnv)
	prevInteractive := Interactive
	Interactive = false
	t.Cleanup(func() {
		Interactive = prevInteractive
		_ = os.Unsetenv(anthropicAPIKeyEnv)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dir := t.TempDir()
		specPath := filepath.Join(dir, "agent.yaml")
		require.NoError(t, os.WriteFile(specPath, []byte(
			"agent: claude-code\nenv:\n  ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}\n"), 0o600))

		tm.hostedAgents.EXPECT().
			CreateAgentConfig(gomock.Any()).
			DoAndReturn(func(req *godo.HostedAgentConfigCreateRequest) (*godo.HostedAgentConfig, error) {
				assert.Contains(t, req.ManifestYAML, "sk-ant-from-secret")
				assert.NotContains(t, req.ManifestYAML, "${ANTHROPIC_API_KEY}")
				return &godo.HostedAgentConfig{ID: "cfg_claude", Name: "reviewer"}, nil
			})

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentName, "reviewer")
		config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"ANTHROPIC_API_KEY=sk-ant-from-secret"})
		require.NoError(t, RunAgentsConfigCreate(config))
	})
}

func TestRejectStdinSecretWithStdinManifest(t *testing.T) {
	err := rejectStdinSecretWithStdinManifest("-", []string{"ANTHROPIC_API_KEY=-"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined")

	assert.NoError(t, rejectStdinSecretWithStdinManifest("agents.yaml", []string{"ANTHROPIC_API_KEY=-"}))
	assert.NoError(t, rejectStdinSecretWithStdinManifest("-", []string{"ANTHROPIC_API_KEY=@/tmp/key"}))
}

func TestParseAgentSecretPairs_RejectsEmptyValue(t *testing.T) {
	_, err := parseAgentSecretPairs([]string{"ANTHROPIC_API_KEY="})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

// The card is the handoff point to the next command, so it should name the verb
// that now exists rather than the retired run/attach spelling.
func TestAgentConfigCreate_CardPointsAtLaunch(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var buf bytes.Buffer
	printAgentConfigCard(&buf, &godo.HostedAgentConfig{ID: "cfg_1", Name: "reviewer"}, true)
	out := buf.String()
	assert.Contains(t, out, agentCLI+" launch --"+doctl.ArgAgentFromConfig)
	assert.NotContains(t, out, "--config-id")
}

func TestAgentConfigList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListAgentConfigs(&godo.HostedAgentConfigListOptions{PageSize: 10}).
			Return([]godo.HostedAgentConfigSummary{{ID: "cfg_1", Name: "a"}}, "", nil)

		config.Doit.Set(config.NS, doctl.ArgAgentPageSize, 10)
		require.NoError(t, RunAgentsConfigList(config))
	})
}

func TestPrintAgentConfigsList(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })
	prevVerbose := Verbose
	t.Cleanup(func() { Verbose = prevVerbose })

	created := time.Now().Add(-5 * time.Minute)
	configs := []godo.HostedAgentConfigSummary{{
		ID:          "01a01e58-6209-7c75-8b31-6cb80f7301ff",
		Name:        "session-opencode-20260820-084503",
		CreatedAt:   godo.Timestamp{Time: created},
		ContentHash: "c0c95e3b975398c69bd0b235d32f21647171bf825aa08369de820ee240569b35",
	}}

	Verbose = false
	var buf bytes.Buffer
	printAgentConfigsList(&buf, configs)
	out := buf.String()
	assert.Contains(t, out, "1 config")
	assert.Contains(t, out, "session-opencode-20260820-084503")
	assert.Contains(t, out, "01a01e58-6209-7c75-8b31-6cb80f7301ff")
	assert.Contains(t, out, "5m ago")
	assert.NotContains(t, out, "c0c95e3b975398c69bd0b235d32f21647171bf825aa08369de820ee240569b35")

	Verbose = true
	var verboseBuf bytes.Buffer
	printAgentConfigsList(&verboseBuf, configs)
	verboseOut := verboseBuf.String()
	assert.Contains(t, verboseOut, created.UTC().Format("2006-01-02 15:04"))
	assert.NotContains(t, verboseOut, "5m ago")
}

func TestTruncateMiddle(t *testing.T) {
	assert.Equal(t, "", truncateMiddle("", 20))
	assert.Equal(t, "short", truncateMiddle("short", 20))
	assert.Equal(t, "c0c95e3b9…240569b35", truncateMiddle("c0c95e3b975398c69bd0b235d32f21647171bf825aa08369de820ee240569b35", 20))
}

func TestAgentConfigGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			GetAgentConfig("cfg_1").
			Return(&godo.HostedAgentConfig{ID: "cfg_1", Name: "a"}, nil)

		config.Args = append(config.Args, "cfg_1")
		require.NoError(t, RunAgentsConfigGet(config))
	})
}

func TestAgentConfigGet_MissingArg(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		assert.Error(t, RunAgentsConfigGet(config))
	})
}

func godoStatusErr(status int, message string) error {
	return &godo.ErrorResponse{
		Response: &http.Response{
			StatusCode: status,
			Request:    httptest.NewRequest(http.MethodGet, "http://harness/v2/agents/configs/cfg_1", nil),
		},
		Message: message,
	}
}

func TestAgentConfigDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().DeleteAgentConfig("cfg_1").Return(nil)

		config.Args = append(config.Args, "cfg_1")
		require.NoError(t, RunAgentsConfigDelete(config))
	})
}

func TestAgentConfigDelete_ActiveSessions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().DeleteAgentConfig("cfg_1").Return(godoStatusErr(http.StatusConflict, "agent config has active sessions"))

		config.Args = append(config.Args, "cfg_1")
		err := RunAgentsConfigDelete(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "active sessions")
		assert.Contains(t, err.Error(), "doctl harness-runtime remove")
	})
}

func TestAgentConfigListSessions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListAgentConfigSessions("cfg_1", &godo.HostedAgentSessionListOptions{
				Status: godo.HostedAgentSessionStatus("SESSION_STATUS_READY"),
			}).
			Return([]do.HostedAgentSession{{
				HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"},
			}}, "", nil)

		config.Args = append(config.Args, "cfg_1")
		config.Doit.Set(config.NS, doctl.ArgAgentStatus, "SESSION_STATUS_READY")
		require.NoError(t, RunAgentsConfigListSessions(config))
	})
}

func TestAgentConfigListSessions_NotFound(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListAgentConfigSessions("cfg_missing", &godo.HostedAgentSessionListOptions{}).
			Return(nil, "", godoStatusErr(http.StatusNotFound, "agent config not found"))

		config.Args = append(config.Args, "cfg_missing")
		err := RunAgentsConfigListSessions(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent config not found")
	})
}

func TestAgentConfigStartSession(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
				Name:     "my-session",
				ConfigID: "cfg_1",
			}).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1", Name: "my-session"},
			}, nil)

		config.Args = append(config.Args, "cfg_1")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-session")
		require.NoError(t, RunAgentsConfigStartSession(config))
	})
}

// Server-side warnings are advisory, so they must not be allowed to append
// prose to the document that -o json callers are parsing.
func TestAgentConfigCreate_WarningsKeepJSONParseable(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dir := t.TempDir()
		specPath := filepath.Join(dir, "agent.yaml")
		require.NoError(t, os.WriteFile(specPath, []byte("agent: opencode\n"), 0o600))

		tm.hostedAgents.EXPECT().
			CreateAgentConfig(gomock.Any()).
			Return(&godo.HostedAgentConfig{
				ID:       "cfg_warned",
				Name:     "my-config",
				Warnings: []string{"adapter opencode is not yet supported for session create"},
			}, nil)

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-config")

		var stdout bytes.Buffer
		config.Out = &stdout

		prev := Output
		Output = "json"
		defer func() { Output = prev }()

		stderr := captureProcessStderr(t, func() {
			require.NoError(t, RunAgentsConfigCreate(config))
		})

		raw := stdout.String()
		var parsed map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &parsed), "stdout must be valid JSON in -o json mode, got: %q", raw)
		assert.Equal(t, "cfg_warned", parsed["id"])

		// The warning is reported as data in the payload rather than as prose
		// appended after it, which is what made the document unparseable.
		assert.Equal(t,
			[]any{"adapter opencode is not yet supported for session create"},
			parsed["warnings"],
			"warnings must be carried in the JSON payload")

		assert.Empty(t, stderr, "nothing may be written to stderr in -o json mode")
	})
}

// In text mode there is no payload to carry the warning, so it is printed —
// and to stderr, keeping the card on stdout pipeable on its own.
func TestAgentConfigCreate_WarningsPrintedToStderrInTextMode(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dir := t.TempDir()
		specPath := filepath.Join(dir, "agent.yaml")
		require.NoError(t, os.WriteFile(specPath, []byte("agent: opencode\n"), 0o600))

		tm.hostedAgents.EXPECT().
			CreateAgentConfig(gomock.Any()).
			Return(&godo.HostedAgentConfig{
				ID:       "cfg_warned",
				Name:     "my-config",
				Warnings: []string{"adapter opencode is not yet supported for session create"},
			}, nil)

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-config")

		var stdout bytes.Buffer
		config.Out = &stdout

		stderr := captureProcessStderr(t, func() {
			require.NoError(t, RunAgentsConfigCreate(config))
		})

		assert.Contains(t, stderr, "not yet supported", "warning must reach the operator")
		assert.NotContains(t, stdout.String(), "not yet supported", "warning must not be mixed into the card on stdout")
	})
}
