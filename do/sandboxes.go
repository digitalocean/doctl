/*
Copyright 2026 The Doctl Authors All rights reserved.
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

// Sandbox wraps a godo Sandbox.
type Sandbox struct {
	*godo.Sandbox
}

// Sandboxes is a slice of Sandbox.
type Sandboxes []Sandbox

// SandboxesService is an interface for interacting with DigitalOcean's
// sandbox API.
type SandboxesService interface {
	List(state godo.SandboxState) (Sandboxes, error)
	Get(sandboxID string) (*Sandbox, error)
	Create(req *godo.SandboxCreateRequest) (*Sandbox, error)
	Delete(sandboxID string) error
	Pause(sandboxID string) (*Sandbox, error)
	Resume(sandboxID string) (*Sandbox, error)

	ListCommands(sandboxID string) (SandboxCommands, error)
	StartCommand(sandboxID string, req *godo.SandboxCommandStartRequest) (int64, error)
	StreamCommand(ctx context.Context, sandboxID string, pid int64) (SandboxCommandStream, error)
	SignalCommand(sandboxID string, pid int64, signal godo.SandboxCommandSignal) error
	WriteCommandStdin(sandboxID string, pid int64, data []byte, eof bool) error

	ListTemplates() (SandboxTemplates, error)
	GetTemplate(templateID string) (*SandboxTemplate, error)
	CreateTemplate(req *godo.SandboxTemplateCreateRequest) (*SandboxTemplate, error)
	UpdateTemplate(templateID string, req *godo.SandboxTemplateUpdateRequest) (*SandboxTemplate, error)
	DeleteTemplate(templateID string) error
	ListTemplateBuilds(templateID string) (SandboxTemplateBuilds, error)
	GetTemplateBuild(templateID, buildID string) (*SandboxTemplateBuild, error)

	ListSnapshots(sandboxID string) (SandboxSnapshots, error)
	GetSnapshot(snapshotID string) (*SandboxSnapshot, error)
	CreateSnapshot(sandboxID string, req *godo.SandboxSnapshotCreateRequest) (*SandboxSnapshot, error)
	DeleteSnapshot(snapshotID string) error
}

// SandboxSnapshot wraps a godo SandboxSnapshot.
type SandboxSnapshot struct {
	*godo.SandboxSnapshot
}

// SandboxSnapshots is a slice of SandboxSnapshot.
type SandboxSnapshots []SandboxSnapshot

// SandboxTemplate wraps a godo SandboxTemplate.
type SandboxTemplate struct {
	*godo.SandboxTemplate
}

// SandboxTemplates is a slice of SandboxTemplate.
type SandboxTemplates []SandboxTemplate

// SandboxTemplateBuild wraps a godo SandboxTemplateBuild.
type SandboxTemplateBuild struct {
	*godo.SandboxTemplateBuild
}

// SandboxTemplateBuilds is a slice of SandboxTemplateBuild.
type SandboxTemplateBuilds []SandboxTemplateBuild

// SandboxCommand wraps a godo SandboxCommand.
type SandboxCommand struct {
	*godo.SandboxCommand
}

// SandboxCommands is a slice of SandboxCommand.
type SandboxCommands []SandboxCommand

// SandboxCommandStream reads the events of a running command. Next returns
// io.EOF when the stream ends. Close it when done.
type SandboxCommandStream interface {
	Next() (*godo.SandboxCommandEvent, error)
	Close() error
}

type sandboxesService struct {
	client *godo.Client
}

var _ SandboxesService = &sandboxesService{}

// NewSandboxesService builds an instance of SandboxesService.
func NewSandboxesService(client *godo.Client) SandboxesService {
	return &sandboxesService{client: client}
}

// List returns every sandbox of the team, following the next-page token, or
// only those in state when it is set.
func (s *sandboxesService) List(state godo.SandboxState) (Sandboxes, error) {
	var out Sandboxes
	opt := &godo.SandboxListOptions{State: state}
	for {
		page, _, err := s.client.Sandboxes.List(context.TODO(), opt)
		if err != nil {
			return nil, err
		}
		for i := range page.Sandboxes {
			out = append(out, Sandbox{Sandbox: &page.Sandboxes[i]})
		}
		if page.NextPageToken == "" {
			return out, nil
		}
		opt.PageToken = page.NextPageToken
	}
}

func (s *sandboxesService) Get(sandboxID string) (*Sandbox, error) {
	sb, _, err := s.client.Sandboxes.Get(context.TODO(), sandboxID)
	if err != nil {
		return nil, err
	}
	return &Sandbox{Sandbox: sb}, nil
}

func (s *sandboxesService) Create(req *godo.SandboxCreateRequest) (*Sandbox, error) {
	sb, _, err := s.client.Sandboxes.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &Sandbox{Sandbox: sb}, nil
}

func (s *sandboxesService) Delete(sandboxID string) error {
	_, err := s.client.Sandboxes.Delete(context.TODO(), sandboxID)
	return err
}

func (s *sandboxesService) Pause(sandboxID string) (*Sandbox, error) {
	sb, _, err := s.client.Sandboxes.Pause(context.TODO(), sandboxID)
	if err != nil {
		return nil, err
	}
	return &Sandbox{Sandbox: sb}, nil
}

func (s *sandboxesService) Resume(sandboxID string) (*Sandbox, error) {
	sb, _, err := s.client.Sandboxes.Resume(context.TODO(), sandboxID)
	if err != nil {
		return nil, err
	}
	return &Sandbox{Sandbox: sb}, nil
}

func (s *sandboxesService) ListCommands(sandboxID string) (SandboxCommands, error) {
	list, _, err := s.client.Sandboxes.ListCommands(context.TODO(), sandboxID)
	if err != nil {
		return nil, err
	}
	out := make(SandboxCommands, len(list))
	for i := range list {
		out[i] = SandboxCommand{SandboxCommand: &list[i]}
	}
	return out, nil
}

func (s *sandboxesService) StartCommand(sandboxID string, req *godo.SandboxCommandStartRequest) (int64, error) {
	pid, _, err := s.client.Sandboxes.StartCommand(context.TODO(), sandboxID, req)
	return pid, err
}

func (s *sandboxesService) StreamCommand(ctx context.Context, sandboxID string, pid int64) (SandboxCommandStream, error) {
	stream, _, err := s.client.Sandboxes.StreamCommand(ctx, sandboxID, pid)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (s *sandboxesService) SignalCommand(sandboxID string, pid int64, signal godo.SandboxCommandSignal) error {
	_, err := s.client.Sandboxes.SignalCommand(context.TODO(), sandboxID, pid, signal)
	return err
}

func (s *sandboxesService) WriteCommandStdin(sandboxID string, pid int64, data []byte, eof bool) error {
	_, err := s.client.Sandboxes.WriteCommandStdin(context.TODO(), sandboxID, pid,
		&godo.SandboxCommandStdinRequest{Data: data, EOF: eof})
	return err
}

// ListTemplates returns every custom template of the team, following the
// next-page token.
func (s *sandboxesService) ListTemplates() (SandboxTemplates, error) {
	var out SandboxTemplates
	opt := &godo.SandboxTemplateListOptions{}
	for {
		page, _, err := s.client.Sandboxes.ListTemplates(context.TODO(), opt)
		if err != nil {
			return nil, err
		}
		for i := range page.Templates {
			out = append(out, SandboxTemplate{SandboxTemplate: &page.Templates[i]})
		}
		if page.NextPageToken == "" {
			return out, nil
		}
		opt.PageToken = page.NextPageToken
	}
}

func (s *sandboxesService) GetTemplate(templateID string) (*SandboxTemplate, error) {
	t, _, err := s.client.Sandboxes.GetTemplate(context.TODO(), templateID)
	if err != nil {
		return nil, err
	}
	return &SandboxTemplate{SandboxTemplate: t}, nil
}

func (s *sandboxesService) CreateTemplate(req *godo.SandboxTemplateCreateRequest) (*SandboxTemplate, error) {
	t, _, err := s.client.Sandboxes.CreateTemplate(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &SandboxTemplate{SandboxTemplate: t}, nil
}

func (s *sandboxesService) UpdateTemplate(templateID string, req *godo.SandboxTemplateUpdateRequest) (*SandboxTemplate, error) {
	t, _, err := s.client.Sandboxes.UpdateTemplate(context.TODO(), templateID, req)
	if err != nil {
		return nil, err
	}
	return &SandboxTemplate{SandboxTemplate: t}, nil
}

func (s *sandboxesService) DeleteTemplate(templateID string) error {
	_, err := s.client.Sandboxes.DeleteTemplate(context.TODO(), templateID)
	return err
}

// ListTemplateBuilds returns every build of a template, following the
// next-page token.
func (s *sandboxesService) ListTemplateBuilds(templateID string) (SandboxTemplateBuilds, error) {
	var out SandboxTemplateBuilds
	opt := &godo.SandboxTemplateListOptions{}
	for {
		page, _, err := s.client.Sandboxes.ListTemplateBuilds(context.TODO(), templateID, opt)
		if err != nil {
			return nil, err
		}
		for i := range page.Builds {
			out = append(out, SandboxTemplateBuild{SandboxTemplateBuild: &page.Builds[i]})
		}
		if page.NextPageToken == "" {
			return out, nil
		}
		opt.PageToken = page.NextPageToken
	}
}

func (s *sandboxesService) GetTemplateBuild(templateID, buildID string) (*SandboxTemplateBuild, error) {
	b, _, err := s.client.Sandboxes.GetTemplateBuild(context.TODO(), templateID, buildID)
	if err != nil {
		return nil, err
	}
	return &SandboxTemplateBuild{SandboxTemplateBuild: b}, nil
}

// ListSnapshots returns every snapshot of the team, newest first, or only the
// snapshots of sandboxID when it is not empty. It follows the next-page token.
func (s *sandboxesService) ListSnapshots(sandboxID string) (SandboxSnapshots, error) {
	var out SandboxSnapshots
	opt := &godo.SandboxSnapshotListOptions{SandboxID: sandboxID}
	for {
		page, _, err := s.client.Sandboxes.ListSnapshots(context.TODO(), opt)
		if err != nil {
			return nil, err
		}
		for i := range page.Snapshots {
			out = append(out, SandboxSnapshot{SandboxSnapshot: &page.Snapshots[i]})
		}
		if page.NextPageToken == "" {
			return out, nil
		}
		opt.PageToken = page.NextPageToken
	}
}

func (s *sandboxesService) GetSnapshot(snapshotID string) (*SandboxSnapshot, error) {
	snap, _, err := s.client.Sandboxes.GetSnapshot(context.TODO(), snapshotID)
	if err != nil {
		return nil, err
	}
	return &SandboxSnapshot{SandboxSnapshot: snap}, nil
}

func (s *sandboxesService) CreateSnapshot(sandboxID string, req *godo.SandboxSnapshotCreateRequest) (*SandboxSnapshot, error) {
	snap, _, err := s.client.Sandboxes.CreateSnapshot(context.TODO(), sandboxID, req)
	if err != nil {
		return nil, err
	}
	return &SandboxSnapshot{SandboxSnapshot: snap}, nil
}

func (s *sandboxesService) DeleteSnapshot(snapshotID string) error {
	_, err := s.client.Sandboxes.DeleteSnapshot(context.TODO(), snapshotID)
	return err
}
