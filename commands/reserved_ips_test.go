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

func TestReservedIPCommands(t *testing.T) {
	cmd := ReservedIP()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list")
}

func TestReservedIPsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPs.EXPECT().List().Return(testReservedIPList, nil)

		RunReservedIPList(config)
	})
}

func TestReservedIPsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPs.EXPECT().Get("127.0.0.1").Return(&testReservedIP, nil)

		config.Args = append(config.Args, "127.0.0.1")

		RunReservedIPGet(config)
	})
}

func TestReservedIPsCreate_Droplet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ficr := &godo.ReservedIPCreateRequest{DropletID: 1}
		tm.reservedIPs.EXPECT().Create(ficr).Return(&testReservedIP, nil)

		config.Doit.Set(config.NS, doctl.ArgDropletID, 1)

		err := RunReservedIPCreate(config)
		assert.NoError(t, err)
	})
}

func TestReservedIPsCreate_Region(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ficr := &godo.ReservedIPCreateRequest{Region: "dev0"}
		tm.reservedIPs.EXPECT().Create(ficr).Return(&testReservedIP, nil)

		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "dev0")

		err := RunReservedIPCreate(config)
		assert.NoError(t, err)
	})
}

func TestReservedIPsCreate_fail_with_no_args(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunReservedIPCreate(config)
		assert.Error(t, err)
	})
}

func TestReservedIPsCreate_fail_with_both_args(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgDropletID, 1)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "dev0")

		err := RunReservedIPCreate(config)
		assert.Error(t, err)
	})
}

func TestReservedIPsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPs.EXPECT().Delete("127.0.0.1").Return(nil)

		config.Args = append(config.Args, "127.0.0.1")

		config.Doit.Set(config.NS, doctl.ArgForce, true)

		RunReservedIPDelete(config)
	})
}
