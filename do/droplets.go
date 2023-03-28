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
	"github.com/digitalocean/godo/util"
)

// DropletIPTable is a table of interface IPS.
type DropletIPTable map[InterfaceType]string

// InterfaceType is an interface type.
type InterfaceType string

const (
	// InterfacePublic is a public interface.
	InterfacePublic InterfaceType = "public"
	// InterfacePrivate is a private interface.
	InterfacePrivate InterfaceType = "private"
)

// Droplet is a wrapper for godo.Droplet
type Droplet struct {
	*godo.Droplet
}

// Droplets is a slice of Droplet.
type Droplets []Droplet

// Kernel is a wrapper for godo.Kernel
type Kernel struct {
	*godo.Kernel
}

// Kernels is a slice of Kernel.
type Kernels []Kernel

// DropletsService is an interface for interacting with DigitalOcean's droplet api.
type DropletsService interface {
	List() (Droplets, error)
	ListByTag(string) (Droplets, error)
	Get(int) (*Droplet, error)
	Create(*godo.DropletCreateRequest, bool) (*Droplet, error)
	CreateMultiple(*godo.DropletMultiCreateRequest) (Droplets, error)
	Delete(int) error
	DeleteByTag(string) error
	Kernels(int) (Kernels, error)
	Snapshots(int) (Images, error)
	Backups(int) (Images, error)
	Actions(int) (Actions, error)
	Neighbors(int) (Droplets, error)
}

type dropletsService struct {
	client *godo.Client
}

var _ DropletsService = &dropletsService{}

// NewDropletsService builds a DropletsService instance.
func NewDropletsService(client *godo.Client) DropletsService {
	return &dropletsService{
		client: client,
	}
}

func (ds *dropletsService) List() (Droplets, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Droplets.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(Droplets, len(si))
	for i := range si {
		a := si[i].(godo.Droplet)
		list[i] = Droplet{Droplet: &a}
	}

	return list, nil
}

func (ds *dropletsService) ListByTag(tagName string) (Droplets, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Droplets.ListByTag(context.TODO(), tagName, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(Droplets, len(si))
	for i := range si {
		a := si[i].(godo.Droplet)
		list[i] = Droplet{Droplet: &a}
	}

	return list, nil
}

func (ds *dropletsService) Get(id int) (*Droplet, error) {
	d, _, err := ds.client.Droplets.Get(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	return &Droplet{Droplet: d}, nil
}

func (ds *dropletsService) Create(dcr *godo.DropletCreateRequest, wait bool) (*Droplet, error) {
	d, resp, err := ds.client.Droplets.Create(context.TODO(), dcr)
	if err != nil {
		return nil, err
	}

	if wait {
		var action *godo.LinkAction
		for _, a := range resp.Links.Actions {
			if a.Rel == "create" {
				action = &a
				break
			}
		}

		if action != nil {
			_ = util.WaitForActive(context.TODO(), ds.client, action.HREF)
			doDroplet, err := ds.Get(d.ID)
			if err != nil {
				return nil, err
			}
			d = doDroplet.Droplet
		}
	}

	return &Droplet{Droplet: d}, nil
}

func (ds *dropletsService) CreateMultiple(dmcr *godo.DropletMultiCreateRequest) (Droplets, error) {
	godoDroplets, _, err := ds.client.Droplets.CreateMultiple(context.TODO(), dmcr)
	if err != nil {
		return nil, err
	}

	droplets := make(Droplets, 0, len(godoDroplets))
	for _, d := range godoDroplets {
		droplets = append(droplets, Droplet{Droplet: &d})
	}

	return droplets, nil
}

func (ds *dropletsService) Delete(id int) error {
	_, err := ds.client.Droplets.Delete(context.TODO(), id)
	return err
}

func (ds *dropletsService) DeleteByTag(tag string) error {
	_, err := ds.client.Droplets.DeleteByTag(context.TODO(), tag)
	return err
}

func (ds *dropletsService) Kernels(id int) (Kernels, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Droplets.Kernels(context.TODO(), id, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(Kernels, len(si))
	for i := range si {
		a := si[i].(godo.Kernel)
		list[i] = Kernel{Kernel: &a}
	}

	return list, nil
}

func (ds *dropletsService) Snapshots(id int) (Images, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Droplets.Snapshots(context.TODO(), id, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(Images, len(si))
	for i := range si {
		a := si[i].(godo.Image)
		list[i] = Image{Image: &a}
	}

	return list, nil
}

func (ds *dropletsService) Backups(id int) (Images, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Droplets.Backups(context.TODO(), id, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(Images, len(si))
	for i := range si {
		a := si[i].(godo.Image)
		list[i] = Image{Image: &a}
	}

	return list, nil
}

func (ds *dropletsService) Actions(id int) (Actions, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Droplets.Actions(context.TODO(), id, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(Actions, len(si))
	for i := range si {
		a := si[i].(godo.Action)
		list[i] = Action{Action: &a}
	}

	return list, nil
}

func (ds *dropletsService) Neighbors(id int) (Droplets, error) {
	list, _, err := ds.client.Droplets.Neighbors(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	droplets := make(Droplets, len(list))
	for i := range list {
		droplets[i] = Droplet{&list[i]}
	}

	return droplets, nil
}
