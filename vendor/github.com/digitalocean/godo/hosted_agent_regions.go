package godo

import (
	"context"
	"net/http"
)

// HostedAgentRegion is one entry of the region catalog returned by
// ListRegions (GET /v2/harness-runtime/regions). Submit Slug as the region a
// session or environment is created in; a slug returned here is accepted by
// placement, and a slug missing from the catalog is refused.
type HostedAgentRegion struct {
	// Slug is the DO datacenter slug, e.g. "ric1".
	Slug string `json:"slug,omitempty"`
	// IsDefault marks the region a create lands in when none is named.
	// Exactly one listed region is the default, and every team sees it.
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

// DefaultRegion returns the slug of the region a create lands in when none is
// named, and whether the catalog named one. Callers rendering a region picker
// use it to preselect an entry instead of hardcoding a slug.
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
