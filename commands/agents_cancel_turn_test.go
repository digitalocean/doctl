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
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	domocks "github.com/digitalocean/doctl/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func drainFrames(t *testing.T, body string, thinking *thinkingState) {
	t.Helper()
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)
	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()
	var buf bytes.Buffer
	drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, thinking, nil, &tokenDeduper{})
}

func TestDrainStream_OpenTurnIsCancellableByItsRunID(t *testing.T) {
	thinking := newThinkingState(io.Discard)
	drainFrames(t,
		promptFrame("evt-1", "run-1", godo.HostedAgentEventKindRunStarted, `{}`)+
			promptTokenFrame("evt-2", "run-1", "working", false),
		thinking)

	assert.Equal(t, "run-1", thinking.claimTurnCancel())
	assert.Equal(t, "", thinking.claimTurnCancel(), "a second Esc must not send a second cancel")
}

func TestDrainStream_EndedTurnIsNotCancellable(t *testing.T) {
	thinking := newThinkingState(io.Discard)
	drainFrames(t,
		promptFrame("evt-1", "run-1", godo.HostedAgentEventKindRunStarted, `{}`)+
			promptFrame("evt-2", "run-1", godo.HostedAgentEventKindRunFailed, `{"code":7,"message":"turn cancelled by me"}`),
		thinking)

	assert.Equal(t, "", thinking.claimTurnCancel())
}

func TestHandlePendingEscTimeout_CancelsTurnOnlyOnEmptyPrompt(t *testing.T) {
	cases := []struct {
		name       string
		line       string
		pendingID  string
		hookResult bool
		wantCalled bool
		wantResult bool
		wantLine   string
	}{
		{name: "text typed clears it and leaves the turn alone", line: "draft", wantLine: ""},
		{name: "empty prompt cancels the turn", hookResult: true, wantCalled: true, wantResult: true},
		{name: "empty prompt with nothing to cancel", hookResult: false, wantCalled: true, wantResult: false},
		{name: "pending approval is never cancelled from", pendingID: "hitl_1", hookResult: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pending := &pendingHITL{}
			if tc.pendingID != "" {
				pending.set(tc.pendingID)
			}
			state := newAttachState(io.Discard, pending)
			state.display.setRaw(true)
			state.lineBuf = []byte(tc.line)
			state.cursor = len(tc.line)
			called := false
			state.cancelTurn = func() bool {
				called = true
				return tc.hookResult
			}

			state.escSeq = []byte{0x1b}
			got := state.handlePendingEscTimeout()

			if tc.line != "" {
				assert.True(t, got)
			} else {
				assert.Equal(t, tc.wantResult, got)
			}
			assert.Equal(t, tc.wantCalled, called)
			assert.Equal(t, tc.wantLine, string(state.lineBuf))
		})
	}
}

func TestHandleAttachByte_CtrlCCancelsTheTurnThenDetaches(t *testing.T) {
	cases := []struct {
		name       string
		line       string
		hook       bool
		startTurn  string
		wantStops  []bool
		wantCancel int
		wantLine   string
	}{
		{name: "open turn: first cancels, second detaches", hook: true, startTurn: "run-1", wantStops: []bool{false, true}, wantCancel: 1},
		{name: "draft survives the cancel", hook: true, startTurn: "run-1", line: "next", wantStops: []bool{false}, wantCancel: 1, wantLine: "next"},
		{name: "no turn open detaches", hook: true, wantStops: []bool{true}},
		{name: "no hook installed detaches", startTurn: "run-1", wantStops: []bool{true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				thinking := newThinkingState(io.Discard)
				if tc.startTurn != "" {
					thinking.startTurn(tc.startTurn)
				}
				state := newAttachState(io.Discard, &pendingHITL{})
				state.display.setRaw(true)
				state.lineBuf = []byte(tc.line)
				state.cursor = len(tc.line)
				cancels := 0
				if tc.hook {
					state.cancelTurn = func() bool {
						if thinking.claimTurnCancel() == "" {
							return false
						}
						cancels++
						return true
					}
				}

				for i, want := range tc.wantStops {
					stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 0x03, state, nil, thinking)
					assert.NoError(t, err)
					assert.Equal(t, want, stop, "Ctrl-C #%d", i+1)
				}
				assert.Equal(t, tc.wantCancel, cancels)
				assert.Equal(t, tc.wantLine, string(state.lineBuf))
			})
		})
	}
}

func TestRunAgentsCancel(t *testing.T) {
	cases := []struct {
		name    string
		runID   string
		result  do.HostedAgentCancelTurnResult
		err     error
		wantOut string
		wantErr string
	}{
		{
			name:    "no run id cancels the turn in flight and names it",
			result:  do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnAcked, RunID: "run-live"},
			wantOut: "Cancelled run run-live on session " + testPromptSessionID,
		},
		{
			name:    "a named run is passed through",
			runID:   "run-7",
			result:  do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnAcked, RunID: "run-7"},
			wantOut: "Cancelled run run-7",
		},
		{
			name:    "nothing running is not an error",
			result:  do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnNoTurn},
			wantOut: "Nothing is running on session " + testPromptSessionID,
		},
		{
			name:    "a named run that already finished is not an error",
			runID:   "run-7",
			result:  do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnNoTurn, RunID: "run-7"},
			wantOut: "Run run-7 is not running",
		},
		{
			name:    "an agent that cannot cancel is an error: the turn keeps going",
			result:  do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnUnsupported},
			wantErr: "can't cancel a turn; it keeps running",
		},
		{
			name:    "a delivery failure is an error",
			err:     errors.New("409 session is paused"),
			wantErr: "session is paused",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				var out bytes.Buffer
				config.Out = &out
				tm.hostedAgents.EXPECT().CancelTurn(testPromptSessionID, tc.runID).Return(tc.result, tc.err)
				config.Args = []string{testPromptSessionID}
				if tc.runID != "" {
					config.Doit.Set(config.NS, doctl.ArgAgentCancelRunID, "  "+tc.runID+" ")
				}

				err := RunAgentsCancel(config)
				if tc.wantErr != "" {
					assert.ErrorContains(t, err, tc.wantErr)
					return
				}
				assert.NoError(t, err)
				assert.Contains(t, out.String(), tc.wantOut)
			})
		})
	}
}

func TestRunAgentsCancel_JSON(t *testing.T) {
	prev := Output
	Output = "json"
	t.Cleanup(func() { Output = prev })

	cases := []struct {
		name    string
		result  do.HostedAgentCancelTurnResult
		want    string
		wantErr bool
	}{
		{
			name:   "names the turn the server resolved",
			result: do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnAcked, RunID: "run-live"},
			want:   `{"session_id":"` + testPromptSessionID + `","run_id":"run-live","outcome":"acked"}`,
		},
		{
			name:   "nothing running omits the run id and still succeeds",
			result: do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnNoTurn},
			want:   `{"session_id":"` + testPromptSessionID + `","outcome":"no_turn"}`,
		},
		{
			name:    "unsupported prints the document and fails",
			result:  do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnUnsupported},
			want:    `{"session_id":"` + testPromptSessionID + `","outcome":"unsupported"}`,
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				var out bytes.Buffer
				config.Out = &out
				tm.hostedAgents.EXPECT().CancelTurn(testPromptSessionID, "").Return(tc.result, nil)
				config.Args = []string{testPromptSessionID}

				err := RunAgentsCancel(config)
				if tc.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
				assert.JSONEq(t, tc.want, out.String())
			})
		})
	}
}

func TestRequestTurnCancel(t *testing.T) {
	cases := []struct {
		name          string
		outcome       do.HostedAgentCancelTurnOutcome
		err           error
		wantOut       string
		wantReclaim   string
		wantHintAfter bool
	}{
		{name: "acked prints nothing and holds the claim", outcome: do.HostedAgentCancelTurnAcked, wantReclaim: ""},
		{name: "no turn prints nothing", outcome: do.HostedAgentCancelTurnNoTurn, wantReclaim: ""},
		{
			name:    "unsupported explains and retires the hint",
			outcome: do.HostedAgentCancelTurnUnsupported,
			wantOut: "This agent can't cancel a turn; it keeps running. Ctrl-C or Ctrl-D detaches.\n",
		},
		{
			name:          "a failed request lets Esc try again",
			err:           errors.New("boom"),
			wantOut:       "cancel failed: boom\n",
			wantReclaim:   "run-1",
			wantHintAfter: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc := domocks.NewMockHostedAgentsService(ctrl)
			svc.EXPECT().CancelTurn("sess_x", "run-1").Return(do.HostedAgentCancelTurnResult{Outcome: tc.outcome, RunID: "run-1"}, tc.err)

			thinking := newThinkingState(io.Discard)
			thinking.startTurn("run-1")
			assert.Equal(t, "run-1", thinking.claimTurnCancel())

			var out bytes.Buffer
			requestTurnCancel(&out, svc, "sess_x", "run-1", thinking)

			assert.Equal(t, tc.wantOut, out.String())
			assert.Equal(t, tc.wantReclaim, thinking.claimTurnCancel())
			if tc.outcome == do.HostedAgentCancelTurnUnsupported {
				assert.Equal(t, defaultThinkingLabel, thinking.frameLabel())
			}
			if tc.wantHintAfter {
				assert.Contains(t, thinking.frameLabel(), cancellingTurnTag, "re-claimed by the assertion above")
			}
		})
	}
}

func TestThinkingFrameLabel(t *testing.T) {
	thinking := newThinkingState(io.Discard)
	assert.Equal(t, defaultThinkingLabel, thinking.frameLabel(), "no turn open")

	thinking.startTurn("run-1")
	assert.Equal(t, defaultThinkingLabel+" · "+cancelTurnHint, thinking.frameLabel())

	thinking.claimTurnCancel()
	assert.Equal(t, defaultThinkingLabel+" · "+cancellingTurnTag, thinking.frameLabel())

	thinking.startTurn("run-2")
	assert.Equal(t, defaultThinkingLabel+" · "+cancelTurnHint, thinking.frameLabel(), "a new turn is cancellable again")

	thinking.setTurnRunning(false)
	assert.Equal(t, defaultThinkingLabel, thinking.frameLabel(), "turn ended")

	thinking.startTurn("")
	assert.Equal(t, defaultThinkingLabel, thinking.frameLabel(), "no run_id to cancel by")
	assert.Equal(t, "", thinking.claimTurnCancel())

	thinking.startTurn("run-3")
	thinking.markCancelUnsupported()
	assert.Equal(t, defaultThinkingLabel, thinking.frameLabel(), "unsupported agents get no hint")
	assert.Equal(t, "", thinking.claimTurnCancel())
}
