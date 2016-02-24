package commands

import (
	"testing"

	"github.com/bryanl/doit"
	domocks "github.com/bryanl/doit/do/mocks"
	"github.com/stretchr/testify/assert"
)

func TestDropletActionCommand(t *testing.T) {
	cmd := DropletAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "change-kernel", "disable-backups", "enable-ipv6", "enable-private-networking", "get", "power-cycle", "power-off", "power-on", "power-reset", "reboot", "rebuild", "rename", "resize", "restore", "shutdown", "snapshot", "upgrade")
}

func TestDropletActionsChangeKernel(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("ChangeKernel", 1, 2).Return(&testAction, nil)

		config.doitConfig.Set(config.ns, doit.ArgKernelID, 2)
		config.args = append(config.args, "1")

		RunDropletActionChangeKernel(config)
	})
}
func TestDropletActionsDisableBackups(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("DisableBackups", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		RunDropletActionDisableBackups(config)
	})

}
func TestDropletActionsEnableIPv6(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("EnableIPv6", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		RunDropletActionEnableIPv6(config)
	})
}

func TestDropletActionsEnablePrivateNetworking(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("EnablePrivateNetworking", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		RunDropletActionEnablePrivateNetworking(config)
	})
}
func TestDropletActionsGet(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("Get", 1, 2).Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgActionID, 2)

		RunDropletActionGet(config)
	})
}

func TestDropletActionsPasswordReset(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("PasswordReset", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		RunDropletActionPasswordReset(config)
	})
}

func TestDropletActionsPowerCycle(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("PowerCycle", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		RunDropletActionPowerCycle(config)
	})

}
func TestDropletActionsPowerOff(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("PowerOff", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		RunDropletActionPowerOff(config)
	})
}
func TestDropletActionsPowerOn(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("PowerOn", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		RunDropletActionPowerOn(config)
	})

}
func TestDropletActionsReboot(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("Reboot", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		RunDropletActionReboot(config)
	})
}

func TestDropletActionsRebuildByImageID(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("RebuildByImageID", 1, 2).Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgImage, "2")

		RunDropletActionRebuild(config)
	})
}

func TestDropletActionsRebuildByImageSlug(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("RebuildByImageSlug", 1, "slug").Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgImage, "slug")

		RunDropletActionRebuild(config)
	})

}
func TestDropletActionsRename(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("Rename", 1, "name").Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgDropletName, "name")

		RunDropletActionRename(config)
	})
}

func TestDropletActionsResize(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("Resize", 1, "1gb", true).Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgSizeSlug, "1gb")
		config.doitConfig.Set(config.ns, doit.ArgResizeDisk, true)

		RunDropletActionResize(config)
	})
}

func TestDropletActionsRestore(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("Restore", 1, 2).Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgImageID, 2)

		RunDropletActionRestore(config)
	})
}

func TestDropletActionsShutdown(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("Shutdown", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		RunDropletActionShutdown(config)
	})
}

func TestDropletActionsSnapshot(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("Snapshot", 1, "name").Return(&testAction, nil)

		config.args = append(config.args, "1")

		config.doitConfig.Set(config.ns, doit.ArgSnapshotName, "name")

		RunDropletActionSnapshot(config)
	})
}

func TestDropletActionsUpgrade(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		das := &domocks.DropletActionsService{}
		config.das = das

		das.On("Upgrade", 1).Return(&testAction, nil)

		config.args = append(config.args, "1")

		RunDropletActionUpgrade(config)
	})
}
