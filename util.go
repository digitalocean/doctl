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

package doctl

import (
	"context"

	"github.com/digitalocean/doctl/pkg/listen"
	"github.com/digitalocean/doctl/pkg/runner"
	"github.com/digitalocean/doctl/pkg/terminal"
)

// MockRunner is an implementation of Runner for mocking.
type MockRunner struct {
	Err error
}

var _ runner.Runner = &MockRunner{}

// Run mock runs things.
func (tr *MockRunner) Run() error {
	return tr.Err
}

// MockListener is an implementation of ListenerService for mocking.
type MockListener struct {
	Err error
}

var _ listen.ListenerService = &MockListener{}

// Listen mocks ListenerService.Listen
func (tr *MockListener) Listen(ctx context.Context) error {
	return tr.Err
}

// MockTerminal is an implementation of Terminal for mocking.
type MockTerminal struct {
	Err error
}

var _ terminal.Terminal = &MockTerminal{}

// ReadRawStdin mocks Terminal.ReadRawStdin
func (tr *MockTerminal) ReadRawStdin(ctx context.Context, stdinCh chan<- string) error {
	return tr.Err
}

// MonitorResizeEvents mocks Terminal.MonitorResizeEvents
func (tr *MockTerminal) MonitorResizeEvents(ctx context.Context, resizeEvents chan<- terminal.TerminalSize) error {
	return tr.Err
}
