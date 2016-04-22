package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	testDrive = do.Drive{
		Drive: &godo.Drive{
			ID:            uuid.New(),
			SizeGigaBytes: 100,
			Name:          "test-drive",
			Description:   "test description",
			Region:        &godo.Region{Slug: "atlantis"},
		},
	}
	testDriveList = []do.Drive{
		testDrive,
	}
)

func TestDriveCommand(t *testing.T) {
	cmd := Drive()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list")
}

func TestDrivesGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.drives.On("Get", "test-drive").Return(&testDrive, nil)

		config.Args = append(config.Args, "test-drive")

		err := RunDriveGet(config)
		assert.NoError(t, err)
	})
}

func TestDrivesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.drives.On("List").Return(testDriveList, nil)

		err := RunDriveList(config)
		assert.NoError(t, err)
	})
}

func TestDriveCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tcr := godo.DriveCreateRequest{
			Name:          "test-drive",
			SizeGibiBytes: 100,
			Region:        "atlantis",
			Description:   "test description",
		}
		tm.drives.On("CreateDrive", &tcr).Return(&testDrive, nil)

		config.Args = append(config.Args, "test-drive")

		config.Doit.Set(config.NS, doctl.ArgDriveRegion, "atlantis")
		config.Doit.Set(config.NS, doctl.ArgDriveSize, "100GiB")
		config.Doit.Set(config.NS, doctl.ArgDriveDesc, "test description")

		err := RunDriveCreate(config)
		assert.NoError(t, err)
	})
}

func TestDrivesDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.drives.On("DeleteDrive", "test-drive").Return(nil)

		config.Args = append(config.Args, "test-drive")

		err := RunDriveDelete(config)
		assert.NoError(t, err)
	})
}
