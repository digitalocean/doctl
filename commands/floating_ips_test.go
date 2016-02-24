package commands

import (
	"testing"

	"github.com/bryanl/doit"
	domocks "github.com/bryanl/doit/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestFloatingIPCommands(t *testing.T) {
	cmd := FloatingIP()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list")
}

func TestFloatingIPsList(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		fis := &domocks.FloatingIPsService{}
		config.fis = fis

		fis.On("List").Return(testFloatingIPList, nil)

		RunFloatingIPList(config)
	})
}

func TestFloatingIPsGet(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		fis := &domocks.FloatingIPsService{}
		config.fis = fis

		fis.On("Get", "127.0.0.1").Return(&testFloatingIP, nil)

		config.args = append(config.args, "127.0.0.1")

		RunFloatingIPGet(config)
	})
}

func TestFloatingIPsCreate(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		fis := &domocks.FloatingIPsService{}
		config.fis = fis

		ficr := &godo.FloatingIPCreateRequest{Region: "dev0", DropletID: 1}
		fis.On("Create", ficr).Return(&testFloatingIP, nil)

		config.doitConfig.Set(config.ns, doit.ArgRegionSlug, "dev0")
		config.doitConfig.Set(config.ns, doit.ArgDropletID, 1)

		RunFloatingIPCreate(config)
	})
}

func TestFloatingIPsDelete(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		fis := &domocks.FloatingIPsService{}
		config.fis = fis

		fis.On("Delete", "127.0.0.1").Return(nil)

		config.args = append(config.args, "127.0.0.1")

		RunFloatingIPDelete(config)
	})
}
