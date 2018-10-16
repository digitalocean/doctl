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

// DropletActionsService is an interface for interacting with DigitalOcean's droplet action api.
type DropletActionsService interface {
	Shutdown(int) (*Action, error)
	ShutdownByTag(string) (Actions, error)
	PowerOff(int) (*Action, error)
	PowerOffByTag(string) (Actions, error)
	PowerOn(int) (*Action, error)
	PowerOnByTag(string) (Actions, error)
	PowerCycle(int) (*Action, error)
	PowerCycleByTag(string) (Actions, error)
	Reboot(int) (*Action, error)
	Restore(int, int) (*Action, error)
	Resize(int, string, bool) (*Action, error)
	Rename(int, string) (*Action, error)
	Snapshot(int, string) (*Action, error)
	SnapshotByTag(string, string) (Actions, error)
	EnableBackups(int) (*Action, error)
	EnableBackupsByTag(string) (Actions, error)
	DisableBackups(int) (*Action, error)
	DisableBackupsByTag(string) (Actions, error)
	PasswordReset(int) (*Action, error)
	RebuildByImageID(int, int) (*Action, error)
	RebuildByImageSlug(int, string) (*Action, error)
	ChangeKernel(int, int) (*Action, error)
	EnableIPv6(int) (*Action, error)
	EnableIPv6ByTag(string) (Actions, error)
	EnablePrivateNetworking(int) (*Action, error)
	EnablePrivateNetworkingByTag(string) (Actions, error)
	Get(int, int) (*Action, error)
	GetByURI(string) (*Action, error)
}

type dropletActionsService struct {
	client *godo.Client
}

var _ DropletActionsService = &dropletActionsService{}

// NewDropletActionsService builds an instance of DropletActionsService.
func NewDropletActionsService(godoClient *godo.Client) DropletActionsService {
	return &dropletActionsService{
		client: godoClient,
	}
}

func (das *dropletActionsService) handleActionResponse(a *godo.Action, err error) (*Action, error) {
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}

func (das *dropletActionsService) handleTagActionResponse(a []godo.Action, err error) (Actions, error) {
	if err != nil {
		return nil, err
	}

	actions := make([]Action, 0, len(a))

	for _, action := range a {
		actions = append(actions, Action{Action: &action})
	}

	return actions, nil
}

func (das *dropletActionsService) Shutdown(id int) (*Action, error) {
	a, _, err := das.client.DropletActions.Shutdown(context.TODO(), id)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) ShutdownByTag(tag string) (Actions, error) {
	a, _, err := das.client.DropletActions.ShutdownByTag(context.TODO(), tag)
	return das.handleTagActionResponse(a, err)
}

func (das *dropletActionsService) PowerOff(id int) (*Action, error) {
	a, _, err := das.client.DropletActions.PowerOff(context.TODO(), id)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) PowerOffByTag(tag string) (Actions, error) {
	a, _, err := das.client.DropletActions.PowerOffByTag(context.TODO(), tag)
	return das.handleTagActionResponse(a, err)
}

func (das *dropletActionsService) PowerOn(id int) (*Action, error) {
	a, _, err := das.client.DropletActions.PowerOn(context.TODO(), id)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) PowerOnByTag(tag string) (Actions, error) {
	a, _, err := das.client.DropletActions.PowerOnByTag(context.TODO(), tag)
	return das.handleTagActionResponse(a, err)
}

func (das *dropletActionsService) PowerCycle(id int) (*Action, error) {
	a, _, err := das.client.DropletActions.PowerCycle(context.TODO(), id)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) PowerCycleByTag(tag string) (Actions, error) {
	a, _, err := das.client.DropletActions.PowerCycleByTag(context.TODO(), tag)
	return das.handleTagActionResponse(a, err)
}

func (das *dropletActionsService) Reboot(id int) (*Action, error) {
	a, _, err := das.client.DropletActions.Reboot(context.TODO(), id)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) Restore(id, imageID int) (*Action, error) {
	a, _, err := das.client.DropletActions.Restore(context.TODO(), id, imageID)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) Resize(id int, sizeSlug string, resizeDisk bool) (*Action, error) {
	a, _, err := das.client.DropletActions.Resize(context.TODO(), id, sizeSlug, resizeDisk)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) Rename(id int, name string) (*Action, error) {
	a, _, err := das.client.DropletActions.Rename(context.TODO(), id, name)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) Snapshot(id int, name string) (*Action, error) {
	a, _, err := das.client.DropletActions.Snapshot(context.TODO(), id, name)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) SnapshotByTag(tag string, name string) (Actions, error) {
	a, _, err := das.client.DropletActions.SnapshotByTag(context.TODO(), tag, name)
	return das.handleTagActionResponse(a, err)
}

func (das *dropletActionsService) EnableBackups(id int) (*Action, error) {
	a, _, err := das.client.DropletActions.EnableBackups(context.TODO(), id)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) EnableBackupsByTag(tag string) (Actions, error) {
	a, _, err := das.client.DropletActions.EnableBackupsByTag(context.TODO(), tag)
	return das.handleTagActionResponse(a, err)
}

func (das *dropletActionsService) DisableBackups(id int) (*Action, error) {
	a, _, err := das.client.DropletActions.DisableBackups(context.TODO(), id)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) DisableBackupsByTag(tag string) (Actions, error) {
	a, _, err := das.client.DropletActions.DisableBackupsByTag(context.TODO(), tag)
	return das.handleTagActionResponse(a, err)
}

func (das *dropletActionsService) PasswordReset(id int) (*Action, error) {
	a, _, err := das.client.DropletActions.PasswordReset(context.TODO(), id)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) RebuildByImageID(id, imageID int) (*Action, error) {
	a, _, err := das.client.DropletActions.RebuildByImageID(context.TODO(), id, imageID)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) RebuildByImageSlug(id int, slug string) (*Action, error) {
	a, _, err := das.client.DropletActions.RebuildByImageSlug(context.TODO(), id, slug)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) ChangeKernel(id, kernelID int) (*Action, error) {
	a, _, err := das.client.DropletActions.ChangeKernel(context.TODO(), id, kernelID)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) EnableIPv6(id int) (*Action, error) {
	a, _, err := das.client.DropletActions.EnableIPv6(context.TODO(), id)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) EnableIPv6ByTag(tag string) (Actions, error) {
	a, _, err := das.client.DropletActions.EnableIPv6ByTag(context.TODO(), tag)
	return das.handleTagActionResponse(a, err)
}

func (das *dropletActionsService) EnablePrivateNetworking(id int) (*Action, error) {
	a, _, err := das.client.DropletActions.EnablePrivateNetworking(context.TODO(), id)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) EnablePrivateNetworkingByTag(tag string) (Actions, error) {
	a, _, err := das.client.DropletActions.EnablePrivateNetworkingByTag(context.TODO(), tag)
	return das.handleTagActionResponse(a, err)
}

func (das *dropletActionsService) Get(id int, actionID int) (*Action, error) {
	a, _, err := das.client.DropletActions.Get(context.TODO(), id, actionID)
	return das.handleActionResponse(a, err)
}

func (das *dropletActionsService) GetByURI(uri string) (*Action, error) {
	a, _, err := das.client.DropletActions.GetByURI(context.TODO(), uri)
	return das.handleActionResponse(a, err)
}
