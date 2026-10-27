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
	"net/url"
	"strings"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/deviceid"
	"github.com/digitalocean/godo"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// The proxy's one job: the browser's request reaches the public URL with the
// token and device id added, the path and query intact, the Host rewritten,
// and nothing of the token comes back.
func TestCanvasProxyAddsTokenAndForwardsPath(t *testing.T) {
	var got *http.Request
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		w.Header().Set("X-Upstream", "yes")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>canvas</html>"))
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	require.NoError(t, err)

	local := httptest.NewServer(newCanvasProxy(target, "T0K", "dev-1"))
	defer local.Close()

	req, err := http.NewRequest(http.MethodGet, local.URL+"/canvas/assets/app.js?v=1", nil)
	require.NoError(t, err)
	req.Header.Set("Proxy-Connection", "keep-alive")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "yes", resp.Header.Get("X-Upstream"))
	require.NotNil(t, got)
	assert.Equal(t, "Bearer T0K", got.Header.Get("Authorization"))
	assert.Equal(t, "dev-1", got.Header.Get(deviceid.HeaderName))
	assert.Equal(t, "/canvas/assets/app.js", got.URL.Path)
	assert.Equal(t, "v=1", got.URL.RawQuery)
	assert.Equal(t, target.Host, got.Host, "Host is the public hostname, not localhost")
	assert.Empty(t, got.Header.Get("Proxy-Connection"), "hop-by-hop headers are not forwarded")
}

// Canvas streams events over a WebSocket on the same origin; the proxy must
// carry the upgrade through with the token and splice the connection.
func TestCanvasProxyPassesWebSocketUpgradeWithToken(t *testing.T) {
	upgrader := websocket.Upgrader{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer T0K" {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
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
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	require.NoError(t, err)

	local := httptest.NewServer(newCanvasProxy(target, "T0K", ""))
	defer local.Close()

	wsURL := "ws" + strings.TrimPrefix(local.URL, "http") + "/sockets/events/abc"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err, "upgrade through the proxy")
	defer conn.Close()
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("hello")))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, "echo:hello", string(msg))
}

// An environment without public URLs is a known state: the error says so and
// gives the port-forward route to the same Canvas.
func TestCanvasIngressError_NotImplementedPointsAtPortForward(t *testing.T) {
	err := canvasIngressError("my-session", &godo.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusNotImplemented},
		Message:  "not implemented",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not publish a public URL")
	assert.Contains(t, err.Error(), "port-forward my-session 8000:8000")
	assert.Contains(t, err.Error(), "http://127.0.0.1:8000/canvas")
}

func TestCanvasIngressError_OtherErrorsPassThrough(t *testing.T) {
	err := canvasIngressError("s", errors.New("boom"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestResolveCanvasTarget_ReadyURL(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().GetSessionIngress("sess_1").Return(&do.HostedAgentSessionIngress{
			SessionID: "sess_1",
			Port:      8000,
			URL:       "https://abc.ric1.sandbox.ondigitalocean.com",
			State:     do.IngressReady,
		}, nil)
		target, err := resolveCanvasTarget(config.HostedAgents(), "my-session", "sess_1")
		require.NoError(t, err)
		assert.Equal(t, "https", target.Scheme)
		assert.Equal(t, "abc.ric1.sandbox.ondigitalocean.com", target.Host)
	})
}

func TestResolveCanvasTarget_NotReadyIsAnError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().GetSessionIngress("sess_1").Return(&do.HostedAgentSessionIngress{
			URL:   "https://abc.ric1.sandbox.ondigitalocean.com",
			State: "INGRESS_URL_STATE_PROVISIONING",
		}, nil)
		_, err := resolveCanvasTarget(config.HostedAgents(), "my-session", "sess_1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not ready yet")
		assert.Contains(t, err.Error(), "PROVISIONING")
	})
}

func TestResolveCanvasTarget_IngressNotEnabled(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().GetSessionIngress("sess_1").Return(nil, &godo.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusNotImplemented},
		})
		_, err := resolveCanvasTarget(config.HostedAgents(), "my-session", "sess_1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "port-forward my-session 8000:8000")
	})
}

func TestEnsureSessionAwake_RejectsTerminalWithVerb(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().GetSession("sess_done").Return(&do.HostedAgentSession{
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID: "sess_done",
				Name:      "old",
				Status:    godo.HostedAgentSessionStatusDestroyed,
			},
		}, nil)
		err := ensureSessionAwake(config, "sess_done", "served to Agent Canvas")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be served to Agent Canvas")
	})
}

var _ = gomock.Any // keep gomock imported for parity with sibling tests
