/*
Copyright 2024 The Doctl Authors All rights reserved.
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

// ReservedIPv6ActionsService is an interface for interacting with
// DigitalOcean's reserved IPv6 action api.
type ReservedIPv6ActionsService interface {
	Assign(ip string, dropletID int) (*Action, error)
	Unassign(ip string) (*Action, error)
}

type reservedIPv6ActionsService struct {
	client *godo.Client
}

var _ ReservedIPv6ActionsService = &reservedIPv6ActionsService{}

// NewReservedIPv6ActionsService builds a ReservedIPv6ActionsService instance.
func NewReservedIPv6ActionsService(godoClient *godo.Client) ReservedIPv6ActionsService {
	return &reservedIPv6ActionsService{
		client: godoClient,
	}
}

func (fia *reservedIPv6ActionsService) Assign(ip string, dropletID int) (*Action, error) {
	a, _, err := fia.client.ReservedIPV6Actions.Assign(context.TODO(), ip, dropletID)
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}

func (fia *reservedIPv6ActionsService) Unassign(ip string) (*Action, error) {
	a, _, err := fia.client.ReservedIPV6Actions.Unassign(context.TODO(), ip)
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}
