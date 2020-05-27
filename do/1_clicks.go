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
	"net/http"
)

// // VPC wraps a godo VPC.
// type VPC struct {
// 	*godo.VPC
// }

// // VPCs is a slice of VPC.
// type VPCs []VPC

// OneClickService is the godo VPCsService interface.
type OneClickService interface {
	List(string) ([]string, error)
}

type oneClickService struct {
	Client *http.Client
}

// NewOneClickService builds an instance of OneClickService.
func NewOneClickService(client *http.Client) OneClickService {
	ocs := &oneClickService{
		Client: client,
	}

	return ocs
}

func (ocs *oneClickService) List(oneClickType string) ([]string, error) {
	list := []string{}
	return list, nil
}
