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
	assertCommandNames(t, cmd, "create", "list", "get", "delete", "list-sessions", "start-session")
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

	var buf bytes.Buffer
	printAgentConfigsList(&buf, []godo.HostedAgentConfigSummary{{
		ID:          "01a01e58-6209-7c75-8b31-6cb80f7301ff",
		Name:        "session-opencode-20260820-084503",
		CreatedAt:   godo.Timestamp{Time: time.Date(2026, 8, 20, 8, 45, 3, 0, time.UTC)},
		ContentHash: "c0c95e3b975398c69bd0b235d32f21647171bf825aa08369de820ee240569b35",
	}})

	out := buf.String()
	assert.Contains(t, out, "1 config")
	assert.Contains(t, out, "session-opencode-20260820-084503")
	assert.Contains(t, out, "01a01e58-6209-7c75-8b31-6cb80f7301ff")
	assert.Contains(t, out, "2026-08-20 08:45")
	assert.NotContains(t, out, "c0c95e3b975398c69bd0b235d32f21647171bf825aa08369de820ee240569b35")
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
		assert.Contains(t, err.Error(), "doctl agent remove")
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
