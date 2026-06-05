package deviceid

import (
	"net/http"
	"strings"
)

// HeaderName is coordinated with the harness-api server. Rename in lockstep.
const HeaderName = "X-Device-UUID"

// agentsSessionsBase mirrors godo's hostedAgentsSessionsBasePath.
const agentsSessionsBase = "/v2/agents/sessions"

// Transport stamps HeaderName on requests under agentsSessionsBase.
type Transport struct {
	Base http.RoundTripper
	ID   string
}

// NewTransport returns base unchanged when id is empty so callers can chain
// unconditionally.
func NewTransport(base http.RoundTripper, id string) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if id == "" {
		return base
	}
	return &Transport{Base: base, ID: id}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.ID == "" || !inAgentsScope(req) {
		return t.Base.RoundTrip(req)
	}
	// RoundTripper contract: don't mutate the caller's request.
	r := req.Clone(req.Context())
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Set(HeaderName, t.ID)
	return t.Base.RoundTrip(r)
}

// inAgentsScope matches the base path exactly or any sub-resource, but not
// a prefix-only match like "/v2/agents/sessionsfoo".
func inAgentsScope(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	p := req.URL.Path
	return p == agentsSessionsBase || strings.HasPrefix(p, agentsSessionsBase+"/")
}
