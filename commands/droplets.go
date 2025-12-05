package godo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const dropletBasePath = "v2/droplets"

// errNoNetworks is returned by convenience accessors when the Droplet
// has no attached networks information (nil Networks).
var errNoNetworks = errors.New("no networks have been defined")

// DropletsService is an interface for interacting with the Droplet endpoints
// of the DigitalOcean API.
//
// The methods mirror the REST actions available at the DigitalOcean API:
// - List lists droplets optionally using pagination options.
// - Get returns the details for a single droplet ID.
// - Create and CreateMultiple will instantiate new droplets.
// - Delete and DeleteByTag remove droplets.
//
// Implementations should return a *Response from the underlying HTTP call
// and any returned error. The interface is used to allow easy testing/mocking.
type DropletsService interface {
	// List returns all droplets accessible for the account, optionally
	// constrained by ListOptions (page, per_page).
	List(context.Context, *ListOptions) ([]Droplet, *Response, error)

	// ListWithGPUs returns droplets with GPU enabled.
	ListWithGPUs(context.Context, *ListOptions) ([]Droplet, *Response, error)

	// ListByName returns droplets whose name exactly matches the provided
	// name (case-insensitive).
	ListByName(context.Context, string, *ListOptions) ([]Droplet, *Response, error)

	// ListByTag returns droplets that have been tagged with the given tag.
	ListByTag(context.Context, string, *ListOptions) ([]Droplet, *Response, error)

	// Get obtains the droplet details for the given droplet ID.
	Get(context.Context, int) (*Droplet, *Response, error)

	// Create creates a single droplet using the provided DropletCreateRequest.
	Create(context.Context, *DropletCreateRequest) (*Droplet, *Response, error)

	// CreateMultiple creates multiple droplets in a single request.
	CreateMultiple(context.Context, *DropletMultiCreateRequest) ([]Droplet, *Response, error)

	// Delete deletes a droplet by ID.
	Delete(context.Context, int) (*Response, error)

	// DeleteByTag deletes all droplets which have the provided tag name.
	DeleteByTag(context.Context, string) (*Response, error)

	// Kernels lists available kernels for a droplet.
	Kernels(context.Context, int, *ListOptions) ([]Kernel, *Response, error)

	// Snapshots lists snapshots taken from a droplet.
	Snapshots(context.Context, int, *ListOptions) ([]Image, *Response, error)

	// Backups lists backup images for a droplet.
	Backups(context.Context, int, *ListOptions) ([]Image, *Response, error)

	// Actions lists the actions (reboot, resize, snapshot, etc.) for a droplet.
	Actions(context.Context, int, *ListOptions) ([]Action, *Response, error)

	// Neighbors returns any Droplets on the same physical host as a droplet.
	// The DigitalOcean API endpoint is: GET /v2/droplets/{droplet_id}/neighbors.
	Neighbors(context.Context, int) ([]Droplet, *Response, error)

	// GetBackupPolicy returns the backup policy for the droplet.
	GetBackupPolicy(context.Context, int) (*DropletBackupPolicy, *Response, error)

	// ListBackupPolicies lists all droplet backup policies.
	ListBackupPolicies(context.Context, *ListOptions) (map[int]*DropletBackupPolicy, *Response, error)

	// ListSupportedBackupPolicies lists available backup policy configurations
	// supported by DigitalOcean.
	ListSupportedBackupPolicies(context.Context) ([]*SupportedBackupPolicy, *Response, error)
}

// DropletsServiceOp handles communication with the Droplet related methods of the
// DigitalOcean API. It implements the DropletsService interface.
type DropletsServiceOp struct {
	client *Client
}

var _ DropletsService = &DropletsServiceOp{}

// Droplet represents a DigitalOcean Droplet (VM instance).
//
// Note: fields mirror the JSON returned by the DO API and should not be
// rearranged or removed without updating the JSON tags.
type Droplet struct {
	ID               int           `json:"id,float64,omitempty"`
	Name             string        `json:"name,omitempty"`
	Memory           int           `json:"memory,omitempty"`
	Vcpus            int           `json:"vcpus,omitempty"`
	Disk             int           `json:"disk,omitempty"`
	Region           *Region       `json:"region,omitempty"`
	Image            *Image        `json:"image,omitempty"`
	Size             *Size         `json:"size,omitempty"`
	SizeSlug         string        `json:"size_slug,omitempty"`
	BackupIDs        []int         `json:"backup_ids,omitempty"`
	NextBackupWindow *BackupWindow `json:"next_backup_window,omitempty"`
	SnapshotIDs      []int         `json:"snapshot_ids,omitempty"`
	Features         []string      `json:"features,omitempty"`
	Locked           bool          `json:"locked,bool,omitempty"`
	Status           string        `json:"status,omitempty"`
	Networks         *Networks     `json:"networks,omitempty"`
	Created          string        `json:"created_at,omitempty"`
	Kernel           *Kernel       `json:"kernel,omitempty"`
	Tags             []string      `json:"tags,omitempty"`
	VolumeIDs        []string      `json:"volume_ids"`
	VPCUUID          string        `json:"vpc_uuid,omitempty"`
}

// PublicIPv4 returns the public IPv4 address for the Droplet.
//
// If the droplet has no network information an error is returned. If no
// public IPv4 address is present an empty string is returned.
func (d *Droplet) PublicIPv4() (string, error) {
	if d.Networks == nil {
		return "", errNoNetworks
	}

	for _, v4 := range d.Networks.V4 {
		if v4.Type == "public" {
			return v4.IPAddress, nil
		}
	}

	// No public IPv4 found; return empty string with no error.
	return "", nil
}

// PrivateIPv4 returns the private IPv4 address for the Droplet.
//
// If the droplet has no networks an error is returned. If no private IPv4 is
// present an empty string is returned.
func (d *Droplet) PrivateIPv4() (string, error) {
	if d.Networks == nil {
		return "", errNoNetworks
	}

	for _, v4 := range d.Networks.V4 {
		if v4.Type == "private" {
			return v4.IPAddress, nil
		}
	}

	return "", nil
}

// PublicIPv6 returns the public IPv6 address for the Droplet.
//
// If the droplet has no networks an error is returned. If no public IPv6 is
// present an empty string is returned.
func (d *Droplet) PublicIPv6() (string, error) {
	if d.Networks == nil {
		return "", errNoNetworks
	}

	for _, v6 := range d.Networks.V6 {
		if v6.Type == "public" {
			return v6.IPAddress, nil
		}
	}

	return "", nil
}

// Kernel object returned by droplet kernel endpoints.
type Kernel struct {
	ID      int    `json:"id,float64,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// BackupWindow represents the time window for the droplet backup.
type BackupWindow struct {
	Start *Timestamp `json:"start,omitempty"`
	End   *Timestamp `json:"end,omitempty"`
}

// String returns a compact JSON representation of the droplet.
func (d Droplet) String() string {
	return Stringify(d)
}

// URN returns the droplet ID in a valid DO API URN form.
//
// Example: ToURN("Droplet", 123) might return "do:droplet:123".
func (d Droplet) URN() string {
	return ToURN("Droplet", d.ID)
}

// dropletRoot is the JSON response wrapper for a single droplet.
type dropletRoot struct {
	Droplet *Droplet `json:"droplet"`
	Links   *Links   `json:"links,omitempty"`
}

// dropletsRoot is the JSON response wrapper for a list of droplets.
type dropletsRoot struct {
	Droplets []Droplet `json:"droplets"`
	Links    *Links    `json:"links"`
	Meta     *Meta     `json:"meta"`
}

// DropletCreateImage identifies an image for the create request. It prefers slug over ID.
// The custom MarshalJSON ensures a slug string is sent if present, otherwise the numeric id.
//
// Example JSON sent to API: "ubuntu-20-04-x64" or 1234
type DropletCreateImage struct {
	ID   int
	Slug string
}

// MarshalJSON returns either the slug or id of the image. It returns the id
// if the slug is empty. This is used to support the API's flexible payload.
func (d DropletCreateImage) MarshalJSON() ([]byte, error) {
	if d.Slug != "" {
		return json.Marshal(d.Slug)
	}

	return json.Marshal(d.ID)
}

// DropletCreateVolume identifies a volume to attach for the create request.
// Prefer ID over name. The MarshalJSON sends either { "id": "<id>" } or { "name": "<name>" }.
type DropletCreateVolume struct {
	ID string
	// Deprecated: Name is kept for compatibility. ID is preferred.
	Name string
}

// MarshalJSON returns an object with either the ID or name of the volume. It
// prefers the ID over the name.
func (d DropletCreateVolume) MarshalJSON() ([]byte, error) {
	if d.ID != "" {
		return json.Marshal(struct {
			ID string `json:"id"`
		}{ID: d.ID})
	}

	return json.Marshal(struct {
		Name string `json:"name"`
	}{Name: d.Name})
}

// DropletCreateSSHKey identifies a SSH Key for the create request. Prefer fingerprint over ID.
type DropletCreateSSHKey struct {
	ID          int
	Fingerprint string
}

// MarshalJSON returns either the fingerprint or id of the ssh key. It returns
// the id if the fingerprint is empty. This matches the DO API which allows
// either a fingerprint string or numeric id.
func (d DropletCreateSSHKey) MarshalJSON() ([]byte, error) {
	if d.Fingerprint != "" {
		return json.Marshal(d.Fingerprint)
	}

	return json.Marshal(d.ID)
}

// DropletCreateRequest represents the payload to create a single droplet.
//
// Fields correspond to the DigitalOcean API create endpoint. Note that
// WithDropletAgent is optional and if provided will instruct the API whether
// to install the droplet agent.
type DropletCreateRequest struct {
	Name              string                      `json:"name"`
	Region            string                      `json:"region"`
	Size              string                      `json:"size"`
	Image             DropletCreateImage          `json:"image"`
	SSHKeys           []DropletCreateSSHKey       `json:"ssh_keys"`
	Backups           bool                        `json:"backups"`
	IPv6              bool                        `json:"ipv6"`
	PrivateNetworking bool                        `json:"private_networking"`
	Monitoring        bool                        `json:"monitoring"`
	UserData          string                      `json:"user_data,omitempty"`
	Volumes           []DropletCreateVolume       `json:"volumes,omitempty"`
	Tags              []string                    `json:"tags"`
	VPCUUID           string                      `json:"vpc_uuid,omitempty"`
	WithDropletAgent  *bool                       `json:"with_droplet_agent,omitempty"`
	BackupPolicy      *DropletBackupPolicyRequest `json:"backup_policy,omitempty"`
}

// DropletMultiCreateRequest is a request to create multiple Droplets at once.
// The API accepts arrays of names and will return the created droplets.
type DropletMultiCreateRequest struct {
	Names             []string                    `json:"names"`
	Region            string                      `json:"region"`
	Size              string                      `json:"size"`
	Image             DropletCreateImage          `json:"image"`
	SSHKeys           []DropletCreateSSHKey       `json:"ssh_keys"`
	Backups           bool                        `json:"backups"`
	IPv6              bool                        `json:"ipv6"`
	PrivateNetworking bool                        `json:"private_networking"`
	Monitoring        bool                        `json:"monitoring"`
	UserData          string                      `json:"user_data,omitempty"`
	Tags              []string                    `json:"tags"`
	VPCUUID           string                      `json:"vpc_uuid,omitempty"`
	WithDropletAgent  *bool                       `json:"with_droplet_agent,omitempty"`
	BackupPolicy      *DropletBackupPolicyRequest `json:"backup_policy,omitempty"`
}

func (d DropletCreateRequest) String() string {
	return Stringify(d)
}

func (d DropletMultiCreateRequest) String() string {
	return Stringify(d)
}

// DropletBackupPolicyRequest defines the backup policy when creating a Droplet.
type DropletBackupPolicyRequest struct {
	Plan    string `json:"plan,omitempty"`
	Weekday string `json:"weekday,omitempty"`
	Hour    *int   `json:"hour,omitempty"`
}

func (d DropletCreateRequest) String() string {
	return Stringify(d)
}

func (d DropletMultiCreateRequest) String() string {
	return Stringify(d)
}

// Networks represents the Droplet's Networks (both IPv4 and IPv6).
type Networks struct {
	V4 []NetworkV4 `json:"v4,omitempty"`
	V6 []NetworkV6 `json:"v6,omitempty"`
}

// NetworkV4 represents a DigitalOcean IPv4 Network record for a droplet.
type NetworkV4 struct {
	IPAddress string `json:"ip_address,omitempty"`
	Netmask   string `json:"netmask,omitempty"`
	Gateway   string `json:"gateway,omitempty"`
	Type      string `json:"type,omitempty"` // "public" or "private"
}

func (n NetworkV4) String() string {
	return Stringify(n)
}

// NetworkV6 represents a DigitalOcean IPv6 network.
type NetworkV6 struct {
	IPAddress string `json:"ip_address,omitempty"`
	Netmask   int    `json:"netmask,omitempty"`
	Gateway   string `json:"gateway,omitempty"`
	Type      string `json:"type,omitempty"` // "public"
}

func (n NetworkV6) String() string {
	return Stringify(n)
}
