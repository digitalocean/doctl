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

const sampleManifest = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
metadata:
  name: test-agent
spec:
  adapter: opencode
`

func TestAgentsCommand(t *testing.T) {
	cmd := Agents()
	assert.NotNil(t, cmd)

	assertCommandNames(t, cmd, "start", "attach", "list", "show", "logs", "approve", "destroy")
}

func TestAgents_helpers(t *testing.T) {
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

func TestReadManifest(t *testing.T) {
	t.Run("from file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "agent.yaml")
		assert.NoError(t, os.WriteFile(path, []byte(sampleManifest), 0o644))

		raw, err := readManifest(nil, path)
		assert.NoError(t, err)
		assert.Equal(t, sampleManifest, string(raw))
	})

	t.Run("from stdin", func(t *testing.T) {
		raw, err := readManifest(strings.NewReader(sampleManifest), "-")
		assert.NoError(t, err)
		assert.Equal(t, sampleManifest, string(raw))
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readManifest(nil, filepath.Join(t.TempDir(), "nope.yaml"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("empty manifest", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.yaml")
		assert.NoError(t, os.WriteFile(path, []byte("   \n  \t\n"), 0o644))

		_, err := readManifest(nil, path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

// TestRunAgentsStart covers --spec on disk uploaded as a raw application/x-yaml
// body via CreateSessionFromManifest.
func TestRunAgentsStart(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	assert.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest([]byte(sampleManifest)).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_test",
					AgentKind: godo.HostedAgentKindOpenCode,
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
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
