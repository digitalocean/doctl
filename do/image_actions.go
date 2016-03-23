
/*
Copyright 2016 The Doctl Authors All rights reserved.
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

import "github.com/digitalocean/godo"

// ImageActionsService is an interface for interacting with DigitalOcean's image action api.
type ImageActionsService interface {
	Get(int, int) (*Action, error)
	Transfer(int, *godo.ActionRequest) (*Action, error)
}

type imageActionsService struct {
	client *godo.Client
}

var _ ImageActionsService = &imageActionsService{}

// NewImageActionsService builds an ImageActionsService instance.
func NewImageActionsService(client *godo.Client) ImageActionsService {
	return &imageActionsService{
		client: client,
	}
}

func (ia *imageActionsService) Get(imageID, actionID int) (*Action, error) {
	a, _, err := ia.client.ImageActions.Get(imageID, actionID)
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}

func (ia *imageActionsService) Transfer(imageID int, transferRequest *godo.ActionRequest) (*Action, error) {
	a, _, err := ia.client.ImageActions.Transfer(imageID, transferRequest)
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}
