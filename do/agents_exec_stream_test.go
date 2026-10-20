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

package do

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadExecSSE_HappyPath(t *testing.T) {
	stdoutB64 := base64.StdEncoding.EncodeToString([]byte("hello\n"))
	stderrB64 := base64.StdEncoding.EncodeToString([]byte("warn\n"))
	body := strings.Join([]string{
		"event: stdout",
		`data: {"data":"` + stdoutB64 + `"}`,
		"",
		"event: stderr",
		`data: {"data":"` + stderrB64 + `"}`,
		"",
		": keepalive",
		"",
		"event: exit",
		`data: {"exit_code":9,"duration_ms":12}`,
		"",
	}, "\n")

	var frames []SandboxExecEvent
	err := readExecSSE(strings.NewReader(body), func(ev SandboxExecEvent) error {
		frames = append(frames, ev)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, frames, 3)
	assert.Equal(t, []byte("hello\n"), frames[0].Stdout)
	assert.Equal(t, []byte("warn\n"), frames[1].Stderr)
	require.NotNil(t, frames[2].ExitCode)
	assert.Equal(t, 9, *frames[2].ExitCode)
}

func TestReadExecSSE_ErrorEvent(t *testing.T) {
	body := "event: error\ndata: {\"message\":\"exec timed out after 3s (requested)\"}\n\n"
	err := readExecSSE(strings.NewReader(body), func(SandboxExecEvent) error { return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out after 3s (requested)")
}

func TestEmitBufferedExecAsStream(t *testing.T) {
	payload, err := json.Marshal(godo.HostedAgentSandboxExecResponse{
		ExitCode: 2,
		Stdout:   "out",
		Stderr:   "err",
	})
	require.NoError(t, err)

	var frames []SandboxExecEvent
	require.NoError(t, emitBufferedExecAsStream(bytes.NewReader(payload), func(ev SandboxExecEvent) error {
		frames = append(frames, ev)
		return nil
	}))
	require.Len(t, frames, 3)
	assert.Equal(t, []byte("out"), frames[0].Stdout)
	assert.Equal(t, []byte("err"), frames[1].Stderr)
	require.NotNil(t, frames[2].ExitCode)
	assert.Equal(t, 2, *frames[2].ExitCode)
}

func TestStreamExecInSandbox_FallsBackToJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.Header.Get("Accept"), "text/event-stream")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(godo.HostedAgentSandboxExecResponse{
			ExitCode: 0,
			Stdout:   "buffered\n",
		})
	}))
	t.Cleanup(server.Close)

	client, err := godo.New(server.Client(), godo.SetBaseURL(server.URL+"/"))
	require.NoError(t, err)
	svc := NewHostedAgentsService(client).(*hostedAgentsService)

	var stdout bytes.Buffer
	var code *int
	err = svc.StreamExecInSandbox(context.Background(), "sess-1", &godo.HostedAgentSandboxExecRequest{
		Argv: []string{"echo", "buffered"},
	}, func(ev SandboxExecEvent) error {
		if len(ev.Stdout) > 0 {
			_, _ = stdout.Write(ev.Stdout)
		}
		if ev.ExitCode != nil {
			c := *ev.ExitCode
			code = &c
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "buffered\n", stdout.String())
	require.NotNil(t, code)
	assert.Equal(t, 0, *code)
}

func TestStreamExecInSandbox_SSE(t *testing.T) {
	stdoutB64 := base64.StdEncoding.EncodeToString([]byte("live\n"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)
		_, _ = io.WriteString(w, "event: stdout\ndata: {\"data\":\""+stdoutB64+"\"}\n\n")
		flusher.Flush()
		_, _ = io.WriteString(w, "event: exit\ndata: {\"exit_code\":0,\"duration_ms\":1}\n\n")
		flusher.Flush()
	}))
	t.Cleanup(server.Close)

	client, err := godo.New(server.Client(), godo.SetBaseURL(server.URL+"/"))
	require.NoError(t, err)
	svc := NewHostedAgentsService(client).(*hostedAgentsService)

	var stdout bytes.Buffer
	err = svc.StreamExecInSandbox(context.Background(), "sess-1", &godo.HostedAgentSandboxExecRequest{
		Argv: []string{"echo", "live"},
	}, func(ev SandboxExecEvent) error {
		if len(ev.Stdout) > 0 {
			_, _ = stdout.Write(ev.Stdout)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "live\n", stdout.String())
}

// Pre-header failures (504 timeout, 429 capacity) must surface as typed API
// errors rather than being misread as a buffered JSON body.
func TestStreamExecInSandbox_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Header.Get("Accept"), "text/event-stream")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = io.WriteString(w, `{"id":"req","message":"exec timed out after 3s (requested)"}`)
	}))
	t.Cleanup(server.Close)

	client, err := godo.New(server.Client(), godo.SetBaseURL(server.URL+"/"))
	require.NoError(t, err)
	svc := NewHostedAgentsService(client).(*hostedAgentsService)

	err = svc.StreamExecInSandbox(context.Background(), "sess-1", &godo.HostedAgentSandboxExecRequest{
		Argv: []string{"sleep", "120"},
	}, func(SandboxExecEvent) error { return nil })
	require.Error(t, err)
	var er *godo.ErrorResponse
	require.ErrorAs(t, err, &er)
	assert.Equal(t, http.StatusGatewayTimeout, er.Response.StatusCode)
	assert.Contains(t, er.Message, "exec timed out after 3s (requested)")
}
