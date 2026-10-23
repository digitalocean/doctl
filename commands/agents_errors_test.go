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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/doctl/internal/ui"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripGodoTransportNoise(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			in:   `POST https://api.digitalocean.com/v2/harness-runtime/sessions: 400 invalid harness`,
			want: "invalid harness",
		},
		{
			in:   `GET https://api.digitalocean.com/v2/harness-runtime/sessions/sess_x: 404 (request "abc-123") session not found`,
			want: "session not found",
		},
		{
			in:   "plain local validation error",
			want: "plain local validation error",
		},
		{
			in:   "",
			want: "",
		},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, stripGodoTransportNoise(tc.in))
	}
}

func TestBeautifyAgentError_APIResponse(t *testing.T) {
	er := &godo.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusConflict,
			Request:    httptest.NewRequest(http.MethodPost, "https://api.digitalocean.com/v2/harness-runtime/sessions", nil),
		},
		Message: "team is at the limit of 4 active sessions",
	}

	out := beautifyAgentError(er)
	var pretty *agentPrettyError
	require.True(t, errors.As(out, &pretty))
	assert.Equal(t, "Session limit reached", pretty.title)
	assert.Equal(t, "team is at the limit of 4 active sessions", pretty.reason)
	assert.Equal(t, http.StatusConflict, pretty.status)
	assert.Contains(t, pretty.tips, "doctl harness-runtime list")

	display := formatStructuredError(ui.Plain(&bytes.Buffer{}, &bytes.Buffer{}), pretty)
	assert.Contains(t, display, "Error: Session limit reached")
	assert.Contains(t, display, "team is at the limit of 4 active sessions")
	assert.Contains(t, display, "409")
	assert.NotContains(t, display, "POST https://")
}

// Sending input to a bare sandbox is the one 409 a user reaches by doing the
// obvious thing, so it gets a card rather than the HTTP reason phrase. The
// exact server copy is the fixture: matching on a substring of it is what makes
// this fragile, and a test that never sees the real string would not notice.
// Both manifest spellings are fixtures, since either may appear in the copy.
func TestBeautifyAgentError_NoAgentConflictGetsItsOwnCard(t *testing.T) {
	for _, spelling := range []string{"none", "sandbox"} {
		t.Run(spelling, func(t *testing.T) {
			er := &godo.ErrorResponse{
				Response: &http.Response{
					StatusCode: http.StatusConflict,
					Request:    httptest.NewRequest(http.MethodPost, "https://api.digitalocean.com/v2/harness-runtime/sessions/x/input", nil),
				},
				Message: "this session runs no agent (agent: " + spelling + "); drive it with exec, workspace upload/download, or port-forward instead of sending input",
			}

			out := beautifyAgentError(er)
			var pretty *agentPrettyError
			require.True(t, errors.As(out, &pretty))
			assert.Equal(t, "This session runs no agent", pretty.title)
			assert.NotEqual(t, "Conflict", pretty.title, "the HTTP reason phrase is not a title")
			assert.Contains(t, pretty.tips, agentCLI+" exec <session> -- <command>")
			assert.Contains(t, pretty.tips, agentCLI+" port-forward <session> <port>")
		})
	}
}

// The narrower 409s are matched before it, so adding the no-agent case must not
// swallow them — "no agent" is a loose substring and these share a status.
func TestBeautifyAgentError_NoAgentCardDoesNotSwallowOther409s(t *testing.T) {
	cases := map[string]string{
		"team is at the limit of 4 active sessions": "Session limit reached",
		"run is terminal":             "Session run has ended",
		"session is already attached": "Session already attached elsewhere",
	}
	for msg, wantTitle := range cases {
		t.Run(wantTitle, func(t *testing.T) {
			title, _ := agentErrorTitleAndTips(msg, http.StatusConflict)
			assert.Equal(t, wantTitle, title)
		})
	}
}

func TestBeautifyAgentError_LocalValidation(t *testing.T) {
	err := errors.New(`POST https://api.digitalocean.com/v2/x: 400 --harness and --config-id are mutually exclusive`)
	out := beautifyAgentError(err)
	var pretty *agentPrettyError
	require.True(t, errors.As(out, &pretty))
	assert.Equal(t, "Invalid arguments", pretty.title)
	assert.Equal(t, "--harness and --config-id are mutually exclusive", pretty.reason)
	assert.NotContains(t, formatStructuredError(ui.Plain(&bytes.Buffer{}, &bytes.Buffer{}), pretty), "POST https://")
}

// A prepay 402 must never fall through to the generic "Request failed" card,
// and must name the one thing that resolves it.
func TestBeautifyAgentError_PrepayBlocked(t *testing.T) {
	out := beautifyAgentError(harnessAPIErr(http.StatusPaymentRequired, prepayBlockedWireMessage))

	var pretty *agentPrettyError
	require.True(t, errors.As(out, &pretty))
	assert.Equal(t, prepayBlockedTitle, pretty.title)
	assert.Equal(t, http.StatusPaymentRequired, pretty.status)

	display := formatStructuredError(ui.Plain(&bytes.Buffer{}, &bytes.Buffer{}), pretty)
	assert.Contains(t, display, "Error: "+prepayBlockedTitle)
	assert.Contains(t, display, "402")
	assert.Contains(t, display, prepayTopUpURL)
	assert.Contains(t, display, prepayBalanceCmd)
	assert.Contains(t, display, "work is saved")
	assert.NotContains(t, display, "POST https://")
}

// The gate failing closed is a retry, not a top-up. Telling a solvent user to
// add funds because a mirror lookup timed out would be actively misleading.
func TestBeautifyAgentError_PrepayUnknown(t *testing.T) {
	out := beautifyAgentError(harnessAPIErr(http.StatusServiceUnavailable, prepayUnknownWireMessage))

	var pretty *agentPrettyError
	require.True(t, errors.As(out, &pretty))
	assert.Equal(t, prepayUnknownTitle, pretty.title)

	display := formatStructuredError(ui.Plain(&bytes.Buffer{}, &bytes.Buffer{}), pretty)
	assert.Contains(t, display, "503")
	assert.Contains(t, display, "Retry in a moment")
	assert.NotContains(t, display, "Add funds")
}

func TestBeautifyAgentError_Idempotent(t *testing.T) {
	first := beautifyAgentError(errors.New("no agent session goes by the name foo"))
	second := beautifyAgentError(first)
	assert.Same(t, first, second)
}

func TestBeautifyAgentError_SilentExitPassthrough(t *testing.T) {
	assert.Equal(t, ErrExitSilently, beautifyAgentError(ErrExitSilently))
}

// MARSOHS-1627: agentspec unknown-field errors must not show raw \u00a0 escapes
// or an internal contracts/ path customers cannot open.
func TestSanitizeAgentAPIMessage_UnknownFieldWhitespace(t *testing.T) {
	in := `agentspec: unknown field "\u00a0\u00a0HARNESS_INFERENCE_API_KEY" (unknown or misspelled fields are rejected; see contracts/agent.flat.yaml)`
	got := sanitizeAgentAPIMessage(in)
	assert.Contains(t, got, `unknown field "HARNESS_INFERENCE_API_KEY"`)
	assert.Contains(t, got, "unexpected whitespace")
	assert.Contains(t, got, "copy-paste")
	assert.NotContains(t, got, `\u00a0`)
	assert.NotContains(t, got, "contracts/")
	assert.Contains(t, got, "unknown or misspelled fields are rejected")
}

func TestBeautifyAgentError_AgentspecUnknownField(t *testing.T) {
	er := harnessAPIErr(http.StatusBadRequest,
		`agentspec: unknown field "\u00a0\u00a0HARNESS_INFERENCE_API_KEY" (unknown or misspelled fields are rejected; see contracts/agent.flat.yaml)`)

	out := beautifyAgentError(er)
	var pretty *agentPrettyError
	require.True(t, errors.As(out, &pretty))
	assert.Equal(t, "Invalid request", pretty.title)
	assert.Contains(t, pretty.reason, `unknown field "HARNESS_INFERENCE_API_KEY"`)
	assert.Contains(t, pretty.reason, "unexpected whitespace")
	assert.NotContains(t, pretty.reason, `\u00a0`)
	assert.NotContains(t, pretty.reason, "contracts/")
	assert.Contains(t, pretty.tips, "doctl harness-runtime validate")

	display := formatStructuredError(ui.Plain(&bytes.Buffer{}, &bytes.Buffer{}), pretty)
	assert.Contains(t, display, "HARNESS_INFERENCE_API_KEY")
	assert.NotContains(t, display, "contracts/")
	assert.NotContains(t, display, `\u00a0`)
}

// EX15 returns HTTP 504 with an applied-budget reason. That must not look like
// a generic "Service temporarily unavailable" retry tip.
func TestBeautifyAgentError_ExecTimeout504(t *testing.T) {
	out := beautifyAgentError(harnessAPIErr(http.StatusGatewayTimeout, "exec timed out after 3s (requested)"))

	var pretty *agentPrettyError
	require.True(t, errors.As(out, &pretty))
	assert.Equal(t, "Exec timed out", pretty.title)
	assert.Equal(t, "exec timed out after 3s (requested)", pretty.reason)
	assert.Equal(t, http.StatusGatewayTimeout, pretty.status)
	assert.Contains(t, pretty.tips, "Pass a larger --timeout")
	assert.NotContains(t, pretty.tips, "Retry in a moment")
}

func TestBeautifyAgentError_ExecStreamCapacity429(t *testing.T) {
	out := beautifyAgentError(harnessAPIErr(http.StatusTooManyRequests, "sandbox exec stream capacity exhausted, retry later"))

	var pretty *agentPrettyError
	require.True(t, errors.As(out, &pretty))
	assert.Equal(t, "Exec stream capacity exhausted", pretty.title)
	assert.Contains(t, pretty.tips, "Wait a moment and retry")
}

func TestBeautifyAgentError_SSEExecTimeoutMessage(t *testing.T) {
	out := beautifyAgentError(errors.New("exec timed out after 900s (default)"))

	var pretty *agentPrettyError
	require.True(t, errors.As(out, &pretty))
	assert.Equal(t, "Exec timed out", pretty.title)
	assert.Contains(t, pretty.tips, "Pass a larger --timeout")
	assert.NotContains(t, pretty.tips, "--wait-timeout")
}
