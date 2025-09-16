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

type FunctionRoute struct {
	*godo.AgentFunction
}

type FunctionRoutes []FunctionRoute
type ApiKeyInfo struct {
	*godo.ApiKeyInfo
}

type AgentVersion struct {
	*godo.AgentVersion
}

type OpenAiApiKey struct {
	*godo.OpenAiApiKey
}

// OpenAiApiKeys is a slice of OpenAiApiKey.
type OpenAiApiKeys []OpenAiApiKey

// ApiKeys is a slice of ApiKey.
type ApiKeys []ApiKeyInfo

// Agents is a slice of Agent.
type Agents []Agent

type AgentVersions []AgentVersion

// DatacenterRegion represents a datacenter region for GenAI services.
type DatacenterRegion struct {
	*godo.DatacenterRegions
}

// DatacenterRegions is a slice of DatacenterRegion.
type DatacenterRegions []DatacenterRegion

// Model represents an available model for GenAI services.
type Model struct {
	*godo.Model
}

// Models is a slice of Model.
type Models []Model

// KnowledgeBases for Agents
type KnowledgeBases []KnowledgeBase

// KnowledgeBase DataSources for Agents
type KnowledgeBaseDataSources []KnowledgeBaseDataSource

// IndexingJob represents a job for indexing knowledge base content.
type IndexingJob struct {
	*godo.LastIndexingJob
}

// IndexingJobs is a slice of IndexingJob
type IndexingJobs []IndexingJob

// IndexingJobDataSource represents an indexed data source within an indexing job
type IndexingJobDataSource struct {
	*godo.IndexedDataSource
}

// IndexingJobDataSources is a slice of IndexingJobDataSource
type IndexingJobDataSources []IndexingJobDataSource

// GenAIService is an interface for interacting with DigitalOcean's Agent API.
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
	CreateFunctionRoute(id string, req *godo.FunctionRouteCreateRequest) (*Agent, error)
	DeleteFunctionRoute(agent_id string, function_id string) (*Agent, error)
	UpdateFunctionRoute(agent_id string, function_id string, req *godo.FunctionRouteUpdateRequest) (*Agent, error)
	ListAgentVersions(agentID string) (AgentVersions, error)
	ListAgentAPIKeys(agentId string) (ApiKeys, error)
	CreateAgentAPIKey(agentID string, req *godo.AgentAPIKeyCreateRequest) (*ApiKeyInfo, error)
	UpdateAgentAPIKey(agentID string, apikeyID string, req *godo.AgentAPIKeyUpdateRequest) (*ApiKeyInfo, error)
	DeleteAgentAPIKey(agentID string, apikeyID string) error
	RegenerateAgentAPIKey(agentID string, apikeyID string) (*ApiKeyInfo, error)
	ListOpenAIAPIKeys() (OpenAiApiKeys, error)
	CreateOpenAIAPIKey(openaiAPIKeyCreate *godo.OpenAIAPIKeyCreateRequest) (*OpenAiApiKey, error)
	GetOpenAIAPIKey(openaiApiKeyId string) (*OpenAiApiKey, error)
	UpdateOpenAIAPIKey(openaiApiKeyId string, openaiAPIKeyUpdate *godo.OpenAIAPIKeyUpdateRequest) (*OpenAiApiKey, error)
	DeleteOpenAIAPIKey(openaiApiKeyId string) (*OpenAiApiKey, error)
	ListAgentsByOpenAIAPIKey(openaiApiKeyId string) (Agents, error)
	ListDatacenterRegions(servesInference, servesBatch *bool) (DatacenterRegions, error)
	ListAvailableModels() (Models, error)
	ListIndexingJobs() (IndexingJobs, error)
	GetIndexingJob(indexingJobID string) (*IndexingJob, error)
	CancelIndexingJob(indexingJobID string) (*IndexingJob, error)
	ListIndexingJobDataSources(indexingJobID string) (IndexingJobDataSources, error)
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

// CreateFunctionRoute creates a new function route for the specified agent
func (s *genAIService) CreateFunctionRoute(id string, cr *godo.FunctionRouteCreateRequest) (*Agent, error) {
	agent, _, err := s.client.GenAI.CreateFunctionRoute(context.TODO(), id, cr)
	if err != nil {
		return nil, err
	}
	return &Agent{Agent: agent}, nil
}

// DeleteFunctionRoute deletes a function route for the specified agent
func (s *genAIService) DeleteFunctionRoute(agent_id string, function_id string) (*Agent, error) {
	agent, _, err := s.client.GenAI.DeleteFunctionRoute(context.TODO(), agent_id, function_id)
	if err != nil {
		return nil, err
	}
	return &Agent{agent}, nil
}

// Update FunctionRoute updates a function route for the specified agent
func (s *genAIService) UpdateFunctionRoute(agent_id string, function_id string, cr *godo.FunctionRouteUpdateRequest) (*Agent, error) {
	agent, _, err := s.client.GenAI.UpdateFunctionRoute(context.TODO(), agent_id, function_id, cr)
	if err != nil {
		return nil, err
	}
	return &Agent{agent}, nil
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

	// Checking if there are no API keys we don't need to paginate
	opt := &godo.ListOptions{Page: 1, PerPage: perPage}
	res, _, err := f(opt)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return ApiKeys{}, nil
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	if len(si) == 0 {
		return ApiKeys{}, nil
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

func (a *genAIService) ListOpenAIAPIKeys() (OpenAiApiKeys, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := a.client.GenAI.ListOpenAIAPIKeys(context.TODO(), opt)
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

	list := make([]OpenAiApiKey, len(si))
	for i := range si {
		ok := si[i].(*godo.OpenAiApiKey)
		list[i] = OpenAiApiKey{OpenAiApiKey: ok}
	}

	return list, nil
}

func (a *genAIService) CreateOpenAIAPIKey(openaiAPIKeyCreate *godo.OpenAIAPIKeyCreateRequest) (*OpenAiApiKey, error) {
	openaiApiKey, _, err := a.client.GenAI.CreateOpenAIAPIKey(context.TODO(), openaiAPIKeyCreate)
	if err != nil {
		return nil, err
	}
	return &OpenAiApiKey{OpenAiApiKey: openaiApiKey}, nil
}

func (a *genAIService) GetOpenAIAPIKey(openaiApiKeyId string) (*OpenAiApiKey, error) {
	openaiApiKey, _, err := a.client.GenAI.GetOpenAIAPIKey(context.TODO(), openaiApiKeyId)
	if err != nil {
		return nil, err
	}
	return &OpenAiApiKey{OpenAiApiKey: openaiApiKey}, nil
}

func (a *genAIService) UpdateOpenAIAPIKey(openaiApiKeyId string, openaiAPIKeyUpdate *godo.OpenAIAPIKeyUpdateRequest) (*OpenAiApiKey, error) {
	openaiApiKey, _, err := a.client.GenAI.UpdateOpenAIAPIKey(context.TODO(), openaiApiKeyId, openaiAPIKeyUpdate)
	if err != nil {
		return nil, err
	}
	return &OpenAiApiKey{OpenAiApiKey: openaiApiKey}, nil
}

func (a *genAIService) DeleteOpenAIAPIKey(openaiApiKeyId string) (*OpenAiApiKey, error) {
	openaiApiKey, _, err := a.client.GenAI.DeleteOpenAIAPIKey(context.TODO(), openaiApiKeyId)
	if err != nil {
		return nil, err
	}
	return &OpenAiApiKey{OpenAiApiKey: openaiApiKey}, nil
}

func (a *genAIService) ListAgentsByOpenAIAPIKey(openaiApiKeyId string) (Agents, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		agents, resp, err := a.client.GenAI.ListAgentsByOpenAIAPIKey(context.TODO(), openaiApiKeyId, opt)
		if err != nil {
			return nil, nil, err
		}
		list := make([]any, len(agents))
		for i := range agents {
			list[i] = agents[i]
		}
		return list, resp, nil
	}

	// Checking if there are no API keys we don't need to paginate
	opt := &godo.ListOptions{Page: 1, PerPage: perPage}
	res, _, err := f(opt)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return Agents{}, nil
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

func (a *genAIService) ListDatacenterRegions(servesInference, servesBatch *bool) (DatacenterRegions, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := a.client.GenAI.ListDatacenterRegions(context.TODO(), servesInference, servesBatch)
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

	list := make([]DatacenterRegion, len(si))
	for i := range si {
		dc := si[i].(*godo.DatacenterRegions)
		list[i] = DatacenterRegion{DatacenterRegions: dc}
	}

	return list, nil
}

func (a *genAIService) ListAvailableModels() (Models, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := a.client.GenAI.ListAvailableModels(context.TODO(), opt)
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

	list := make([]Model, len(si))
	for i := range si {
		m := si[i].(*godo.Model)
		list[i] = Model{Model: m}
	}

	return list, nil
}

// ListIndexingJobs lists all indexing jobs for knowledge bases.
func (a *genAIService) ListIndexingJobs() (IndexingJobs, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		resp, godoResp, err := a.client.GenAI.ListIndexingJobs(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}
		si := make([]any, len(resp.Jobs))
		for i := range resp.Jobs {
			si[i] = &resp.Jobs[i]
		}
		return si, godoResp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]IndexingJob, len(si))
	for i := range si {
		job := si[i].(*godo.LastIndexingJob)
		list[i] = IndexingJob{LastIndexingJob: job}
	}

	return list, nil
}

// GetIndexingJob retrieves the status of a specific indexing job.
func (a *genAIService) GetIndexingJob(indexingJobID string) (*IndexingJob, error) {
	resp, _, err := a.client.GenAI.GetIndexingJob(context.TODO(), indexingJobID)
	if err != nil {
		return nil, err
	}
	return &IndexingJob{LastIndexingJob: &resp.Job}, nil
}

// CancelIndexingJob cancels a specific indexing job.
func (a *genAIService) CancelIndexingJob(indexingJobID string) (*IndexingJob, error) {
	resp, _, err := a.client.GenAI.CancelIndexingJob(context.TODO(), indexingJobID)
	if err != nil {
		return nil, err
	}
	return &IndexingJob{LastIndexingJob: &resp.Job}, nil
}

// ListIndexingJobDataSources lists all data sources for a specific indexing job.
func (a *genAIService) ListIndexingJobDataSources(indexingJobID string) (IndexingJobDataSources, error) {
	resp, _, err := a.client.GenAI.ListIndexingJobDataSources(context.TODO(), indexingJobID)
	if err != nil {
		return nil, err
	}

	list := make([]IndexingJobDataSource, len(resp.IndexedDataSources))
	for i := range resp.IndexedDataSources {
		list[i] = IndexingJobDataSource{IndexedDataSource: &resp.IndexedDataSources[i]}
	}

	return list, nil
}
