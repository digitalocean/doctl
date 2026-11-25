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

// ADSService is an interface for interacting with the Agent Data Services
// (ADS) API: workspaces, tokens, workspace secrets, and data flows.
type ADSService interface {
	ListPlans() (*godo.ADSPlansResponse, error)

	CreateWorkspace(region string, req *godo.ADSWorkspaceCreateRequest) (*godo.ADSWorkspace, error)
	ListWorkspaces(region string) (*godo.ADSWorkspacesResponse, error)
	GetWorkspace(region, uuid string) (*godo.ADSWorkspace, error)
	DeleteWorkspace(region, uuid string) error
	GetWorkspaceQueryCredentials(region, uuid string) (*godo.ADSQueryCredentials, error)

	CreateToken(req *godo.ADSTokenCreateRequest) (*godo.ADSTokenCreateResponse, error)
	ListTokens(req *godo.ADSTokenListRequest) (*godo.ADSListTokensResponse, error)

	PutWorkspaceSecret(req *godo.ADSPutWorkspaceSecretRequest) (*godo.ADSPutWorkspaceSecretResponse, error)
	GetWorkspaceSecret(region, uuid, secretKey string) (*godo.ADSGetWorkspaceSecretResponse, error)
	ListWorkspaceSecrets(region, uuid string) (*godo.ADSListWorkspaceSecretsResponse, error)
	DeleteWorkspaceSecret(region, uuid, secretKey string) error

	CreateDataFlow(req *godo.ADSDataFlowCreateRequest) (*godo.ADSDataFlowCreateResponse, error)
	ListDataFlows(req *godo.ADSDataFlowListRequest) (*godo.ADSListDataFlowsResponse, error)
	GetDataFlow(region, workspace, id string) (*godo.ADSDataFlow, error)
	UpdateDataFlow(req *godo.ADSDataFlowUpdateRequest) (*godo.ADSDataFlowUpdateResponse, error)
	DeleteDataFlow(region, workspace, id string) error
	PauseDataFlow(region, workspace, id string) error
	ResumeDataFlow(region, workspace, id string) error
	RestartDataFlow(region, workspace, id string) error
	RewindDataFlow(req *godo.ADSDataFlowRewindRequest) error
	CreateFchainUploadURL(req *godo.ADSFchainUploadURLRequest) (*godo.ADSFchainUploadURLResponse, error)
	AttachDataFlowFchain(req *godo.ADSAttachDataFlowFchainRequest) error

	GetWorkspaceMetrics(req *godo.ADSWorkspaceMetricsRequest) (*godo.ADSWorkspaceMetricsResponse, error)
}

type adsService struct {
	client *godo.Client
}

var _ ADSService = &adsService{}

// NewADSService builds an ADSService instance.
func NewADSService(client *godo.Client) ADSService {
	return &adsService{client: client}
}

func (s *adsService) ListPlans() (*godo.ADSPlansResponse, error) {
	resp, _, err := s.client.ADS.ListPlans(context.TODO())
	return resp, err
}

func (s *adsService) CreateWorkspace(region string, req *godo.ADSWorkspaceCreateRequest) (*godo.ADSWorkspace, error) {
	ws, _, err := s.client.ADS.CreateWorkspace(context.TODO(), region, req)
	return ws, err
}

func (s *adsService) ListWorkspaces(region string) (*godo.ADSWorkspacesResponse, error) {
	resp, _, err := s.client.ADS.ListWorkspaces(context.TODO(), region)
	return resp, err
}

func (s *adsService) GetWorkspace(region, uuid string) (*godo.ADSWorkspace, error) {
	ws, _, err := s.client.ADS.GetWorkspace(context.TODO(), region, uuid)
	return ws, err
}

func (s *adsService) DeleteWorkspace(region, uuid string) error {
	_, err := s.client.ADS.DeleteWorkspace(context.TODO(), region, uuid)
	return err
}

func (s *adsService) GetWorkspaceQueryCredentials(region, uuid string) (*godo.ADSQueryCredentials, error) {
	creds, _, err := s.client.ADS.GetWorkspaceQueryCredentials(context.TODO(), region, uuid)
	return creds, err
}

func (s *adsService) CreateToken(req *godo.ADSTokenCreateRequest) (*godo.ADSTokenCreateResponse, error) {
	resp, _, err := s.client.ADS.CreateToken(context.TODO(), req)
	return resp, err
}

func (s *adsService) ListTokens(req *godo.ADSTokenListRequest) (*godo.ADSListTokensResponse, error) {
	resp, _, err := s.client.ADS.ListTokens(context.TODO(), req)
	return resp, err
}

func (s *adsService) PutWorkspaceSecret(req *godo.ADSPutWorkspaceSecretRequest) (*godo.ADSPutWorkspaceSecretResponse, error) {
	resp, _, err := s.client.ADS.PutWorkspaceSecret(context.TODO(), req)
	return resp, err
}

func (s *adsService) GetWorkspaceSecret(region, uuid, secretKey string) (*godo.ADSGetWorkspaceSecretResponse, error) {
	resp, _, err := s.client.ADS.GetWorkspaceSecret(context.TODO(), region, uuid, secretKey)
	return resp, err
}

func (s *adsService) ListWorkspaceSecrets(region, uuid string) (*godo.ADSListWorkspaceSecretsResponse, error) {
	resp, _, err := s.client.ADS.ListWorkspaceSecrets(context.TODO(), region, uuid)
	return resp, err
}

func (s *adsService) DeleteWorkspaceSecret(region, uuid, secretKey string) error {
	_, err := s.client.ADS.DeleteWorkspaceSecret(context.TODO(), region, uuid, secretKey)
	return err
}

func (s *adsService) CreateDataFlow(req *godo.ADSDataFlowCreateRequest) (*godo.ADSDataFlowCreateResponse, error) {
	resp, _, err := s.client.ADS.CreateDataFlow(context.TODO(), req)
	return resp, err
}

func (s *adsService) ListDataFlows(req *godo.ADSDataFlowListRequest) (*godo.ADSListDataFlowsResponse, error) {
	resp, _, err := s.client.ADS.ListDataFlows(context.TODO(), req)
	return resp, err
}

func (s *adsService) GetDataFlow(region, workspace, id string) (*godo.ADSDataFlow, error) {
	flow, _, err := s.client.ADS.GetDataFlow(context.TODO(), region, workspace, id)
	return flow, err
}

func (s *adsService) UpdateDataFlow(req *godo.ADSDataFlowUpdateRequest) (*godo.ADSDataFlowUpdateResponse, error) {
	resp, _, err := s.client.ADS.UpdateDataFlow(context.TODO(), req)
	return resp, err
}

func (s *adsService) DeleteDataFlow(region, workspace, id string) error {
	_, err := s.client.ADS.DeleteDataFlow(context.TODO(), region, workspace, id)
	return err
}

func (s *adsService) PauseDataFlow(region, workspace, id string) error {
	_, err := s.client.ADS.PauseDataFlow(context.TODO(), region, workspace, id)
	return err
}

func (s *adsService) ResumeDataFlow(region, workspace, id string) error {
	_, err := s.client.ADS.ResumeDataFlow(context.TODO(), region, workspace, id)
	return err
}

func (s *adsService) RestartDataFlow(region, workspace, id string) error {
	_, err := s.client.ADS.RestartDataFlow(context.TODO(), region, workspace, id)
	return err
}

func (s *adsService) RewindDataFlow(req *godo.ADSDataFlowRewindRequest) error {
	_, err := s.client.ADS.RewindDataFlow(context.TODO(), req)
	return err
}

func (s *adsService) CreateFchainUploadURL(req *godo.ADSFchainUploadURLRequest) (*godo.ADSFchainUploadURLResponse, error) {
	resp, _, err := s.client.ADS.CreateFchainUploadURL(context.TODO(), req)
	return resp, err
}

func (s *adsService) AttachDataFlowFchain(req *godo.ADSAttachDataFlowFchainRequest) error {
	_, err := s.client.ADS.AttachDataFlowFchain(context.TODO(), req)
	return err
}

func (s *adsService) GetWorkspaceMetrics(req *godo.ADSWorkspaceMetricsRequest) (*godo.ADSWorkspaceMetricsResponse, error) {
	resp, _, err := s.client.ADS.GetWorkspaceMetrics(context.TODO(), req)
	return resp, err
}
