/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package do

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
)

const registryHostname = "registry.digitalocean.com"

// Registry wraps a godo Project.
type Registry struct {
	*godo.Registry
}

// Endpoint returns the registry endpoint for image tagging
func (r *Registry) Endpoint() string {
	return fmt.Sprintf("%s/%s", registryHostname, r.Registry.Name)
}

// RegistryService is the godo RegistryService interface.
type RegistryService interface {
	Get() (*Registry, error)
	Create(*godo.RegistryCreateRequest) (*Registry, error)
	Delete() error
	DockerCredentials(*godo.RegistryDockerCredentialsRequest) (*godo.DockerCredentials, error)
	Endpoint() string
}

type registryService struct {
	client *godo.Client
	ctx    context.Context
}

var _ RegistryService = &registryService{}

// NewRegistryService builds an instance of RegistryService.
func NewRegistryService(client *godo.Client) RegistryService {
	return &registryService{
		client: client,
		ctx:    context.Background(),
	}
}

func (rs *registryService) Get() (*Registry, error) {
	r, _, err := rs.client.Registry.Get(rs.ctx)
	if err != nil {
		return nil, err
	}

	return &Registry{Registry: r}, nil
}

func (rs *registryService) Create(cr *godo.RegistryCreateRequest) (*Registry, error) {
	r, _, err := rs.client.Registry.Create(rs.ctx, cr)
	if err != nil {
		return nil, err
	}

	return &Registry{Registry: r}, nil
}

func (rs *registryService) Delete() error {
	_, err := rs.client.Registry.Delete(rs.ctx)
	return err
}

func (rs *registryService) DockerCredentials(request *godo.RegistryDockerCredentialsRequest) (*godo.DockerCredentials, error) {
	dockerConfig, _, err := rs.client.Registry.DockerCredentials(rs.ctx, request)
	if err != nil {
		return nil, err
	}

	return dockerConfig, nil
}

func (rs *registryService) Endpoint() string {
	return registryHostname
}
