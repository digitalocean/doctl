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

// Agents is a slice of Agent.
type Agents []Agent

// KnowledgeBases for Agents
type KnowledgeBases []KnowledgeBase

// KnowledgeBase DataSources for Agents
type KnowledgeBaseDataSources []KnowledgeBaseDataSource

// AgentService is an interface for interacting with DigitalOcean's Agent API.
type GenAIService interface {
	ListKnowledgeBases() (KnowledgeBases, error)
	GetKnowledgeBase(knowledgeBaseID string) (*KnowledgeBase, error)
	CreateKnowledgeBase(req *godo.KnowledgeBaseCreateRequest) (*KnowledgeBase, error)
	UpdateKnowledgebase(knowledgeBaseID string, req *godo.UpdateKnowledgeBaseRequest) (*KnowledgeBase, error)
	DeleteKnowledgebase(knowledgeBaseID string) error
	AddKnowledgeBaseDataSource(knowledgeBaseID string, req *godo.AddDataSourceRequest) (*KnowledgeBaseDataSource, error)
	ListKnowledgeBaseDataSources(knowledgeBaseID string) (KnowledgeBaseDataSources, error)
	DeleteKnowledgeBaseDataSource(knowledgeBaseID string, dataSourceID string) error
	AttachKnowledgebase(agentId string, knowledgeBaseID string) (*Agent, error)
	DetachKnowledgebase(agentId string, knowledgeBaseID string) (*Agent, error)
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
	kb, _, err := a.client.GenAI.GetKnowledgeBase(context.TODO(), knowledgeBaseID)
	if err != nil {
		return nil, err
	}
	return &KnowledgeBase{KnowledgeBase: kb}, nil
}

func (a *genAIService) ListKnowledgeBaseDataSources(knowledgeBaseID string) (KnowledgeBaseDataSources, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := a.client.GenAI.ListDataSources(context.TODO(), knowledgeBaseID, opt)
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

func (a *genAIService) UpdateKnowledgebase(knowledgeBaseID string, req *godo.UpdateKnowledgeBaseRequest) (*KnowledgeBase, error) {
	kb, _, err := a.client.GenAI.UpdateKnowledgeBase(context.TODO(), knowledgeBaseID, req)
	if err != nil {
		return nil, err
	}
	return &KnowledgeBase{KnowledgeBase: kb}, nil
}

func (a *genAIService) AddKnowledgeBaseDataSource(knowledgeBaseID string, req *godo.AddDataSourceRequest) (*KnowledgeBaseDataSource, error) {
	kb, _, err := a.client.GenAI.AddDataSource(context.TODO(), knowledgeBaseID, req)
	if err != nil {
		return nil, err
	}
	return &KnowledgeBaseDataSource{KnowledgeBaseDataSource: kb}, nil
}

func (a *genAIService) DeleteKnowledgeBaseDataSource(knowledgeBaseID string, dataSourceID string) error {
	_, _, _, err := a.client.GenAI.DeleteDataSource(context.TODO(), knowledgeBaseID, dataSourceID)
	return err
}

func (a *genAIService) DeleteKnowledgebase(knowledgeBaseID string) error {
	_, _, err := a.client.GenAI.DeleteKnowledgeBase(context.TODO(), knowledgeBaseID)
	return err
}

func (a *genAIService) AttachKnowledgebase(agentId string, knowledgeBaseID string) (*Agent, error) {
	agent, _, err := a.client.GenAI.AttachKnowledgeBase(context.TODO(), agentId, knowledgeBaseID)
	if err != nil {
		return &Agent{}, err
	}
	return &Agent{Agent: agent}, nil
}

func (a *genAIService) DetachKnowledgebase(agentId string, knowledgeBaseID string) (*Agent, error) {
	agent, _, err := a.client.GenAI.DetachKnowledgeBase(context.TODO(), agentId, knowledgeBaseID)
	if err != nil {
		return &Agent{}, err
	}
	return &Agent{Agent: agent}, nil
}
