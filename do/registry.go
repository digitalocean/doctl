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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/digitalocean/godo"
)

// RegistryHostname is the hostname for the DO registry
const RegistryHostname = "registry.digitalocean.com"

// Registry wraps a godo Registry.
type Registry struct {
	*godo.Registry
}

// Repository wraps a godo Repository
type Repository struct {
	*godo.Repository
}

// RepositoryV2 wraps a godo RepositoryV2
type RepositoryV2 struct {
	*godo.RepositoryV2
}

// RepositoryManifest wraps a godo RepositoryManifest
type RepositoryManifest struct {
	*godo.RepositoryManifest
}

// RepositoryTag wraps a godo RepositoryTag
type RepositoryTag struct {
	*godo.RepositoryTag
}

// GarbageCollection wraps a godo GarbageCollection
type GarbageCollection struct {
	*godo.GarbageCollection
}

// RegistrySubscriptionTier wraps a godo RegistrySubscriptionTier
type RegistrySubscriptionTier struct {
	*godo.RegistrySubscriptionTier
}

// Endpoint returns the registry endpoint for image tagging
func (r *Registry) Endpoint() string {
	return fmt.Sprintf("%s/%s", RegistryHostname, r.Registry.Name)
}

// RegistryService is the godo RegistryService interface.
type RegistryService interface {
	Get() (*Registry, error)
	Create(*godo.RegistryCreateRequest) (*Registry, error)
	Delete() error
	DockerCredentials(*godo.RegistryDockerCredentialsRequest) (*godo.DockerCredentials, error)
	ListRepositoryTags(string, string) ([]RepositoryTag, error)
	ListRepositoryManifests(string, string) ([]RepositoryManifest, error)
	ListRepositories(string) ([]Repository, error)
	ListRepositoriesV2(string) ([]RepositoryV2, error)
	DeleteTag(string, string, string) error
	DeleteManifest(string, string, string) error
	Endpoint() string
	StartGarbageCollection(string, *godo.StartGarbageCollectionRequest) (*GarbageCollection, error)
	GetGarbageCollection(string) (*GarbageCollection, error)
	ListGarbageCollections(string) ([]GarbageCollection, error)
	CancelGarbageCollection(string, string) (*GarbageCollection, error)
	GetSubscriptionTiers() ([]RegistrySubscriptionTier, error)
	GetAvailableRegions() ([]string, error)
	RevokeOAuthToken(token string, endpoint string) error
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

func (rs *registryService) ListRepositories(registry string) ([]Repository, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := rs.client.Registry.ListRepositories(rs.ctx, registry, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]Repository, len(si))
	for i := range si {
		a := si[i].(*godo.Repository)
		list[i] = Repository{Repository: a}
	}

	return list, nil
}

func (rs *registryService) ListRepositoriesV2(registry string) ([]RepositoryV2, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := rs.client.Registry.ListRepositoriesV2(rs.ctx, registry, &godo.TokenListOptions{
			Page:    opt.Page,
			PerPage: opt.PerPage,
			// there's an opportunity for optimization here by using page token instead of page
		})
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]RepositoryV2, len(si))
	for i := range si {
		a := si[i].(*godo.RepositoryV2)
		list[i] = RepositoryV2{RepositoryV2: a}
	}

	return list, nil
}

func (rs *registryService) ListRepositoryTags(registry, repository string) ([]RepositoryTag, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := rs.client.Registry.ListRepositoryTags(rs.ctx, registry, repository, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]RepositoryTag, len(si))
	for i := range si {
		a := si[i].(*godo.RepositoryTag)
		list[i] = RepositoryTag{RepositoryTag: a}
	}

	return list, nil
}

func (rs *registryService) ListRepositoryManifests(registry, repository string) ([]RepositoryManifest, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := rs.client.Registry.ListRepositoryManifests(rs.ctx, registry, repository, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]RepositoryManifest, len(si))
	for i := range si {
		a := si[i].(*godo.RepositoryManifest)
		list[i] = RepositoryManifest{RepositoryManifest: a}
	}

	return list, nil
}

func (rs *registryService) DeleteTag(registry, repository, tag string) error {
	_, err := rs.client.Registry.DeleteTag(rs.ctx, registry, repository, tag)
	return err
}

func (rs *registryService) DeleteManifest(registry, repository, digest string) error {
	_, err := rs.client.Registry.DeleteManifest(rs.ctx, registry, repository, digest)
	return err
}

func (rs *registryService) StartGarbageCollection(registry string, gcRequest *godo.StartGarbageCollectionRequest) (*GarbageCollection, error) {
	gc, _, err := rs.client.Registry.StartGarbageCollection(rs.ctx, registry, gcRequest)
	if err != nil {
		return nil, err
	}

	return &GarbageCollection{gc}, nil
}

func (rs *registryService) GetGarbageCollection(registry string) (*GarbageCollection, error) {
	gc, _, err := rs.client.Registry.GetGarbageCollection(rs.ctx, registry)
	if err != nil {
		return nil, err
	}

	return &GarbageCollection{gc}, nil
}

func (rs *registryService) ListGarbageCollections(registry string) ([]GarbageCollection, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := rs.client.Registry.ListGarbageCollections(rs.ctx, registry, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]GarbageCollection, len(si))
	for i := range si {
		a := si[i].(*godo.GarbageCollection)
		list[i] = GarbageCollection{a}
	}

	return list, err
}

func (rs *registryService) CancelGarbageCollection(registry, garbageCollection string) (*GarbageCollection, error) {
	gc, _, err := rs.client.Registry.UpdateGarbageCollection(rs.ctx, registry,
		garbageCollection, &godo.UpdateGarbageCollectionRequest{
			Cancel: true,
		})
	if err != nil {
		return nil, err
	}

	return &GarbageCollection{gc}, nil
}

func (rs *registryService) Endpoint() string {
	return RegistryHostname
}

func (rs *registryService) GetSubscriptionTiers() ([]RegistrySubscriptionTier, error) {
	opts, _, err := rs.client.Registry.GetOptions(rs.ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]RegistrySubscriptionTier, len(opts.SubscriptionTiers))
	for i, tier := range opts.SubscriptionTiers {
		ret[i] = RegistrySubscriptionTier{tier}
	}

	return ret, nil
}

func (rs *registryService) GetAvailableRegions() ([]string, error) {
	opts, _, err := rs.client.Registry.GetOptions(rs.ctx)
	if err != nil {
		return nil, err
	}

	return opts.AvailableRegions, nil
}

func (rs *registryService) RevokeOAuthToken(token string, endpoint string) error {
	data := url.Values{}
	data.Set("token", token)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("error revoking token: " + http.StatusText(resp.StatusCode))
	}

	return err
}

// RegistriesService is the godo RegistriesService interface.
type RegistriesService interface {
	Get(string) (*Registry, error)
	List() ([]Registry, error)
	Create(*godo.RegistryCreateRequest) (*Registry, error)
	Delete(string) error
	DockerCredentials(string, *godo.RegistryDockerCredentialsRequest) (*godo.DockerCredentials, error)
	ListRepositories(string) ([]Repository, error)
	ListRepositoriesV2(string) ([]RepositoryV2, error)
	ListRepositoryTags(string, string) ([]RepositoryTag, error)
	DeleteTag(string, string, string) error
	ListRepositoryManifests(string, string) ([]RepositoryManifest, error)
	DeleteManifest(string, string, string) error
	StartGarbageCollection(string, *godo.StartGarbageCollectionRequest) (*GarbageCollection, error)
	GetGarbageCollection(string) (*GarbageCollection, error)
	ListGarbageCollections(string) ([]GarbageCollection, error)
	UpdateGarbageCollection(string, string, *godo.UpdateGarbageCollectionRequest) (*GarbageCollection, error)
	GetOptions() (*godo.RegistryOptions, error)
	GetSubscriptionTiers() ([]RegistrySubscriptionTier, error)
	GetAvailableRegions() ([]string, error)
}

type registriesService struct {
	client *godo.Client
	ctx    context.Context
}

var _ RegistriesService = &registriesService{}

// NewRegistriesService builds an instance of RegistriesService.
func NewRegistriesService(client *godo.Client) RegistriesService {
	return &registriesService{
		client: client,
		ctx:    context.Background(),
	}
}

// Get retrieves a registry by name.
func (rs *registriesService) Get(name string) (*Registry, error) {
	r, _, err := rs.client.Registries.Get(rs.ctx, name)
	if err != nil {
		return nil, err
	}

	return &Registry{Registry: r}, nil
}

// List retrieves all secondary registries.
func (rs *registriesService) List() ([]Registry, error) {
	list, _, err := rs.client.Registries.List(rs.ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]Registry, len(list))
	for i, r := range list {
		ret[i] = Registry{Registry: r}
	}

	return ret, nil
}

// Create creates a secondary registry.
func (rs *registriesService) Create(cr *godo.RegistryCreateRequest) (*Registry, error) {
	r, _, err := rs.client.Registries.Create(rs.ctx, cr)
	if err != nil {
		return nil, err
	}

	return &Registry{Registry: r}, nil
}

// Delete deletes a secondary registry.
func (rs *registriesService) Delete(name string) error {
	_, err := rs.client.Registries.Delete(rs.ctx, name)
	return err
}

// DockerCredentials retrieves docker credentials for a secondary registry.
func (rs *registriesService) DockerCredentials(name string, request *godo.RegistryDockerCredentialsRequest) (*godo.DockerCredentials, error) {
	dockerConfig, _, err := rs.client.Registries.DockerCredentials(rs.ctx, name, request)
	if err != nil {
		return nil, err
	}

	return dockerConfig, nil
}

// ListRepositories lists repositories in a registry.
func (rs *registriesService) ListRepositories(registryName string) ([]Repository, error) {
	opts := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	repositories, _, err := rs.client.Registry.ListRepositories(rs.ctx, registryName, opts)
	if err != nil {
		return nil, err
	}

	ret := make([]Repository, len(repositories))
	for i, r := range repositories {
		ret[i] = Repository{Repository: r}
	}

	return ret, nil
}

// ListRepositoriesV2 lists repositories in a registry using the V2 API.
func (rs *registriesService) ListRepositoriesV2(registryName string) ([]RepositoryV2, error) {
	opts := &godo.TokenListOptions{
		Page:    1,
		PerPage: 200,
	}

	repositories, _, err := rs.client.Registries.ListRepositoriesV2(rs.ctx, registryName, opts)
	if err != nil {
		return nil, err
	}

	ret := make([]RepositoryV2, len(repositories))
	for i, r := range repositories {
		ret[i] = RepositoryV2{RepositoryV2: r}
	}

	return ret, nil
}

// ListRepositoryTags lists tags for a repository.
func (rs *registriesService) ListRepositoryTags(registryName, repositoryName string) ([]RepositoryTag, error) {
	opts := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	tags, _, err := rs.client.Registries.ListRepositoryTags(rs.ctx, registryName, repositoryName, opts)
	if err != nil {
		return nil, err
	}

	ret := make([]RepositoryTag, len(tags))
	for i, t := range tags {
		ret[i] = RepositoryTag{RepositoryTag: t}
	}

	return ret, nil
}

// DeleteTag deletes a tag from a repository.
func (rs *registriesService) DeleteTag(registryName, repositoryName, tag string) error {
	_, err := rs.client.Registries.DeleteTag(rs.ctx, registryName, repositoryName, tag)
	return err
}

// ListRepositoryManifests lists manifests for a repository.
func (rs *registriesService) ListRepositoryManifests(registryName, repositoryName string) ([]RepositoryManifest, error) {
	opts := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	manifests, _, err := rs.client.Registries.ListRepositoryManifests(rs.ctx, registryName, repositoryName, opts)
	if err != nil {
		return nil, err
	}

	ret := make([]RepositoryManifest, len(manifests))
	for i, m := range manifests {
		ret[i] = RepositoryManifest{RepositoryManifest: m}
	}

	return ret, nil
}

// DeleteManifest deletes a manifest from a repository.
func (rs *registriesService) DeleteManifest(registryName, repositoryName, digest string) error {
	_, err := rs.client.Registries.DeleteManifest(rs.ctx, registryName, repositoryName, digest)
	return err
}

// StartGarbageCollection starts a garbage collection for a registry.
func (rs *registriesService) StartGarbageCollection(registryName string, request *godo.StartGarbageCollectionRequest) (*GarbageCollection, error) {
	gc, _, err := rs.client.Registries.StartGarbageCollection(rs.ctx, registryName, request)
	if err != nil {
		return nil, err
	}

	return &GarbageCollection{GarbageCollection: gc}, nil
}

// GetGarbageCollection gets the active garbage collection for a registry.
func (rs *registriesService) GetGarbageCollection(registryName string) (*GarbageCollection, error) {
	gc, _, err := rs.client.Registries.GetGarbageCollection(rs.ctx, registryName)
	if err != nil {
		return nil, err
	}

	return &GarbageCollection{GarbageCollection: gc}, nil
}

// ListGarbageCollections lists garbage collections for a registry.
func (rs *registriesService) ListGarbageCollections(registryName string) ([]GarbageCollection, error) {
	opts := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	gcs, _, err := rs.client.Registries.ListGarbageCollections(rs.ctx, registryName, opts)
	if err != nil {
		return nil, err
	}

	ret := make([]GarbageCollection, len(gcs))
	for i, gc := range gcs {
		ret[i] = GarbageCollection{GarbageCollection: gc}
	}

	return ret, nil
}

// UpdateGarbageCollection updates a garbage collection for a registry.
func (rs *registriesService) UpdateGarbageCollection(registryName, gcUUID string, request *godo.UpdateGarbageCollectionRequest) (*GarbageCollection, error) {
	gc, _, err := rs.client.Registries.UpdateGarbageCollection(rs.ctx, registryName, gcUUID, request)
	if err != nil {
		return nil, err
	}

	return &GarbageCollection{GarbageCollection: gc}, nil
}

// GetOptions gets the available options for registries.
func (rs *registriesService) GetOptions() (*godo.RegistryOptions, error) {
	opts, _, err := rs.client.Registries.GetOptions(rs.ctx)
	if err != nil {
		return nil, err
	}

	return opts, nil
}

// GetSubscriptionTiers gets the available subscription tiers for registries.
func (rs *registriesService) GetSubscriptionTiers() ([]RegistrySubscriptionTier, error) {
	opts, _, err := rs.client.Registries.GetOptions(rs.ctx)
	if err != nil {
		return nil, err
	}

	tiers := make([]RegistrySubscriptionTier, len(opts.SubscriptionTiers))
	for i, tier := range opts.SubscriptionTiers {
		tiers[i] = RegistrySubscriptionTier{RegistrySubscriptionTier: tier}
	}

	return tiers, nil
}

// GetAvailableRegions gets the available regions for registries.
func (rs *registriesService) GetAvailableRegions() ([]string, error) {
	opts, _, err := rs.client.Registries.GetOptions(rs.ctx)
	if err != nil {
		return nil, err
	}

	return opts.AvailableRegions, nil
}
