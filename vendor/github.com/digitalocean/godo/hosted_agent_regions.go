package godo

import (
	"context"
	"net/http"
)

// HostedAgentRegion is one entry of the region catalog returned by
// ListRegions (GET /v2/harness-runtime/regions). A region is chosen by the API
// host called, not by a field in a request body, so Slug names the host to
// point a client at. A create is placed in the region of the host that served
// it; the optional region field on a create request is a check against that
// host, not a choice of region.
type HostedAgentRegion struct {
	// Slug is the DO datacenter slug, e.g. "ric1". It names the regional API
	// host for this region: "ric1" is https://api-ric1.digitalocean.com. Use
	// HostedAgentsRegionalBaseURL to build that host.
	Slug string `json:"slug,omitempty"`
	// IsDefault marks the region the region-less host,
	// https://api.digitalocean.com, serves. Exactly one listed region is the
	// default, and every team sees it.
	IsDefault bool `json:"is_default,omitempty"`
}

// HostedAgentRegionsResponse is returned by ListRegions.
type HostedAgentRegionsResponse struct {
	// Regions are the regions the team may place resources in, sorted by slug.
	// Never empty: the default region is listed for every team.
	Regions []HostedAgentRegion `json:"regions"`
}

// ListRegions returns the per-team region catalog. See HostedAgentsService for
// the contract it guarantees against create-time placement.
func (s *HostedAgentsServiceOp) ListRegions(ctx context.Context) (*HostedAgentRegionsResponse, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, hostedAgentRegionsPath, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentRegionsResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// DefaultRegion returns the slug of the region the region-less host serves,
// and whether the catalog named one. Callers rendering a region picker use it
// to preselect an entry instead of hardcoding a slug.
func (r *HostedAgentRegionsResponse) DefaultRegion() (string, bool) {
	if r == nil {
		return "", false
	}
	for _, region := range r.Regions {
		if region.IsDefault {
			return region.Slug, true
		}
	}
	return "", false
}
