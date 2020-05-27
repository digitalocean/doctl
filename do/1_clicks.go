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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// OneClickService is the godo VPCsService interface.
type OneClickService interface {
	List(string) (OneClicks, error)
}

type oneClickService struct {
	Client *http.Client
}

// OneClick represents the structre of a 1-click
type OneClick struct {
	Slug string
	Type string
}

// OneClicks is a set of OneClick structs
type OneClicks []OneClick

// OneClickResp is the struct representing the json payload for a 1-click
type OneClickResp struct {
	Slug string `json:"slug"`
	Type string `json:"type"`
}

// OneClicksResp is a struct representing the json payload for a list of 1-clicks
type OneClicksResp struct {
	List []OneClickResp `json:"1_click"`
}

// NewOneClickService builds an instance of OneClickService.
func NewOneClickService(client *http.Client) OneClickService {
	ocs := &oneClickService{
		Client: client,
	}

	return ocs
}

func (ocs *oneClickService) List(oneClickType string) (OneClicks, error) {
	url := fmt.Sprintf("%s/v2/1-click?type=%s", "https://api.digitalocean.com", oneClickType)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating request for 1-click")
	}

	req = req.WithContext(context.Background())
	req.Header.Set("Content-Type", "application/json")

	resp, err := ocs.Client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "request 1-click list")
	}

	body := bytes.NewBuffer(nil)
	_, err = io.Copy(body, resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading 1-click list from response")
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, errors.Wrap(err, "closing 1-click response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("received non 200 response")
	}

	oneClicksResp := &OneClicksResp{}
	err = json.Unmarshal(body.Bytes(), oneClicksResp)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal body")
	}

	list := OneClicks{}
	for _, ocr := range oneClicksResp.List {
		oc := OneClick{
			Slug: ocr.Slug,
			Type: ocr.Type,
		}

		list = append(list, oc)
	}

	return list, nil
}
