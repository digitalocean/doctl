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

	"github.com/stretchr/testify/assert"
)

func TestReservedIPActionCommand(t *testing.T) {
	cmd := ReservedIPAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "assign", "get", "unassign")
}

func TestReservedIPActionsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPActions.EXPECT().Get("127.0.0.1", 2).Return(&testAction, nil)

		config.Args = append(config.Args, "127.0.0.1", "2")

		err := RunReservedIPActionsGet(config)
		assert.NoError(t, err)
	})

}

func TestReservedIPActionsAssign(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPActions.EXPECT().Assign("127.0.0.1", 2).Return(&testAction, nil)

		config.Args = append(config.Args, "127.0.0.1", "2")

		err := RunReservedIPActionsAssign(config)
		assert.NoError(t, err)
	})
}

func TestReservedIPActionsUnassign(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.reservedIPActions.EXPECT().Unassign("127.0.0.1").Return(&testAction, nil)

		config.Args = append(config.Args, "127.0.0.1")

		err := RunReservedIPActionsUnassign(config)
		assert.NoError(t, err)
	})
}
