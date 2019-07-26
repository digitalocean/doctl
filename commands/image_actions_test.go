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

func TestImageActionCommand(t *testing.T) {
	cmd := ImageAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "transfer")
}

func TestImageActionsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.imageActions.EXPECT().Get(1, 2).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgActionID, 2)

		err := RunImageActionsGet(config)
		assert.NoError(t, err)
	})

}

func TestImageActionsTransfer(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ar := &godo.ActionRequest{"type": "transfer", "region": "dev0"}
		tm.imageActions.EXPECT().Transfer(1, ar).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "dev0")

		err := RunImageActionsTransfer(config)
		assert.NoError(t, err)
	})
}
