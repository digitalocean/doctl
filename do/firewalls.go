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

	"github.com/digitalocean/godo"
)

// Firewall wraps a godo Firewall.
type Firewall struct {
	*godo.Firewall
}

// Firewalls is a slice of Firewall.
type Firewalls []Firewall

// FirewallsService is the godo FirewallsService interface.
type FirewallsService interface {
	Get(fID string) (*Firewall, error)
	Create(fr *godo.FirewallRequest) (*Firewall, error)
	Update(fID string, fr *godo.FirewallRequest) (*Firewall, error)
	List() (Firewalls, error)
	ListByDroplet(dID int) (Firewalls, error)
	Delete(fID string) error
	AddDroplets(fID string, dIDs ...int) error
	RemoveDroplets(fID string, dIDs ...int) error
	AddTags(fID string, tags ...string) error
	RemoveTags(fID string, tags ...string) error
	AddRules(fID string, rr *godo.FirewallRulesRequest) error
	RemoveRules(fID string, rr *godo.FirewallRulesRequest) error
}

var _ FirewallsService = &firewallsService{}

type firewallsService struct {
	client *godo.Client
}

// NewFirewallsService builds an instance of FirewallsService.
func NewFirewallsService(client *godo.Client) FirewallsService {
	return &firewallsService{client: client}
}

func (fs *firewallsService) Get(fID string) (*Firewall, error) {
	f, _, err := fs.client.Firewalls.Get(context.TODO(), fID)
	if err != nil {
		return nil, err
	}

	return &Firewall{Firewall: f}, nil
}

func (fs *firewallsService) Create(fr *godo.FirewallRequest) (*Firewall, error) {
	f, _, err := fs.client.Firewalls.Create(context.TODO(), fr)
	if err != nil {
		return nil, err
	}

	return &Firewall{Firewall: f}, nil
}

func (fs *firewallsService) Update(fID string, fr *godo.FirewallRequest) (*Firewall, error) {
	f, _, err := fs.client.Firewalls.Update(context.TODO(), fID, fr)
	if err != nil {
		return nil, err
	}

	return &Firewall{Firewall: f}, nil
}

func (fs *firewallsService) List() (Firewalls, error) {
	listFn := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := fs.client.Firewalls.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	return paginatedListHelper(listFn)
}

func (fs *firewallsService) ListByDroplet(dID int) (Firewalls, error) {
	listFn := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := fs.client.Firewalls.ListByDroplet(context.TODO(), dID, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	return paginatedListHelper(listFn)
}

func (fs *firewallsService) Delete(fID string) error {
	_, err := fs.client.Firewalls.Delete(context.TODO(), fID)
	return err
}

func (fs *firewallsService) AddDroplets(fID string, dIDs ...int) error {
	_, err := fs.client.Firewalls.AddDroplets(context.TODO(), fID, dIDs...)
	return err
}

func (fs *firewallsService) RemoveDroplets(fID string, dIDs ...int) error {
	_, err := fs.client.Firewalls.RemoveDroplets(context.TODO(), fID, dIDs...)
	return err
}

func (fs *firewallsService) AddTags(fID string, tags ...string) error {
	_, err := fs.client.Firewalls.AddTags(context.TODO(), fID, tags...)
	return err
}

func (fs *firewallsService) RemoveTags(fID string, tags ...string) error {
	_, err := fs.client.Firewalls.RemoveTags(context.TODO(), fID, tags...)
	return err
}

func (fs *firewallsService) AddRules(fID string, rr *godo.FirewallRulesRequest) error {
	_, err := fs.client.Firewalls.AddRules(context.TODO(), fID, rr)
	return err
}

func (fs *firewallsService) RemoveRules(fID string, rr *godo.FirewallRulesRequest) error {
	_, err := fs.client.Firewalls.RemoveRules(context.TODO(), fID, rr)
	return err
}

func paginatedListHelper(listFn func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error)) (Firewalls, error) {
	si, err := PaginateResp(listFn)
	if err != nil {
		return nil, err
	}

	list := make([]Firewall, len(si))
	for i := range si {
		a, ok := si[i].(godo.Firewall)
		if !ok {
			return nil, errors.New("unexpected value in response")
		}

		list[i] = Firewall{Firewall: &a}
	}

	return list, nil
}
