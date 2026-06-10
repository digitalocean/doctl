package deviceid

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// captureHandler records the value of HeaderName seen on the inbound request.
type captureHandler struct {
	gotHeader string
	gotPath   string
}

func (c *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.gotHeader = r.Header.Get(HeaderName)
	c.gotPath = r.URL.Path
	w.WriteHeader(http.StatusOK)
}

func newClient(t *testing.T, id string) (*http.Client, *captureHandler, func()) {
	t.Helper()
	h := &captureHandler{}
	srv := httptest.NewServer(h)
	c := &http.Client{Transport: NewTransport(http.DefaultTransport, id)}
	return c, h, srv.Close
}

func doGET(t *testing.T, c *http.Client, urlStr string) {
	t.Helper()
	resp, err := c.Get(urlStr)
	assert.NoError(t, err)
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func TestTransport_StampsHeaderInAgentsScope(t *testing.T) {
	cases := []string{
		"/v2/agents",                                // bare root
		"/v2/agents/",                               // trailing slash
		"/v2/agents/sessions",                       // POST create / GET list
		"/v2/agents/sessions/sess_abc",              // GET / DELETE single
		"/v2/agents/sessions/sess_abc/input",        // POST input
		"/v2/agents/sessions/sess_abc/stream",       // SSE
		"/v2/agents/sessions/sess_abc/hitl/req_xyz", // HITL resolve
		"/v2/agents/sessions/sess_abc/sandbox/exec", // sandbox exec
		"/v2/agents/templates",                      // any future hosted-agents sub-resource
	}
	for _, p := range cases {
		t.Run(strings.TrimPrefix(p, "/"), func(t *testing.T) {
			h := &captureHandler{}
			srv := httptest.NewServer(h)
			defer srv.Close()

			c := &http.Client{Transport: NewTransport(http.DefaultTransport, "test-uuid")}
			doGET(t, c, srv.URL+p)

			assert.Equal(t, "test-uuid", h.gotHeader, "header must be stamped on %s", p)
			assert.Equal(t, p, h.gotPath)
		})
	}
}

func TestTransport_OmitsHeaderOutOfScope(t *testing.T) {
	cases := []string{
		"/v2/droplets",
		"/v2/kubernetes/clusters",
		"/v2/registry",
		"/v2/gen-ai/agents",     // GradientAI agents, a different feature — must NOT leak
		"/v2/gen-ai/agents/abc", //
		"/v2/agentsfoo",         // prefix-only match — must NOT match base
		"/v2/agentsfoo/",        //
		"/",                     // root
	}
	for _, p := range cases {
		t.Run(strings.TrimPrefix(p, "/"), func(t *testing.T) {
			h := &captureHandler{}
			srv := httptest.NewServer(h)
			defer srv.Close()

			c := &http.Client{Transport: NewTransport(http.DefaultTransport, "test-uuid")}
			doGET(t, c, srv.URL+p)

			assert.Empty(t, h.gotHeader, "header must NOT leak to %s", p)
		})
	}
}

func TestTransport_EmptyIDReturnsBaseUnchanged(t *testing.T) {
	base := http.DefaultTransport
	got := NewTransport(base, "")
	assert.Same(t, base, got, "empty id must short-circuit to base transport")
}

func TestTransport_NilBaseFallsBackToDefault(t *testing.T) {
	got := NewTransport(nil, "test-uuid")
	tr, ok := got.(*Transport)
	if assert.True(t, ok) {
		assert.Same(t, http.DefaultTransport, tr.Base)
	}
}

func TestTransport_DoesNotMutateCallerHeaders(t *testing.T) {
	h := &captureHandler{}
	srv := httptest.NewServer(h)
	defer srv.Close()

	tr := NewTransport(http.DefaultTransport, "test-uuid")
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/v2/agents/sessions", nil)
	assert.NoError(t, err)

	original := req.Header.Clone()
	resp, err := tr.RoundTrip(req)
	assert.NoError(t, err)
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Caller's header map must remain untouched (no X-Device-UUID written
	// into it). The header is only present on the cloned request the
	// transport actually sent.
	assert.Equal(t, original, req.Header)
	assert.Equal(t, "test-uuid", h.gotHeader)
}
