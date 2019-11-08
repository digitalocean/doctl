package godo

import (
	"context"
	"net/http"
)

const (
	registryPath = "/v2/registry"
)

// RegistryService is an interface for interfacing with the Registry endpoints
// of the DigitalOcean API.
// See: https://developers.digitalocean.com/documentation/v2#registry
type RegistryService interface {
	Create(context.Context, *RegistryCreateRequest) (*Registry, *Response, error)
	Get(context.Context) (*Registry, *Response, error)
	Delete(context.Context) (*Response, error)
}

var _ RegistryService = &RegistryServiceOp{}

// RegistryServiceOp handles communication with Registry methods of the DigitalOcean API.
type RegistryServiceOp struct {
	client *Client
}

// RegistryCreateRequest represents a request to create a registry.
type RegistryCreateRequest struct {
	Name string `json:"name,omitempty"`
}

// Registry represents a registry.
type Registry struct {
	Name string `json:"name,omitempty"`
}

type registryRoot struct {
	Registry *Registry `json:"registry,omitempty"`
}

// Get retrieves the details of a Registry.
func (svc *RegistryServiceOp) Get(ctx context.Context) (*Registry, *Response, error) {
	req, err := svc.client.NewRequest(ctx, http.MethodGet, registryPath, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(registryRoot)
	resp, err := svc.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.Registry, resp, nil
}

// Create creates a registry.
func (svc *RegistryServiceOp) Create(ctx context.Context, create *RegistryCreateRequest) (*Registry, *Response, error) {
	req, err := svc.client.NewRequest(ctx, http.MethodPost, registryPath, create)
	if err != nil {
		return nil, nil, err
	}
	root := new(registryRoot)
	resp, err := svc.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.Registry, resp, nil
}

// Delete deletes a registry. There is no way to recover a registry once it has
// been destroyed.
func (svc *RegistryServiceOp) Delete(ctx context.Context) (*Response, error) {
	req, err := svc.client.NewRequest(ctx, http.MethodDelete, registryPath, nil)
	if err != nil {
		return nil, err
	}
	resp, err := svc.client.Do(ctx, req, nil)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
