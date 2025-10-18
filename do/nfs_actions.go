/*
Copyright 2025 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/godo"
)

// NfsActionsService is an interface for interacting with DigitalOcean's NFS Actions API.
type NfsActionsService interface {
	Resize(id string, size uint64, region string) (*godo.Action, error)
	Snapshot(id, name, region string) (*godo.Action, error)
}

type nfsActionsService struct {
	client *godo.Client
}

var _ NfsActionsService = &nfsActionsService{}

// NewNfsActionsService builds a NewNfsActionsService instance.
func NewNfsActionsService(godoClient *godo.Client) NfsActionsService {
	return &nfsActionsService{
		client: godoClient,
	}
}

func (s *nfsActionsService) Resize(id string, size uint64, region string) (*godo.Action, error) {
	action, _, err := s.client.NfsActions.Resize(context.TODO(), id, size, region)
	if err != nil {
		return nil, err
	}
	return action, nil
}

func (s *nfsActionsService) Snapshot(id, name, region string) (*godo.Action, error) {
	action, _, err := s.client.NfsActions.Snapshot(context.TODO(), id, name, region)
	if err != nil {
		return nil, err
	}
	return action, nil
}
