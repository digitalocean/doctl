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

func TestReservedIPv6Commands(t *testing.T) {
	cmd := ReservedIPv6()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list")
}

func TestReservedIPv6sList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPv6s.EXPECT().List().Return(testReservedIPv6List, nil)

		RunReservedIPv6List(config)
	})
}

func TestReservedIPv6sGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPv6s.EXPECT().Get("5a11:a:b0a7").Return(&testReservedIPv6, nil)

		config.Args = append(config.Args, "5a11:a:b0a7")

		RunReservedIPv6Get(config)
	})
}

func TestReservedIPv6sCreate_Region(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ficr := &godo.ReservedIPV6CreateRequest{Region: "dev0"}
		tm.reservedIPv6s.EXPECT().Create(ficr).Return(&testReservedIPv6, nil)

		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "dev0")

		err := RunReservedIPv6Create(config)
		assert.NoError(t, err)
	})
}

func TestReservedIPv6sCreate_fail_with_no_args(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunReservedIPv6Create(config)
		assert.Error(t, err)
	})
}

func TestReservedIPv6sDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPv6s.EXPECT().Delete("5a11:a:b0a7").Return(nil)

		config.Args = append(config.Args, "5a11:a:b0a7")

		config.Doit.Set(config.NS, doctl.ArgForce, true)

		RunReservedIPv6Delete(config)
	})
}

func TestReservedIPv6ActionsAssign(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPv6s.EXPECT().Assign("5a11:a:b0a7", 2).Return(&testAction, nil)

		config.Args = append(config.Args, "5a11:a:b0a7", "2")

		err := RunReservedIPv6sAssign(config)
		assert.NoError(t, err)
	})
}

func TestReservedIPv6ActionsUnassign(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPv6s.EXPECT().Unassign("5a11:a:b0a7").Return(&testAction, nil)

		config.Args = append(config.Args, "5a11:a:b0a7")

		err := RunReservedIPv6sUnassign(config)
		assert.NoError(t, err)
	})
}
