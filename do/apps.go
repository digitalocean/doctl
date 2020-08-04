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

	CreateDeployment(appID string) (*godo.Deployment, error)
	GetDeployment(appID, deploymentID string) (*godo.Deployment, error)
	ListDeployments(appID string) ([]*godo.Deployment, error)

	GetLogs(appID, deploymentID, component string, logType godo.AppLogType, follow bool) (*godo.AppLogs, error)
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
	apps, _, err := s.client.Apps.List(s.ctx, nil)
	if err != nil {
		return nil, err
	}
	return apps, nil
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

func (s *appsService) CreateDeployment(appID string) (*godo.Deployment, error) {
	deployment, _, err := s.client.Apps.CreateDeployment(s.ctx, appID)
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
	deployments, _, err := s.client.Apps.ListDeployments(s.ctx, appID, nil)
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func (s *appsService) GetLogs(appID, deploymentID, component string, logType godo.AppLogType, follow bool) (*godo.AppLogs, error) {
	logs, _, err := s.client.Apps.GetLogs(s.ctx, appID, deploymentID, component, logType, follow)
	if err != nil {
		return nil, err
	}
	return logs, nil
}
