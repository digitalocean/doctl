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

	"github.com/digitalocean/godo"
)

type DropletAutoscaleService interface {
	Create(*godo.DropletAutoscalePoolRequest) (*godo.DropletAutoscalePool, error)
	Get(string) (*godo.DropletAutoscalePool, error)
	List(*godo.ListOptions) ([]*godo.DropletAutoscalePool, error)
	ListMembers(string, *godo.ListOptions) ([]*godo.DropletAutoscaleResource, error)
	ListHistory(string, *godo.ListOptions) ([]*godo.DropletAutoscaleHistoryEvent, error)
	Update(string, *godo.DropletAutoscalePoolRequest) (*godo.DropletAutoscalePool, error)
	Delete(string) error
	DeleteDangerous(string) error
}

var _ DropletAutoscaleService = &dropletAutoscaleService{}

type dropletAutoscaleService struct {
	client *godo.Client
}

// NewDropletAutoscaleService creates an instance of DropletAutoscaleService
func NewDropletAutoscaleService(client *godo.Client) DropletAutoscaleService {
	return &dropletAutoscaleService{
		client: client,
	}
}

// Create creates a new droplet autoscale pool
func (d *dropletAutoscaleService) Create(createReq *godo.DropletAutoscalePoolRequest) (*godo.DropletAutoscalePool, error) {
	pool, _, err := d.client.DropletAutoscale.Create(context.Background(), createReq)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// Get retrieves an existing droplet autoscale pool
func (d *dropletAutoscaleService) Get(poolID string) (*godo.DropletAutoscalePool, error) {
	pool, _, err := d.client.DropletAutoscale.Get(context.Background(), poolID)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// List lists all existing droplet autoscale pools
func (d *dropletAutoscaleService) List(opts *godo.ListOptions) ([]*godo.DropletAutoscalePool, error) {
	listResp, err := PaginateResp(func(options *godo.ListOptions) ([]any, *godo.Response, error) {
		pools, resp, err := d.client.DropletAutoscale.List(context.Background(), opts)
		if err != nil {
			return nil, nil, err
		}
		anyResp := make([]any, 0, len(pools))
		for _, pool := range pools {
			anyResp = append(anyResp, pool)
		}
		return anyResp, resp, err
	})
	if err != nil {
		return nil, err
	}
	pools := make([]*godo.DropletAutoscalePool, 0, len(listResp))
	for _, pool := range listResp {
		pools = append(pools, pool.(*godo.DropletAutoscalePool))
	}
	return pools, nil
}

// ListMembers lists all droplet autoscale resources for an existing pool
func (d *dropletAutoscaleService) ListMembers(poolID string, opts *godo.ListOptions) ([]*godo.DropletAutoscaleResource, error) {
	listResp, err := PaginateResp(func(options *godo.ListOptions) ([]any, *godo.Response, error) {
		members, resp, err := d.client.DropletAutoscale.ListMembers(context.Background(), poolID, opts)
		if err != nil {
			return nil, nil, err
		}
		anyResp := make([]any, 0, len(members))
		for _, member := range members {
			anyResp = append(anyResp, member)
		}
		return anyResp, resp, err
	})
	if err != nil {
		return nil, err
	}
	members := make([]*godo.DropletAutoscaleResource, 0, len(listResp))
	for _, member := range listResp {
		members = append(members, member.(*godo.DropletAutoscaleResource))
	}
	return members, nil
}

// ListHistory lists all droplet autoscale history events for an existing pool
func (d *dropletAutoscaleService) ListHistory(poolID string, opts *godo.ListOptions) ([]*godo.DropletAutoscaleHistoryEvent, error) {
	listResp, err := PaginateResp(func(options *godo.ListOptions) ([]any, *godo.Response, error) {
		events, resp, err := d.client.DropletAutoscale.ListHistory(context.Background(), poolID, opts)
		if err != nil {
			return nil, nil, err
		}
		anyResp := make([]any, 0, len(events))
		for _, event := range events {
			anyResp = append(anyResp, event)
		}
		return anyResp, resp, err
	})
	if err != nil {
		return nil, err
	}
	events := make([]*godo.DropletAutoscaleHistoryEvent, 0, len(listResp))
	for _, event := range listResp {
		events = append(events, event.(*godo.DropletAutoscaleHistoryEvent))
	}
	return events, nil
}

// Update updates an existing droplet autoscale pool
func (d *dropletAutoscaleService) Update(poolID string, updateReq *godo.DropletAutoscalePoolRequest) (*godo.DropletAutoscalePool, error) {
	pool, _, err := d.client.DropletAutoscale.Update(context.Background(), poolID, updateReq)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// Delete deletes an existing droplet autoscale pool
func (d *dropletAutoscaleService) Delete(poolID string) error {
	_, err := d.client.DropletAutoscale.Delete(context.Background(), poolID)
	return err
}

// DeleteDangerous deletes an existing droplet autoscale pool and all its underlying resources
func (d *dropletAutoscaleService) DeleteDangerous(poolID string) error {
	_, err := d.client.DropletAutoscale.DeleteDangerous(context.Background(), poolID)
	return err
}
