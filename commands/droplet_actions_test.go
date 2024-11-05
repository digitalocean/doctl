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
	"encoding/json"
	"os"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDropletActionCommand(t *testing.T) {
	cmd := DropletAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "change-kernel", "change_backup_policy", "enable-backups", "disable-backups", "enable-ipv6", "enable-private-networking", "get", "power-cycle", "power-off", "power-on", "password-reset", "reboot", "rebuild", "rename", "resize", "restore", "shutdown", "snapshot")
}

func TestDropletActionsChangeKernel(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().ChangeKernel(1, 2).Return(&testAction, nil)

		config.Doit.Set(config.NS, doctl.ArgKernelID, 2)
		config.Args = append(config.Args, "1")

		err := RunDropletActionChangeKernel(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionChangeKernel(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsEnableBackups(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().EnableBackups(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionEnableBackups(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionEnableBackups(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsDisableBackups(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().DisableBackups(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionDisableBackups(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionDisableBackups(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletChangeBackupPolicy(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dropletPolicy := godo.DropletBackupPolicyRequest{
			Plan:    "weekly",
			Weekday: "SAT",
			Hour:    godo.PtrTo(0),
		}

		policyFile, err := os.CreateTemp(t.TempDir(), "policy-cfg")
		require.NoError(t, err)
		defer policyFile.Close()

		err = json.NewEncoder(policyFile).Encode(&dropletPolicy)
		require.NoError(t, err)

		policyReq := &godo.DropletBackupPolicyRequest{
			Plan:    dropletPolicy.Plan,
			Weekday: dropletPolicy.Weekday,
			Hour:    dropletPolicy.Hour,
		}

		tm.dropletActions.EXPECT().ChangeBackupPolicy(1, policyReq).Times(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")
		config.Doit.Set(config.NS, doctl.ArgDropletBackupPolicy, policyFile.Name())

		err = RunDropletActionChangeBackupPolicy(config)
		require.NoError(t, err)
	})
}

func TestDropletActionsEnableIPv6(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().EnableIPv6(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionEnableIPv6(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionEnableIPv6(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsEnablePrivateNetworking(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().EnablePrivateNetworking(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionEnablePrivateNetworking(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionEnablePrivateNetworking(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().Get(1, 2).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgActionID, 2)

		err := RunDropletActionGet(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionGet(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsPasswordReset(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().PasswordReset(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionPasswordReset(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionPasswordReset(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsPowerCycle(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().PowerCycle(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionPowerCycle(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionPowerCycle(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsPowerOff(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().PowerOff(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionPowerOff(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionPowerOff(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsPowerOn(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().PowerOn(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionPowerOn(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionPowerOn(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsReboot(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().Reboot(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionReboot(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionReboot(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsRebuildByImageID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().RebuildByImageID(1, 2).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgImage, "2")

		err := RunDropletActionRebuild(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionRebuild(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsRebuildByImageSlug(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().RebuildByImageSlug(1, "slug").Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgImage, "slug")

		err := RunDropletActionRebuild(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionRebuild(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsRename(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().Rename(1, "name").Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgDropletName, "name")

		err := RunDropletActionRename(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionRename(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsResize(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().Resize(1, "1gb", true).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "1gb")
		config.Doit.Set(config.NS, doctl.ArgResizeDisk, true)

		err := RunDropletActionResize(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionResize(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsRestore(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().Restore(1, 2).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgImageID, 2)

		err := RunDropletActionRestore(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionRestore(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsShutdown(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().Shutdown(1).Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActionShutdown(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionShutdown(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}

func TestDropletActionsSnapshot(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dropletActions.EXPECT().Snapshot(1, "name").Return(&testAction, nil)

		config.Args = append(config.Args, "1")

		config.Doit.Set(config.NS, doctl.ArgSnapshotName, "name")

		err := RunDropletActionSnapshot(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-test-id")

		err := RunDropletActionSnapshot(config)
		assert.EqualError(t, err, `expected <droplet-id> to be a positive integer, got "my-test-id"`)
	})
}
