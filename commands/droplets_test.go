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
	"fmt"
	"strconv"
	"testing"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
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
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Actions", 1).Return(testActionList, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActions(config)
		assert.NoError(t, err)
	})
}

func TestDropletBackupList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Backups", 1).Return(testImageList, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletBackups(config)
		assert.NoError(t, err)
	})
}

func TestDropletCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dcr := &godo.DropletCreateRequest{Name: "droplet", Region: "dev0", Size: "1gb", Image: godo.DropletCreateImage{ID: 0, Slug: "image"}, SSHKeys: []godo.DropletCreateSSHKey{}, Backups: false, IPv6: false, PrivateNetworking: false, UserData: "#cloud-config"}
		tm.droplets.On("Create", dcr, false).Return(&testDroplet, nil)

		config.Args = append(config.Args, "droplet")

		config.Doit.Set(config.NS, doit.ArgRegionSlug, "dev0")
		config.Doit.Set(config.NS, doit.ArgSizeSlug, "1gb")
		config.Doit.Set(config.NS, doit.ArgImage, "image")
		config.Doit.Set(config.NS, doit.ArgUserData, "#cloud-config")

		err := RunDropletCreate(config)
		assert.NoError(t, err)
	})
}

func TestDropletCreateUserDataFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dcr := &godo.DropletCreateRequest{Name: "droplet", Region: "dev0", Size: "1gb", Image: godo.DropletCreateImage{ID: 0, Slug: "image"}, SSHKeys: []godo.DropletCreateSSHKey{}, Backups: false, IPv6: false, PrivateNetworking: false, UserData: "#cloud-config\n\ncoreos:\n  etcd2:\n    # generate a new token for each unique cluster from https://discovery.etcd.io/new?size=5\n    # specify the initial size of your cluster with ?size=X\n    discovery: https://discovery.etcd.io/<token>\n    # multi-region and multi-cloud deployments need to use $public_ipv4\n    advertise-client-urls: http://$private_ipv4:2379,http://$private_ipv4:4001\n    initial-advertise-peer-urls: http://$private_ipv4:2380\n    # listen on both the official ports and the legacy ports\n    # legacy ports can be omitted if your application doesn't depend on them\n    listen-client-urls: http://0.0.0.0:2379,http://0.0.0.0:4001\n    listen-peer-urls: http://$private_ipv4:2380\n  units:\n    - name: etcd2.service\n      command: start\n    - name: fleet.service\n      command: start\n"}
		tm.droplets.On("Create", dcr, false).Return(&testDroplet, nil)

		config.Args = append(config.Args, "droplet")

		config.Doit.Set(config.NS, doit.ArgRegionSlug, "dev0")
		config.Doit.Set(config.NS, doit.ArgSizeSlug, "1gb")
		config.Doit.Set(config.NS, doit.ArgImage, "image")
		config.Doit.Set(config.NS, doit.ArgUserDataFile, "../testdata/cloud-config.yml")

		err := RunDropletCreate(config)
		assert.NoError(t, err)
	})
}

func TestDropletDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Delete", 1).Return(nil)

		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID))

		err := RunDropletDelete(config)
		assert.NoError(t, err)
	})
}

func TestDropletDeleteByName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("List").Return(testDropletList, nil)
		tm.droplets.On("Delete", 1).Return(nil)

		config.Args = append(config.Args, testDroplet.Name)

		err := RunDropletDelete(config)
		assert.NoError(t, err)
	})
}

func TestDropletGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Get", testDroplet.ID).Return(&testDroplet, nil)

		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID))

		err := RunDropletGet(config)
		assert.NoError(t, err)
	})
}

func TestDropletKernelList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Kernels", testDroplet.ID).Return(testKernelList, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletKernels(config)
		assert.NoError(t, err)
	})
}

func TestDropletNeighbors(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Neighbors", testDroplet.ID).Return(testDropletList, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletNeighbors(config)
		assert.NoError(t, err)
	})
}

func TestDropletSnapshotList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Snapshots", testDroplet.ID).Return(testImageList, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletSnapshots(config)
		assert.NoError(t, err)
	})
}

func TestDropletsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("List").Return(testDropletList, nil)

		err := RunDropletList(config)
		assert.NoError(t, err)
	})
}

func Test_extractSSHKey(t *testing.T) {
	cases := []struct {
		in       string
		expected []godo.DropletCreateSSHKey
	}{
		{
			in:       "1",
			expected: []godo.DropletCreateSSHKey{{ID: 1}},
		},
		{
			in:       "fingerprint",
			expected: []godo.DropletCreateSSHKey{{Fingerprint: "fingerprint"}},
		},
		{
			in:       "1,2",
			expected: []godo.DropletCreateSSHKey{{ID: 1}, {ID: 2}},
		},
		{
			in:       "1,fingerprint",
			expected: []godo.DropletCreateSSHKey{{ID: 1}, {Fingerprint: "fingerprint"}},
		},
	}

	for _, c := range cases {
		got := extractSSHKeys([]string{fmt.Sprintf("[%s]", c.in)})
		assert.Equal(t, c.expected, got)
	}
}
