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
	"github.com/digitalocean/doctl/pkg/listen"
	"github.com/digitalocean/doctl/pkg/runner"
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

// Start mocks ListenerService.Start
func (tr *MockListener) Start() error {
	return tr.Err
}

// Stop mocks ListenerService.Stop
func (tr *MockListener) Stop() {
	return
}
