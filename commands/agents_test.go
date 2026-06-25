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
	"errors"
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

// TestHostedAgentEventDecodesSPIWire regression-guards the SPI envelope
// (type/data/timestamp/tenant_id) -> godo HostedAgentEvent mapping.
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
		{"run completed", godo.HostedAgentEventKindRunCompleted, "", `{"total_tokens_in":3,"total_tokens_out":5,"run_cost_micros":1234}`, "\n[run done: 3 in / 5 out tokens, $0.0012]\n" + runSeparator + "\n"},
		{"run failed", godo.HostedAgentEventKindRunFailed, "", `{"code":5,"message":"hitl rejected"}`, "\n[run failed: hitl rejected (code 5)]\n" + runSeparator + "\n"},
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

// TestErrorResponseSurfacesNestedMessage pins the {"error":{"code","message"}}
// decode that harness-api uses (vs godo's historical top-level {"message"}).
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

// TestAttachLoopAcknowledgesSend: a successful submit prints "(queued ...)"
// immediately so the multi-second wait for the first token doesn't read as a hang.
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

// TestAttachLoopHITLLetterShortcut covers the line-mode HITL path. The
// raw-mode keystroke path needs a real PTY; see TestReadHITLKeystroke.
func TestAttachLoopHITLLetterShortcut(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		outcome godo.HostedAgentHITLOutcome
	}{
		{"approve y", "y\n", godo.HostedAgentHITLOutcomeApprove},
		{"approve YES", "YES\n", godo.HostedAgentHITLOutcomeApprove},
		{"approve a", "a\n", godo.HostedAgentHITLOutcomeApprove},
		{"reject n", "n\n", godo.HostedAgentHITLOutcomeReject},
		{"reject no", "no\n", godo.HostedAgentHITLOutcomeReject},
		{"reject r", "r\n", godo.HostedAgentHITLOutcomeReject},
		{"defer d", "d\n", godo.HostedAgentHITLOutcomeDefer},
		{"defer defer", "defer\n", godo.HostedAgentHITLOutcomeDefer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				var buf bytes.Buffer
				config.Out = &buf
				pending := &pendingHITL{}
				pending.set("hitl_42")

				tm.hostedAgents.EXPECT().
					ResolveHITL("sess_x", "hitl_42", &godo.HostedAgentResolveHITLRequest{
						Outcome: tc.outcome,
						Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
					}).Return(nil)

				err := attachLoop(config, config.HostedAgents(), "sess_x",
					strings.NewReader(tc.input), pending)
				assert.NoError(t, err)
				assert.Contains(t, buf.String(), "[y/n/d] > ")
			})
		})
	}
}

// TestAttachLoopClearsPendingAfterResolve guards the "press y, then had to
// press Enter again" bug: after a successful ResolveHITL, attachLoop must
// clear `pending` client-side instead of waiting for the SSE HITLResolved
// echo. Otherwise the next iteration re-enters the HITL branch and blocks.
func TestAttachLoopClearsPendingAfterResolve(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		pending := &pendingHITL{}
		pending.set("hitl_42")

		tm.hostedAgents.EXPECT().
			ResolveHITL("sess_x", "hitl_42", &godo.HostedAgentResolveHITLRequest{
				Outcome: godo.HostedAgentHITLOutcomeApprove,
				Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
			}).Return(nil)

		// Line-mode `y\n` exercises the same resolve+clear path; the raw-mode
		// branch shares the clearIf call.
		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("y\n"), pending)
		assert.NoError(t, err)
		assert.Equal(t, "", pending.get(), "pending must be cleared after successful resolve")
	})
}

// TestAttachLoopKeepsPendingOnResolveError: if ResolveHITL fails, we must NOT
// clear pending — otherwise the user loses their chance to retry the approval.
func TestAttachLoopKeepsPendingOnResolveError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		pending := &pendingHITL{}
		pending.set("hitl_42")

		tm.hostedAgents.EXPECT().
			ResolveHITL("sess_x", "hitl_42", &godo.HostedAgentResolveHITLRequest{
				Outcome: godo.HostedAgentHITLOutcomeApprove,
				Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
			}).Return(errors.New("boom"))

		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("y\n"), pending)
		assert.NoError(t, err)
		assert.Equal(t, "hitl_42", pending.get(), "pending must survive a failed resolve")
		assert.Contains(t, buf.String(), "resolve failed: boom")
	})
}

// TestAttachLoopHITLShortcutIgnoredWithoutPending: a bare `y` with no pending
// HITL is sent as regular input, not silently swallowed as an approval.
func TestAttachLoopHITLShortcutIgnoredWithoutPending(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		tm.hostedAgents.EXPECT().
			SendInput("sess_x", &godo.HostedAgentSendInputRequest{Text: "y"}).
			Return(&godo.HostedAgentSendInputResponse{RunID: "run-1"}, nil)

		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("y\n"), &pendingHITL{})
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "(queued as run-1")
	})
}

func TestHitlLetterShortcut(t *testing.T) {
	cases := []struct {
		in   string
		want godo.HostedAgentHITLOutcome
		ok   bool
	}{
		{"y", godo.HostedAgentHITLOutcomeApprove, true},
		{"YES", godo.HostedAgentHITLOutcomeApprove, true},
		{"a", godo.HostedAgentHITLOutcomeApprove, true},
		{"A", godo.HostedAgentHITLOutcomeApprove, true},
		{"n", godo.HostedAgentHITLOutcomeReject, true},
		{"No", godo.HostedAgentHITLOutcomeReject, true},
		{"r", godo.HostedAgentHITLOutcomeReject, true},
		{"R", godo.HostedAgentHITLOutcomeReject, true},
		{"d", godo.HostedAgentHITLOutcomeDefer, true},
		{"defer", godo.HostedAgentHITLOutcomeDefer, true},
		{"yes please", "", false},
		{"", "", false},
		{"approve", "", false},
	}
	for _, tc := range cases {
		got, ok := hitlLetterShortcut(tc.in)
		assert.Equal(t, tc.ok, ok, "input=%q", tc.in)
		if ok {
			assert.Equal(t, tc.want, got, "input=%q", tc.in)
		}
	}
}

func TestAttachPrompt(t *testing.T) {
	p := &pendingHITL{}
	assert.Equal(t, "> ", attachPrompt(p))
	p.set("hitl_42")
	assert.Equal(t, "[y/n/d] > ", attachPrompt(p))
	p.clearIf("hitl_42")
	assert.Equal(t, "> ", attachPrompt(p))
}

// TestReadHITLKeystroke pins the non-TTY fallback contract: no raw mode,
// no bytes consumed. The raw-mode path needs a real PTY and is verified live.
func TestReadHITLKeystroke(t *testing.T) {
	t.Run("non-file reader falls back", func(t *testing.T) {
		outcome, key, action := readHITLKeystroke(strings.NewReader("y"))
		assert.Equal(t, godo.HostedAgentHITLOutcome(""), outcome)
		assert.Equal(t, byte(0), key)
		assert.Equal(t, hitlKeyFallback, action)
	})

	t.Run("pipe (file but not a TTY) falls back without consuming input", func(t *testing.T) {
		r, w, err := os.Pipe()
		assert.NoError(t, err)
		t.Cleanup(func() { r.Close(); w.Close() })

		_, err = w.Write([]byte("y"))
		assert.NoError(t, err)

		outcome, key, action := readHITLKeystroke(r)
		assert.Equal(t, godo.HostedAgentHITLOutcome(""), outcome)
		assert.Equal(t, byte(0), key)
		assert.Equal(t, hitlKeyFallback, action)

		var buf [1]byte
		n, err := r.Read(buf[:])
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, byte('y'), buf[0], "byte must not be consumed when falling back")
	})
}

// TestPromptDisplay pins the prompt-aware writer: streaming tokens (no \n)
// must not get wiped, newline-terminated events drop to a fresh line, and
// the spinner draws above the prompt without disturbing it.
func TestPromptDisplay(t *testing.T) {
	t.Run("non-raw is pass-through", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		_, err := s.display.Write([]byte("hello\nworld\n"))
		assert.NoError(t, err)
		assert.Equal(t, "hello\nworld\n", buf.String())
	})

	t.Run("raw + newline-terminated event clears, CRLFs, redraws prompt+lineBuf", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.mu.Lock()
		s.lineBuf = []byte("partial input")
		s.mu.Unlock()
		s.display.setRaw(true)

		_, err := s.display.Write([]byte("event\n"))
		assert.NoError(t, err)
		assert.Equal(t, "\r\x1b[Kevent\r\n> partial input", buf.String())
	})

	t.Run("raw + streaming tokens preserve previous content (no clear, no redraw)", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.Write([]byte("I'll"))
		s.display.Write([]byte(" create"))
		s.display.Write([]byte(" hello.txt"))

		// First token clears the prompt line, later tokens append.
		assert.Equal(t, "\r\x1b[KI'll create hello.txt", buf.String())
	})

	t.Run("raw + event after stream lands on a fresh line below the stream", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.Write([]byte("I'll create"))
		s.display.Write([]byte("(thinking...)\n"))
		assert.Contains(t, buf.String(), "I'll create\r\n(thinking...)\r\n> ",
			"event must land on a fresh line, not glue onto the streaming text")
	})

	t.Run("raw mode redraw flips prompt when HITL becomes pending", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.redraw()
		assert.Equal(t, "\r\x1b[K> ", buf.String())

		buf.Reset()
		pending.set("hitl_99")
		s.display.redraw()
		assert.Equal(t, "\r\x1b[K[y/n/d] > ", buf.String())
	})

	t.Run("echo is silent mid-stream so keystrokes don't land on the agent's line", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.Write([]byte("streaming"))
		buf.Reset()
		s.display.echo([]byte("h"))
		assert.Equal(t, "", buf.String(), "echo must not paint onto the streaming line")
	})

	t.Run("spinnerFrame draws the line above the prompt via DECSC/DECRC", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.spinnerFrame("⠋")
		assert.Equal(t, "\x1b7\x1b[A\r\x1b[K⠋ thinking...\x1b8", buf.String())
	})

	t.Run("spinnerFrame is a no-op mid-stream", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.Write([]byte("tokens streaming"))
		buf.Reset()
		s.display.spinnerFrame("⠋")
		assert.Equal(t, "", buf.String(), "spinner must not animate while tokens stream")
	})
}

func TestEventCursor(t *testing.T) {
	var c eventCursor
	assert.Equal(t, "", c.get())

	c.set("evt_1")
	assert.Equal(t, "evt_1", c.get())

	c.set("evt_2")
	assert.Equal(t, "evt_2", c.get())

	// Empty must not reset — protects the cursor against events missing EventID.
	c.set("")
	assert.Equal(t, "evt_2", c.get())
}

// TestRunAgentsAttachAuthFailure: a 401 from pre-attach GetSession surfaces
// the friendly "Authentication failed" message, not the raw HTTP error.
func TestRunAgentsAttachAuthFailure(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		authErr := &godo.ErrorResponse{
			Response: &http.Response{
				StatusCode: http.StatusUnauthorized,
				Request:    httptest.NewRequest(http.MethodGet, "http://harness/v2/agents/sessions/sess_x", nil),
			},
			Message: "Unable to authenticate you",
		}
		tm.hostedAgents.EXPECT().GetSession("sess_x").Return(nil, authErr)

		config.Args = []string{"sess_x"}
		err := RunAgentsAttach(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Authentication failed")
		assert.Contains(t, err.Error(), "doctl auth init")
	})
}

// TestClassifyStreamError pins the apierr contract: 401/403/404/409 are
// terminal (409 == V0 single-connection rejection); 5xx/other are transient.
func TestClassifyStreamError(t *testing.T) {
	mkErr := func(status int, message string) error {
		return &godo.ErrorResponse{
			Response: &http.Response{
				StatusCode: status,
				Request:    httptest.NewRequest(http.MethodGet, "http://harness/v2/agents/sessions/sess_x/stream", nil),
			},
			Message: message,
		}
	}

	cases := []struct {
		name        string
		err         error
		wantTermini bool
		wantSubstr  string
	}{
		{
			name:        "401 unauthorized is terminal",
			err:         mkErr(http.StatusUnauthorized, "token expired"),
			wantTermini: true,
			wantSubstr:  "Authentication failed",
		},
		{
			name:        "403 forbidden is terminal",
			err:         mkErr(http.StatusForbidden, "session does not belong to your team"),
			wantTermini: true,
			wantSubstr:  "Access denied",
		},
		{
			name:        "404 not found is terminal",
			err:         mkErr(http.StatusNotFound, "session not found"),
			wantTermini: true,
			wantSubstr:  "Session not found",
		},
		{
			name:        "409 conflict is the V0 single-connection rejection",
			err:         mkErr(http.StatusConflict, "already attached on device abc-123 since 2026-06-24T10:00:00Z"),
			wantTermini: true,
			wantSubstr:  "Session already attached on another device",
		},
		{
			name:        "500 is transient",
			err:         mkErr(http.StatusInternalServerError, "internal"),
			wantTermini: false,
		},
		{
			name:        "503 is transient",
			err:         mkErr(http.StatusServiceUnavailable, "unavailable"),
			wantTermini: false,
		},
		{
			name:        "non-godo error is transient",
			err:         errors.New("network: connection reset"),
			wantTermini: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg, terminal := classifyStreamError(tc.err)
			assert.Equal(t, tc.wantTermini, terminal)
			if tc.wantSubstr != "" {
				assert.Contains(t, msg, tc.wantSubstr)
			}
		})
	}
}

func TestNextBackoff(t *testing.T) {
	cur := initialReconnectBackoff
	for i := 0; i < 10; i++ {
		cur = nextBackoff(cur)
		if cur > maxReconnectBackoff {
			t.Fatalf("backoff exceeded cap: %s > %s", cur, maxReconnectBackoff)
		}
	}
	assert.Equal(t, maxReconnectBackoff, cur)
}
