package deviceid

import (
	"net/http"
	"strings"
)

// HeaderName is coordinated with the harness-api server. Rename in lockstep.
const HeaderName = "X-Device-UUID"

// The harness-api middleware is mounted at both prefixes: /v2/harness-runtime is
// the public name and /v2/agents the original one, which the edge still serves.
// Every request under either receives the header. The legacy prefix has to stay
// until nothing sends it, or the header silently stops being stamped.
const (
	harnessRuntimeBase = "/v2/harness-runtime"
	agentsBase         = "/v2/agents"
)

// Transport stamps HeaderName on requests under harnessRuntimeBase or agentsBase.
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

// inAgentsScope matches either base path exactly or any sub-resource, but not
// a prefix-only match like "/v2/agentsfoo".
func inAgentsScope(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	p := req.URL.Path
	for _, base := range []string{harnessRuntimeBase, agentsBase} {
		if p == base || strings.HasPrefix(p, base+"/") {
			return true
		}
	}
	return false
}
