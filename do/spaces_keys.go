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
	"strings"

	"github.com/digitalocean/godo"
)

// SpacesKey wraps a godo SpacesKey.
type SpacesKey struct {
	*godo.SpacesKey
}

// GrantString returns a string representation of the grants.
func (s *SpacesKey) GrantString() string {
	var grants []string
	for _, grant := range s.Grants {
		grants = append(grants, fmt.Sprintf("bucket=%s;permission=%s", grant.Bucket, grant.Permission))
	}
	return strings.Join(grants, ",")
}

// SpacesKeysService is an interface for interfacing with the Spaces Keys
type SpacesKeysService interface {
	Create(*godo.SpacesKeyCreateRequest) (*SpacesKey, error)
	Delete(string) error
	Update(string, *godo.SpacesKeyUpdateRequest) (*SpacesKey, error)
	List() ([]SpacesKey, error)
	Get(string) (*SpacesKey, error)
}

type spacesKeysService struct {
	client *godo.Client
	ctx    context.Context
}

// NewSpacesKeysService returns a new instance of SpacesKeysService.
func NewSpacesKeysService(client *godo.Client) SpacesKeysService {
	return &spacesKeysService{client, context.Background()}
}

// Create creates a new Spaces key.
func (s *spacesKeysService) Create(cr *godo.SpacesKeyCreateRequest) (*SpacesKey, error) {
	key, _, err := s.client.SpacesKeys.Create(s.ctx, cr)
	if err != nil {
		return nil, err
	}

	return &SpacesKey{key}, nil
}

// Delete deletes a Spaces key.
func (s *spacesKeysService) Delete(accessKey string) error {
	_, err := s.client.SpacesKeys.Delete(s.ctx, accessKey)
	return err
}

// Get gets a Spaces key.
func (s *spacesKeysService) Get(accessKey string) (*SpacesKey, error) {
	key, _, err := s.client.SpacesKeys.Get(s.ctx, accessKey)
	if err != nil {
		return nil, err
	}
	return &SpacesKey{key}, nil
}

// List returns all Spaces keys.
func (s *spacesKeysService) List() ([]SpacesKey, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.SpacesKeys.List(s.ctx, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	keys, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]SpacesKey, len(keys))
	for i := range keys {
		k := keys[i].(*godo.SpacesKey)
		list[i] = SpacesKey{k}
	}

	return list, nil
}

// Update updates a Spaces key.
func (s *spacesKeysService) Update(accessKey string, ur *godo.SpacesKeyUpdateRequest) (*SpacesKey, error) {
	key, _, err := s.client.SpacesKeys.Update(s.ctx, accessKey, ur)
	if err != nil {
		return nil, err
	}

	return &SpacesKey{key}, nil
}
