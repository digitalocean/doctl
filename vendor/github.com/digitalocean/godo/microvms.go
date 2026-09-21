package godo

import (
	"context"
	"fmt"
	"net/http"
)

const (
	microVMBasePath            = "v2/microvms"
	microVMCheckpointsBasePath = "v2/microvms/checkpoints"
	microVMOptionsPath         = "v2/microvms/options"
)

// MicroVMState represents the lifecycle state of a MicroVM.
type MicroVMState string

// Possible lifecycle states for a MicroVM.
const (
	MicroVMStateUnknown     = MicroVMState("unknown")
	MicroVMStateCreating    = MicroVMState("creating")
	MicroVMStateRunning     = MicroVMState("running")
	MicroVMStatePausing     = MicroVMState("pausing")
	MicroVMStatePaused      = MicroVMState("paused")
	MicroVMStateResuming    = MicroVMState("resuming")
	MicroVMStateTerminating = MicroVMState("terminating")
	MicroVMStateTerminated  = MicroVMState("terminated")
	MicroVMStateFailed      = MicroVMState("failed")
)

// MicroVMNetworking represents the networking mode of a MicroVM.
type MicroVMNetworking string

// Possible networking modes for a MicroVM.
const (
	MicroVMNetworkingUnknown = MicroVMNetworking("unknown")
	MicroVMNetworkingPublic  = MicroVMNetworking("public")
	MicroVMNetworkingVPC     = MicroVMNetworking("vpc")
)

// MicroVMHTTPProtocol represents the HTTP protocol option for a MicroVM.
type MicroVMHTTPProtocol string

// Possible HTTP protocol values for a MicroVM.
const (
	MicroVMHTTPProtocolHTTP  = MicroVMHTTPProtocol("http")
	MicroVMHTTPProtocolHTTP2 = MicroVMHTTPProtocol("http2")
)

// MicroVMURLStatus represents the lifecycle of a MicroVM URL.
type MicroVMURLStatus string

// Possible statuses for a MicroVM URL.
const (
	MicroVMURLStatusPending = MicroVMURLStatus("PENDING")
	MicroVMURLStatusActive  = MicroVMURLStatus("ACTIVE")
)

// MicroVMCheckpointStatus represents the status of a MicroVM checkpoint.
type MicroVMCheckpointStatus string

// Possible states for a MicroVM checkpoint.
const (
	MicroVMCheckpointStatusUnknown   = MicroVMCheckpointStatus("CHECKPOINT_UNKNOWN")
	MicroVMCheckpointStatusCreating  = MicroVMCheckpointStatus("CHECKPOINT_CREATING")
	MicroVMCheckpointStatusAvailable = MicroVMCheckpointStatus("CHECKPOINT_AVAILABLE")
	MicroVMCheckpointStatusFailed    = MicroVMCheckpointStatus("CHECKPOINT_FAILED")
	MicroVMCheckpointStatusDeleted   = MicroVMCheckpointStatus("CHECKPOINT_DELETED")
	MicroVMCheckpointStatusDeleting  = MicroVMCheckpointStatus("CHECKPOINT_DELETING")
)

// MicroVMsService is an interface for interfacing with the MicroVM
// endpoints of the DigitalOcean API.
// See: https://docs.digitalocean.com/reference/api/api-reference/#tag/MicroVMs
type MicroVMsService interface {
	List(ctx context.Context, opt *ListOptions) ([]MicroVM, *Response, error)
	ListByRegion(ctx context.Context, region string, opt *ListOptions) ([]MicroVM, *Response, error)
	ListByName(ctx context.Context, name string, opt *ListOptions) ([]MicroVM, *Response, error)
	Get(ctx context.Context, id string) (*MicroVM, *Response, error)
	Create(ctx context.Context, createRequest *MicroVMCreateRequest) (*MicroVM, *Response, error)
	Pause(ctx context.Context, id string) (*MicroVM, *Response, error)
	Resume(ctx context.Context, id string) (*MicroVM, *Response, error)
	Delete(ctx context.Context, id string) (*Response, error)

	ListCheckpoints(ctx context.Context, opt *ListMicroVMCheckpointsOptions) ([]MicroVMCheckpoint, *Response, error)
	CreateCheckpoint(ctx context.Context, microVMID string, createRequest *MicroVMCheckpointCreateRequest) (*MicroVMCheckpoint, *Response, error)
	GetCheckpoint(ctx context.Context, id string) (*MicroVMCheckpoint, *Response, error)
	DeleteCheckpoint(ctx context.Context, id string) (*Response, error)

	GetCreateOptions(ctx context.Context) (*MicroVMCreateOptions, *Response, error)
}

// MicroVMsServiceOp handles communication with the MicroVM related
// methods of the DigitalOcean API.
type MicroVMsServiceOp struct {
	client *Client
}

var _ MicroVMsService = &MicroVMsServiceOp{}

// MicroVMSize is the compute capacity reported on a MicroVM.
type MicroVMSize struct {
	CPU    uint32 `json:"cpu,omitempty"`
	Memory uint32 `json:"memory,omitempty"`
	Disk   uint64 `json:"disk,omitempty"`
}

// MicroVMSizeRequest is the compute capacity requested at create time.
// Disk is provisioned with the size and is not requested.
type MicroVMSizeRequest struct {
	CPU    uint32 `json:"cpu"`
	Memory uint32 `json:"memory"`
}

// MicroVMSource names what a MicroVM runs. Exactly one of OCIRef or
// CheckpointID must be set on create; the API rejects both or neither.
type MicroVMSource struct {
	OCIRef       string `json:"oci_ref,omitempty"`
	CheckpointID string `json:"checkpoint_id,omitempty"`
}

// MicroVMURL is one ingress URL attached to a MicroVM.
type MicroVMURL struct {
	Hostname string           `json:"hostname,omitempty"`
	Port     int              `json:"port,omitempty"`
	Default  bool             `json:"default,omitempty"`
	Status   MicroVMURLStatus `json:"status,omitempty"`
}

// MicroVM represents a DigitalOcean MicroVM.
type MicroVM struct {
	ID            string            `json:"id,omitempty"`
	Name          string            `json:"name,omitempty"`
	Region        string            `json:"region,omitempty"`
	State         MicroVMState      `json:"state,omitempty"`
	Size          *MicroVMSize      `json:"size,omitempty"`
	URLs          []MicroVMURL      `json:"urls,omitempty"`
	Ports         []uint32          `json:"ports,omitempty"`
	FailureReason string            `json:"failure_reason,omitempty"`
	Networking    MicroVMNetworking `json:"networking,omitempty"`
	Source        *MicroVMSource    `json:"source,omitempty"`
	AutoPause     *AutoPauseConfig  `json:"auto_pause,omitempty"`
	AutoResume    *bool             `json:"auto_resume,omitempty"`
	Created       string            `json:"created_at,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
}

// AutoPauseConfig configures MicroVM auto-pause behavior. IdleTimeout is
// a Go duration string (e.g. "5m", "30s") describing how long the MicroVM
// must be idle before it is paused.
type AutoPauseConfig struct {
	Enabled     *bool  `json:"enabled,omitempty"`
	IdleTimeout string `json:"idle_timeout,omitempty"`
}

// MicroVMCheckpoint represents a checkpoint of a MicroVM
// (persisted memory + disk state).
type MicroVMCheckpoint struct {
	ID          string                  `json:"id,omitempty"`
	MicroVMID   string                  `json:"microvm_id,omitempty"`
	MicroVMName string                  `json:"microvm_name,omitempty"`
	Name        string                  `json:"name,omitempty"`
	Region      string                  `json:"region,omitempty"`
	Status      MicroVMCheckpointStatus `json:"status,omitempty"`
	MemoryBytes uint64                  `json:"memory_bytes,omitempty"`
	DiskBytes   uint64                  `json:"disk_bytes,omitempty"`
	Created     string                  `json:"created_at,omitempty"`
}

// MicroVMCreateRequest represents a request to create a MicroVM.
// Size and Region may be omitted when restoring from a checkpoint so the API
// can inherit them from the checkpoint (api-v2 C6).
type MicroVMCreateRequest struct {
	Name         string              `json:"name"`
	Region       string              `json:"region,omitempty"`
	Size         *MicroVMSizeRequest `json:"size,omitempty"`
	Source       *MicroVMSource      `json:"source"`
	Networking   MicroVMNetworking   `json:"networking,omitempty"`
	VPCUUID      string              `json:"vpc_uuid,omitempty"`
	AutoPause    *AutoPauseConfig    `json:"auto_pause,omitempty"`
	AutoResume   *bool               `json:"auto_resume,omitempty"`
	HTTPPort     uint32              `json:"http_port,omitempty"`
	HTTPProtocol MicroVMHTTPProtocol `json:"http_protocol,omitempty"`
	Ports        []uint32            `json:"ports,omitempty"`
	Environment  map[string]string   `json:"environment,omitempty"`
	Tags         []string            `json:"tags,omitempty"`
}

// MicroVMCheckpointCreateRequest represents a request to create a checkpoint.
type MicroVMCheckpointCreateRequest struct {
	Name string `json:"name,omitempty"`
}

// ListMicroVMCheckpointsOptions are the optional parameters for listing
// checkpoints. MicroVMID filters to checkpoints captured from that
// MicroVM; omit it to list all checkpoints for the team.
type ListMicroVMCheckpointsOptions struct {
	ListOptions
	MicroVMID string `url:"microvm_id,omitempty"`
}

// MicroVMCreateOptions is the response from GET /v2/microvms/options.
type MicroVMCreateOptions struct {
	DefaultRegion string                 `json:"default_region,omitempty"`
	Sizes         []MicroVMSizeOption    `json:"sizes,omitempty"`
	Features      []MicroVMFeatureOption `json:"features,omitempty"`
	AccountLimits *MicroVMAccountLimits  `json:"account_limits,omitempty"`
}

// MicroVMSizeOption is one supported size evaluated for the team. Regions
// holds the slugs where the team can create this size; a region it cannot use
// is absent rather than listed as unavailable.
type MicroVMSizeOption struct {
	CPU       uint32              `json:"cpu"`
	Memory    uint32              `json:"memory"`
	Disk      uint64              `json:"disk"`
	Available bool                `json:"available"`
	Regions   []string            `json:"regions"`
	Pricing   *MicroVMSizePricing `json:"pricing,omitempty"`
}

// MicroVMSizePricing is the USD price of one size.
type MicroVMSizePricing struct {
	PricePerHour  float64 `json:"price_per_hour"`
	PricePerMonth float64 `json:"price_per_month"`
}

// MicroVMFeatureOption is one product feature gate for the team.
type MicroVMFeatureOption struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled"`
}

// MicroVMAccountLimits are the team's effective MicroVM limits.
type MicroVMAccountLimits struct {
	MaxConcurrentRunning  uint64 `json:"max_concurrent_running,omitempty"`
	MaxTotalCount         uint64 `json:"max_total_count,omitempty"`
	MaxMemoryBytes        uint64 `json:"max_memory_bytes,omitempty"`
	MaxDiskBytes          uint64 `json:"max_disk_bytes,omitempty"`
	MaxIdleTimeoutSeconds uint64 `json:"max_idle_timeout_seconds,omitempty"`
}

// String returns a human-readable description of a MicroVM.
func (m MicroVM) String() string {
	return Stringify(m)
}

// URN returns the MicroVM ID in a valid DO API URN form.
//
// The collection is still "microdroplet" on purpose. URNs are persisted by
// tags and emitted in billing events, and the collection name comes from a
// proto enum shared across those consumers, so renaming it is a coordinated
// change rather than part of this one.
func (m MicroVM) URN() string {
	return ToURN("MicroDroplet", m.ID)
}

// String returns a human-readable description of a MicroVMCheckpoint.
func (c MicroVMCheckpoint) String() string {
	return Stringify(c)
}

// String returns a human-readable description of a MicroVMCreateRequest.
func (r MicroVMCreateRequest) String() string {
	return Stringify(r)
}

type microVMRoot struct {
	MicroVM *MicroVM `json:"microvm"`
}

type microVMsRoot struct {
	MicroVMs []MicroVM `json:"microvms"`
	Links    *Links    `json:"links"`
	Meta     *Meta     `json:"meta"`
}

type microVMCheckpointRoot struct {
	Checkpoint *MicroVMCheckpoint `json:"checkpoint"`
}

type microVMCheckpointsRoot struct {
	Checkpoints []MicroVMCheckpoint `json:"checkpoints"`
	Links       *Links              `json:"links"`
	Meta        *Meta               `json:"meta"`
}

// listMicroVMOptions holds MicroVM-specific list filters that are
// not part of the shared ListOptions.
type listMicroVMOptions struct {
	Region string `url:"region,omitempty"`
	Name   string `url:"name,omitempty"`
}

// List lists all MicroVMs, with optional pagination.
func (s *MicroVMsServiceOp) List(ctx context.Context, opt *ListOptions) ([]MicroVM, *Response, error) {
	return s.list(ctx, opt, nil)
}

// ListByRegion lists MicroVMs filtered by region slug, with optional pagination.
func (s *MicroVMsServiceOp) ListByRegion(ctx context.Context, region string, opt *ListOptions) ([]MicroVM, *Response, error) {
	if region == "" {
		return nil, nil, NewArgError("region", "cannot be empty")
	}
	return s.list(ctx, opt, &listMicroVMOptions{Region: region})
}

// ListByName lists MicroVMs filtered by exact name match, with optional pagination.
func (s *MicroVMsServiceOp) ListByName(ctx context.Context, name string, opt *ListOptions) ([]MicroVM, *Response, error) {
	if name == "" {
		return nil, nil, NewArgError("name", "cannot be empty")
	}
	return s.list(ctx, opt, &listMicroVMOptions{Name: name})
}

func (s *MicroVMsServiceOp) list(ctx context.Context, opt *ListOptions, listOpt *listMicroVMOptions) ([]MicroVM, *Response, error) {
	path := microVMBasePath
	path, err := addOptions(path, opt)
	if err != nil {
		return nil, nil, err
	}
	path, err = addOptions(path, listOpt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(microVMsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if l := root.Links; l != nil {
		resp.Links = l
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.MicroVMs, resp, nil
}

// Get retrieves a MicroVM by its ID.
func (s *MicroVMsServiceOp) Get(ctx context.Context, id string) (*MicroVM, *Response, error) {
	if id == "" {
		return nil, nil, NewArgError("id", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", microVMBasePath, id)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(microVMRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.MicroVM, resp, nil
}

// Create provisions a new MicroVM with the provided configuration.
func (s *MicroVMsServiceOp) Create(ctx context.Context, createRequest *MicroVMCreateRequest) (*MicroVM, *Response, error) {
	if createRequest == nil {
		return nil, nil, NewArgError("createRequest", "cannot be nil")
	}

	req, err := s.client.NewRequest(ctx, http.MethodPost, microVMBasePath, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(microVMRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.MicroVM, resp, nil
}

// Pause synchronously transitions a RUNNING MicroVM to PAUSED. It blocks
// until the platform has durably paused the MicroVM and returns the
// updated resource. The call is idempotent: pausing a MicroVM that is
// already PAUSED returns the current MicroVM with no side effects.
func (s *MicroVMsServiceOp) Pause(ctx context.Context, id string) (*MicroVM, *Response, error) {
	return s.doTransition(ctx, id, "pause")
}

// Resume synchronously transitions a PAUSED MicroVM to RUNNING. It blocks
// until the platform has durably resumed the MicroVM and returns the
// updated resource. The call is idempotent: resuming a MicroVM that is
// already RUNNING returns the current MicroVM with no side effects.
func (s *MicroVMsServiceOp) Resume(ctx context.Context, id string) (*MicroVM, *Response, error) {
	return s.doTransition(ctx, id, "resume")
}

// doTransition posts an empty body to a MicroVM transition sub-resource
// (e.g. /pause, /resume) and decodes the returned MicroVM.
func (s *MicroVMsServiceOp) doTransition(ctx context.Context, id, action string) (*MicroVM, *Response, error) {
	if id == "" {
		return nil, nil, NewArgError("id", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s", microVMBasePath, id, action)

	req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(microVMRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.MicroVM, resp, nil
}

// Delete removes a MicroVM by its ID. The DigitalOcean API returns a 204
// on success and does not include a response body.
func (s *MicroVMsServiceOp) Delete(ctx context.Context, id string) (*Response, error) {
	if id == "" {
		return nil, NewArgError("id", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", microVMBasePath, id)

	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// ListCheckpoints lists checkpoints for the authenticated team. Pass a
// MicroVMID on opt to filter to checkpoints captured from that MicroVM.
func (s *MicroVMsServiceOp) ListCheckpoints(ctx context.Context, opt *ListMicroVMCheckpointsOptions) ([]MicroVMCheckpoint, *Response, error) {
	path := microVMCheckpointsBasePath
	path, err := addOptions(path, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(microVMCheckpointsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if l := root.Links; l != nil {
		resp.Links = l
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.Checkpoints, resp, nil
}

// CreateCheckpoint starts an asynchronous checkpoint of a running MicroVM.
func (s *MicroVMsServiceOp) CreateCheckpoint(ctx context.Context, microVMID string, createRequest *MicroVMCheckpointCreateRequest) (*MicroVMCheckpoint, *Response, error) {
	if microVMID == "" {
		return nil, nil, NewArgError("microVMID", "cannot be empty")
	}
	if createRequest == nil {
		createRequest = &MicroVMCheckpointCreateRequest{}
	}

	path := fmt.Sprintf("%s/%s/checkpoints", microVMBasePath, microVMID)

	req, err := s.client.NewRequest(ctx, http.MethodPost, path, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(microVMCheckpointRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Checkpoint, resp, nil
}

// GetCheckpoint retrieves a checkpoint by its ID.
func (s *MicroVMsServiceOp) GetCheckpoint(ctx context.Context, id string) (*MicroVMCheckpoint, *Response, error) {
	if id == "" {
		return nil, nil, NewArgError("id", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", microVMCheckpointsBasePath, id)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(microVMCheckpointRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Checkpoint, resp, nil
}

// DeleteCheckpoint releases the state stored by a checkpoint. Returns a 204
// on success with no response body.
func (s *MicroVMsServiceOp) DeleteCheckpoint(ctx context.Context, id string) (*Response, error) {
	if id == "" {
		return nil, NewArgError("id", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", microVMCheckpointsBasePath, id)

	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// GetCreateOptions returns the sizes, the regions each one can be created in,
// the feature gates, and the account limits available to the authenticated
// team when creating a MicroVM.
func (s *MicroVMsServiceOp) GetCreateOptions(ctx context.Context) (*MicroVMCreateOptions, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, microVMOptionsPath, nil)
	if err != nil {
		return nil, nil, err
	}

	opts := new(MicroVMCreateOptions)
	resp, err := s.client.Do(ctx, req, opts)
	if err != nil {
		return nil, resp, err
	}

	return opts, resp, nil
}
