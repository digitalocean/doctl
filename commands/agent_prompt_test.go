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
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// A UUID-shaped ref so resolveSessionRef short-circuits instead of listing.
const testPromptSessionID = "01a01e58-6209-7c75-8b31-6cb80f7301ff"

const testPromptRunID = "run_abc123"

// capturePrompt points stdout, stderr, and the exit call at the test instead of
// the process, and returns the recorded exit code by pointer.
func capturePrompt(t *testing.T, config *CmdConfig) (stdout, stderr *bytes.Buffer, exitCode *int) {
	t.Helper()
	stdout, stderr = &bytes.Buffer{}, &bytes.Buffer{}
	config.Out = stdout

	prevErr, prevExit := promptStderr, promptExit
	promptStderr = stderr
	code := noExit
	promptExit = func(c int) { code = c }
	t.Cleanup(func() { promptStderr, promptExit = prevErr, prevExit })

	return stdout, stderr, &code
}

// promptFrame builds one SSE frame carrying a run ID, which the plain sseFrame
// helper omits and this command's run filtering depends on.
func promptFrame(eventID, runID string, kind godo.HostedAgentEventKind, dataJSON string) string {
	return fmt.Sprintf(
		"id: %s\ndata: {\"event_id\":\"%s\",\"run_id\":\"%s\",\"type\":\"%s\",\"data\":%s}\n\n",
		eventID, eventID, runID, kind, dataJSON,
	)
}

func promptTokenFrame(eventID, runID, text string, reasoning bool) string {
	data, _ := json.Marshal(map[string]any{"text": text, "is_reasoning": reasoning})
	return promptFrame(eventID, runID, godo.HostedAgentEventKindTokenChunk, string(data))
}

// promptStream serves the given SSE body and returns a service mock wired to
// stream it, with the session reported READY so the runner does not resume it.
func promptStream(t *testing.T, tm *tcMocks, sse string) {
	t.Helper()
	stubReconnectSleep(t)

	srv := httptest.NewServer(hostedAgentSSEHandler(sse, nil))
	t.Cleanup(srv.Close)
	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	tm.hostedAgents.EXPECT().
		GetSession(testPromptSessionID).
		Return(&do.HostedAgentSession{
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID: testPromptSessionID,
				Status:    godo.HostedAgentSessionStatusReady,
			},
		}, nil).AnyTimes()

	tm.hostedAgents.EXPECT().
		StreamSession(gomock.Any(), testPromptSessionID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
			stream, _, err := client.HostedAgents.StreamSession(context.Background(), testPromptSessionID, opt)
			return stream, err
		}).AnyTimes()
}

// expectSendInput asserts the text that reached the API and hands back a run ID.
func expectSendInput(tm *tcMocks, want string) *gomock.Call {
	return tm.hostedAgents.EXPECT().
		SendInput(testPromptSessionID, gomock.Any()).
		DoAndReturn(func(_ string, req *godo.HostedAgentSendInputRequest) (*godo.HostedAgentSendInputResponse, error) {
			if want != "" {
				if req.Text != want {
					return nil, fmt.Errorf("unexpected prompt text %q, want %q", req.Text, want)
				}
			}
			return &godo.HostedAgentSendInputResponse{RunID: testPromptRunID}, nil
		})
}

// completedRun is the minimal happy-path stream: one answer chunk, then done.
func completedRun(text string) string {
	return promptTokenFrame("evt-1", testPromptRunID, text, false) +
		promptFrame("evt-2", testPromptRunID, godo.HostedAgentEventKindRunCompleted,
			`{"total_tokens_in":12,"total_tokens_out":3}`)
}

// prompt is a flat verb like exec: a session-scoped action, not a sub-resource
// with its own CRUD.
func TestAgentsPromptIsAFlatVerb(t *testing.T) {
	cmd := Agents()
	require.NotNil(t, cmd)

	var prompt *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "prompt" {
			prompt = c
		}
	}
	require.NotNil(t, prompt, "prompt must be registered on the root tree")
	assert.False(t, prompt.HasSubCommands(), "prompt takes text to send, not subcommands")
	assert.Contains(t, prompt.Aliases, "ask")
}

// The session comes first, matching every other session command in the tree.
func TestAgentsPromptTakesSessionFirst(t *testing.T) {
	cmd := Agents()
	for _, c := range cmd.Commands() {
		if c.Name() == "prompt" {
			assert.True(t, strings.HasPrefix(c.Use, "prompt <session>"),
				"the session must be the first positional, as it is for show/exec/upload; got %q", c.Use)
		}
	}
}

// The split between the streams is the contract that makes this usable
// headlessly: stdout carries the answer alone so `$(...)` captures it exactly.
func TestAgentsPromptAnswerOnStdoutProgressOnStderr(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, stderr, exitCode := capturePrompt(t, config)

		sse := promptTokenFrame("evt-1", testPromptRunID, "Paris", false) +
			promptFrame("evt-2", testPromptRunID, godo.HostedAgentEventKindToolCallStarted,
				`{"tool_call_id":"t1","name":"bash","arguments":{"command":"ls"}}`) +
			promptFrame("evt-3", testPromptRunID, godo.HostedAgentEventKindRunCompleted,
				`{"total_tokens_in":12,"total_tokens_out":3}`)
		promptStream(t, tm, sse)
		expectSendInput(tm, "What is the capital of France?")

		config.Args = []string{testPromptSessionID, "What is the capital of France?"}

		require.NoError(t, RunAgentsPrompt(config))
		assert.Equal(t, "Paris\n", stdout.String(), "stdout is the answer and nothing else")
		assert.Contains(t, stderr.String(), "run complete", "the usage summary belongs on stderr")
		assert.NotContains(t, stdout.String(), "run complete")
		assert.Equal(t, noExit, *exitCode, "a completed run must not force an exit status")
	})
}

// An unquoted prompt is the reason the text is variadic; forgetting the quotes
// should still send the whole sentence.
func TestAgentsPromptJoinsUnquotedArgs(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		capturePrompt(t, config)
		promptStream(t, tm, completedRun("Paris"))
		expectSendInput(tm, "What is the capital of France?")

		config.Args = []string{testPromptSessionID, "What", "is", "the", "capital", "of", "France?"}
		require.NoError(t, RunAgentsPrompt(config))
	})
}

func TestAgentsPromptReadsStdinForDash(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		capturePrompt(t, config)
		promptStream(t, tm, completedRun("ok"))
		expectSendInput(tm, "review the diff")

		prev := promptStdin
		promptStdin = strings.NewReader("review the diff\n")
		t.Cleanup(func() { promptStdin = prev })

		config.Args = []string{testPromptSessionID, "-"}
		require.NoError(t, RunAgentsPrompt(config))
	})
}

// Subscribe-then-send. Sending first would race the subscription: the run's
// opening events could land before the stream is live, and the command would
// then wait out its timeout for a terminal event it had already missed.
func TestAgentsPromptSubscribesBeforeSending(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		capturePrompt(t, config)
		stubReconnectSleep(t)

		srv := httptest.NewServer(hostedAgentSSEHandler(completedRun("Paris"), nil))
		t.Cleanup(srv.Close)
		client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
		require.NoError(t, err)

		tm.hostedAgents.EXPECT().
			GetSession(testPromptSessionID).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: testPromptSessionID,
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil).AnyTimes()

		subscribed := false
		tm.hostedAgents.EXPECT().
			StreamSession(gomock.Any(), testPromptSessionID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				subscribed = true
				stream, _, err := client.HostedAgents.StreamSession(context.Background(), testPromptSessionID, opt)
				return stream, err
			}).AnyTimes()

		tm.hostedAgents.EXPECT().
			SendInput(testPromptSessionID, gomock.Any()).
			DoAndReturn(func(_ string, _ *godo.HostedAgentSendInputRequest) (*godo.HostedAgentSendInputResponse, error) {
				assert.True(t, subscribed, "the stream must be open before the prompt is sent")
				return &godo.HostedAgentSendInputResponse{RunID: testPromptRunID}, nil
			})

		config.Args = []string{testPromptSessionID, "hi"}
		require.NoError(t, RunAgentsPrompt(config))
	})
}

// A session can be driven from elsewhere while this runs, and the stream
// replays events from before the send. Only this run's text is the answer.
func TestAgentsPromptIgnoresOtherRuns(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, _, _ := capturePrompt(t, config)

		sse := promptTokenFrame("evt-0", "run_someone_else", "NOT MINE", false) +
			promptTokenFrame("evt-1", testPromptRunID, "Paris", false) +
			promptFrame("evt-2", "run_someone_else", godo.HostedAgentEventKindRunCompleted, `{}`) +
			promptFrame("evt-3", testPromptRunID, godo.HostedAgentEventKindRunCompleted, `{}`)
		promptStream(t, tm, sse)
		expectSendInput(tm, "")

		config.Args = []string{testPromptSessionID, "hi"}
		require.NoError(t, RunAgentsPrompt(config))
		assert.Equal(t, "Paris\n", stdout.String(),
			"another run's output must not contaminate this answer, nor end the wait early")
	})
}

// Reasoning is not the answer, so it must stay off stdout unless asked for,
// and even then it goes to stderr.
func TestAgentsPromptExcludesReasoningFromStdout(t *testing.T) {
	sse := promptTokenFrame("evt-1", testPromptRunID, "thinking hard", true) +
		promptTokenFrame("evt-2", testPromptRunID, "Paris", false) +
		promptFrame("evt-3", testPromptRunID, godo.HostedAgentEventKindRunCompleted, `{}`)

	t.Run("omitted by default", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			stdout, stderr, _ := capturePrompt(t, config)
			promptStream(t, tm, sse)
			expectSendInput(tm, "")

			config.Args = []string{testPromptSessionID, "hi"}
			require.NoError(t, RunAgentsPrompt(config))
			assert.Equal(t, "Paris\n", stdout.String())
			assert.NotContains(t, stderr.String(), "thinking hard")
		})
	})

	t.Run("on stderr with --include-reasoning", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			stdout, stderr, _ := capturePrompt(t, config)
			promptStream(t, tm, sse)
			expectSendInput(tm, "")

			config.Args = []string{testPromptSessionID, "hi"}
			config.Doit.Set(config.NS, doctl.ArgAgentPromptIncludeReasoning, true)
			require.NoError(t, RunAgentsPrompt(config))
			assert.Equal(t, "Paris\n", stdout.String(), "reasoning must never reach stdout")
			assert.Contains(t, stderr.String(), "thinking hard")
		})
	})
}

// Blocking on an approval would hang until the timeout and look identical to a
// slow agent, so a request with no policy stops the command and says so.
func TestAgentsPromptFailsFastOnApprovalWithoutPolicy(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		capturePrompt(t, config)

		sse := promptFrame("evt-1", testPromptRunID, godo.HostedAgentEventKindHITLRequested,
			`{"hitl_id":"hitl_1","command":"rm -rf /"}`)
		promptStream(t, tm, sse)
		expectSendInput(tm, "")

		config.Args = []string{testPromptSessionID, "clean up"}

		err := RunAgentsPrompt(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--on-hitl", "the error must name the flag that fixes it")
		assert.Contains(t, err.Error(), "hitl_1", "and the request ID needed to resolve it out of band")
	})
}

// A data-form elicitation needs values --on-hitl cannot supply, even when
// set: it must fail loudly and point at `approve ... --content`, not
// silently submit an empty answer.
func TestAgentsPromptDataFormFailsLoudlyEvenWithPolicy(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		capturePrompt(t, config)

		sse := promptFrame("evt-1", testPromptRunID, godo.HostedAgentEventKindHITLRequested,
			`{"hitl_id":"hitl_1","details":{"kind":"mcp_elicitation","mode":"form","serverName":"jira","requestedSchema":{"type":"object","properties":{"site_url":{"type":"string"}},"required":["site_url"]}}}`)
		promptStream(t, tm, sse)
		expectSendInput(tm, "")

		config.Args = []string{testPromptSessionID, "connect jira"}
		config.Doit.Set(config.NS, doctl.ArgAgentOnHITL, "approve")

		err := RunAgentsPrompt(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--content")
		assert.Contains(t, err.Error(), "hitl_1")
	})
}

func TestAgentsPromptAutoResolvesApprovalWithPolicy(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, stderr, _ := capturePrompt(t, config)

		sse := promptFrame("evt-1", testPromptRunID, godo.HostedAgentEventKindHITLRequested,
			`{"hitl_id":"hitl_1","command":"ls"}`) +
			promptTokenFrame("evt-2", testPromptRunID, "done", false) +
			promptFrame("evt-3", testPromptRunID, godo.HostedAgentEventKindRunCompleted, `{}`)
		promptStream(t, tm, sse)
		expectSendInput(tm, "")

		tm.hostedAgents.EXPECT().
			ResolveHITL(testPromptSessionID, "hitl_1", gomock.Any()).
			DoAndReturn(func(_, _ string, req *godo.HostedAgentResolveHITLRequest) error {
				assert.Equal(t, godo.HostedAgentHITLOutcomeApprove, req.Outcome)
				return nil
			})

		config.Args = []string{testPromptSessionID, "go"}
		config.Doit.Set(config.NS, doctl.ArgAgentOnHITL, "approve")

		require.NoError(t, RunAgentsPrompt(config))
		assert.Contains(t, stderr.String(), "auto-approve")
	})
}

// A run that ran and failed is the agent's result, not a doctl error: the
// status is reproduced as the exit code with nothing printed on top.
func TestAgentsPromptFailedRunExitsNonZero(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, stderr, exitCode := capturePrompt(t, config)

		sse := promptFrame("evt-1", testPromptRunID, godo.HostedAgentEventKindRunFailed,
			`{"code":3,"message":"the harness crashed"}`)
		promptStream(t, tm, sse)
		expectSendInput(tm, "")

		config.Args = []string{testPromptSessionID, "hi"}

		err := RunAgentsPrompt(config)
		assert.ErrorIs(t, err, ErrExitSilently)
		assert.Equal(t, 1, *exitCode)
		assert.Contains(t, stderr.String(), "the harness crashed")
	})
}

// "Show me this session" and "it happens to be asleep" should not be two
// commands an unattended script has to sequence itself.
func TestAgentsPromptResumesPausedSession(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		capturePrompt(t, config)
		stubReconnectSleep(t)

		srv := httptest.NewServer(hostedAgentSSEHandler(completedRun("Paris"), nil))
		t.Cleanup(srv.Close)
		client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
		require.NoError(t, err)

		paused := &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
			SessionID: testPromptSessionID, Status: godo.HostedAgentSessionStatusPaused,
		}}
		ready := &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
			SessionID: testPromptSessionID, Status: godo.HostedAgentSessionStatusReady,
		}}
		gomock.InOrder(
			tm.hostedAgents.EXPECT().GetSession(testPromptSessionID).Return(paused, nil),
			tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).Return(nil),
			tm.hostedAgents.EXPECT().GetSession(testPromptSessionID).Return(ready, nil),
		)
		tm.hostedAgents.EXPECT().
			StreamSession(gomock.Any(), testPromptSessionID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				stream, _, err := client.HostedAgents.StreamSession(context.Background(), testPromptSessionID, opt)
				return stream, err
			}).AnyTimes()
		expectSendInput(tm, "hi")

		config.Args = []string{testPromptSessionID, "hi"}
		require.NoError(t, RunAgentsPrompt(config))
	})
}

func TestAgentsPromptJSONOutput(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, _, _ := capturePrompt(t, config)
		promptStream(t, tm, completedRun("Paris"))
		expectSendInput(tm, "")

		prev := Output
		Output = "json"
		t.Cleanup(func() { Output = prev })

		config.Args = []string{testPromptSessionID, "hi"}
		require.NoError(t, RunAgentsPrompt(config))

		var got map[string]any
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &got), "a bare object, not an array")
		assert.Equal(t, "Paris", got["text"])
		assert.Equal(t, testPromptRunID, got["run_id"])
		assert.Equal(t, promptStatusCompleted, got["status"])
	})
}

// A timeout is not a failure: 124 (GNU timeout's status) lets a caller tell
// "still working" apart from "the agent failed" without parsing stderr.
func TestAgentsPromptTimeoutExits124(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, stderr, exitCode := capturePrompt(t, config)

		expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		t.Cleanup(cancel)

		col := &promptCollector{sessionID: testPromptSessionID}
		col.answer.WriteString("partial answer")

		err := col.report(config, expired)
		assert.ErrorIs(t, err, ErrExitSilently)
		assert.Equal(t, promptTimeoutExit, *exitCode)
		assert.Equal(t, "partial answer\n", stdout.String(), "whatever the agent did say is still worth printing")
		assert.Contains(t, stderr.String(), "timed out")
	})
}

// A stream that ends without a terminal event is not a completed run, so it
// must not report success to an `&&` chain.
func TestAgentsPromptIncompleteRunExitsNonZero(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, stderr, exitCode := capturePrompt(t, config)

		col := &promptCollector{sessionID: testPromptSessionID}
		err := col.report(config, context.Background())

		assert.ErrorIs(t, err, ErrExitSilently)
		assert.Equal(t, 1, *exitCode)
		assert.Contains(t, stderr.String(), "did not finish")
	})
}

func TestPromptTextFrom(t *testing.T) {
	t.Run("rejects an empty prompt", func(t *testing.T) {
		_, err := promptTextFrom([]string{"   "})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("keeps interior whitespace", func(t *testing.T) {
		got, err := promptTextFrom([]string{"fix", "the", "test"})
		require.NoError(t, err)
		assert.Equal(t, "fix the test", got)
	})
}
