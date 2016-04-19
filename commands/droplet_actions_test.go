/*
Copyright 2016 The Doctl Authors All rights reserved.
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
	"github.com/stretchr/testify/assert"
)

func TestDropletActionCommand(t *testing.T) {
	cmd := DropletAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "change-kernel", "disable-backups", "enable-ipv6", "enable-private-networking", "get", "power-cycle", "power-off", "power-on", "power-reset", "reboot", "rebuild", "rename", "resize", "restore", "shutdown", "snapshot", "upgrade")
}

func TestDropletActionsChangeKernel(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("ChangeKernel", 1, 2).Return(&testAction, nil)

		config.Doit.Set(config.NS, doctl.ArgKernelID, 2)
		config.Args = append(config.Args, "1")

		err := RunDropletActionChangeKernel(config)
		assert.NoError(t, err)
	})
}
func TestDropletActionsDisableBackups(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("DisableBackups", 1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionDisableBackups(config)
		assert.NoError(t, err)
	})

}
func TestDropletActionsEnableIPv6(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("EnableIPv6", 1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionEnableIPv6(config)
		assert.NoError(t, err)
	})
}

func TestDropletActionsEnablePrivateNetworking(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("EnablePrivateNetworking", 1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionEnablePrivateNetworking(config)
		assert.NoError(t, err)
	})
}
func TestDropletActionsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("Get", 1, 2).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgActionID, 2)

		err := RunDropletActionGet(config)
		assert.NoError(t, err)
	})
}

func TestDropletActionsPasswordReset(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("PasswordReset", 1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionPasswordReset(config)
		assert.NoError(t, err)
	})
}

func TestDropletActionsPowerCycle(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("PowerCycle", 1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionPowerCycle(config)
		assert.NoError(t, err)
	})

}
func TestDropletActionsPowerOff(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("PowerOff", 1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionPowerOff(config)
		assert.NoError(t, err)
	})
}
func TestDropletActionsPowerOn(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("PowerOn", 1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionPowerOn(config)
		assert.NoError(t, err)
	})

}
func TestDropletActionsReboot(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("Reboot", 1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionReboot(config)
		assert.NoError(t, err)
	})
}

func TestDropletActionsRebuildByImageID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("RebuildByImageID", 1, 2).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgImage, "2")

		err := RunDropletActionRebuild(config)
		assert.NoError(t, err)

		assert.True(t, tm.dropletActions.AssertExpectations(t))
	})
}

func TestDropletActionsRebuildByImageSlug(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("RebuildByImageSlug", 1, "slug").Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgImage, "slug")

		err := RunDropletActionRebuild(config)
		assert.NoError(t, err)

		assert.True(t, tm.dropletActions.AssertExpectations(t))
	})

}
func TestDropletActionsRename(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("Rename", 1, "name").Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgDropletName, "name")

		err := RunDropletActionRename(config)
		assert.NoError(t, err)
	})
}

func TestDropletActionsResize(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("Resize", 1, "1gb", true).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "1gb")
		config.Doit.Set(config.NS, doctl.ArgResizeDisk, true)

		err := RunDropletActionResize(config)
		assert.NoError(t, err)
	})
}

func TestDropletActionsRestore(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("Restore", 1, 2).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgImageID, 2)

		err := RunDropletActionRestore(config)
		assert.NoError(t, err)
	})
}

func TestDropletActionsShutdown(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("Shutdown", 1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionShutdown(config)
		assert.NoError(t, err)
	})
}

func TestDropletActionsSnapshot(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("Snapshot", 1, "name").Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgSnapshotName, "name")

		err := RunDropletActionSnapshot(config)
		assert.NoError(t, err)
	})
}

func TestDropletActionsUpgrade(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.On("Upgrade", 1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionUpgrade(config)
		assert.NoError(t, err)
	})
}
