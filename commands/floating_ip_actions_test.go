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

func TestFloatingIPActionCommand(t *testing.T) {
	cmd := FloatingIPAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "assign", "get", "unassign")
}

func TestFloatingIPActionsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.floatingIPActions.EXPECT().Get("127.0.0.1", 2).Return(&testAction, nil)

		config.Args = append(config.Args, "127.0.0.1", "2")

		err := RunFloatingIPActionsGet(config)
		assert.NoError(t, err)
	})

}

func TestFloatingIPActionsAssign(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.floatingIPActions.EXPECT().Assign("127.0.0.1", 2).Return(&testAction, nil)

		config.Args = append(config.Args, "127.0.0.1", "2")

		err := RunFloatingIPActionsAssign(config)
		assert.NoError(t, err)
	})
}

func TestFloatingIPActionsUnassign(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.floatingIPActions.EXPECT().Unassign("127.0.0.1").Return(&testAction, nil)

		config.Args = append(config.Args, "127.0.0.1")

		err := RunFloatingIPActionsUnassign(config)
		assert.NoError(t, err)
	})
}
