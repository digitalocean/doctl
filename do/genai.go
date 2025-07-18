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

type KnowledgeBase struct {
	*godo.KnowledgeBase
}

type KnowledgeBaseDataSource struct {
	*godo.KnowledgeBaseDataSource
}

type AgentRouteResponse struct {
	*godo.AgentRouteResponse
}

// Agents is a slice of Agent.
type Agents []Agent

// KnowledgeBases for Agents
type KnowledgeBases []KnowledgeBase

// KnowledgeBase DataSources for Agents
type KnowledgeBaseDataSources []KnowledgeBaseDataSource

// AgentService is an interface for interacting with DigitalOcean's Agent API.
type GenAIService interface {
	ListAgents() (Agents, error)
	CreateAgent(req *godo.AgentCreateRequest) (*Agent, error)
	GetAgent(agentID string) (*Agent, error)
	UpdateAgent(agentID string, req *godo.AgentUpdateRequest) (*Agent, error)
	DeleteAgent(agentID string) error
	UpdateAgentVisibility(agentID string, req *godo.AgentVisibilityUpdateRequest) (*Agent, error)
	ListKnowledgeBases() (KnowledgeBases, error)
	GetKnowledgeBase(knowledgeBaseID string) (*KnowledgeBase, error)
	CreateKnowledgeBase(req *godo.KnowledgeBaseCreateRequest) (*KnowledgeBase, error)
	UpdateKnowledgeBase(knowledgeBaseID string, req *godo.UpdateKnowledgeBaseRequest) (*KnowledgeBase, error)
	DeleteKnowledgeBase(knowledgeBaseID string) error
	AddKnowledgeBaseDataSource(knowledgeBaseID string, req *godo.AddKnowledgeBaseDataSourceRequest) (*KnowledgeBaseDataSource, error)
	ListKnowledgeBaseDataSources(knowledgeBaseID string) (KnowledgeBaseDataSources, error)
	DeleteKnowledgeBaseDataSource(knowledgeBaseID string, dataSourceID string) error
	AttachKnowledgeBaseToAgent(agentId string, knowledgeBaseID string) (*Agent, error)
	DetachKnowledgeBaseToAgent(agentId string, knowledgeBaseID string) (*Agent, error)
	AddAgentRoute(parentAgentID string, childAgentID string) (*AgentRouteResponse, error)
	UpdateAgentRoute(parentAgentID string, childAgentID string, req *godo.AgentRouteUpdateRequest) (*AgentRouteResponse, error)
	DeleteAgentRoute(parentAgentID string, childAgentID string) error
}

var _ GenAIService = &genAIService{}

type genAIService struct {
	client *godo.Client
}

// NewAgentService builds an instance of AgentService.
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

// ListKnowledgeBases lists all knowledge bases for an agent.
func (a *genAIService) ListKnowledgeBases() (KnowledgeBases, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := a.client.GenAI.ListKnowledgeBases(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}
		si := make([]any, len(list))
		for i := range list {
			si[i] = &list[i]
		}
		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]KnowledgeBase, len(si))
	for i := range si {
		kb := si[i].(*godo.KnowledgeBase)
		list[i] = KnowledgeBase{KnowledgeBase: kb}
	}

	return list, nil
}

func (a *genAIService) GetKnowledgeBase(knowledgeBaseID string) (*KnowledgeBase, error) {
	kb, _, _, err := a.client.GenAI.GetKnowledgeBase(context.TODO(), knowledgeBaseID)
	if err != nil {
		return nil, err
	}
	return &KnowledgeBase{KnowledgeBase: kb}, nil
}

func (a *genAIService) ListKnowledgeBaseDataSources(knowledgeBaseID string) (KnowledgeBaseDataSources, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := a.client.GenAI.ListKnowledgeBaseDataSources(context.TODO(), knowledgeBaseID, opt)
		if err != nil {
			return nil, nil, err
		}
		si := make([]any, len(list))
		for i := range list {
			si[i] = &list[i]
		}
		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]KnowledgeBaseDataSource, len(si))
	for i := range si {
		kb := si[i].(*godo.KnowledgeBaseDataSource)
		list[i] = KnowledgeBaseDataSource{KnowledgeBaseDataSource: kb}
	}

	return list, nil
}

func (a *genAIService) CreateKnowledgeBase(req *godo.KnowledgeBaseCreateRequest) (*KnowledgeBase, error) {
	kb, _, err := a.client.GenAI.CreateKnowledgeBase(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &KnowledgeBase{KnowledgeBase: kb}, nil
}

func (a *genAIService) UpdateKnowledgeBase(knowledgeBaseID string, req *godo.UpdateKnowledgeBaseRequest) (*KnowledgeBase, error) {
	kb, _, err := a.client.GenAI.UpdateKnowledgeBase(context.TODO(), knowledgeBaseID, req)
	if err != nil {
		return nil, err
	}
	return &KnowledgeBase{KnowledgeBase: kb}, nil
}

func (a *genAIService) AddKnowledgeBaseDataSource(knowledgeBaseID string, req *godo.AddKnowledgeBaseDataSourceRequest) (*KnowledgeBaseDataSource, error) {
	kb, _, err := a.client.GenAI.AddKnowledgeBaseDataSource(context.TODO(), knowledgeBaseID, req)
	if err != nil {
		return nil, err
	}
	return &KnowledgeBaseDataSource{KnowledgeBaseDataSource: kb}, nil
}

func (a *genAIService) DeleteKnowledgeBaseDataSource(knowledgeBaseID string, dataSourceID string) error {
	_, _, _, err := a.client.GenAI.DeleteKnowledgeBaseDataSource(context.TODO(), knowledgeBaseID, dataSourceID)
	return err
}

func (a *genAIService) DeleteKnowledgeBase(knowledgeBaseID string) error {
	_, _, err := a.client.GenAI.DeleteKnowledgeBase(context.TODO(), knowledgeBaseID)
	return err
}

func (a *genAIService) AttachKnowledgeBaseToAgent(agentId string, knowledgeBaseID string) (*Agent, error) {
	agent, _, err := a.client.GenAI.AttachKnowledgeBaseToAgent(context.TODO(), agentId, knowledgeBaseID)
	if err != nil {
		return &Agent{}, err
	}
	return &Agent{Agent: agent}, nil
}

func (a *genAIService) DetachKnowledgeBaseToAgent(agentId string, knowledgeBaseID string) (*Agent, error) {
	agent, _, err := a.client.GenAI.DetachKnowledgeBaseToAgent(context.TODO(), agentId, knowledgeBaseID)
	if err != nil {
		return &Agent{}, err
	}
	return &Agent{Agent: agent}, nil
}

func (a *genAIService) AddAgentRoute(parentAgentID string, childAgentID string) (*AgentRouteResponse, error) {
	// Create the request object
	req := &godo.AgentRouteCreateRequest{
		ParentAgentUuid: parentAgentID,
		ChildAgentUuid:  childAgentID,
	}

	routeResponse, _, err := a.client.GenAI.AddAgentRoute(context.TODO(), parentAgentID, childAgentID, req)
	if err != nil {
		return nil, err
	}
	return &AgentRouteResponse{AgentRouteResponse: routeResponse}, nil
}

func (a *genAIService) UpdateAgentRoute(parentAgentID string, childAgentID string, req *godo.AgentRouteUpdateRequest) (*AgentRouteResponse, error) {
	routeResponse, _, err := a.client.GenAI.UpdateAgentRoute(context.TODO(), parentAgentID, childAgentID, req)
	if err != nil {
		return nil, err
	}
	return &AgentRouteResponse{AgentRouteResponse: routeResponse}, nil
}

func (a *genAIService) DeleteAgentRoute(parentAgentID string, childAgentID string) error {
	_, _, err := a.client.GenAI.DeleteAgentRoute(context.TODO(), parentAgentID, childAgentID)
	return err
}
