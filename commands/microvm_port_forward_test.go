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
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestParseMicroVMForwardPair(t *testing.T) {
	for arg, want := range map[string]microVMForwardPair{
		"80":      {local: 80, remote: 80},
		"8080:80": {local: 8080, remote: 80},
		"0:5432":  {local: 0, remote: 5432},
	} {
		got, err := parseMicroVMForwardPair(arg)
		require.NoError(t, err, arg)
		assert.Equal(t, want, got, arg)
	}
	for arg, want := range map[string]string{
		"0":        "between 1 and 65535",
		"70000":    "between 1 and 65535",
		"50051":    "reserved for the MicroVM agent",
		"abc":      "invalid port pair",
		"x:80":     "invalid port pair",
		"70000:80": "invalid port pair",
	} {
		_, err := parseMicroVMForwardPair(arg)
		assert.ErrorContains(t, err, want, arg)
	}
}

func TestMicroVMTunnelWarnerSuppressesRepeats(t *testing.T) {
	w := &microVMTunnelWarner{remotePort: 80, seen: map[string]int{}}
	err := &microVMTunnelError{code: "agent_error", message: "connection refused"}
	for i := 0; i < 4; i++ {
		w.report(err)
	}
	assert.Equal(t, 4, w.seen[explainMicroVMTunnelErr(err, 80)])
}

func TestRunMicroVMPortForwardRefusesAgentPort(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testMicroVMID, "50051"}
		err := RunMicroVMPortForward(config)
		assert.ErrorContains(t, err, "reserved for the MicroVM agent")
	})
}

func microVMInState(s godo.MicroVMState) *do.MicroVM {
	return &do.MicroVM{MicroVM: &godo.MicroVM{ID: testMicroVMID, State: s}}
}

func TestEnsureMicroVMRunningResumesPaused(t *testing.T) {
	defer func(d time.Duration) { microVMPollInterval = d }(microVMPollInterval)
	microVMPollInterval = 0

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		gomock.InOrder(
			tm.microVMs.EXPECT().Get(testMicroVMID).Return(microVMInState(godo.MicroVMStatePaused), nil),
			tm.microVMs.EXPECT().Resume(testMicroVMID).Return(microVMInState(godo.MicroVMStateResuming), nil),
			tm.microVMs.EXPECT().Get(testMicroVMID).Return(microVMInState(godo.MicroVMStateResuming), nil),
			tm.microVMs.EXPECT().Get(testMicroVMID).Return(microVMInState(godo.MicroVMStateRunning), nil),
		)
		require.NoError(t, ensureMicroVMRunning(config, testMicroVMID))
	})
}

func TestEnsureMicroVMRunningRefusesFailed(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().Get(testMicroVMID).Return(microVMInState(godo.MicroVMStateFailed), nil)
		err := ensureMicroVMRunning(config, testMicroVMID)
		assert.ErrorContains(t, err, "is failed")
	})
}

// fakeMicroVMTunnel serves the port_forward WebSocket the way microdroplet-api
// does: upgrade, then one text control frame, then raw bytes.
func fakeMicroVMTunnel(t *testing.T, first string, echo bool) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		_ = ws.WriteMessage(websocket.TextMessage, []byte(first))
		if !echo {
			_ = ws.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
		for {
			mt, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.BinaryMessage {
				_ = ws.WriteMessage(websocket.BinaryMessage, data)
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/v2/microvms/" + testMicroVMID + "/port_forward?port=80"
}

func TestBridgeMicroVMConnCopiesBytes(t *testing.T) {
	wsURL := fakeMicroVMTunnel(t, `{"status":{"state":"connected"}}`, true)
	local, remote := net.Pipe()
	done := make(chan error, 1)
	go func() { done <- bridgeMicroVMConn(context.Background(), remote, wsURL, http.Header{}, nil) }()

	_, err := local.Write([]byte("ping"))
	require.NoError(t, err)
	buf := make([]byte, 4)
	_, err = io.ReadFull(local, buf)
	require.NoError(t, err)
	assert.Equal(t, "ping", string(buf))

	local.Close()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("bridge did not return after the local side closed")
	}
}

func TestBridgeMicroVMConnReportsAgentError(t *testing.T) {
	wsURL := fakeMicroVMTunnel(t,
		`{"error":{"code":"agent_error","message":"dial: connect: connection refused"}}`, false)
	local, remote := net.Pipe()
	defer local.Close()

	err := bridgeMicroVMConn(context.Background(), remote, wsURL, http.Header{}, nil)

	var te *microVMTunnelError
	require.True(t, errors.As(err, &te), "got %T: %v", err, err)
	msg := explainMicroVMTunnelErr(err, 80)
	assert.Contains(t, msg, "port 80: the MicroVM could not connect to that port")
	assert.Contains(t, msg, "connection refused")
}
