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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
			in:   `POST https://api.digitalocean.com/v2/agents/sessions: 400 invalid harness`,
			want: "invalid harness",
		},
		{
			in:   `GET https://api.digitalocean.com/v2/agents/sessions/sess_x: 404 (request "abc-123") session not found`,
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
			Request:    httptest.NewRequest(http.MethodPost, "https://api.digitalocean.com/v2/agents/sessions", nil),
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

	display := pretty.DisplayError()
	assert.Contains(t, display, "Session limit reached")
	assert.Contains(t, display, "team is at the limit of 4 active sessions")
	assert.Contains(t, display, "409")
	assert.NotContains(t, display, "POST https://")
	assert.NotContains(t, strings.ToLower(display), "error:")
}

func TestBeautifyAgentError_LocalValidation(t *testing.T) {
	err := errors.New(`POST https://api.digitalocean.com/v2/x: 400 --harness and --config-id are mutually exclusive`)
	out := beautifyAgentError(err)
	var pretty *agentPrettyError
	require.True(t, errors.As(out, &pretty))
	assert.Equal(t, "Invalid arguments", pretty.title)
	assert.Equal(t, "--harness and --config-id are mutually exclusive", pretty.reason)
	assert.NotContains(t, pretty.DisplayError(), "POST https://")
}

func TestBeautifyAgentError_Idempotent(t *testing.T) {
	first := beautifyAgentError(errors.New("no agent session goes by the name foo"))
	second := beautifyAgentError(first)
	assert.Same(t, first, second)
}

func TestBeautifyAgentError_SilentExitPassthrough(t *testing.T) {
	assert.Equal(t, ErrExitSilently, beautifyAgentError(ErrExitSilently))
}
