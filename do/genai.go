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

// Agent wraps a godo.Agent.
type Agent struct {
	*godo.Agent
}
type ApiKeyInfo struct {
	*godo.ApiKeyInfo
}

type AgentVersion struct {
	*godo.AgentVersion
}

// ApiKeys is a slice of ApiKey.
type ApiKeys []ApiKeyInfo

// Agents is a slice of Agent.
type Agents []Agent

type AgentVersions []AgentVersion

// GenAIService is an interface for interacting with DigitalOcean's Agent API.
type GenAIService interface {
	ListAgents() (Agents, error)
	CreateAgent(req *godo.AgentCreateRequest) (*Agent, error)
	GetAgent(agentID string) (*Agent, error)
	UpdateAgent(agentID string, req *godo.AgentUpdateRequest) (*Agent, error)
	DeleteAgent(agentID string) error
	UpdateAgentVisibility(agentID string, req *godo.AgentVisibilityUpdateRequest) (*Agent, error)
	ListAgentVersions(agentID string) (AgentVersions, error)
	ListAgentAPIKeys(agentId string) (ApiKeys, error)
	CreateAgentAPIKey(agentID string, req *godo.AgentAPIKeyCreateRequest) (*ApiKeyInfo, error)
	UpdateAgentAPIKey(agentID string, apikeyID string, req *godo.AgentAPIKeyUpdateRequest) (*ApiKeyInfo, error)
	DeleteAgentAPIKey(agentID string, apikeyID string) error
	RegenerateAgentAPIKey(agentID string, apikeyID string) (*ApiKeyInfo, error)
}

var _ GenAIService = &genAIService{}

type genAIService struct {
	client *godo.Client
}

// NewGenAIService builds an instance of GenAIService.
func NewGenAIService(client *godo.Client) GenAIService {
	return &genAIService{
		client: client,
	}
}

// List lists all agents.
func (a *genAIService) ListAgents() (Agents, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := a.client.GenAI.ListAgents(context.TODO(), opt)
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

	list := make([]Agent, len(si))
	for i := range si {
		a := si[i].(*godo.Agent)
		list[i] = Agent{Agent: a}
	}

	return list, nil
}

// Create creates a new agent.
func (a *genAIService) CreateAgent(req *godo.AgentCreateRequest) (*Agent, error) {
	agent, _, err := a.client.GenAI.CreateAgent(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &Agent{Agent: agent}, nil
}

// Get retrieves an agent by ID.
func (a *genAIService) GetAgent(agentID string) (*Agent, error) {
	agent, _, err := a.client.GenAI.GetAgent(context.TODO(), agentID)
	if err != nil {
		return nil, err
	}
	return &Agent{Agent: agent}, nil
}

// Update updates an agent by ID.
func (a *genAIService) UpdateAgent(agentID string, req *godo.AgentUpdateRequest) (*Agent, error) {
	agent, _, err := a.client.GenAI.UpdateAgent(context.TODO(), agentID, req)
	if err != nil {
		return nil, err
	}
	return &Agent{Agent: agent}, nil
}

func (a *genAIService) DeleteAgent(agentID string) error {
	_, _, err := a.client.GenAI.DeleteAgent(context.TODO(), agentID)
	return err
}

// UpdateVisibility updates the visibility of an agent by ID.
func (a *genAIService) UpdateAgentVisibility(agentID string, req *godo.AgentVisibilityUpdateRequest) (*Agent, error) {
	agent, _, err := a.client.GenAI.UpdateAgentVisibility(context.TODO(), agentID, req)
	if err != nil {
		return nil, err
	}
	return &Agent{Agent: agent}, nil
}

// CreateAgentAPIKey implements GenAIService.
func (a *genAIService) CreateAgentAPIKey(agentID string, req *godo.AgentAPIKeyCreateRequest) (*ApiKeyInfo, error) {
	apikeyInfo, _, err := a.client.GenAI.CreateAgentAPIKey(context.TODO(), agentID, req)
	if err != nil {
		return nil, err
	}
	return &ApiKeyInfo{ApiKeyInfo: apikeyInfo}, nil
}

// DeleteAgentAPIKey implements GenAIService.
func (a *genAIService) DeleteAgentAPIKey(agentID string, apikeyID string) error {
	_, _, err := a.client.GenAI.DeleteAgentAPIKey(context.TODO(), agentID, apikeyID)
	return err
}

// ListAgentAPIKeys implements GenAIService.
func (a *genAIService) ListAgentAPIKeys(agentId string) (ApiKeys, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := a.client.GenAI.ListAgentAPIKeys(context.TODO(), agentId, opt)
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

	list := make([]ApiKeyInfo, len(si))
	for i := range si {
		a := si[i].(*godo.ApiKeyInfo)
		list[i] = ApiKeyInfo{ApiKeyInfo: a}
	}

	return list, nil
}

// RegenerateAgentAPIKey implements GenAIService.
func (a *genAIService) RegenerateAgentAPIKey(agentID string, apikeyID string) (*ApiKeyInfo, error) {
	apikeyInfo, _, err := a.client.GenAI.RegenerateAgentAPIKey(context.TODO(), agentID, apikeyID)
	if err != nil {
		return nil, err
	}
	return &ApiKeyInfo{ApiKeyInfo: apikeyInfo}, nil
}

// UpdateAgentAPIKey implements GenAIService.
func (a *genAIService) UpdateAgentAPIKey(agentID string, apikeyID string, req *godo.AgentAPIKeyUpdateRequest) (*ApiKeyInfo, error) {
	apikeyInfo, _, err := a.client.GenAI.UpdateAgentAPIKey(context.TODO(), agentID, apikeyID, req)
	if err != nil {
		return nil, err
	}
	return &ApiKeyInfo{ApiKeyInfo: apikeyInfo}, nil
}

func (a *genAIService) ListAgentVersions(agentID string) (AgentVersions, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := a.client.GenAI.ListAgentVersions(context.TODO(), agentID, opt)
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

	list := make([]AgentVersion, len(si))
	for i := range si {
		a := si[i].(*godo.AgentVersion)
		list[i] = AgentVersion{AgentVersion: a}
	}

	return list, nil

}
