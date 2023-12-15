/*
Copyright 2023 The Doctl Authors All rights reserved.
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

// UptimeCheck is a wrapper for godo.UptimeCheck.
type UptimeCheck struct {
	*godo.UptimeCheck
}

// UptimeAlert is a wrapper for godo.UptimeAlert.
type UptimeAlert struct {
	*godo.UptimeAlert
}

// UptimeCheckState is a wrapper for godo.UptimeCheckState.
type UptimeCheckState struct {
	*godo.UptimeCheckState
}

// UptimeChecksService is an interface for interacting with DigitalOcean's uptime check api.
type UptimeChecksService interface {
	Create(*godo.CreateUptimeCheckRequest) (*UptimeCheck, error)
	List() ([]UptimeCheck, error)
	Get(string) (*UptimeCheck, error)
	GetState(string) (*UptimeCheckState, error)
	Update(string, *godo.UpdateUptimeCheckRequest) (*UptimeCheck, error)
	Delete(string) error
	CreateAlert(string, *godo.CreateUptimeAlertRequest) (*UptimeAlert, error)
	ListAlerts(string) ([]UptimeAlert, error)
	GetAlert(string, string) (*UptimeAlert, error)
	UpdateAlert(string, string, *godo.UpdateUptimeAlertRequest) (*UptimeAlert, error)
	DeleteAlert(string, string) error
}

type uptimeChecksService struct {
	client *godo.Client
}

var _ UptimeChecksService = &uptimeChecksService{}

// NewUptimeChecksService builds an NewUptimeChecksService instance.
func NewUptimeChecksService(godoClient *godo.Client) UptimeChecksService {
	return &uptimeChecksService{
		client: godoClient,
	}
}

func (ucs *uptimeChecksService) Create(req *godo.CreateUptimeCheckRequest) (*UptimeCheck, error) {
	uptimeCheck, _, err := ucs.client.UptimeChecks.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &UptimeCheck{uptimeCheck}, nil
}

func (ucs *uptimeChecksService) List() ([]UptimeCheck, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := ucs.client.UptimeChecks.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]UptimeCheck, len(si))
	for i := range si {
		uc := si[i].(godo.UptimeCheck)
		list[i] = UptimeCheck{UptimeCheck: &uc}
	}
	return list, nil
}

func (ucs *uptimeChecksService) Get(id string) (*UptimeCheck, error) {
	uptimeCheck, _, err := ucs.client.UptimeChecks.Get(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &UptimeCheck{uptimeCheck}, nil
}

func (ucs *uptimeChecksService) GetState(id string) (*UptimeCheckState, error) {
	uptimeCheckState, _, err := ucs.client.UptimeChecks.GetState(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &UptimeCheckState{uptimeCheckState}, nil
}

func (ucs *uptimeChecksService) Update(id string, req *godo.UpdateUptimeCheckRequest) (*UptimeCheck, error) {
	uptimeCheck, _, err := ucs.client.UptimeChecks.Update(context.TODO(), id, req)
	if err != nil {
		return nil, err
	}
	return &UptimeCheck{uptimeCheck}, nil
}

func (ucs *uptimeChecksService) Delete(id string) error {
	_, err := ucs.client.UptimeChecks.Delete(context.TODO(), id)
	return err
}

func (ucs *uptimeChecksService) CreateAlert(id string, req *godo.CreateUptimeAlertRequest) (*UptimeAlert, error) {
	uptimeAlert, _, err := ucs.client.UptimeChecks.CreateAlert(context.TODO(), id, req)
	if err != nil {
		return nil, err
	}

	return &UptimeAlert{uptimeAlert}, nil
}

func (ucs *uptimeChecksService) ListAlerts(id string) ([]UptimeAlert, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := ucs.client.UptimeChecks.ListAlerts(context.TODO(), id, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]UptimeAlert, len(si))
	for i := range si {
		ua := si[i].(godo.UptimeAlert)
		list[i] = UptimeAlert{UptimeAlert: &ua}
	}
	return list, nil
}

func (ucs *uptimeChecksService) GetAlert(checkID string, alertID string) (*UptimeAlert, error) {
	uptimeAlert, _, err := ucs.client.UptimeChecks.GetAlert(context.TODO(), checkID, alertID)
	if err != nil {
		return nil, err
	}
	return &UptimeAlert{uptimeAlert}, nil
}

func (ucs *uptimeChecksService) UpdateAlert(checkID string, alertID string, req *godo.UpdateUptimeAlertRequest) (*UptimeAlert, error) {
	uptimeAlert, _, err := ucs.client.UptimeChecks.UpdateAlert(context.TODO(), checkID, alertID, req)
	if err != nil {
		return nil, err
	}
	return &UptimeAlert{uptimeAlert}, nil
}

func (ucs *uptimeChecksService) DeleteAlert(checkID string, alertID string) error {
	_, err := ucs.client.UptimeChecks.DeleteAlert(context.TODO(), checkID, alertID)
	return err
}
