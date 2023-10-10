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

// UptimeAlert is a wrapper for godo.UptimeAlert.
type UptimeAlert struct {
	*godo.UptimeAlert
}

// Notifications is a wrapper for godo.Notifications.
type Notifications struct {
	*godo.Notifications
}

// UptimeAlertsService is an interface for interacting with DigitalOcean's uptime alerts API.
type UptimeAlertsService interface {
	Create(string, *godo.CreateUptimeAlertRequest) (*UptimeAlert, error)
	List(string) ([]UptimeAlert, error)
	Get(string, string) (*UptimeAlert, error)
	GetState(string) (*UptimeCheckState, error)
	Update(string, string, *godo.UpdateUptimeAlertRequest) (*UptimeAlert, error)
	Delete(string, string) error
}

type uptimeAlertsService struct {
	client *godo.Client
}

var _ UptimeAlertsService = &uptimeAlertsService{}

// NewUptimeAlertsService builds an UptimeAlertsService instance.
func NewUptimeAlertsService(godoClient *godo.Client) UptimeAlertsService {
	return &uptimeAlertsService{
		client: godoClient,
	}
}

func (uas *uptimeAlertsService) Create(checkID string, req *godo.CreateUptimeAlertRequest) (*UptimeAlert, error) {
	uptimeAlert, _, err := uas.client.UptimeChecks.CreateAlert(context.TODO(), checkID, req)
	if err != nil {
		return nil, err
	}
	return &UptimeAlert{uptimeAlert}, nil
}

func (uas *uptimeAlertsService) List(checkID string) ([]UptimeAlert, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := uas.client.UptimeChecks.ListAlerts(context.TODO(), checkID, opt)
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

	list := make([]UptimeAlert, len(si))
	for i := range si {
		ua := si[i].(godo.UptimeAlert)
		list[i] = UptimeAlert{UptimeAlert: &ua}
	}
	return list, nil
}

func (uas *uptimeAlertsService) Get(checkID, alertID string) (*UptimeAlert, error) {
	uptimeAlert, _, err := uas.client.UptimeChecks.GetAlert(context.TODO(), checkID, alertID)
	if err != nil {
		return nil, err
	}
	return &UptimeAlert{uptimeAlert}, nil
}

func (uas *uptimeAlertsService) GetState(id string) (*UptimeCheckState, error) {
	uptimeCheckState, _, err := uas.client.UptimeChecks.GetState(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &UptimeCheckState{uptimeCheckState}, nil
}

func (uas *uptimeAlertsService) Update(checkID, alertID string, req *godo.UpdateUptimeAlertRequest) (*UptimeAlert, error) {
	uptimeAlert, _, err := uas.client.UptimeChecks.UpdateAlert(context.TODO(), checkID, alertID, req)
	if err != nil {
		return nil, err
	}
	return &UptimeAlert{uptimeAlert}, nil
}

func (uas *uptimeAlertsService) Delete(checkID, alertID string) error {
	_, err := uas.client.UptimeChecks.DeleteAlert(context.TODO(), checkID, alertID)
	return err
}
