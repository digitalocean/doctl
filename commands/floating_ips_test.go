package commands

import (
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestFloatingIPCommands(t *testing.T) {
	cmd := FloatingIP()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list")
}

func TestFloatingIPsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.floatingIPs.On("List").Return(testFloatingIPList, nil)

		RunFloatingIPList(config)
	})
}

func TestFloatingIPsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.floatingIPs.On("Get", "127.0.0.1").Return(&testFloatingIP, nil)

		config.Args = append(config.Args, "127.0.0.1")

		RunFloatingIPGet(config)
	})
}

func TestFloatingIPsCreate_Droplet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ficr := &godo.FloatingIPCreateRequest{DropletID: 1}
		tm.floatingIPs.On("Create", ficr).Return(&testFloatingIP, nil)

		config.Doit.Set(config.NS, doit.ArgDropletID, 1)

		err := RunFloatingIPCreate(config)
		assert.NoError(t, err)
	})
}

func TestFloatingIPsCreate_Region(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ficr := &godo.FloatingIPCreateRequest{Region: "dev0"}
		tm.floatingIPs.On("Create", ficr).Return(&testFloatingIP, nil)

		config.Doit.Set(config.NS, doit.ArgRegionSlug, "dev0")

		err := RunFloatingIPCreate(config)
		assert.NoError(t, err)
	})
}

func TestFloatingIPsCreate_fail_with_no_args(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunFloatingIPCreate(config)
		assert.Error(t, err)
	})
}

func TestFloatingIPsCreate_fail_with_both_args(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doit.ArgDropletID, 1)
		config.Doit.Set(config.NS, doit.ArgRegionSlug, "dev0")

		err := RunFloatingIPCreate(config)
		assert.Error(t, err)
	})
}

func TestFloatingIPsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.floatingIPs.On("Delete", "127.0.0.1").Return(nil)

		config.Args = append(config.Args, "127.0.0.1")

		RunFloatingIPDelete(config)
	})
}
