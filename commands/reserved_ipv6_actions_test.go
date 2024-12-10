/*
Copyright 2024 The Doctl Authors All rights reserved.
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

	"github.com/stretchr/testify/assert"
)

func TestReservedIPv6ActionCommand(t *testing.T) {
	cmd := ReservedIPv6Action()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "assign", "unassign")
}

func TestReservedIPv6ActionsAssign(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPv6Actions.EXPECT().Assign("5a11:a:b0a7", 2).Return(&testAction, nil)

		config.Args = append(config.Args, "5a11:a:b0a7", "2")

		err := RunReservedIPv6ActionsAssign(config)
		assert.NoError(t, err)
	})
}

func TestReservedIPv6ActionsUnassign(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPv6Actions.EXPECT().Unassign("5a11:a:b0a7").Return(&testAction, nil)

		config.Args = append(config.Args, "5a11:a:b0a7")

		err := RunReservedIPv6ActionsUnassign(config)
		assert.NoError(t, err)
	})
}
