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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/deviceid"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/digitalocean/godo"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
			// Unset --api-url falls back to the same default the rest of the
			// agents commands use, so the tunnel and the session that created
			// it stay on one host. The trailing slash must not survive into
			// the joined path.
			name:   "default api host",
			apiURL: "",
			want:   "wss://api.digitalocean.com/v2/agents/sessions/sess-1/port-forward/9119",
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
		{"unparsable url", "://nope"},
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

func TestTunnelHeader(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		deviceID string
		want     map[string]string
	}{
		{
			name:     "token and device id",
			token:    "tok",
			deviceID: "DEV-UUID",
			want: map[string]string{
				"Authorization":     "Bearer tok",
				deviceid.HeaderName: "DEV-UUID",
			},
		},
		{
			name:     "unauthenticated context still carries the device id",
			deviceID: "DEV-UUID",
			want:     map[string]string{deviceid.HeaderName: "DEV-UUID"},
		},
		{
			// deviceid.Get returns "" when the lookup fails or the user opted
			// out with DOCTL_DISABLE_DEVICE_ID; the header must be absent
			// rather than empty.
			name:  "device id unavailable",
			token: "tok",
			want:  map[string]string{"Authorization": "Bearer tok"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tunnelHeader(tt.token, tt.deviceID)

			assert.Len(t, got, len(tt.want))
			for k, v := range tt.want {
				assert.Equal(t, v, got.Get(k))
			}
		})
	}
}

func TestRedactDeviceID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty stays empty", in: "", want: ""},
		{name: "whitespace only", in: "   ", want: ""},
		{name: "short values are fully masked", in: "12345678", want: "***"},
		{
			name: "long values keep a correlatable prefix and suffix",
			in:   "564D8C1A-1234-5678-9ABC-DEF012345678",
			want: "564D***5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, redactDeviceID(tt.in))
		})
	}
}

func TestFormatHeaderRedactsDeviceID(t *testing.T) {
	h := http.Header{}
	h.Set(deviceid.HeaderName, "564D8C1A-1234-5678-9ABC-DEF012345678")

	got := formatHeader(h)

	assert.NotContains(t, got, "564D8C1A-1234-5678-9ABC-DEF012345678")
	// Header.Set canonicalizes the key to X-Device-Uuid, matching what the
	// deviceid transport already puts on the wire for every other agents call.
	assert.Contains(t, got, http.CanonicalHeaderKey(deviceid.HeaderName)+": 564D***5678")
}

// TestBridgeLocalConnSendsDeviceHeader proves the header survives the gorilla
// dial rather than only being present in the map we hand it.
func TestBridgeLocalConnSendsDeviceHeader(t *testing.T) {
	got := make(chan http.Header, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.Header.Clone()
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") +
		"/v2/agents/sessions/sess-1/port-forward/9119"

	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()

	header := tunnelHeader("tok", "DEV-UUID")
	_ = bridgeLocalConn(context.Background(), remote, wsURL, header, nil)

	select {
	case h := <-got:
		assert.Equal(t, "DEV-UUID", h.Get(deviceid.HeaderName))
		assert.Equal(t, "Bearer tok", h.Get("Authorization"))
	default:
		t.Fatal("server never received the upgrade request")
	}
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

func TestExplainTunnelClose(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "dial failed names the port and what to do",
			err:  &websocket.CloseError{Code: 4002, Text: "guest port unreachable"},
			want: "port 8080: nothing is listening on that port inside the session — start the process that serves it, then retry",
		},
		{
			name: "session ended points at restarting the session",
			err:  &websocket.CloseError{Code: 4001, Text: "session ended"},
			want: "port 8080: the session ended while forwarding — start it again, then re-run doctl harness-runtime port-forward",
		},
		{
			name: "server error is retryable",
			err:  &websocket.CloseError{Code: 4011, Text: "tunnel closed"},
			want: "port 8080: the tunnel failed inside the service — retry; if it keeps happening, re-run with --trace",
		},
		{
			name: "abnormal closure is a network drop",
			err:  &websocket.CloseError{Code: websocket.CloseAbnormalClosure, Text: ""},
			want: "port 8080: the tunnel dropped without closing cleanly — check your network; new connections still work",
		},
		{
			name: "an unknown close code keeps the server's own words",
			err:  &websocket.CloseError{Code: 4099, Text: "something new"},
			want: "port 8080: connection closed: websocket: close 4099: something new",
		},
		{
			name: "a non-close error is passed through",
			err:  errors.New("read tcp: connection reset by peer"),
			want: "port 8080: connection closed: read tcp: connection reset by peer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, explainTunnelClose(tt.err, 8080))
		})
	}
}

// TestTunnelWarnerSuppressesRepeats covers the case that prompted the message
// rewrite: one browser page load opens several connections, and an unstarted
// server made each of them print the same line.
func TestTunnelWarnerSuppressesRepeats(t *testing.T) {
	var buf bytes.Buffer
	withUIEnv(t, ui.Plain(&buf, &buf))

	w := newTunnelWarner(8080)
	unreachable := &websocket.CloseError{Code: 4002, Text: "guest port unreachable"}
	for range 4 {
		w.report(unreachable)
	}

	out := buf.String()
	assert.Equal(t, 1, strings.Count(out, "nothing is listening on that port"))
	assert.Equal(t, 1, strings.Count(out, "further identical warnings are suppressed"))

	// A different failure is different information, so it still prints.
	w.report(&websocket.CloseError{Code: 4001, Text: "session ended"})
	assert.Contains(t, buf.String(), "the session ended while forwarding")
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

func TestEnsureSessionAwakeForPortForward_ResumesPaused(t *testing.T) {
	prev := sessionReadyPollInterval
	sessionReadyPollInterval = time.Millisecond
	t.Cleanup(func() { sessionReadyPollInterval = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		gomock.InOrder(
			tm.hostedAgents.EXPECT().
				GetSession("sess_paused").
				Return(&do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_paused",
						Name:      "bug044-test",
						Status:    godo.HostedAgentSessionStatusPaused,
					},
				}, nil),
			tm.hostedAgents.EXPECT().ResumeSession("sess_paused").Return(nil),
			tm.hostedAgents.EXPECT().
				GetSession("sess_paused").
				Return(&do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_paused",
						Name:      "bug044-test",
						Status:    godo.HostedAgentSessionStatusReady,
					},
				}, nil),
		)

		var buf bytes.Buffer
		config.Out = &buf
		_, err := ensureSessionAwakeForPortForward(config, "sess_paused")
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "is paused — resuming")
	})
}

func TestEnsureSessionAwakeForPortForward_ReadyIsNoop(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			GetSession("sess_ready").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_ready",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)
		// No ResumeSession expectation: gomock fails if it is called.

		_, err := ensureSessionAwakeForPortForward(config, "sess_ready")
		require.NoError(t, err)
	})
}

func TestEnsureSessionAwakeForPortForward_RejectsTerminal(t *testing.T) {
	for _, status := range []godo.HostedAgentSessionStatus{
		godo.HostedAgentSessionStatusDestroyed,
		godo.HostedAgentSessionStatusDestroying,
		godo.HostedAgentSessionStatusFailed,
	} {
		t.Run(humanSessionStatus(status), func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.hostedAgents.EXPECT().
					GetSession("sess_gone").
					Return(&do.HostedAgentSession{
						HostedAgentSession: &godo.HostedAgentSession{
							SessionID: "sess_gone",
							Name:      "gone",
							Status:    status,
						},
					}, nil)

				_, err := ensureSessionAwakeForPortForward(config, "sess_gone")
				require.Error(t, err)
				assert.Contains(t, err.Error(), "cannot be port-forwarded")
			})
		})
	}
}

// The Canvas port of an openhands session is served as HTTP: a page request
// crosses the tunnel to the guest as it came (path, Host, body), a WebSocket
// upgrade is spliced through, and /api/automation never leaves the machine.
func TestCanvasForwardHandler_ProxiesThroughTheTunnelAndAnswersAutomationLocally(t *testing.T) {
	upgrader := websocket.Upgrader{}
	// The guest: Canvas's front, answering a page and echoing on a socket.
	guest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/canvas/":
			fmt.Fprintf(w, "canvas page host=%s", r.Host)
		case "/api/automation/health":
			http.Error(w, "the guest must never see this", http.StatusTeapot)
		case "/sockets/events":
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			_ = conn.WriteMessage(mt, append([]byte("echo:"), msg...))
		default:
			http.NotFound(w, r)
		}
	}))
	defer guest.Close()
	guestAddr := strings.TrimPrefix(guest.URL, "http://")

	// harness-api's port-forward: one WebSocket per TCP connection, binary
	// frames relayed byte for byte to the guest port.
	tunnel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer T0K" {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		tcp, err := net.Dial("tcp", guestAddr)
		if err != nil {
			return
		}
		defer tcp.Close()
		go func() {
			buf := make([]byte, 32<<10)
			for {
				n, err := tcp.Read(buf)
				if n > 0 {
					if werr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
						return
					}
				}
				if err != nil {
					_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					return
				}
			}
		}()
		for {
			mt, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.BinaryMessage {
				if _, err := tcp.Write(data); err != nil {
					return
				}
			}
		}
	}))
	defer tunnel.Close()
	wsURL := "ws" + strings.TrimPrefix(tunnel.URL, "http") + "/v2/agents/sessions/sess_1/port-forward/8000"

	automation := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status":"ok","from":"doctl","path":%q}`, r.URL.Path)
	})
	local := httptest.NewServer(canvasForwardHandler(wsURL, tunnelHeader("T0K", ""), nil, nil, automation))
	defer local.Close()

	resp, err := http.Get(local.URL + "/canvas/")
	require.NoError(t, err)
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "canvas page host="+strings.TrimPrefix(local.URL, "http://"), string(page), "the guest sees the browser's Host, as the raw tunnel would carry it")

	resp, err = http.Get(local.URL + "/api/automation/health")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `"from":"doctl"`)

	conn, wsResp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(local.URL, "http")+"/sockets/events", nil)
	require.NoError(t, err, "upgrade through the tunnel")
	defer conn.Close()
	assert.Equal(t, http.StatusSwitchingProtocols, wsResp.StatusCode)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("hello")))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, "echo:hello", string(msg))
}

// Without an automation handler (a non-loopback bind) the prefix is plain
// proxied traffic: the guest answers it, as a raw tunnel would let it.
func TestCanvasForwardHandler_NoAutomationHandlerProxiesThePrefix(t *testing.T) {
	h := canvasForwardHandler("ws://127.0.0.1:1/never", nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8000/api/automation/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadGateway, rec.Code, "the request went for the tunnel, which is not there in this test")
}
