package godo

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const resourceV6Type = "ReservedIPv6"
const reservedIPV6sBasePath = "v2/reserved_ipv6"

// ReservedIPV6sService is an interface for interfacing with the reserved IPV6s
// endpoints of the Digital Ocean API.
type ReservedIPV6sService interface {
	List(context.Context, *ListOptions) ([]ReservedIPV6, *Response, error)
	Get(context.Context, string) (*ReservedIPV6, *Response, error)
	Create(context.Context, *ReservedIPV6CreateRequest) (*ReservedIPV6, *Response, error)
	Delete(context.Context, string) (*Response, error)
}

// ReservedIPV6sServiceOp handles communication with the reserved IPs related methods of the
// DigitalOcean API.
type ReservedIPV6sServiceOp struct {
	client *Client
}

var _ ReservedIPV6sService = (*ReservedIPV6sServiceOp)(nil)

// ReservedIPV6 represents a Digital Ocean reserved IP.
type ReservedIPV6 struct {
	RegionSlug string    `json:"region_slug"`
	IP         string    `json:"ip"`
	ReservedAt time.Time `json:"reserved_at"`
	Droplet    *Droplet  `json:"droplet,omitempty"`
}

func (f ReservedIPV6) String() string {
	return Stringify(f)
}

// URN returns the reserved IP in a valid DO API URN form.
func (f ReservedIPV6) URN() string {
	return ToURN(resourceV6Type, f.IP)
}

type reservedIPV6sRoot struct {
	ReservedIPs []ReservedIPV6 `json:"reserved_ips"`
	Links       *Links         `json:"links"`
	Meta        *Meta          `json:"meta"`
}

// ReservedIPV6CreateRequest represents a request to reserve a reserved IP.
type ReservedIPV6CreateRequest struct {
	Region string `json:"region_slug,omitempty"`
}

// List all reserved IPV6s.
func (r *ReservedIPV6sServiceOp) List(ctx context.Context, opt *ListOptions) ([]ReservedIPV6, *Response, error) {
	path := reservedIPV6sBasePath
	path, err := addOptions(path, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := r.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(reservedIPV6sRoot)
	resp, err := r.client.Do(ctx, req, root)
	if err != nil {
		return nil, nil, err
	}
	if l := root.Links; l != nil {
		resp.Links = l
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.ReservedIPs, resp, err
}

// Get an individual reserved IPv6.
func (r *ReservedIPV6sServiceOp) Get(ctx context.Context, ip string) (*ReservedIPV6, *Response, error) {
	path := fmt.Sprintf("%s/%s", reservedIPV6sBasePath, ip)

	req, err := r.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(ReservedIPV6)
	resp, err := r.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, err
}

// Create a new IPv6
func (r *ReservedIPV6sServiceOp) Create(ctx context.Context, reserveRequest *ReservedIPV6CreateRequest) (*ReservedIPV6, *Response, error) {
	path := reservedIPV6sBasePath

	req, err := r.client.NewRequest(ctx, http.MethodPost, path, reserveRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(ReservedIPV6)
	resp, err := r.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, err
}

// Delete a reserved IPv6.
func (r *ReservedIPV6sServiceOp) Delete(ctx context.Context, ip string) (*Response, error) {
	path := fmt.Sprintf("%s/%s", reservedIPV6sBasePath, ip)

	req, err := r.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return r.client.Do(ctx, req, nil)
}
