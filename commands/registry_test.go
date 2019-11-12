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

package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testRegistryName = "container-registry"
	testRegistry     = do.Registry{Registry: &godo.Registry{Name: testRegistryName}}
)

func TestRegistryCommand(t *testing.T) {
	cmd := Registry()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "get", "delete", "login")
}

func TestRegistryCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		rcr := godo.RegistryCreateRequest{Name: testRegistryName}
		tm.registry.EXPECT().Create(&rcr).Return(&testRegistry, nil)
		config.Args = append(config.Args, testRegistryName)

		err := RunRegistryCreate(config)
		assert.NoError(t, err)
	})
}

func TestRegistryGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Get().Return(&testRegistry, nil)

		err := RunRegistryGet(config)
		assert.NoError(t, err)
	})
}

func TestRegistryDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.registry.EXPECT().Delete().Return(nil)

		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunRegistryDelete(config)
		assert.NoError(t, err)
	})
}
