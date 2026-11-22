package godo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

const sandboxSnapshotsBasePath = sandboxesBasePath + "/snapshots"

// SandboxSnapshotState is where a snapshot is in its life.
type SandboxSnapshotState string

const (
	// SandboxSnapshotStateUnspecified is the zero state.
	SandboxSnapshotStateUnspecified SandboxSnapshotState = "SNAPSHOT_STATE_UNSPECIFIED"
	// SandboxSnapshotStateCreating means the capture is in progress.
	SandboxSnapshotStateCreating SandboxSnapshotState = "SNAPSHOT_STATE_CREATING"
	// SandboxSnapshotStateUploading means the capture finished and the
	// artifacts are uploading to storage.
	SandboxSnapshotStateUploading SandboxSnapshotState = "SNAPSHOT_STATE_UPLOADING"
	// SandboxSnapshotStateComplete means the snapshot is ready.
	SandboxSnapshotStateComplete SandboxSnapshotState = "SNAPSHOT_STATE_COMPLETE"
	// SandboxSnapshotStateFailed means the snapshot failed. See ErrorMessage.
	SandboxSnapshotStateFailed SandboxSnapshotState = "SNAPSHOT_STATE_FAILED"
)

// SandboxSnapshot is a snapshot of a sandbox.
type SandboxSnapshot struct {
	ID        string `json:"snapshot_id"`
	SandboxID string `json:"sandbox_id"`
	// Name is the name given when the snapshot was created, if any.
	Name  string               `json:"name,omitempty"`
	State SandboxSnapshotState `json:"state"`
	// SizeBytes is the size in bytes, encoded as a decimal string by the API.
	// It is set when the snapshot is complete. Size parses it.
	SizeBytes string    `json:"size_bytes,omitempty"`
	CreatedAt Timestamp `json:"created_at"`
	// ErrorMessage is why the snapshot failed. It is set when the state is
	// SandboxSnapshotStateFailed.
	ErrorMessage string `json:"error_message,omitempty"`
	// TTL is how long the snapshot is kept, such as "168h". Empty when the
	// platform default applies.
	TTL string `json:"ttl,omitempty"`
}

// Size is SizeBytes as a number. It is 0 when the size is not set or is not a
// number.
func (s SandboxSnapshot) Size() uint64 {
	n, err := strconv.ParseUint(s.SizeBytes, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// SandboxSnapshotCreateRequest is the body of CreateSnapshot. Both fields are
// optional.
type SandboxSnapshotCreateRequest struct {
	// Name is 1 to 63 characters: lowercase letters, digits and hyphens.
	Name string `json:"name,omitempty"`
	// TTL is how long to keep the snapshot, such as "168h". When empty, the
	// platform default applies.
	TTL string `json:"ttl,omitempty"`
}

// SandboxSnapshotListOptions are the options of ListSnapshots.
type SandboxSnapshotListOptions struct {
	// SandboxID returns only the snapshots of this sandbox. When empty, every
	// snapshot of your team is returned.
	SandboxID string `url:"sandbox_id,omitempty"`
	// PageSize is the maximum number of snapshots to return.
	PageSize int `url:"page_size,omitempty"`
	// PageToken is the NextPageToken of the previous page. Empty starts from
	// the beginning.
	PageToken string `url:"page_token,omitempty"`
}

// SandboxSnapshotList is one page of snapshots, newest first.
type SandboxSnapshotList struct {
	Snapshots []SandboxSnapshot `json:"snapshots"`
	// NextPageToken is empty on the last page.
	NextPageToken string `json:"next_page_token"`
}

type sandboxSnapshotRoot struct {
	Snapshot *SandboxSnapshot `json:"snapshot"`
}

// CreateSnapshot starts a snapshot of a running or paused sandbox. The
// snapshot is created asynchronously: the returned snapshot is usually in the
// creating state, so poll GetSnapshot until it is complete or failed. A nil
// create request uses the platform defaults.
func (s *SandboxesServiceOp) CreateSnapshot(ctx context.Context, sandboxID string, create *SandboxSnapshotCreateRequest) (*SandboxSnapshot, *Response, error) {
	if sandboxID == "" {
		return nil, nil, errors.New("sandboxes: sandbox id is required")
	}
	var body interface{}
	if create != nil {
		body = create
	}
	return s.snapshotRequest(ctx, http.MethodPost, fmt.Sprintf("%s/%s/snapshots", sandboxesBasePath, sandboxID), body)
}

// GetSnapshot retrieves a snapshot by ID.
func (s *SandboxesServiceOp) GetSnapshot(ctx context.Context, snapshotID string) (*SandboxSnapshot, *Response, error) {
	if snapshotID == "" {
		return nil, nil, errors.New("sandboxes: snapshot id is required")
	}
	return s.snapshotRequest(ctx, http.MethodGet, sandboxSnapshotPath(snapshotID), nil)
}

// ListSnapshots lists your team's snapshots, newest first.
func (s *SandboxesServiceOp) ListSnapshots(ctx context.Context, opt *SandboxSnapshotListOptions) (*SandboxSnapshotList, *Response, error) {
	path, err := addOptions(sandboxSnapshotsBasePath, opt)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(SandboxSnapshotList)
	resp, err := s.client.Do(ctx, req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// DeleteSnapshot deletes a snapshot. Sandboxes already started from it are
// not affected.
func (s *SandboxesServiceOp) DeleteSnapshot(ctx context.Context, snapshotID string) (*Response, error) {
	if snapshotID == "" {
		return nil, errors.New("sandboxes: snapshot id is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodDelete, sandboxSnapshotPath(snapshotID), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

func (s *SandboxesServiceOp) snapshotRequest(ctx context.Context, method, path string, body interface{}) (*SandboxSnapshot, *Response, error) {
	req, err := s.client.NewRequest(ctx, method, path, body)
	if err != nil {
		return nil, nil, err
	}
	root := new(sandboxSnapshotRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Snapshot == nil {
		return nil, resp, errors.New("sandboxes: the response did not include the snapshot")
	}
	return root.Snapshot, resp, nil
}

func sandboxSnapshotPath(snapshotID string) string {
	return fmt.Sprintf("%s/%s", sandboxSnapshotsBasePath, snapshotID)
}
