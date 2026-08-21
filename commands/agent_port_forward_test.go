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
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseForwardPair(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want forwardPair
	}{
		{"bare port forwards both ends", "8080", forwardPair{local: 8080, remote: 8080}},
		{"explicit pair", "12222:2222", forwardPair{local: 12222, remote: 2222}},
		{"zero local lets the OS pick", "0:9119", forwardPair{local: 0, remote: 9119}},
		{"lowest allowed remote", "1024", forwardPair{local: 1024, remote: 1024}},
		{"highest allowed remote", "65535", forwardPair{local: 65535, remote: 65535}},
		// Empty local is treated as "same port", not as an OS-assigned one --
		// only an explicit 0 does that. kubectl reads ":<remote>" as random.
		{"empty local mirrors the remote", ":8080", forwardPair{local: 8080, remote: 8080}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseForwardPair(tt.arg)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseForwardPairErrors(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{"empty", ""},
		{"not a number", "http"},
		{"remote below the privileged-port floor", "80"},
		{"remote just below the floor", "1023"},
		{"remote above the range", "65536"},
		{"remote negative", "-1"},
		{"missing remote", "8080:"},
		{"local not a number", "abc:8080"},
		{"local above the range", "70000:8080"},
		{"local negative", "-1:8080"},
		{"too many segments", "1:2:3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseForwardPair(tt.arg)
			assert.Error(t, err)
		})
	}
}

// setAPIURL overrides the global --api-url viper key for one test and restores
// whatever was there before. hostedAgentsWSURL reads it directly, and viper is
// process-global, so these tests cannot run in parallel.
func setAPIURL(t *testing.T, v string) {
	t.Helper()
	prev := viper.GetString("api-url")
	viper.Set("api-url", v)
	t.Cleanup(func() { viper.Set("api-url", prev) })
}

func TestHostedAgentsWSURL(t *testing.T) {
	tests := []struct {
		name   string
		apiURL string
		want   string
	}{
		{
			// Unset --api-url falls back to doctl.HostedAgentsAPIURL, whose
			// trailing slash must not survive into the joined path.
			name:   "default hosted-agents host",
			apiURL: "",
			want:   "wss://ohr-agent.do-ai.run/v2/agents/sessions/sess-1/port-forward/9119",
		},
		{
			name:   "preview host via --api-url",
			apiURL: "https://ohr-agent.do-ai-test.run",
			want:   "wss://ohr-agent.do-ai-test.run/v2/agents/sessions/sess-1/port-forward/9119",
		},
		{
			name:   "https upgrades to wss",
			apiURL: "https://api.example.com/",
			want:   "wss://api.example.com/v2/agents/sessions/sess-1/port-forward/9119",
		},
		{
			name:   "http downgrades to ws for a local dev stack",
			apiURL: "http://127.0.0.1:8080",
			want:   "ws://127.0.0.1:8080/v2/agents/sessions/sess-1/port-forward/9119",
		},
		{
			name:   "base path prefix is preserved",
			apiURL: "https://api.example.com/edge/",
			want:   "wss://api.example.com/edge/v2/agents/sessions/sess-1/port-forward/9119",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setAPIURL(t, tt.apiURL)
			got, err := hostedAgentsWSURL("sess-1", 9119)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHostedAgentsWSURLErrors(t *testing.T) {
	tests := []struct {
		name   string
		apiURL string
	}{
		{"unparseable url", "://nope"},
		{"unsupported scheme", "ftp://api.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setAPIURL(t, tt.apiURL)
			_, err := hostedAgentsWSURL("sess-1", 9119)
			assert.Error(t, err)
		})
	}
}

func TestRejectionMessage(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "api error json yields just the message",
			body: `{"id": "bad_gateway", "message": "could not reach guest port"}`,
			want: "could not reach guest port",
		},
		{
			name: "proxy plain text passes through",
			body: "upstream connect error or disconnect/reset before headers. reset reason: protocol error",
			want: "upstream connect error or disconnect/reset before headers. reset reason: protocol error",
		},
		{
			name: "multi-line body collapses to one line",
			body: "<html>\n  <body>502 Bad Gateway</body>\n</html>",
			want: "<html> <body>502 Bad Gateway</body> </html>",
		},
		{"empty body yields nothing", "", ""},
		{"whitespace-only body yields nothing", "   \n\t ", ""},
		{"json without a message falls back to raw", `{"id": "x"}`, `{"id": "x"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, rejectionMessage(tt.body))
		})
	}
}

func TestRejectionMessageTruncatesLongBodies(t *testing.T) {
	got := rejectionMessage(strings.Repeat("a", rejectionMessageMax+50))
	assert.Equal(t, strings.Repeat("a", rejectionMessageMax)+"...", got)
}

func TestServerRejection(t *testing.T) {
	dialErr := errors.New("websocket: bad handshake")

	t.Run("no response reports the dial failure", func(t *testing.T) {
		err := serverRejection(nil, "", dialErr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dialing tunnel")
	})

	t.Run("server message replaces the opaque handshake error", func(t *testing.T) {
		resp := &http.Response{Status: "502 Bad Gateway", StatusCode: 502}
		err := serverRejection(resp, `{"message": "could not reach guest port"}`, dialErr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "502 Bad Gateway")
		assert.Contains(t, err.Error(), "could not reach guest port")
		// The gorilla error carries no diagnostic value once we have the
		// server's own words; it should not crowd them out.
		assert.NotContains(t, err.Error(), "bad handshake")
	})

	t.Run("empty body falls back to the handshake error", func(t *testing.T) {
		resp := &http.Response{Status: "503 Service Unavailable", StatusCode: 503}
		err := serverRejection(resp, "", dialErr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "503 Service Unavailable")
		assert.Contains(t, err.Error(), "bad handshake")
	})
}

func TestHandshakeBody(t *testing.T) {
	t.Run("nil response reads nothing", func(t *testing.T) {
		assert.Equal(t, "", handshakeBody(nil))
	})

	t.Run("nil body reads nothing", func(t *testing.T) {
		assert.Equal(t, "", handshakeBody(&http.Response{}))
	})

	t.Run("body is returned and closed", func(t *testing.T) {
		body := &closeTrackingBody{Reader: strings.NewReader("could not reach guest port")}
		resp := &http.Response{Body: body}
		assert.Equal(t, "could not reach guest port", handshakeBody(resp))
		assert.True(t, body.closed, "handshake body must be closed")
	})

	t.Run("oversized body is bounded", func(t *testing.T) {
		resp := &http.Response{
			Body: io.NopCloser(strings.NewReader(strings.Repeat("a", handshakeBodyLimit*2))),
		}
		assert.Len(t, handshakeBody(resp), handshakeBodyLimit)
	})
}

func TestFormatHeaderRedactsAuthorization(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer super-secret-token")
	h.Set("User-Agent", "doctl/test")

	got := formatHeader(h)

	assert.NotContains(t, got, "super-secret-token")
	assert.Contains(t, got, "Authorization: [redacted]")
	assert.Contains(t, got, "User-Agent: doctl/test")
}

// TestBridgeLocalConnSurfacesServerMessage exercises the real gorilla dial:
// the unit tests above cover the parsing, but only this proves the message
// survives the handshake path instead of being replaced by "bad handshake".
func TestBridgeLocalConnSurfacesServerMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, `{"id":"bad_gateway","message":"could not reach guest port"}`)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") +
		"/v2/agents/sessions/sess-1/port-forward/9119"

	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()

	err := bridgeLocalConn(context.Background(), remote, wsURL, http.Header{}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "502 Bad Gateway")
	assert.Contains(t, err.Error(), "could not reach guest port")
}

// closeTrackingBody records whether the reader was closed.
type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}
