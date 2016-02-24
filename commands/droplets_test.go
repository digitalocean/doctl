package commands

import (
	"strconv"
	"testing"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	domocks "github.com/bryanl/doit/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testImage = do.Image{Image: &godo.Image{
		ID:      1,
		Slug:    "slug",
		Regions: []string{"test0"},
	}}
	testImageList = do.Images{testImage}
)

func TestDropletCommand(t *testing.T) {
	cmd := Droplet()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "actions", "backups", "create", "delete", "get", "kernels", "list", "neighbors", "snapshots")
}

func TestDropletActionList(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		ds.On("Actions", 1).Return(testActionList, nil)

		config.args = append(config.args, "1")

		err := RunDropletActions(config)
		assert.NoError(t, err)
	})
}

func TestDropletBackupList(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		ds.On("Backups", 1).Return(testImageList, nil)

		config.args = append(config.args, "1")

		err := RunDropletBackups(config)
		assert.NoError(t, err)
	})
}

func TestDropletCreate(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		dcr := &godo.DropletCreateRequest{Name: "droplet", Region: "dev0", Size: "1gb", Image: godo.DropletCreateImage{ID: 0, Slug: "image"}, SSHKeys: []godo.DropletCreateSSHKey{}, Backups: false, IPv6: false, PrivateNetworking: false, UserData: "#cloud-config"}
		ds.On("Create", dcr, false).Return(&testDroplet, nil)

		config.args = append(config.args, "droplet")

		config.doitConfig.Set(config.ns, doit.ArgRegionSlug, "dev0")
		config.doitConfig.Set(config.ns, doit.ArgSizeSlug, "1gb")
		config.doitConfig.Set(config.ns, doit.ArgImage, "image")
		config.doitConfig.Set(config.ns, doit.ArgUserData, "#cloud-config")

		err := RunDropletCreate(config)
		assert.NoError(t, err)
	})
}

func TestDropletCreateUserDataFile(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		dcr := &godo.DropletCreateRequest{Name: "droplet", Region: "dev0", Size: "1gb", Image: godo.DropletCreateImage{ID: 0, Slug: "image"}, SSHKeys: []godo.DropletCreateSSHKey{}, Backups: false, IPv6: false, PrivateNetworking: false, UserData: "#cloud-config\n\ncoreos:\n  etcd2:\n    # generate a new token for each unique cluster from https://discovery.etcd.io/new?size=5\n    # specify the initial size of your cluster with ?size=X\n    discovery: https://discovery.etcd.io/<token>\n    # multi-region and multi-cloud deployments need to use $public_ipv4\n    advertise-client-urls: http://$private_ipv4:2379,http://$private_ipv4:4001\n    initial-advertise-peer-urls: http://$private_ipv4:2380\n    # listen on both the official ports and the legacy ports\n    # legacy ports can be omitted if your application doesn't depend on them\n    listen-client-urls: http://0.0.0.0:2379,http://0.0.0.0:4001\n    listen-peer-urls: http://$private_ipv4:2380\n  units:\n    - name: etcd2.service\n      command: start\n    - name: fleet.service\n      command: start\n"}
		ds.On("Create", dcr, false).Return(&testDroplet, nil)

		config.args = append(config.args, "droplet")

		config.doitConfig.Set(config.ns, doit.ArgRegionSlug, "dev0")
		config.doitConfig.Set(config.ns, doit.ArgSizeSlug, "1gb")
		config.doitConfig.Set(config.ns, doit.ArgImage, "image")
		config.doitConfig.Set(config.ns, doit.ArgUserDataFile, "../testdata/cloud-config.yml")

		err := RunDropletCreate(config)
		assert.NoError(t, err)
	})
}

func TestDropletDelete(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		ds.On("Delete", 1).Return(nil)

		config.args = append(config.args, strconv.Itoa(testDroplet.ID))

		err := RunDropletDelete(config)
		assert.NoError(t, err)
	})
}

func TestDropletDeleteByName(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		ds.On("List").Return(testDropletList, nil)
		ds.On("Delete", 1).Return(nil)

		config.args = append(config.args, testDroplet.Name)

		err := RunDropletDelete(config)
		assert.NoError(t, err)
	})
}

func TestDropletGet(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		ds.On("Get", testDroplet.ID).Return(&testDroplet, nil)

		config.args = append(config.args, strconv.Itoa(testDroplet.ID))

		err := RunDropletGet(config)
		assert.NoError(t, err)
	})
}

func TestDropletKernelList(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		ds.On("Kernels", testDroplet.ID).Return(testKernelList, nil)

		config.args = append(config.args, "1")

		err := RunDropletKernels(config)
		assert.NoError(t, err)
	})
}

func TestDropletNeighbors(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		ds.On("Neighbors", testDroplet.ID).Return(testDropletList, nil)

		config.args = append(config.args, "1")

		err := RunDropletNeighbors(config)
		assert.NoError(t, err)
	})
}

func TestDropletSnapshotList(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		ds.On("Snapshots", testDroplet.ID).Return(testImageList, nil)

		config.args = append(config.args, "1")

		err := RunDropletSnapshots(config)
		assert.NoError(t, err)
	})
}

func TestDropletsList(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		ds := &domocks.DropletsService{}
		config.ds = ds

		ds.On("List").Return(testDropletList, nil)

		err := RunDropletList(config)
		assert.NoError(t, err)
	})
}
