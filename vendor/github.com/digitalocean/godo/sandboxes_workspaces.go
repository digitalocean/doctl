package godo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

const sandboxWorkspacesBasePath = sandboxesBasePath + "/workspaces"

// SandboxWorkspaceState is the state of a persistent workspace.
type SandboxWorkspaceState string

const (
	// SandboxWorkspaceStateUnspecified is the zero state.
	SandboxWorkspaceStateUnspecified SandboxWorkspaceState = "WORKSPACE_STATE_UNSPECIFIED"
	// SandboxWorkspaceStateCreating means the workspace is being created.
	SandboxWorkspaceStateCreating SandboxWorkspaceState = "WORKSPACE_STATE_CREATING"
	// SandboxWorkspaceStateAvailable means the workspace is ready to attach to a sandbox.
	SandboxWorkspaceStateAvailable SandboxWorkspaceState = "WORKSPACE_STATE_AVAILABLE"
	// SandboxWorkspaceStateAttaching means a sandbox is starting with the workspace attached.
	SandboxWorkspaceStateAttaching SandboxWorkspaceState = "WORKSPACE_STATE_ATTACHING"
	// SandboxWorkspaceStateAttached means the workspace is attached to a sandbox.
	SandboxWorkspaceStateAttached SandboxWorkspaceState = "WORKSPACE_STATE_ATTACHED"
	// SandboxWorkspaceStateReleasing means the workspace is being saved after its sandbox went away.
	SandboxWorkspaceStateReleasing SandboxWorkspaceState = "WORKSPACE_STATE_RELEASING"
	// SandboxWorkspaceStateDeleting means the workspace is being deleted.
	SandboxWorkspaceStateDeleting SandboxWorkspaceState = "WORKSPACE_STATE_DELETING"
	// SandboxWorkspaceStateFailed means the workspace failed.
	SandboxWorkspaceStateFailed SandboxWorkspaceState = "WORKSPACE_STATE_FAILED"
)

// SandboxWorkspace is a persistent disk that outlives any one sandbox. Attach
// it to a sandbox by passing its ID as SandboxCreateRequest.WorkspaceID.
type SandboxWorkspace struct {
	ID string `json:"workspace_id"`
	// Name is empty when none was given.
	Name string `json:"name,omitempty"`
	// SizeLimitGiB is the size limit of the workspace, in GiB.
	SizeLimitGiB int64                 `json:"size_limit_gib"`
	State        SandboxWorkspaceState `json:"state"`
	// AttachedSandboxID is the sandbox the workspace is attached to. Empty when
	// it is not attached.
	AttachedSandboxID string `json:"attached_sandbox_id,omitempty"`
	// LastSavedAt is the last time the workspace data was saved.
	LastSavedAt Timestamp `json:"last_saved_at"`
	CreatedAt   Timestamp `json:"created_at"`
}

// SandboxWorkspaceCreateRequest is the body of CreateWorkspace.
type SandboxWorkspaceCreateRequest struct {
	// Name is an optional label for the workspace.
	Name string `json:"name,omitempty"`
	// SizeLimitGiB is the size limit of the workspace, in GiB. Required; at
	// most 100.
	SizeLimitGiB int64 `json:"size_limit_gib"`
}

// SandboxWorkspaceListOptions are the optional parameters of ListWorkspaces.
type SandboxWorkspaceListOptions struct {
	// State returns only workspaces in this state. Empty returns any state.
	State SandboxWorkspaceState `url:"state,omitempty"`
	// PageSize is the maximum number of workspaces to return.
	PageSize int `url:"page_size,omitempty"`
	// PageToken is the NextPageToken of the previous page. Empty starts from
	// the beginning.
	PageToken string `url:"page_token,omitempty"`
}

// SandboxWorkspaceList is one page of workspaces.
type SandboxWorkspaceList struct {
	Workspaces []SandboxWorkspace `json:"workspaces"`
	// NextPageToken is empty on the last page.
	NextPageToken string `json:"next_page_token"`
}

type sandboxWorkspaceRoot struct {
	Workspace *SandboxWorkspace `json:"workspace"`
}

// ListWorkspaces lists your team's persistent workspaces.
func (s *SandboxesServiceOp) ListWorkspaces(ctx context.Context, opt *SandboxWorkspaceListOptions) (*SandboxWorkspaceList, *Response, error) {
	path, err := addOptions(sandboxWorkspacesBasePath, opt)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(SandboxWorkspaceList)
	resp, err := s.client.Do(ctx, req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// CreateWorkspace creates a persistent workspace. The workspace can still be
// creating when this returns: poll GetWorkspace until it is available.
func (s *SandboxesServiceOp) CreateWorkspace(ctx context.Context, create *SandboxWorkspaceCreateRequest) (*SandboxWorkspace, *Response, error) {
	if create == nil {
		return nil, nil, errors.New("sandboxes: create workspace request is required")
	}
	return s.workspaceRequest(ctx, http.MethodPost, sandboxWorkspacesBasePath, create)
}

// GetWorkspace retrieves a workspace by ID.
func (s *SandboxesServiceOp) GetWorkspace(ctx context.Context, workspaceID string) (*SandboxWorkspace, *Response, error) {
	if workspaceID == "" {
		return nil, nil, errors.New("sandboxes: workspace id is required")
	}
	return s.workspaceRequest(ctx, http.MethodGet, sandboxWorkspacePath(workspaceID), nil)
}

// DeleteWorkspace deletes a workspace and its data.
func (s *SandboxesServiceOp) DeleteWorkspace(ctx context.Context, workspaceID string) (*Response, error) {
	if workspaceID == "" {
		return nil, errors.New("sandboxes: workspace id is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodDelete, sandboxWorkspacePath(workspaceID), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

func sandboxWorkspacePath(workspaceID string) string {
	return fmt.Sprintf("%s/%s", sandboxWorkspacesBasePath, workspaceID)
}

func (s *SandboxesServiceOp) workspaceRequest(ctx context.Context, method, path string, body interface{}) (*SandboxWorkspace, *Response, error) {
	req, err := s.client.NewRequest(ctx, method, path, body)
	if err != nil {
		return nil, nil, err
	}
	root := new(sandboxWorkspaceRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Workspace == nil {
		return nil, resp, errors.New("sandboxes: the response did not include the workspace")
	}
	return root.Workspace, resp, nil
}
