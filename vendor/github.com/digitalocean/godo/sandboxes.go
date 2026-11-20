package godo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

const sandboxesBasePath = "/v2/sandboxes"

// SandboxesService is an interface for interacting with the Sandboxes
// endpoints of the DigitalOcean API.
//
// For a higher-level client with defaults, waiting and a paginator, see the
// github.com/digitalocean/godo/sandbox package.
type SandboxesService interface {
	List(context.Context, *SandboxListOptions) (*SandboxList, *Response, error)
	Get(context.Context, string) (*Sandbox, *Response, error)
	Create(context.Context, *SandboxCreateRequest) (*Sandbox, *Response, error)
	Delete(context.Context, string) (*Response, error)
	Pause(context.Context, string) (*Sandbox, *Response, error)
	Resume(context.Context, string) (*Sandbox, *Response, error)
}

// SandboxesServiceOp handles communication with the sandbox related methods of
// the DigitalOcean API.
type SandboxesServiceOp struct {
	client *Client
}

var _ SandboxesService = (*SandboxesServiceOp)(nil)

// SandboxState is the state of a sandbox.
type SandboxState string

const (
	// SandboxStateUnspecified is the zero state.
	SandboxStateUnspecified SandboxState = "SANDBOX_STATE_UNSPECIFIED"
	// SandboxStateRunning means the sandbox is ready to use.
	SandboxStateRunning SandboxState = "SANDBOX_STATE_RUNNING"
	// SandboxStatePaused means the sandbox is paused.
	SandboxStatePaused SandboxState = "SANDBOX_STATE_PAUSED"
	// SandboxStateCreating means the sandbox is still being created and cannot
	// be used yet. Create answers only after the sandbox is running, so only a
	// concurrent Get or List sees this state.
	SandboxStateCreating SandboxState = "SANDBOX_STATE_CREATING"
)

// Sandbox represents a DigitalOcean Sandbox.
type Sandbox struct {
	ID        string       `json:"sandbox_id"`
	Template  string       `json:"template"`
	State     SandboxState `json:"state"`
	SizeSlug  string       `json:"size_slug,omitempty"`
	CreatedAt Timestamp    `json:"created_at"`
}

// SandboxCreateRequest represents a request to create a sandbox.
type SandboxCreateRequest struct {
	// Template is the name of the template to create the sandbox from.
	Template string `json:"template"`
	// SizeSlug is the size of the sandbox. It must be an enabled size.
	SizeSlug string `json:"size_slug"`
	// AllowInternetAccess lets the sandbox reach the internet. When nil, egress
	// is unrestricted. When false, the sandbox can reach only the platform
	// file-transfer host.
	AllowInternetAccess *bool `json:"allow_internet_access,omitempty"`
	// EnvVars are environment variables for the sandbox. The values are secrets.
	EnvVars map[string]string `json:"env_vars,omitempty"`
}

// SandboxListOptions specifies the optional parameters to List.
type SandboxListOptions struct {
	// State returns only sandboxes in this state. Empty returns any state.
	State SandboxState `url:"state,omitempty"`
	// PageSize is the maximum number of sandboxes to return.
	PageSize int `url:"page_size,omitempty"`
	// PageToken is the NextPageToken of the previous page. Empty starts from
	// the beginning.
	PageToken string `url:"page_token,omitempty"`
}

// SandboxList is one page of sandboxes.
type SandboxList struct {
	Sandboxes []Sandbox `json:"sandboxes"`
	// NextPageToken is empty on the last page.
	NextPageToken string `json:"next_page_token"`
}

type sandboxRoot struct {
	Sandbox *Sandbox `json:"sandbox"`
}

// List lists the sandboxes of the team. When more results exist, the response
// carries a NextPageToken.
func (s *SandboxesServiceOp) List(ctx context.Context, opt *SandboxListOptions) (*SandboxList, *Response, error) {
	path, err := addOptions(sandboxesBasePath, opt)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(SandboxList)
	resp, err := s.client.Do(ctx, req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// Get retrieves a sandbox by ID.
func (s *SandboxesServiceOp) Get(ctx context.Context, sandboxID string) (*Sandbox, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	return s.do(ctx, http.MethodGet, fmt.Sprintf("%s/%s", sandboxesBasePath, sandboxID), nil)
}

// Create creates a sandbox. It returns once the sandbox is running.
func (s *SandboxesServiceOp) Create(ctx context.Context, create *SandboxCreateRequest) (*Sandbox, *Response, error) {
	if create == nil {
		return nil, nil, errors.New("sandboxes: create request is required")
	}
	return s.do(ctx, http.MethodPost, sandboxesBasePath, create)
}

// Delete deletes a sandbox.
func (s *SandboxesServiceOp) Delete(ctx context.Context, sandboxID string) (*Response, error) {
	if sandboxID == "" {
		return nil, errors.New("sandboxes: sandbox id is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodDelete, fmt.Sprintf("%s/%s", sandboxesBasePath, sandboxID), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// Pause pauses a running sandbox. Resume it to use it again. An ephemeral
// sandbox cannot be paused.
func (s *SandboxesServiceOp) Pause(ctx context.Context, sandboxID string) (*Sandbox, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	return s.do(ctx, http.MethodPost, fmt.Sprintf("%s/%s/pause", sandboxesBasePath, sandboxID), nil)
}

// Resume resumes a paused sandbox.
func (s *SandboxesServiceOp) Resume(ctx context.Context, sandboxID string) (*Sandbox, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	return s.do(ctx, http.MethodPost, fmt.Sprintf("%s/%s/resume", sandboxesBasePath, sandboxID), nil)
}

func (s *SandboxesServiceOp) do(ctx context.Context, method, path string, body interface{}) (*Sandbox, *Response, error) {
	req, err := s.client.NewRequest(ctx, method, path, body)
	if err != nil {
		return nil, nil, err
	}
	root := new(sandboxRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Sandbox == nil {
		return nil, resp, errors.New("sandboxes: the response did not include the sandbox")
	}
	return root.Sandbox, resp, nil
}
