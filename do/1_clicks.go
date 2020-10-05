/*
Copyright 2020 The Doctl Authors All rights reserved.
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

// OneClickService is the godo OneClickService interface.
type OneClickService interface {
	List(string) (OneClicks, error)
	InstallKubernetes(string, []string) (string, error)
}

var _ OneClickService = &oneClickService{}

type oneClickService struct {
	Client *godo.Client
}

// OneClick represents the structure of a 1-click
type OneClick struct {
	*godo.OneClick
}

// OneClicks is a set of OneClick structs
type OneClicks []OneClick

// NewOneClickService builds an instance of OneClickService.
func NewOneClickService(client *godo.Client) OneClickService {
	ocs := &oneClickService{
		Client: client,
	}

	return ocs
}

func (ocs *oneClickService) List(oneClickType string) (OneClicks, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ocs.Client.OneClick.List(context.TODO(), oneClickType)
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

	list := make([]OneClick, len(si))
	for i := range si {
		a := si[i].(*godo.OneClick)
		list[i] = OneClick{OneClick: a}
	}

	return list, nil
}

func (ocs *oneClickService) InstallKubernetes(clusterUUID string, slugs []string) (string, error) {
	installReq := &godo.InstallKubernetesAppsRequest{
		Slugs:       slugs,
		ClusterUUID: clusterUUID,
	}

	responseMessage, _, err := ocs.Client.OneClick.InstallKubernetes(context.TODO(), installReq)
	if err != nil {
		return "", err
	}

	return responseMessage.Message, nil
}
