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

// AlertPolicy is a wrapper for godo.AlertPolicy
type AlertPolicy struct {
	*godo.AlertPolicy
}

// AlertPolicies is a slice of AlertPolicy.
type AlertPolicies []AlertPolicy

// MonitoringService is an interface for interacting with DigitalOcean's monitoring api.
type MonitoringService interface {
	ListAlertPolicies() (AlertPolicies, error)
	GetAlertPolicy(string) (*AlertPolicy, error)
	CreateAlertPolicy(request *godo.AlertPolicyCreateRequest) (*AlertPolicy, error)
	UpdateAlertPolicy(uuid string, request *godo.AlertPolicyUpdateRequest) (*AlertPolicy, error)
	DeleteAlertPolicy(string) error
}

type monitoringService struct {
	client *godo.Client
}

var _ MonitoringService = (*monitoringService)(nil)

// NewMonitoringService builds a MonitoringService instance.
func NewMonitoringService(godoClient *godo.Client) MonitoringService {
	return &monitoringService{
		client: godoClient,
	}
}

func (ms *monitoringService) ListAlertPolicies() (AlertPolicies, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ms.client.Monitoring.ListAlertPolicies(context.TODO(), opt)
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

	list := make(AlertPolicies, len(si))
	for i := range si {
		a := si[i].(godo.AlertPolicy)
		list[i] = AlertPolicy{AlertPolicy: &a}
	}

	return list, nil
}

func (ms *monitoringService) GetAlertPolicy(uuid string) (*AlertPolicy, error) {
	p, _, err := ms.client.Monitoring.GetAlertPolicy(context.TODO(), uuid)
	if err != nil {
		return nil, err
	}

	return &AlertPolicy{AlertPolicy: p}, nil
}

func (ms *monitoringService) CreateAlertPolicy(apcr *godo.AlertPolicyCreateRequest) (*AlertPolicy, error) {
	p, _, err := ms.client.Monitoring.CreateAlertPolicy(context.TODO(), apcr)
	if err != nil {
		return nil, err
	}

	return &AlertPolicy{AlertPolicy: p}, nil
}

func (ms *monitoringService) UpdateAlertPolicy(uuid string, apur *godo.AlertPolicyUpdateRequest) (*AlertPolicy, error) {
	p, _, err := ms.client.Monitoring.UpdateAlertPolicy(context.TODO(), uuid, apur)
	if err != nil {
		return nil, err
	}

	return &AlertPolicy{AlertPolicy: p}, nil
}

func (ms *monitoringService) DeleteAlertPolicy(uuid string) error {
	_, err := ms.client.Monitoring.DeleteAlertPolicy(context.TODO(), uuid)
	return err
}
