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

// Agents is a slice of Agent.
type Agents []Agent

// AgentService is an interface for interacting with DigitalOcean's Agent API.
type AgentService interface {
	List() (Agents, error)
	Create(*godo.AgentCreateRequest) (*Agent, error)
	Get(agentID string) (*Agent, error)
	Update(agentID string, req *godo.AgentUpdateRequest) (*Agent, error)
	Delete(agentID string) error
	UpdateVisibility(agentID string, req *godo.AgentVisibilityUpdateRequest) (*Agent, error)
}

var _ AgentService = &agentService{}

type agentService struct {
	client *godo.Client
}

// NewAgentService builds an instance of AgentService.
func NewAgentService(client *godo.Client) AgentService {
	return &agentService{
		client: client,
	}
}

// List lists all agents.
func (a *agentService) List() (Agents, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := a.client.GenAI.List(context.TODO(), opt)
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
func (a *agentService) Create(req *godo.AgentCreateRequest) (*Agent, error) {
	agent, _, err := a.client.GenAI.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &Agent{Agent: agent}, nil
}

// Get retrieves an agent by ID.
func (a *agentService) Get(agentID string) (*Agent, error) {
	agent, _, err := a.client.GenAI.Get(context.TODO(), agentID)
	if err != nil {
		return nil, err
	}
	return &Agent{Agent: agent}, nil
}

// Update updates an agent by ID.
func (a *agentService) Update(agentID string, req *godo.AgentUpdateRequest) (*Agent, error) {
	agent, _, err := a.client.GenAI.Update(context.TODO(), agentID, req)
	if err != nil {
		return nil, err
	}
	return &Agent{Agent: agent}, nil
}

func (a *agentService) Delete(agentID string) error {
	_, _, err := a.client.GenAI.Delete(context.TODO(), agentID)
	return err
}

// UpdateVisibility updates the visibility of an agent by ID.
func (a *agentService) UpdateVisibility(agentID string, req *godo.AgentVisibilityUpdateRequest) (*Agent, error) {
	agent, _, err := a.client.GenAI.UpdateVisibility(context.TODO(), agentID, req)
	if err != nil {
		return nil, err
	}
	return &Agent{Agent: agent}, nil
}
