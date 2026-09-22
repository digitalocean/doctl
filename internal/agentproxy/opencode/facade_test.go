package opencode

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The attach burst captured from a real `opencode attach` (TestedVersion).
// Every route the TUI hits at startup must answer 200 with valid JSON —
// a 404 here is exactly the "new client version wants more" drift signal.
func TestAttachBurstRoutesAnswer(t *testing.T) {
	srv, _ := newBridgedFacade(t)

	routes := []string{
		"/global/health",
		"/path",
		"/project/current",
		"/project/global/directories",
		"/config",
		"/config/providers",
		"/provider",
		"/provider/auth",
		"/agent",
		"/command",
		"/lsp",
		"/mcp",
		"/formatter",
		"/vcs",
		"/session/status",
		"/session?start=1785667630974&path=",
		"/experimental/capabilities",
		"/experimental/console",
		"/experimental/resource",
		"/experimental/workspace",
		"/experimental/workspace/status",
		"/api/location",
		"/api/skill",
		"/api/reference",
		"/api/provider",
		"/api/integration",
		"/api/model",
		"/api/command",
		"/api/agent",
	}
	for _, route := range routes {
		resp, err := http.Get(srv.URL + route)
		require.NoError(t, err, route)
		assert.Equal(t, http.StatusOK, resp.StatusCode, route)
		var v any
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&v), route)
		resp.Body.Close()
	}
}

func TestHealthReportsTestedVersion(t *testing.T) {
	srv, _ := newBridgedFacade(t)

	resp, err := http.Get(srv.URL + "/global/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	var health struct {
		Healthy bool   `json:"healthy"`
		Version string `json:"version"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&health))
	assert.True(t, health.Healthy)
	assert.Equal(t, TestedVersion, health.Version)
}

// The global stream's first frame must be server.connected in the wrapped
// payload envelope — the TUI treats it as the liveness signal.
func TestGlobalEventStreamsServerConnectedFirst(t *testing.T) {
	// A wired harness with a hanging stream: the SSE handler now runs the
	// live event loop after server.connected, so it needs a real (fake)
	// harness to stream from.
	srv, h := newBridgedFacade(t)
	h.HangStreamAfterEvents(true)

	resp, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(line, "data: "), "SSE frame: %q", line)

	var frame struct {
		Payload struct {
			ID         string         `json:"id"`
			Type       string         `json:"type"`
			Properties map[string]any `json:"properties"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(line), "data: ")), &frame))
	assert.Equal(t, "server.connected", frame.Payload.Type)
	assert.NotEmpty(t, frame.Payload.ID)
}

// One event-stream consumer at a time: same rule as the WebSocket
// transport's single-client slot, enforced on the stream rather than the
// listener since plain REST requests may overlap freely.
func TestSecondEventStreamConsumerConflicts(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.HangStreamAfterEvents(true)

	first, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer first.Body.Close()
	// Wait for the stream to actually be established before racing it.
	_, err = bufio.NewReader(first.Body).ReadString('\n')
	require.NoError(t, err)

	second, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer second.Body.Close()
	assert.Equal(t, http.StatusConflict, second.StatusCode)
}

func TestShareIsRefused(t *testing.T) {
	srv, _ := newBridgedFacade(t)

	resp, err := http.Post(srv.URL+"/session/sess-1/share", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUnknownRouteIs404(t *testing.T) {
	srv, _ := newBridgedFacade(t)

	resp, err := http.Get(srv.URL + "/no/such/route")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// The synthetic catalog must agree with itself: the default selection in
// /config/providers points at the one provider/model pair every other
// catalog route advertises.
func TestSyntheticCatalogIsConsistent(t *testing.T) {
	srv, _ := newBridgedFacade(t)

	resp, err := http.Get(srv.URL + "/config/providers")
	require.NoError(t, err)
	defer resp.Body.Close()
	var cfg struct {
		Default   map[string]string `json:"default"`
		Providers []struct {
			ID     string                    `json:"id"`
			Models map[string]map[string]any `json:"models"`
		} `json:"providers"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&cfg))
	require.Len(t, cfg.Providers, 1)
	assert.Equal(t, providerID, cfg.Providers[0].ID)
	assert.Contains(t, cfg.Providers[0].Models, cfg.Default[providerID])
}
