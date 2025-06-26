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
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestBYOIPPrefixCommands(t *testing.T) {
	cmd := BYOIPPrefix()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list", "resource")
}

func TestBYOIPPrefixList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.byoipPrefixes.EXPECT().List().Return(testBYOIPPrefixList, nil)

		RunBYOIPPrefixList(config)
	})
}

func TestBYOIPPrefixesGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.byoipPrefixes.EXPECT().Get("f00b8f02-11da-424b-b658-ad8cebfc5a56").Return(&testBYOIPPrefix, nil)

		config.Args = append(config.Args, "f00b8f02-11da-424b-b658-ad8cebfc5a56")

		assert.NoError(t, RunBYOIPPrefixGet(config))
	})
}

func TestBYOIPPrefixDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.byoipPrefixes.EXPECT().Delete("f00b8f02-11da-424b-b658-ad8cebfc5a56").Return(nil)

		config.Args = append(config.Args, "f00b8f02-11da-424b-b658-ad8cebfc5a56")

		config.Doit.Set(config.NS, doctl.ArgForce, true)

		assert.NoError(t, RunBYOIPPrefixDelete(config))
	})
}

func TestBYOIPPrefixCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		bpcr := &godo.BYOIPPrefixCreateReq{Region: "nyc1", Prefix: "10.1.1.1/24", Signature: "signature"}
		tm.byoipPrefixes.EXPECT().Create(bpcr).Return(testBYOIPPrefixCreate, nil)

		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, doctl.ArgPrefix, "10.1.1.1/24")
		config.Doit.Set(config.NS, doctl.ArgSignature, "signature")

		assert.NoError(t, RunBYOIPPrefixCreate(config))
	})
}

func TestBYOIPPrefixResourcesGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.byoipPrefixes.EXPECT().GetPrefixResources("f00b8f02-11da-424b-b658-ad8cebfc5a56").Return(testBYOIPPrefixGetResources, nil)

		config.Args = append(config.Args, "f00b8f02-11da-424b-b658-ad8cebfc5a56")

		RunBYOIPPrefixResourcesGet(config)
	})
}
