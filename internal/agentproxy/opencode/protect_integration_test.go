package opencode

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/agentproxy"
	"github.com/digitalocean/doctl/internal/agentproxy/agentproxytest"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ProtectLoopbackHTTP + Facade is how start-proxy wires the OpenCode listener.
func TestProtectedFacadeBlocksUnauthenticatedPrompt(t *testing.T) {
	h := agentproxytest.New(t, testSessionID)
	h.QueueRun("run-protect")
	client, err := godo.New(nil, godo.SetBaseURL(h.Server.URL+"/"))
	require.NoError(t, err)

	password, err := agentproxy.NewProxyPassword()
	require.NoError(t, err)

	facade := &Facade{
		SessionID: testSessionID,
		Sessions:  do.NewHostedAgentsService(client),
		Dir:       "/tmp/ws",
	}
	srv := httptest.NewServer(agentproxy.ProtectLoopbackHTTP(facade, password))
	t.Cleanup(srv.Close)

	body := `{"parts":[{"type":"text","text":"hello"}]}`

	resp, err := http.Post(srv.URL+"/session/any/prompt_async", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Nil(t, h.LastInput())

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/session/any/prompt_async", strings.NewReader(body))
	require.NoError(t, err)
	req.SetBasicAuth(agentproxy.OpenCodeProxyBasicUser, password)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Nil(t, h.LastInput())

	req, err = http.NewRequest(http.MethodPost, srv.URL+testSessionURL("/prompt_async"), strings.NewReader(body))
	require.NoError(t, err)
	req.SetBasicAuth(agentproxy.OpenCodeProxyBasicUser, password)
	req.Header.Set("Content-Type", "text/plain")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	assert.Nil(t, h.LastInput())

	req, err = http.NewRequest(http.MethodPost, srv.URL+testSessionURL("/prompt_async"), strings.NewReader(body))
	require.NoError(t, err)
	req.SetBasicAuth(agentproxy.OpenCodeProxyBasicUser, password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Nil(t, h.LastInput())

	req, err = http.NewRequest(http.MethodPost, srv.URL+testSessionURL("/prompt_async"), strings.NewReader(body))
	require.NoError(t, err)
	req.SetBasicAuth(agentproxy.OpenCodeProxyBasicUser, password)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.NotNil(t, h.LastInput())
	assert.Equal(t, "hello", h.LastInput().Text)
}

func TestProtectedFacadeBlocksUnauthenticatedHistory(t *testing.T) {
	h := agentproxytest.New(t, testSessionID)
	client, err := godo.New(nil, godo.SetBaseURL(h.Server.URL+"/"))
	require.NoError(t, err)

	password := "test-password"
	srv := httptest.NewServer(agentproxy.ProtectLoopbackHTTP(&Facade{
		SessionID: testSessionID,
		Sessions:  do.NewHostedAgentsService(client),
		Dir:       "/tmp/ws",
	}, password))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + testSessionURL("/message"))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	req, err := http.NewRequest(http.MethodGet, srv.URL+testSessionURL("/message"), nil)
	require.NoError(t, err)
	req.SetBasicAuth(agentproxy.OpenCodeProxyBasicUser, password)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
