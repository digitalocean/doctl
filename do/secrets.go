/*
Copyright 2026 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/godo"
)

// Secret wraps a godo.Secret.
type Secret struct {
	*godo.Secret
}

// Secrets is a slice of Secret.
type Secrets []Secret

// SecretVersion wraps a godo.SecretVersion.
type SecretVersion struct {
	*godo.SecretVersion
}

// SecretVersions is a slice of SecretVersion.
type SecretVersions []SecretVersion

// SecretWriteResult wraps a godo.SecretWriteResult.
type SecretWriteResult struct {
	*godo.SecretWriteResult
}

// SecretsList holds paginated secrets and list-level metadata.
type SecretsList struct {
	Secrets            Secrets
	UnavailableRegions []string
}

// SecretsService is an interface for interacting with DigitalOcean's Secrets Manager API.
type SecretsService interface {
	List() (*SecretsList, error)
	Get(name, region string) (*Secret, error)
	ListVersions(name, region string) (SecretVersions, error)
	Create(*godo.SecretCreateRequest) (*SecretWriteResult, error)
	Update(name string, req *godo.SecretUpdateRequest) (*SecretWriteResult, error)
	Delete(name, region string) error
	Restore(name, region string) error
}

type secretsService struct {
	client *godo.Client
}

var _ SecretsService = (*secretsService)(nil)

// NewSecretsService builds a SecretsService instance.
func NewSecretsService(godoClient *godo.Client) SecretsService {
	return &secretsService{
		client: godoClient,
	}
}

func (s *secretsService) List() (*SecretsList, error) {
	var unavailableRegions []string

	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.Secrets.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		if opt.Page == 1 && len(list.UnavailableRegions) > 0 {
			unavailableRegions = list.UnavailableRegions
		}

		si := make([]any, len(list.Secrets))
		for i := range list.Secrets {
			si[i] = list.Secrets[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	secrets := make(Secrets, len(si))
	for i := range si {
		secret, ok := si[i].(*godo.Secret)
		if !ok {
			return nil, errors.New("unexpected value in response")
		}
		secrets[i] = Secret{Secret: secret}
	}

	return &SecretsList{
		Secrets:            secrets,
		UnavailableRegions: unavailableRegions,
	}, nil
}

func (s *secretsService) Get(name, region string) (*Secret, error) {
	secret, _, err := s.client.Secrets.Get(context.TODO(), name, region)
	if err != nil {
		return nil, err
	}

	return &Secret{Secret: secret}, nil
}

func (s *secretsService) ListVersions(name, region string) (SecretVersions, error) {
	versions, _, err := s.client.Secrets.ListVersions(context.TODO(), name, region)
	if err != nil {
		return nil, err
	}

	list := make(SecretVersions, len(versions))
	for i := range versions {
		list[i] = SecretVersion{SecretVersion: versions[i]}
	}

	return list, nil
}

func (s *secretsService) Create(req *godo.SecretCreateRequest) (*SecretWriteResult, error) {
	result, _, err := s.client.Secrets.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}

	return &SecretWriteResult{SecretWriteResult: result}, nil
}

func (s *secretsService) Update(name string, req *godo.SecretUpdateRequest) (*SecretWriteResult, error) {
	result, _, err := s.client.Secrets.Update(context.TODO(), name, req)
	if err != nil {
		return nil, err
	}

	return &SecretWriteResult{SecretWriteResult: result}, nil
}

func (s *secretsService) Delete(name, region string) error {
	_, err := s.client.Secrets.Delete(context.TODO(), name, region)
	return err
}

func (s *secretsService) Restore(name, region string) error {
	_, err := s.client.Secrets.Restore(context.TODO(), name, region)
	return err
}
