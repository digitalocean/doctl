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
}

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
