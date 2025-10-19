package godo

import (
	"context"
	"fmt"
	"net/http"
)

const nfsBasePath = "v2/nfs"

type NfsService interface {
	// List retrieves a list of NFS shares with optional filtering via ListOptions and region
	List(context.Context, *ListOptions, string) ([]*Nfs, *Response, error)
	// Create creates a new NFS share with the provided configuration
	Create(context.Context, *NfsCreateRequest) (*Nfs, *Response, error)
	// Delete removes an NFS share by its ID and region
	Delete(context.Context, string, string) (*Response, error)
	// Get retrieves a specific NFS share by its ID and region
	Get(context.Context, string, string) (*Nfs, *Response, error)
}

// NfsServiceOp handles communication with the NFS related methods of the
// DigitalOcean API.
type NfsServiceOp struct {
	client *Client
}

var _ NfsService = &NfsServiceOp{}

// Nfs represents a DigitalOcean NFS share
type Nfs struct {
	// ID is the unique identifier for the NFS share
	ID string `json:"id"`
	// Name is the human-readable name for the NFS share
	Name string `json:"name"`
	// SizeGib is the size of the NFS share in gibibytes
	SizeGib int `json:"size_gib"`
	// Region is the datacenter region where the NFS share is located
	Region string `json:"region"`
	// Status represents the current state of the NFS share
	Status string `json:"status"`
	// CreatedAt is the timestamp when the NFS share was created
	CreatedAt string `json:"created_at"`
	// VpcIDs is a list of VPC IDs that have access to the NFS share
	VpcIDs []string `json:"vpc_ids"`
}

// NfsCreateRequest represents a request to create an NFS share.
type NfsCreateRequest struct {
	Name    string   `json:"name"`
	SizeGib int      `json:"size_gib"`
	Region  string   `json:"region"`
	VpcIDs  []string `json:"vpc_ids,omitempty"`
}

// nfsRoot represents a response from the DigitalOcean API
type nfsRoot struct {
	Share *Nfs `json:"share"`
}

// nfsListRoot represents a response from the DigitalOcean API
type nfsListRoot struct {
	Shares []*Nfs `json:"shares,omitempty"`
	Links  *Links `json:"links,omitempty"`
	Meta   *Meta  `json:"meta"`
}

// nfsOptions represents the query param options for NFS operations
type nfsOptions struct {
	Region string `url:"region"`
}

// Create creates a new NFS share.
func (s *NfsServiceOp) Create(ctx context.Context, createRequest *NfsCreateRequest) (*Nfs, *Response, error) {
	if createRequest == nil {
		return nil, nil, NewArgError("createRequest", "cannot be nil")
	}

	if createRequest.SizeGib < 50 {
		return nil, nil, NewArgError("size_gib", "it cannot be less than 50Gib")
	}

	req, err := s.client.NewRequest(ctx, http.MethodPost, nfsBasePath, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(nfsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Share, resp, nil
}

// Get retrieves an NFS share by ID and region.
func (s *NfsServiceOp) Get(ctx context.Context, id string, region string) (*Nfs, *Response, error) {
	if id == "" {
		return nil, nil, NewArgError("id", "cannot be empty")
	}
	if region == "" {
		return nil, nil, NewArgError("region", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", nfsBasePath, id)

	getOpts := &nfsOptions{Region: region}
	path, err := addOptions(path, getOpts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(nfsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Share, resp, nil
}

// List returns a list of NFS shares.
func (s *NfsServiceOp) List(ctx context.Context, opts *ListOptions, region string) ([]*Nfs, *Response, error) {
	if region == "" {
		return nil, nil, NewArgError("region", "cannot be empty")
	}

	path, err := addOptions(nfsBasePath, opts)
	if err != nil {
		return nil, nil, err
	}

	listOpts := &nfsOptions{Region: region}
	path, err = addOptions(path, listOpts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(nfsListRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	if root.Links != nil {
		resp.Links = root.Links
	}
	if root.Meta != nil {
		resp.Meta = root.Meta
	}

	return root.Shares, resp, nil
}

// Delete deletes an NFS share by ID and region.
func (s *NfsServiceOp) Delete(ctx context.Context, id string, region string) (*Response, error) {
	if id == "" {
		return nil, NewArgError("id", "cannot be empty")
	}
	if region == "" {
		return nil, NewArgError("region", "cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", nfsBasePath, id)

	deleteOpts := &nfsOptions{Region: region}
	path, err := addOptions(path, deleteOpts)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}
