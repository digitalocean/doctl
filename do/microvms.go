/*
Copyright 2025 The Doctl Authors All rights reserved.
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

// MicroVM wraps a godo.MicroVM.
type MicroVM struct {
	*godo.MicroVM
}

// MicroVMs is a slice of MicroVM.
type MicroVMs []MicroVM

// MicroVMCheckpoint wraps a godo.MicroVMCheckpoint.
type MicroVMCheckpoint struct {
	*godo.MicroVMCheckpoint
}

// MicroVMCheckpoints is a slice of MicroVMCheckpoint.
type MicroVMCheckpoints []MicroVMCheckpoint

//go:generate go run go.uber.org/mock/mockgen -source microvms.go -package=mocks -destination mocks/MicroVMsService.go MicroVMsService

// MicroVMListFilter selects which MicroVMs List returns. Empty fields are
// omitted. Region, Name, and TagName combine: the API ANDs whichever are set.
type MicroVMListFilter struct {
	Region  string
	Name    string
	TagName string
}

// MicroVMsService is an interface for interacting with DigitalOcean's
// MicroVM API.
type MicroVMsService interface {
	List(filter MicroVMListFilter) (MicroVMs, error)
	Get(id string) (*MicroVM, error)
	Create(req *godo.MicroVMCreateRequest) (*MicroVM, error)
	Pause(id string) (*MicroVM, error)
	Resume(id string) (*MicroVM, error)
	Delete(id string) error

	ListCheckpoints(microVMID string) (MicroVMCheckpoints, error)
	CreateCheckpoint(microVMID string, req *godo.MicroVMCheckpointCreateRequest) (*MicroVMCheckpoint, error)
	GetCheckpoint(id string) (*MicroVMCheckpoint, error)
	DeleteCheckpoint(id string) error

	GetCreateOptions() (*godo.MicroVMCreateOptions, error)

	Exec(id string, req *godo.MicroVMExecRequest) (*godo.MicroVMExecResult, error)
	ConsoleURL(id string, opt *godo.MicroVMConsoleOptions) (string, error)
}

type microVMsService struct {
	client *godo.Client
}

var _ MicroVMsService = &microVMsService{}

// NewMicroVMsService builds a MicroVMsService backed by the provided
// godo client.
func NewMicroVMsService(client *godo.Client) MicroVMsService {
	return &microVMsService{client: client}
}

func (s *microVMsService) List(filter MicroVMListFilter) (MicroVMs, error) {
	base := &godo.ListMicroVMsOptions{
		Region:  filter.Region,
		Name:    filter.Name,
		TagName: filter.TagName,
	}
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		listOpt := *base
		if opt != nil {
			listOpt.ListOptions = *opt
		}
		list, resp, err := s.client.MicroVMs.ListFiltered(context.TODO(), &listOpt)
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

	list := make(MicroVMs, len(si))
	for i := range si {
		md := si[i].(godo.MicroVM)
		list[i] = MicroVM{MicroVM: &md}
	}
	return list, nil
}

func (s *microVMsService) Get(id string) (*MicroVM, error) {
	md, _, err := s.client.MicroVMs.Get(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &MicroVM{MicroVM: md}, nil
}

func (s *microVMsService) Create(req *godo.MicroVMCreateRequest) (*MicroVM, error) {
	md, _, err := s.client.MicroVMs.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &MicroVM{MicroVM: md}, nil
}

func (s *microVMsService) Pause(id string) (*MicroVM, error) {
	md, _, err := s.client.MicroVMs.Pause(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &MicroVM{MicroVM: md}, nil
}

func (s *microVMsService) Resume(id string) (*MicroVM, error) {
	md, _, err := s.client.MicroVMs.Resume(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &MicroVM{MicroVM: md}, nil
}

func (s *microVMsService) Delete(id string) error {
	_, err := s.client.MicroVMs.Delete(context.TODO(), id)
	return err
}

func (s *microVMsService) ListCheckpoints(microVMID string) (MicroVMCheckpoints, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		listOpt := &godo.ListMicroVMCheckpointsOptions{
			ListOptions: *opt,
			MicroVMID:   microVMID,
		}
		list, resp, err := s.client.MicroVMs.ListCheckpoints(context.TODO(), listOpt)
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

	list := make(MicroVMCheckpoints, len(si))
	for i := range si {
		cp := si[i].(godo.MicroVMCheckpoint)
		list[i] = MicroVMCheckpoint{MicroVMCheckpoint: &cp}
	}
	return list, nil
}

func (s *microVMsService) CreateCheckpoint(microVMID string, req *godo.MicroVMCheckpointCreateRequest) (*MicroVMCheckpoint, error) {
	cp, _, err := s.client.MicroVMs.CreateCheckpoint(context.TODO(), microVMID, req)
	if err != nil {
		return nil, err
	}
	return &MicroVMCheckpoint{MicroVMCheckpoint: cp}, nil
}

func (s *microVMsService) GetCheckpoint(id string) (*MicroVMCheckpoint, error) {
	cp, _, err := s.client.MicroVMs.GetCheckpoint(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &MicroVMCheckpoint{MicroVMCheckpoint: cp}, nil
}

func (s *microVMsService) DeleteCheckpoint(id string) error {
	_, err := s.client.MicroVMs.DeleteCheckpoint(context.TODO(), id)
	return err
}

func (s *microVMsService) GetCreateOptions() (*godo.MicroVMCreateOptions, error) {
	opts, _, err := s.client.MicroVMs.GetCreateOptions(context.TODO())
	return opts, err
}

func (s *microVMsService) Exec(id string, req *godo.MicroVMExecRequest) (*godo.MicroVMExecResult, error) {
	result, _, err := s.client.MicroVMs.Exec(context.TODO(), id, req)
	return result, err
}

func (s *microVMsService) ConsoleURL(id string, opt *godo.MicroVMConsoleOptions) (string, error) {
	return s.client.MicroVMs.ConsoleURL(id, opt)
}
