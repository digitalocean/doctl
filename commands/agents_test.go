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
		tm.hostedAgents.EXPECT().ListSessions(nil).Return([]do.HostedAgentSession{}, "", nil)
		assert.NoError(t, RunAgentsList(config))
	})
}

func TestRunAgentsList_Pagination(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		want := &godo.HostedAgentSessionListOptions{
			PageSize: 2,
			Status:   godo.HostedAgentSessionStatusReady,
		}
		tm.hostedAgents.EXPECT().ListSessions(want).Return([]do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"}},
		}, "1561", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentPageSize, 2)
		config.Doit.Set(config.NS, doctl.ArgAgentStatus, string(godo.HostedAgentSessionStatusReady))

		assert.NoError(t, RunAgentsList(config))
		assert.Contains(t, buf.String(), "Next page token: 1561")
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

// TestHostedAgentEventDecodesSPIWire guards against the regression where the
// SSE stream delivered events but doctl rendered nothing: the harness-api
// serializes the SPI canonical envelope (type/data/timestamp/tenant_id), which
// must map onto godo.HostedAgentEvent's Kind/Payload/At/TeamID fields. This is
// a verbatim run.token_delta frame from the harness-api demo stream.
func TestHostedAgentEventDecodesSPIWire(t *testing.T) {
	const frame = `{"event_id":"01KTBXPBY60VYC5YKF6AKDX0ZS","run_id":"run-7f16719a-da1c-449d-a4ca-18e524bb63e3","tenant_id":"120","session_id":"sess_5a1ff33e","timestamp":"2026-06-05T12:56:24.774753219Z","seq":0,"type":"run.token_delta","data":{"text":"Paris"}}`

	var ev godo.HostedAgentEvent
	assert.NoError(t, json.Unmarshal([]byte(frame), &ev))

	assert.Equal(t, "01KTBXPBY60VYC5YKF6AKDX0ZS", ev.EventID)
	assert.Equal(t, "run-7f16719a-da1c-449d-a4ca-18e524bb63e3", ev.RunID)
	assert.Equal(t, "sess_5a1ff33e", ev.SessionID)
	assert.Equal(t, uint64(120), ev.TeamID)
	assert.Equal(t, godo.HostedAgentEventKindTokenChunk, ev.Kind)
	assert.False(t, ev.At.IsZero(), "timestamp should be parsed from the wire `timestamp` field")
	assert.JSONEq(t, `{"text":"Paris"}`, string(ev.Payload))

	var buf bytes.Buffer
	renderEvent(&buf, ev)
	assert.Equal(t, "Paris", buf.String())
}

func TestRenderEvent(t *testing.T) {
	cases := []struct {
		name    string
		kind    godo.HostedAgentEventKind
		runID   string
		payload string
		want    string
	}{
		{"token chunk", godo.HostedAgentEventKindTokenChunk, "", `{"text":"Paris"}`, "Paris"},
		{"run started", godo.HostedAgentEventKindRunStarted, "run-1", `{"agent":"codex"}`, "\n[run run-1 started (codex)]\n"},
		{"run started no agent", godo.HostedAgentEventKindRunStarted, "run-2", `{}`, "\n[run run-2 started]\n"},
		{"tool call started", godo.HostedAgentEventKindToolCallStarted, "", `{"tool_call_id":"t1","name":"bash"}`, "\n> bash ...\n"},
		{"tool call completed", godo.HostedAgentEventKindToolCallCompleted, "", `{"ok":true,"duration_ms":12,"summary":"ran ls"}`, "  ran ls (12ms)\n"},
		{"run completed", godo.HostedAgentEventKindRunCompleted, "", `{"total_tokens_in":3,"total_tokens_out":5,"run_cost_micros":1234}`, "\n[run done: 3 in / 5 out tokens, $0.0012]\n"},
		{"run failed", godo.HostedAgentEventKindRunFailed, "", `{"code":5,"message":"hitl rejected"}`, "\n[run failed: hitl rejected (code 5)]\n"},
		{"hitl resolved", godo.HostedAgentEventKindHITLResolved, "", `{"hitl_id":"hitl_1","outcome":1}`, "\n[HITL hitl_1 -> approve]\n"},
		{"session updated", godo.HostedAgentEventKindSessionUpdated, "", `{}`, "\n[session updated]\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			renderEvent(&buf, godo.HostedAgentEvent{
				Kind:    tc.kind,
				RunID:   tc.runID,
				Payload: json.RawMessage(tc.payload),
			})
			assert.Equal(t, tc.want, buf.String())
		})
	}
}

// TestErrorResponseSurfacesNestedMessage guards the fix that lets doctl show
// the harness-api error reason. harness-api returns {"error":{"code","message"}}
// (nested), not the top-level {"message"} godo historically expected, so a
// failed SendInput used to print only "...: 400" with no reason.
func TestErrorResponseSurfacesNestedMessage(t *testing.T) {
	const body = `{"error":{"code":400,"message":"forward input to OHR: ohr attach: connection error"}}`
	er := &godo.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Request:    httptest.NewRequest(http.MethodPost, "http://harness/v2/agents/sessions/sess_x/input", nil),
		},
	}
	assert.NoError(t, json.Unmarshal([]byte(body), er))
	assert.Contains(t, er.Error(), "forward input to OHR: ohr attach: connection error")
}

func TestRenderEventHITLRequested(t *testing.T) {
	var buf bytes.Buffer
	renderEvent(&buf, godo.HostedAgentEvent{
		Kind:    godo.HostedAgentEventKindHITLRequested,
		Payload: json.RawMessage(`{"hitl_id":"hitl_42","payload":{"command":"rm -rf /tmp/x"}}`),
	})
	out := buf.String()
	assert.Contains(t, out, "[HITL] Action requires approval:")
	assert.Contains(t, out, "rm -rf /tmp/x")
	assert.Contains(t, out, "hitl_id: hitl_42")
	assert.Contains(t, out, "/a hitl_42")
}

// TestAttachLoopAcknowledgesSend verifies that a successful submit prints an
// immediate acknowledgement with the run id. The agent's first token can be
// tens of seconds away; without this the silence reads as a hang and users
// re-submit, spawning a duplicate run.
func TestAttachLoopAcknowledgesSend(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		tm.hostedAgents.EXPECT().
			SendInput("sess_x", &godo.HostedAgentSendInputRequest{Text: "What is the capital of France?"}).
			Return(&godo.HostedAgentSendInputResponse{RunID: "run-abc"}, nil)

		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("What is the capital of France?\n"), &pendingHITL{})
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "(queued as run-abc; waiting for the agent...)")
	})
}
