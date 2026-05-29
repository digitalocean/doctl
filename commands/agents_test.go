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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestAgentsCommand(t *testing.T) {
	cmd := Agents()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "start", "attach", "list", "show", "logs", "approve", "auth", "destroy")

	authCmd := findChild(cmd, "auth")
	if assert.NotNil(t, authCmd, "expected `auth` subcommand on `agents`") {
		assertCommandNames(t, authCmd, "github")
	}
}

func TestAgents_helpers(t *testing.T) {
	t.Run("agentKindFor", func(t *testing.T) {
		cases := []struct {
			in      string
			want    godo.HostedAgentKind
			wantErr bool
		}{
			{"claude-code", godo.HostedAgentKindClaudeCode, false},
			{"CLAUDE", godo.HostedAgentKindClaudeCode, false},
			{"opencode", godo.HostedAgentKindOpenCode, false},
			{"none", godo.HostedAgentKindNone, false},
			{"bogus", "", true},
		}
		for _, tc := range cases {
			got, err := agentKindFor(tc.in)
			if tc.wantErr {
				assert.Error(t, err, "input=%q", tc.in)
				continue
			}
			assert.NoError(t, err, "input=%q", tc.in)
			assert.Equal(t, tc.want, got, "input=%q", tc.in)
		}
	})

	t.Run("hitlOutcomeFor", func(t *testing.T) {
		cases := []struct {
			in      string
			want    godo.HostedAgentHITLOutcome
			wantErr bool
		}{
			{"approve", godo.HostedAgentHITLOutcomeApprove, false},
			{"REJECT", godo.HostedAgentHITLOutcomeReject, false},
			{"defer", godo.HostedAgentHITLOutcomeDefer, false},
			{"maybe", "", true},
		}
		for _, tc := range cases {
			got, err := hitlOutcomeFor(tc.in)
			if tc.wantErr {
				assert.Error(t, err, "input=%q", tc.in)
				continue
			}
			assert.NoError(t, err, "input=%q", tc.in)
			assert.Equal(t, tc.want, got, "input=%q", tc.in)
		}
	})
}

func TestRunAgentsStart(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		want := &godo.HostedAgentSessionCreateRequest{
			AgentKind: godo.HostedAgentKindClaudeCode,
			RepoHint:  "acme/payments",
		}
		tm.hostedAgents.EXPECT().CreateSession(want).Return(&do.HostedAgentSession{
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID: "sess_test",
				AgentKind: godo.HostedAgentKindClaudeCode,
				Status:    godo.HostedAgentSessionStatusReady,
			},
		}, nil)

		config.Doit.Set(config.NS, "agent", "claude-code")
		config.Doit.Set(config.NS, "repo", "acme/payments")
		assert.NoError(t, RunAgentsStart(config))
	})
}

func TestReadAgentSpec(t *testing.T) {
	t.Run("yaml from file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "agent.yaml")
		err := os.WriteFile(path, []byte(`agent_kind: AGENT_KIND_CLAUDE_CODE
repo_hint: acme/payments
idle_timeout_seconds: 3600
`), 0o644)
		assert.NoError(t, err)

		req, err := readAgentSpec(nil, path)
		assert.NoError(t, err)
		assert.Equal(t, godo.HostedAgentKindClaudeCode, req.AgentKind)
		assert.Equal(t, "acme/payments", req.RepoHint)
		assert.Equal(t, int64(3600), req.IdleTimeoutSeconds)
	})

	t.Run("json from stdin", func(t *testing.T) {
		req, err := readAgentSpec(strings.NewReader(`{"agent_kind":"AGENT_KIND_OPENCODE","repo_hint":"foo/bar"}`), "-")
		assert.NoError(t, err)
		assert.Equal(t, godo.HostedAgentKindOpenCode, req.AgentKind)
		assert.Equal(t, "foo/bar", req.RepoHint)
	})

	t.Run("unknown field is rejected", func(t *testing.T) {
		_, err := readAgentSpec(strings.NewReader(`agent_kind: AGENT_KIND_CLAUDE_CODE
mystery: yes
`), "-")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mystery")
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readAgentSpec(nil, filepath.Join(t.TempDir(), "nope.yaml"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

// TestRunAgentsStart_FromSpec covers the --spec branch: spec YAML on disk,
// --repo flag overrides the spec's repo_hint.
func TestRunAgentsStart_FromSpec(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	err := os.WriteFile(specPath, []byte(`agent_kind: AGENT_KIND_OPENCODE
repo_hint: spec/repo
idle_timeout_seconds: 1800
`), 0o644)
	assert.NoError(t, err)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		want := &godo.HostedAgentSessionCreateRequest{
			AgentKind:          godo.HostedAgentKindOpenCode,
			RepoHint:           "override/repo", // --repo wins over spec
			IdleTimeoutSeconds: 1800,
		}
		tm.hostedAgents.EXPECT().CreateSession(want).Return(&do.HostedAgentSession{
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID: "sess_test",
				AgentKind: godo.HostedAgentKindOpenCode,
				Status:    godo.HostedAgentSessionStatusReady,
			},
		}, nil)

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, "repo", "override/repo")
		assert.NoError(t, RunAgentsStart(config))
	})
}

func TestRunAgentsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().ListSessions(nil).Return([]do.HostedAgentSession{}, nil)
		assert.NoError(t, RunAgentsList(config))
	})
}

func TestRunAgentsShow(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().GetSession("sess_test").Return(&do.HostedAgentSession{
			HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_test"},
		}, nil)
		config.Args = []string{"sess_test"}
		assert.NoError(t, RunAgentsShow(config))
	})
}

func TestRunAgentsDestroy(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().DestroySession("sess_test").Return(nil)
		config.Args = []string{"sess_test"}
		assert.NoError(t, RunAgentsDestroy(config))
	})
}

func TestRunAgentsApprove(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		want := &godo.HostedAgentResolveHITLRequest{
			Outcome: godo.HostedAgentHITLOutcomeApprove,
			Source:  godo.HostedAgentResolutionSourceOutOfBand,
		}
		tm.hostedAgents.EXPECT().ResolveHITL("sess_test", "req_1", want).Return(nil)
		config.Args = []string{"sess_test", "req_1", "approve"}
		assert.NoError(t, RunAgentsApprove(config))
	})
}

func TestRunAgentsAuthGitHub_NoWait(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			StartOAuthFlow("sess_test", "github", &godo.HostedAgentStartOAuthFlowRequest{}).
			Return(&godo.HostedAgentStartOAuthFlowResponse{
				AuthorizeURL: "https://example.invalid/authorize",
				FlowKind:     godo.HostedAgentOAuthFlowKindWebCallback,
			}, nil)

		config.Doit.Set(config.NS, "session", "sess_test")
		config.Doit.Set(config.NS, "no-open", true)
		config.Doit.Set(config.NS, "no-wait", true)
		assert.NoError(t, RunAgentsAuthGitHub(config))
	})
}

func findChild(cmd *Command, name string) *Command {
	for _, child := range cmd.ChildCommands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}
