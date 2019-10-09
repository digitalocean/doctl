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
	"errors"

	"github.com/digitalocean/godo"
)

// Project wraps a godo Project.
type Project struct {
	*godo.Project
}

// Projects is a slice of Project.
type Projects []Project

// ProjectResource wraps a godo ProjectResource
type ProjectResource struct {
	*godo.ProjectResource
}

// ProjectResources is a slice of ProjectResource.
type ProjectResources []ProjectResource

// ProjectsService is the godo ProjectsService interface.
type ProjectsService interface {
	List() (Projects, error)
	GetDefault() (*Project, error)
	Get(projectUUID string) (*Project, error)
	Create(*godo.CreateProjectRequest) (*Project, error)
	Update(projectUUID string, req *godo.UpdateProjectRequest) (*Project, error)
	Delete(projectUUID string) error

	ListResources(projectUUID string) (ProjectResources, error)
	AssignResources(projectUUID string, resources []string) (ProjectResources, error)
}

type projectsService struct {
	client *godo.Client
	ctx    context.Context
}

var _ ProjectsService = &projectsService{}

// NewProjectsService builds an instance of ProjectsService.
func NewProjectsService(client *godo.Client) ProjectsService {
	return &projectsService{
		client: client,
		ctx:    context.Background(),
	}
}

// List projects.
func (ps *projectsService) List() (Projects, error) {
	listFn := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ps.client.Projects.List(ps.ctx, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	return projectsPaginatedListHelper(listFn)
}

func (ps *projectsService) GetDefault() (*Project, error) {
	f, _, err := ps.client.Projects.GetDefault(ps.ctx)
	if err != nil {
		return nil, err
	}

	return &Project{Project: f}, nil
}

func (ps *projectsService) Get(projectUUID string) (*Project, error) {
	f, _, err := ps.client.Projects.Get(ps.ctx, projectUUID)
	if err != nil {
		return nil, err
	}

	return &Project{Project: f}, nil
}

func (ps *projectsService) Create(cr *godo.CreateProjectRequest) (*Project, error) {
	f, _, err := ps.client.Projects.Create(ps.ctx, cr)
	if err != nil {
		return nil, err
	}

	return &Project{Project: f}, nil
}

func (ps *projectsService) Update(projectUUID string, ur *godo.UpdateProjectRequest) (*Project, error) {
	p, _, err := ps.client.Projects.Update(ps.ctx, projectUUID, ur)
	if err != nil {
		return nil, err
	}

	return &Project{Project: p}, nil
}

func (ps *projectsService) Delete(projectUUID string) error {
	_, err := ps.client.Projects.Delete(ps.ctx, projectUUID)
	return err
}

func (ps *projectsService) ListResources(projectUUID string) (ProjectResources, error) {
	listFn := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ps.client.Projects.ListResources(ps.ctx, projectUUID, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	return projectResourcesPaginatedListHelper(listFn)
}

func (ps *projectsService) AssignResources(projectUUID string, resources []string) (ProjectResources, error) {
	assignableResources := make([]interface{}, len(resources))
	for i, resource := range resources {
		assignableResources[i] = resource
	}

	assignedResources, _, err := ps.client.Projects.AssignResources(ps.ctx, projectUUID, assignableResources...)
	if err != nil {
		return nil, err
	}

	prs := make(ProjectResources, len(assignedResources))
	for i := range assignedResources {
		prs[i] = ProjectResource{&assignedResources[i]}
	}

	return prs, err
}

func projectsPaginatedListHelper(listFn func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error)) (Projects, error) {
	si, err := PaginateResp(listFn)
	if err != nil {
		return nil, err
	}

	list := make([]Project, len(si))
	for i := range si {
		a, ok := si[i].(godo.Project)
		if !ok {
			return nil, errors.New("unexpected value in response")
		}

		list[i] = Project{Project: &a}
	}

	return list, nil
}

func projectResourcesPaginatedListHelper(listFn func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error)) (ProjectResources, error) {
	si, err := PaginateResp(listFn)
	if err != nil {
		return nil, err
	}

	list := make([]ProjectResource, len(si))
	for i := range si {
		a, ok := si[i].(godo.ProjectResource)
		if !ok {
			return nil, errors.New("unexpected value in response")
		}

		list[i] = ProjectResource{ProjectResource: &a}
	}

	return list, nil
}
