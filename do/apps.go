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

// AppsService is the interface that wraps godo AppsService.
type AppsService interface {
	Create(req *godo.AppCreateRequest) (*godo.App, error)
	Get(appID string) (*godo.App, error)
	List() ([]*godo.App, error)
	Update(appID string, req *godo.AppUpdateRequest) (*godo.App, error)
	Delete(appID string) error
	Propose(req *godo.AppProposeRequest) (*godo.AppProposeResponse, error)

	CreateDeployment(appID string, forceRebuild bool) (*godo.Deployment, error)
	GetDeployment(appID, deploymentID string) (*godo.Deployment, error)
	ListDeployments(appID string) ([]*godo.Deployment, error)

	GetLogs(appID, deploymentID, component string, logType godo.AppLogType, follow bool, tail int) (*godo.AppLogs, error)

	ListRegions() ([]*godo.AppRegion, error)

	ListTiers() ([]*godo.AppTier, error)
	GetTier(slug string) (*godo.AppTier, error)

	ListInstanceSizes() ([]*godo.AppInstanceSize, error)
	GetInstanceSize(slug string) (*godo.AppInstanceSize, error)

	ListAlerts(appID string) ([]*godo.AppAlert, error)
	UpdateAlertDestinations(appID, alertID string, update *godo.AlertDestinationUpdateRequest) (*godo.AppAlert, error)
}

type appsService struct {
	client *godo.Client
	ctx    context.Context
}

var _ AppsService = (*appsService)(nil)

// NewAppsService builds an instance of AppsService.
func NewAppsService(client *godo.Client) AppsService {
	return &appsService{
		client: client,
		ctx:    context.Background(),
	}
}

func (s *appsService) Create(req *godo.AppCreateRequest) (*godo.App, error) {
	app, _, err := s.client.Apps.Create(s.ctx, req)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (s *appsService) Get(appID string) (*godo.App, error) {
	app, _, err := s.client.Apps.Get(s.ctx, appID)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (s *appsService) List() ([]*godo.App, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := s.client.Apps.List(s.ctx, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, 0, len(list))
		for _, item := range list {
			si = append(si, item)
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]*godo.App, 0, len(si))
	for _, item := range si {
		a := item.(*godo.App)
		list = append(list, a)
	}

	return list, nil
}

func (s *appsService) Update(appID string, req *godo.AppUpdateRequest) (*godo.App, error) {
	app, _, err := s.client.Apps.Update(s.ctx, appID, req)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (s *appsService) Delete(appID string) error {
	_, err := s.client.Apps.Delete(s.ctx, appID)
	return err
}

func (s *appsService) Propose(req *godo.AppProposeRequest) (*godo.AppProposeResponse, error) {
	res, _, err := s.client.Apps.Propose(s.ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *appsService) CreateDeployment(appID string, forceRebuild bool) (*godo.Deployment, error) {
	deployment, _, err := s.client.Apps.CreateDeployment(s.ctx, appID, &godo.DeploymentCreateRequest{
		ForceBuild: forceRebuild,
	})
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func (s *appsService) GetDeployment(appID, deploymentID string) (*godo.Deployment, error) {
	deployment, _, err := s.client.Apps.GetDeployment(s.ctx, appID, deploymentID)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func (s *appsService) ListDeployments(appID string) ([]*godo.Deployment, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := s.client.Apps.ListDeployments(s.ctx, appID, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, 0, len(list))
		for _, item := range list {
			si = append(si, item)
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]*godo.Deployment, 0, len(si))
	for _, item := range si {
		d := item.(*godo.Deployment)
		list = append(list, d)
	}

	return list, nil
}

func (s *appsService) GetLogs(appID, deploymentID, component string, logType godo.AppLogType, follow bool, tail int) (*godo.AppLogs, error) {
	logs, _, err := s.client.Apps.GetLogs(s.ctx, appID, deploymentID, component, logType, follow, tail)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *appsService) ListRegions() ([]*godo.AppRegion, error) {
	regions, _, err := s.client.Apps.ListRegions(s.ctx)
	if err != nil {
		return nil, err
	}
	return regions, nil
}

func (s *appsService) ListTiers() ([]*godo.AppTier, error) {
	tiers, _, err := s.client.Apps.ListTiers(s.ctx)
	if err != nil {
		return nil, err
	}
	return tiers, nil
}

func (s *appsService) GetTier(slug string) (*godo.AppTier, error) {
	tier, _, err := s.client.Apps.GetTier(s.ctx, slug)
	if err != nil {
		return nil, err
	}
	return tier, nil
}

func (s *appsService) ListInstanceSizes() ([]*godo.AppInstanceSize, error) {
	instanceSizes, _, err := s.client.Apps.ListInstanceSizes(s.ctx)
	if err != nil {
		return nil, err
	}
	return instanceSizes, nil
}

func (s *appsService) GetInstanceSize(slug string) (*godo.AppInstanceSize, error) {
	instanceSize, _, err := s.client.Apps.GetInstanceSize(s.ctx, slug)
	if err != nil {
		return nil, err
	}
	return instanceSize, nil
}

func (s *appsService) ListAlerts(appID string) ([]*godo.AppAlert, error) {
	alerts, _, err := s.client.Apps.ListAlerts(s.ctx, appID)
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

func (s *appsService) UpdateAlertDestinations(appID, alertID string, update *godo.AlertDestinationUpdateRequest) (*godo.AppAlert, error) {
	alert, _, err := s.client.Apps.UpdateAlertDestinations(s.ctx, appID, alertID, update)
	if err != nil {
		return nil, err
	}
	return alert, nil
}
