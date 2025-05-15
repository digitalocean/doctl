package godo

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	egressGatewaysBasePath = "/v2/egress_gateways"
)

// EgressGatewaysService defines an interface for managing Egress Gateways through the DigitalOcean API
type EgressGatewaysService interface {
	Create(context.Context, *EgressGatewayRequest) (*EgressGateway, *Response, error)
	Get(context.Context, string) (*EgressGateway, *Response, error)
	List(context.Context, *EgressGatewaysListOptions) ([]*EgressGateway, *Response, error)
	Update(context.Context, string, *EgressGatewayRequest) (*EgressGateway, *Response, error)
	Delete(context.Context, string) (*Response, error)
}

// EgressGatewayRequest represents a DigitalOcean Egress Gateway create/update request
type EgressGatewayRequest struct {
	Name               string        `json:"name"`
	Type               string        `json:"type"`
	Region             string        `json:"region"`
	VPCs               []*IngressVPC `json:"vpcs"`
	UDPTimeoutSeconds  uint32        `json:"udp_timeout_seconds,omitempty"`
	ICMPTimeoutSeconds uint32        `json:"icmp_timeout_seconds,omitempty"`
	TCPTimeoutSeconds  uint32        `json:"tcp_timeout_seconds,omitempty"`
}

// EgressGateway represents a DigitalOcean Egress Gateway resource
type EgressGateway struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Type               string        `json:"type"`
	State              string        `json:"state"`
	Region             string        `json:"region"`
	VPCs               []*IngressVPC `json:"vpcs"`
	Egresses           *Egresses     `json:"egresses,omitempty"`
	UDPTimeoutSeconds  uint32        `json:"udp_timeout_seconds,omitempty"`
	ICMPTimeoutSeconds uint32        `json:"icmp_timeout_seconds,omitempty"`
	TCPTimeoutSeconds  uint32        `json:"tcp_timeout_seconds,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// IngressVPC defines the ingress configs supported by an Egress Gateway
type IngressVPC struct {
	VpcUUID              string `json:"vpc_uuid"`
	GatewayIP            string `json:"gateway_ip,omitempty"`
	DefaultEgressGateway bool   `json:"default_egress_gateway,omitempty"`
}

// Egresses define egress routes supported by an Egress Gateway
type Egresses struct {
	PublicGateways []*PublicGateway `json:"public_gateways,omitempty"`
}

// PublicGateway defines the public egress supported by an Egress Gateway
type PublicGateway struct {
	IPv4 string `json:"ipv4"`
}

// EgressGatewaysListOptions define custom options for listing Egress Gateways
type EgressGatewaysListOptions struct {
	ListOptions
	State  []string `json:"state,omitempty"`
	Region []string `json:"region,omitempty"`
	Type   []string `json:"type,omitempty"`
	Name   []string `json:"name,omitempty"`
}

type egressGatewayRoot struct {
	EgressGateway *EgressGateway `json:"egress_gateway"`
}

type egressGatewaysRoot struct {
	EgressGateways []*EgressGateway `json:"egress_gateways"`
	Links          *Links           `json:"links"`
	Meta           *Meta            `json:"meta"`
}

// EgressGatewaysServiceOp handles communication with Egress Gateway methods of the DigitalOcean API
type EgressGatewaysServiceOp struct {
	client *Client
}

var _ EgressGatewaysService = &EgressGatewaysServiceOp{}

// Create a new Egress Gateway
func (n *EgressGatewaysServiceOp) Create(ctx context.Context, createReq *EgressGatewayRequest) (*EgressGateway, *Response, error) {
	req, err := n.client.NewRequest(ctx, http.MethodPost, egressGatewaysBasePath, createReq)
	if err != nil {
		return nil, nil, err
	}
	root := new(egressGatewayRoot)
	resp, err := n.client.Do(ctx, req, root)
	if err != nil {
		return nil, nil, err
	}
	return root.EgressGateway, resp, nil
}

// Get an existing Egress Gateway
func (n *EgressGatewaysServiceOp) Get(ctx context.Context, id string) (*EgressGateway, *Response, error) {
	req, err := n.client.NewRequest(ctx, http.MethodGet, fmt.Sprintf("%s/%s", egressGatewaysBasePath, id), nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(egressGatewayRoot)
	resp, err := n.client.Do(ctx, req, root)
	if err != nil {
		return nil, nil, err
	}
	return root.EgressGateway, resp, nil
}

// List all active Egress Gateways
func (n *EgressGatewaysServiceOp) List(ctx context.Context, opts *EgressGatewaysListOptions) ([]*EgressGateway, *Response, error) {
	path, err := addOptions(egressGatewaysBasePath, opts)
	if err != nil {
		return nil, nil, err
	}
	req, err := n.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(egressGatewaysRoot)
	resp, err := n.client.Do(ctx, req, root)
	if err != nil {
		return nil, nil, err
	}
	if root.Links != nil {
		resp.Links = root.Links
	}
	if root.Meta != nil {
		resp.Meta = root.Meta
	}
	return root.EgressGateways, resp, nil
}

// Update an existing Egress Gateway
func (n *EgressGatewaysServiceOp) Update(ctx context.Context, id string, updateReq *EgressGatewayRequest) (*EgressGateway, *Response, error) {
	req, err := n.client.NewRequest(ctx, http.MethodPut, fmt.Sprintf("%s/%s", egressGatewaysBasePath, id), updateReq)
	if err != nil {
		return nil, nil, err
	}
	root := new(egressGatewayRoot)
	resp, err := n.client.Do(ctx, req, root)
	if err != nil {
		return nil, nil, err
	}
	return root.EgressGateway, resp, nil
}

// Delete an existing Egress Gateway
func (n *EgressGatewaysServiceOp) Delete(ctx context.Context, id string) (*Response, error) {
	req, err := n.client.NewRequest(ctx, http.MethodDelete, fmt.Sprintf("%s/%s", egressGatewaysBasePath, id), nil)
	if err != nil {
		return nil, err
	}
	return n.client.Do(ctx, req, nil)
}
