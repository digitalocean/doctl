package godo

import (
	"context"
	"fmt"
	"net/http"
)

const (
	microDropletBasePath            = "v2/microdroplets"
	microDropletCheckpointsBasePath = "v2/microdroplets/checkpoints"
	microDropletOptionsPath         = "v2/microdroplets/options"
)

// MicroDropletState represents the lifecycle state of a MicroDroplet.
type MicroDropletState string

// Possible lifecycle states for a MicroDroplet.
const (
	MicroDropletStateUnknown     = MicroDropletState("unknown")
	MicroDropletStateCreating    = MicroDropletState("creating")
	MicroDropletStateRunning     = MicroDropletState("running")
	MicroDropletStatePausing     = MicroDropletState("pausing")
	MicroDropletStatePaused      = MicroDropletState("paused")
	MicroDropletStateResuming    = MicroDropletState("resuming")
	MicroDropletStateTerminating = MicroDropletState("terminating")
	MicroDropletStateTerminated  = MicroDropletState("terminated")
	MicroDropletStateFailed      = MicroDropletState("failed")
)

// MicroDropletNetworking represents the networking mode of a MicroDroplet.
type MicroDropletNetworking string

// Possible networking modes for a MicroDroplet.
const (
	MicroDropletNetworkingUnknown = MicroDropletNetworking("unknown")
	MicroDropletNetworkingPublic  = MicroDropletNetworking("public")
	MicroDropletNetworkingVPC     = MicroDropletNetworking("vpc")
)

// MicroDropletHTTPProtocol represents the HTTP protocol option for a MicroDroplet.
type MicroDropletHTTPProtocol string

// Possible HTTP protocol values for a MicroDroplet.
const (
	MicroDropletHTTPProtocolHTTP  = MicroDropletHTTPProtocol("http")
	MicroDropletHTTPProtocolHTTP2 = MicroDropletHTTPProtocol("http2")
)

// MicroDropletURLStatus represents the lifecycle of a MicroDroplet URL.
type MicroDropletURLStatus string

// Possible statuses for a MicroDroplet URL.
const (
	MicroDropletURLStatusPending = MicroDropletURLStatus("PENDING")
	MicroDropletURLStatusActive  = MicroDropletURLStatus("ACTIVE")
)

// MicroDropletCheckpointStatus represents the status of a MicroDroplet checkpoint.
type MicroDropletCheckpointStatus string

// Possible states for a MicroDroplet checkpoint.
const (
	MicroDropletCheckpointStatusUnknown   = MicroDropletCheckpointStatus("CHECKPOINT_UNKNOWN")
	MicroDropletCheckpointStatusCreating  = MicroDropletCheckpointStatus("CHECKPOINT_CREATING")
	MicroDropletCheckpointStatusAvailable = MicroDropletCheckpointStatus("CHECKPOINT_AVAILABLE")
	MicroDropletCheckpointStatusFailed    = MicroDropletCheckpointStatus("CHECKPOINT_FAILED")
	MicroDropletCheckpointStatusDeleted   = MicroDropletCheckpointStatus("CHECKPOINT_DELETED")
	MicroDropletCheckpointStatusDeleting  = MicroDropletCheckpointStatus("CHECKPOINT_DELETING")
)

// MicroDropletsService is an interface for interfacing with the MicroDroplet
// endpoints of the DigitalOcean API.
// See: https://docs.digitalocean.com/reference/api/api-reference/#tag/MicroDroplets
type MicroDropletsService interface {
	List(ctx context.Context, opt *ListOptions) ([]MicroDroplet, *Response, error)
	ListByRegion(ctx context.Context, region string, opt *ListOptions) ([]MicroDroplet, *Response, error)
	ListByName(ctx context.Context, name string, opt *ListOptions) ([]MicroDroplet, *Response, error)
	Get(ctx context.Context, id string) (*MicroDroplet, *Response, error)
	Create(ctx context.Context, createRequest *MicroDropletCreateRequest) (*MicroDroplet, *Response, error)
	Pause(ctx context.Context, id string) (*MicroDroplet, *Response, error)
	Resume(ctx context.Context, id string) (*MicroDroplet, *Response, error)
	Delete(ctx context.Context, id string) (*Response, error)

	ListCheckpoints(ctx context.Context, opt *ListMicroDropletCheckpointsOptions) ([]MicroDropletCheckpoint, *Response, error)
	CreateCheckpoint(ctx context.Context, microDropletID string, createRequest *MicroDropletCheckpointCreateRequest) (*MicroDropletCheckpoint, *Response, error)
	GetCheckpoint(ctx context.Context, id string) (*MicroDropletCheckpoint, *Response, error)
	DeleteCheckpoint(ctx context.Context, id string) (*Response, error)

	GetCreateOptions(ctx context.Context) (*MicroDropletCreateOptions, *Response, error)
}

// MicroDropletsServiceOp handles communication with the MicroDroplet related
// methods of the DigitalOcean API.
type MicroDropletsServiceOp struct {
	client *Client
}

var _ MicroDropletsService = &MicroDropletsServiceOp{}

// MicroDropletSize is the compute capacity reported on a MicroDroplet.
type MicroDropletSize struct {
	CPU    uint32 `json:"cpu,omitempty"`
	Memory uint32 `json:"memory,omitempty"`
	Disk   uint64 `json:"disk,omitempty"`
}

// MicroDropletSizeRequest is the compute capacity requested at create time.
// Disk is provisioned with the size and is not requested.
type MicroDropletSizeRequest struct {
	CPU    uint32 `json:"cpu"`
	Memory uint32 `json:"memory"`
}

// MicroDropletSource names what a MicroDroplet runs. Exactly one of OCIRef or
// CheckpointID must be set on create; the API rejects both or neither.
type MicroDropletSource struct {
	OCIRef       string `json:"oci_ref,omitempty"`
	CheckpointID string `json:"checkpoint_id,omitempty"`
}

// MicroDropletURL is one ingress URL attached to a MicroDroplet.
type MicroDropletURL struct {
	Hostname string                `json:"hostname,omitempty"`
	Port     int                   `json:"port,omitempty"`
	Default  bool                  `json:"default,omitempty"`
	Status   MicroDropletURLStatus `json:"status,omitempty"`
}

// MicroDroplet represents a DigitalOcean MicroDroplet.
type MicroDroplet struct {
	ID            string                 `json:"id,omitempty"`
	Name          string                 `json:"name,omitempty"`
	Region        string                 `json:"region,omitempty"`
	State         MicroDropletState      `json:"state,omitempty"`
	Size          *MicroDropletSize      `json:"size,omitempty"`
	URLs          []MicroDropletURL      `json:"urls,omitempty"`
	Ports         []uint32               `json:"ports,omitempty"`
	FailureReason string                 `json:"failure_reason,omitempty"`
	Networking    MicroDropletNetworking `json:"networking,omitempty"`
	Source        *MicroDropletSource    `json:"source,omitempty"`
	AutoPause     *AutoPauseConfig       `json:"auto_pause,omitempty"`
	AutoResume    *bool                  `json:"auto_resume,omitempty"`
	Created       string                 `json:"created_at,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
}

// AutoPauseConfig configures MicroDroplet auto-pause behavior. IdleTimeout is
// a Go duration string (e.g. "5m", "30s") describing how long the MicroDroplet
// must be idle before it is paused.
type AutoPauseConfig struct {
	Enabled     *bool  `json:"enabled,omitempty"`
	IdleTimeout string `json:"idle_timeout,omitempty"`
}

// MicroDropletCheckpoint represents a checkpoint of a MicroDroplet
// (persisted memory + disk state).
type MicroDropletCheckpoint struct {
	ID               string                       `json:"id,omitempty"`
	MicroDropletID   string                       `json:"micro_droplet_id,omitempty"`
	MicroDropletName string                       `json:"micro_droplet_name,omitempty"`
	Name             string                       `json:"name,omitempty"`
	Region           string                       `json:"region,omitempty"`
	Status           MicroDropletCheckpointStatus `json:"status,omitempty"`
	MemoryBytes      uint64                       `json:"memory_bytes,omitempty"`
	DiskBytes        uint64                       `json:"disk_bytes,omitempty"`
	Created          string                       `json:"created_at,omitempty"`
}

// MicroDropletCreateRequest represents a request to create a MicroDroplet.
// Size and Region may be omitted when restoring from a checkpoint so the API
// can inherit them from the checkpoint (api-v2 C6).
type MicroDropletCreateRequest struct {
	Name         string                   `json:"name"`
	Region       string                   `json:"region,omitempty"`
	Size         *MicroDropletSizeRequest `json:"size,omitempty"`
	Source       *MicroDropletSource      `json:"source"`
	Networking   MicroDropletNetworking   `json:"networking,omitempty"`
	VPCUUID      string                   `json:"vpc_uuid,omitempty"`
	AutoPause    *AutoPauseConfig         `json:"auto_pause,omitempty"`
	AutoResume   *bool                    `json:"auto_resume,omitempty"`
	HTTPPort     uint32                   `json:"http_port,omitempty"`
	HTTPProtocol MicroDropletHTTPProtocol `json:"http_protocol,omitempty"`
	Ports        []uint32                 `json:"ports,omitempty"`
	Environment  map[string]string        `json:"environment,omitempty"`
	Tags         []string                 `json:"tags,omitempty"`
}

// MicroDropletCheckpointCreateRequest represents a request to create a checkpoint.
type MicroDropletCheckpointCreateRequest struct {
	Name string `json:"name,omitempty"`
}

// ListMicroDropletCheckpointsOptions are the optional parameters for listing
// checkpoints. MicroDropletID filters to checkpoints captured from that
// MicroDroplet; omit it to list all checkpoints for the team.
type ListMicroDropletCheckpointsOptions struct {
	ListOptions
	MicroDropletID string `url:"micro_droplet_id,omitempty"`
}

// MicroDropletCreateOptions is the response from GET /v2/microdroplets/options.
type MicroDropletCreateOptions struct {
	Regions       []MicroDropletRegionOption  `json:"regions,omitempty"`
	DefaultRegion string                      `json:"default_region,omitempty"`
	Sizes         []MicroDropletSizeOption    `json:"sizes,omitempty"`
	Features      []MicroDropletFeatureOption `json:"features,omitempty"`
	AccountLimits *MicroDropletAccountLimits  `json:"account_limits,omitempty"`
}

// MicroDropletRegionOption is one region evaluated for the authenticated team.
type MicroDropletRegionOption struct {
	Slug              string `json:"slug,omitempty"`
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
}

// MicroDropletSizeOption is one supported size evaluated for the team.
type MicroDropletSizeOption struct {
	Size      MicroDropletSize         `json:"size"`
	Available bool                     `json:"available"`
	Pricing   *MicroDropletSizePricing `json:"pricing,omitempty"`
}

// MicroDropletSizePricing is the USD price of one size.
type MicroDropletSizePricing struct {
	PricePerHour  float64 `json:"price_per_hour"`
	PricePerMonth float64 `json:"price_per_month"`
}

// MicroDropletFeatureOption is one product feature gate for the team.
type MicroDropletFeatureOption struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled"`
}

// MicroDropletAccountLimits are the team's effective MicroDroplet limits.
type MicroDropletAccountLimits struct {
	MaxConcurrentRunning  uint64 `json:"max_concurrent_running,omitempty"`
	MaxTotalCount         uint64 `json:"max_total_count,omitempty"`
	MaxMemoryBytes        uint64 `json:"max_memory_bytes,omitempty"`
	MaxDiskBytes          uint64 `json:"max_disk_bytes,omitempty"`
	MaxIdleTimeoutSeconds uint64 `json:"max_idle_timeout_seconds,omitempty"`
}

// String returns a human-readable description of a MicroDroplet.
func (m MicroDroplet) String() string {
	return Stringify(m)
}

// URN returns the MicroDroplet ID in a valid DO API URN form.
func (m MicroDroplet) URN() string {
	return ToURN("MicroDroplet", m.ID)
}

// String returns a human-readable description of a MicroDropletCheckpoint.
func (c MicroDropletCheckpoint) String() string {
	return Stringify(c)
}

// String returns a human-readable description of a MicroDropletCreateRequest.
func (r MicroDropletCreateRequest) String() string {
	return Stringify(r)
}

type microDropletRoot struct {
	MicroDroplet *MicroDroplet `json:"micro_droplet"`
}

type microDropletsRoot struct {
	MicroDroplets []MicroDroplet `json:"micro_droplets"`
	Links         *Links         `json:"links"`
	Meta          *Meta          `json:"meta"`
}

type microDropletCheckpointRoot struct {
	Checkpoint *MicroDropletCheckpoint `json:"checkpoint"`
}

type microDropletCheckpointsRoot struct {
	Checkpoints []MicroDropletCheckpoint `json:"checkpoints"`
	Links       *Links                   `json:"links"`
	Meta        *Meta                    `json:"meta"`
}

// listMicroDropletOptions holds MicroDroplet-specific list filters that are
// not part of the shared ListOptions.
type listMicroDropletOptions struct {
	Region string `url:"region,omitempty"`
	Name   string `url:"name,omitempty"`
}

// List lists all MicroDroplets, with optional pagination.
func (s *MicroDropletsServiceOp) List(ctx context.Context, opt *ListOptions) ([]MicroDroplet, *Response, error) {
	return s.list(ctx, opt, nil)
}

// ListByRegion lists MicroDroplets filtered by region slug, with optional pagination.
func (s *MicroDropletsServiceOp) ListByRegion(ctx context.Context, region string, opt *ListOptions) ([]MicroDroplet, *Response, error) {
	if region == "" {
		return nil, nil, NewArgError("region", "cannot be empty")
	}
	return s.list(ctx, opt, &listMicroDropletOptions{Region: region})
}

// ListByName lists MicroDroplets filtered by exact name match, with optional pagination.
func (s *MicroDropletsServiceOp) ListByName(ctx context.Context, name string, opt *ListOptions) ([]MicroDroplet, *Response, error) {
	if name == "" {
		return nil, nil, NewArgError("name", "cannot be empty")
	}
	return s.list(ctx, opt, &listMicroDropletOptions{Name: name})
}

func (s *MicroDropletsServiceOp) list(ctx context.Context, opt *ListOptions, listOpt *listMicroDropletOptions) ([]MicroDroplet, *Response, error) {
	path := microDropletBasePath
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

	root := new(microDropletsRoot)
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

	return root.MicroDroplets, resp, nil
}

// Get retrieves a MicroDroplet by its ID.
func (s *MicroDropletsServiceOp) Get(ctx context.Context, id string) (*MicroDroplet, *Response, error) {
	if id == "" {
		return nil, nil, NewArgError("id", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", microDropletBasePath, id)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(microDropletRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.MicroDroplet, resp, nil
}

// Create provisions a new MicroDroplet with the provided configuration.
func (s *MicroDropletsServiceOp) Create(ctx context.Context, createRequest *MicroDropletCreateRequest) (*MicroDroplet, *Response, error) {
	if createRequest == nil {
		return nil, nil, NewArgError("createRequest", "cannot be nil")
	}

	req, err := s.client.NewRequest(ctx, http.MethodPost, microDropletBasePath, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(microDropletRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.MicroDroplet, resp, nil
}

// Pause synchronously transitions a RUNNING MicroDroplet to PAUSED. It blocks
// until the platform has durably paused the MicroDroplet and returns the
// updated resource. The call is idempotent: pausing a MicroDroplet that is
// already PAUSED returns the current MicroDroplet with no side effects.
func (s *MicroDropletsServiceOp) Pause(ctx context.Context, id string) (*MicroDroplet, *Response, error) {
	return s.doTransition(ctx, id, "pause")
}

// Resume synchronously transitions a PAUSED MicroDroplet to RUNNING. It blocks
// until the platform has durably resumed the MicroDroplet and returns the
// updated resource. The call is idempotent: resuming a MicroDroplet that is
// already RUNNING returns the current MicroDroplet with no side effects.
func (s *MicroDropletsServiceOp) Resume(ctx context.Context, id string) (*MicroDroplet, *Response, error) {
	return s.doTransition(ctx, id, "resume")
}

// doTransition posts an empty body to a MicroDroplet transition sub-resource
// (e.g. /pause, /resume) and decodes the returned MicroDroplet.
func (s *MicroDropletsServiceOp) doTransition(ctx context.Context, id, action string) (*MicroDroplet, *Response, error) {
	if id == "" {
		return nil, nil, NewArgError("id", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s", microDropletBasePath, id, action)

	req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(microDropletRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.MicroDroplet, resp, nil
}

// Delete removes a MicroDroplet by its ID. The DigitalOcean API returns a 204
// on success and does not include a response body.
func (s *MicroDropletsServiceOp) Delete(ctx context.Context, id string) (*Response, error) {
	if id == "" {
		return nil, NewArgError("id", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", microDropletBasePath, id)

	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// ListCheckpoints lists checkpoints for the authenticated team. Pass a
// MicroDropletID on opt to filter to checkpoints captured from that MicroDroplet.
func (s *MicroDropletsServiceOp) ListCheckpoints(ctx context.Context, opt *ListMicroDropletCheckpointsOptions) ([]MicroDropletCheckpoint, *Response, error) {
	path := microDropletCheckpointsBasePath
	path, err := addOptions(path, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(microDropletCheckpointsRoot)
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

// CreateCheckpoint starts an asynchronous checkpoint of a running MicroDroplet.
func (s *MicroDropletsServiceOp) CreateCheckpoint(ctx context.Context, microDropletID string, createRequest *MicroDropletCheckpointCreateRequest) (*MicroDropletCheckpoint, *Response, error) {
	if microDropletID == "" {
		return nil, nil, NewArgError("microDropletID", "cannot be empty")
	}
	if createRequest == nil {
		createRequest = &MicroDropletCheckpointCreateRequest{}
	}

	path := fmt.Sprintf("%s/%s/checkpoints", microDropletBasePath, microDropletID)

	req, err := s.client.NewRequest(ctx, http.MethodPost, path, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(microDropletCheckpointRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Checkpoint, resp, nil
}

// GetCheckpoint retrieves a checkpoint by its ID.
func (s *MicroDropletsServiceOp) GetCheckpoint(ctx context.Context, id string) (*MicroDropletCheckpoint, *Response, error) {
	if id == "" {
		return nil, nil, NewArgError("id", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", microDropletCheckpointsBasePath, id)

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(microDropletCheckpointRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Checkpoint, resp, nil
}

// DeleteCheckpoint releases the state stored by a checkpoint. Returns a 204
// on success with no response body.
func (s *MicroDropletsServiceOp) DeleteCheckpoint(ctx context.Context, id string) (*Response, error) {
	if id == "" {
		return nil, NewArgError("id", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", microDropletCheckpointsBasePath, id)

	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// GetCreateOptions returns the regions, sizes, features, and account limits
// available to the authenticated team when creating a MicroDroplet.
func (s *MicroDropletsServiceOp) GetCreateOptions(ctx context.Context) (*MicroDropletCreateOptions, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, microDropletOptionsPath, nil)
	if err != nil {
		return nil, nil, err
	}

	opts := new(MicroDropletCreateOptions)
	resp, err := s.client.Do(ctx, req, opts)
	if err != nil {
		return nil, resp, err
	}

	return opts, resp, nil
}
