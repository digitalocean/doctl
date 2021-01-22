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
	"fmt"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/stretchr/testify/assert"
)

func TestVolumeActionCommand(t *testing.T) {
	cmd := VolumeAction()
	assert.NotNil(t, cmd)

	assertCommandNames(t, cmd, "attach", "detach", "detach-by-droplet-id", "resize", "get", "list")
}
func TestVolumeActionsAttach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumeActions.EXPECT().Attach(testVolume.ID, testDroplet.ID).Return(&testAction, nil)
		config.Args = append(config.Args, testVolume.ID)
		config.Args = append(config.Args, fmt.Sprintf("%d", testDroplet.ID))

		err := RunVolumeAttach(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumeActions.EXPECT().Attach(testVolume.ID, testDroplet.ID).Return(&testAction, nil)
		tm.actions.EXPECT().Get(1).Return(&testAction, nil)

		config.Args = append(config.Args, testVolume.ID)
		config.Args = append(config.Args, fmt.Sprintf("%d", testDroplet.ID))
		config.Doit.Set(config.NS, doctl.ArgCommandWait, true)

		err := RunVolumeAttach(config)
		assert.NoError(t, err)
	})
}

func TestVolumeDetach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumeActions.EXPECT().Detach(testVolume.ID, testDroplet.ID).Return(&testAction, nil)
		config.Args = append(config.Args, testVolume.ID)
		config.Args = append(config.Args, fmt.Sprintf("%d", testDroplet.ID))

		err := RunVolumeDetach(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumeActions.EXPECT().Detach(testVolume.ID, testDroplet.ID).Return(&testAction, nil)
		tm.actions.EXPECT().Get(1).Return(&testAction, nil)

		config.Args = append(config.Args, testVolume.ID)
		config.Args = append(config.Args, fmt.Sprintf("%d", testDroplet.ID))
		config.Doit.Set(config.NS, doctl.ArgCommandWait, true)

		err := RunVolumeDetach(config)
		assert.NoError(t, err)
	})
}

func TestVolumeResize(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumeActions.EXPECT().Resize(testVolume.ID, 150, "dev0").Return(&testAction, nil)
		config.Args = append(config.Args, testVolume.ID)

		config.Doit.Set(config.NS, doctl.ArgSizeSlug, 150)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "dev0")

		err := RunVolumeResize(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumeActions.EXPECT().Resize(testVolume.ID, 150, "dev0").Return(&testAction, nil)
		tm.actions.EXPECT().Get(1).Return(&testAction, nil)

		config.Args = append(config.Args, testVolume.ID)

		config.Doit.Set(config.NS, doctl.ArgSizeSlug, 150)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "dev0")
		config.Doit.Set(config.NS, doctl.ArgCommandWait, true)

		err := RunVolumeResize(config)
		assert.NoError(t, err)
	})
}
