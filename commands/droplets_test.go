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
	"bytes"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	testImage = do.Image{Image: &godo.Image{
		ID:      1,
		Slug:    "slug",
		Regions: []string{"test0"},
	}}
	testImageSecondary = do.Image{Image: &godo.Image{
		ID:      2,
		Slug:    "slug-secondary",
		Regions: []string{"test0"},
	}}
	testImageList = do.Images{testImage, testImageSecondary}
)

func TestDropletCommand(t *testing.T) {
	cmd := Droplet()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "actions", "backups", "create", "delete", "get", "kernels", "list", "neighbors", "snapshots", "tag", "untag")
}

func TestDropletActionList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Actions(1).Return(testActionList, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletActions(config)
		assert.NoError(t, err)
	})
}

func TestDropletBackupList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Backups(1).Return(testImageList, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletBackups(config)
		assert.NoError(t, err)
	})
}

func TestDropletCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		volumeUUID := uuid.New().String()
		dcr := &godo.DropletCreateRequest{
			Name:    "droplet",
			Region:  "dev0",
			Size:    "1gb",
			Image:   godo.DropletCreateImage{ID: 0, Slug: "image"},
			SSHKeys: []godo.DropletCreateSSHKey{},
			Volumes: []godo.DropletCreateVolume{
				{Name: "test-volume"},
				{ID: volumeUUID},
			},
			Backups:           false,
			IPv6:              false,
			PrivateNetworking: false,
			Monitoring:        false,
			UserData:          "#cloud-config",
			Tags:              []string{"one", "two"},
		}
		tm.droplets.EXPECT().Create(dcr, false).Return(&testDroplet, nil)

		config.Args = append(config.Args, "droplet")

		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "dev0")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "1gb")
		config.Doit.Set(config.NS, doctl.ArgImage, "image")
		config.Doit.Set(config.NS, doctl.ArgUserData, "#cloud-config")
		config.Doit.Set(config.NS, doctl.ArgVolumeList, []string{"test-volume", volumeUUID})
		config.Doit.Set(config.NS, doctl.ArgTagNames, []string{"one", "two"})

		err := RunDropletCreate(config)
		assert.NoError(t, err)
	})
}

func TestDropletCreateWithTag(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dcr := &godo.DropletCreateRequest{Name: "droplet", Region: "dev0", Size: "1gb", Image: godo.DropletCreateImage{ID: 0, Slug: "image"}, SSHKeys: []godo.DropletCreateSSHKey{}, Backups: false, IPv6: false, PrivateNetworking: false, UserData: "#cloud-config"}
		tm.droplets.EXPECT().Create(dcr, false).Return(&testDroplet, nil)
		tm.tags.EXPECT().Get("my-tag").Return(&testTag, nil)

		trr := &godo.TagResourcesRequest{
			Resources: []godo.Resource{
				{ID: "1", Type: godo.DropletResourceType},
			},
		}
		tm.tags.EXPECT().TagResources("my-tag", trr).Return(nil)

		config.Args = append(config.Args, "droplet")

		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "dev0")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "1gb")
		config.Doit.Set(config.NS, doctl.ArgImage, "image")
		config.Doit.Set(config.NS, doctl.ArgUserData, "#cloud-config")
		config.Doit.Set(config.NS, doctl.ArgTagName, "my-tag")

		err := RunDropletCreate(config)
		assert.NoError(t, err)
	})
}

func TestDropletCreateUserDataFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		userData := `
coreos:
  etcd2:
    discovery: https://discovery.etcd.io/<token>
    advertise-client-urls: http://$private_ipv4:2379,http://$private_ipv4:4001
    initial-advertise-peer-urls: http://$private_ipv4:2380
    listen-client-urls: http://0.0.0.0:2379,http://0.0.0.0:4001
    listen-peer-urls: http://$private_ipv4:2380
  units:
    - name: etcd2.service
      command: start
    - name: fleet.service
      command: start
`

		tmpFile, err := ioutil.TempFile(os.TempDir(), "doctlDropletsTest-*.yml")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(userData)
		assert.NoError(t, err)

		err = tmpFile.Close()
		assert.NoError(t, err)

		dcr := &godo.DropletCreateRequest{
			Name:   "droplet",
			Region: "dev0",
			Size:   "1gb",
			Image: godo.DropletCreateImage{
				ID:   0,
				Slug: "image",
			},
			SSHKeys:           []godo.DropletCreateSSHKey{},
			Backups:           false,
			IPv6:              false,
			PrivateNetworking: false,
			UserData:          userData,
		}
		tm.droplets.EXPECT().Create(dcr, false).Return(&testDroplet, nil)

		config.Args = append(config.Args, "droplet")

		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "dev0")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "1gb")
		config.Doit.Set(config.NS, doctl.ArgImage, "image")
		config.Doit.Set(config.NS, doctl.ArgUserDataFile, tmpFile.Name())

		err = RunDropletCreate(config)
		assert.NoError(t, err)
	})
}

func TestDropletDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Delete(1).Return(nil)

		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID))
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunDropletDelete(config)
		assert.NoError(t, err)

	})
}

func TestDropletDeleteByTag_DropletsExist(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().ListByTag("my-tag").Return(testDropletList, nil)
		tm.droplets.EXPECT().DeleteByTag("my-tag").Return(nil)

		config.Doit.Set(config.NS, doctl.ArgTagName, "my-tag")
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunDropletDelete(config)
		assert.NoError(t, err)
	})
}

func TestDropletDeleteByTag_DropletsMissing(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().ListByTag("my-tag").Return(do.Droplets{}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgTagName, "my-tag")
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunDropletDelete(config)
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "nothing to delete")
	})
}

func TestDropletDeleteRepeatedID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Delete(1).Return(nil).Times(1)

		id := strconv.Itoa(testDroplet.ID)
		config.Args = append(config.Args, id, id)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunDropletDelete(config)
		assert.NoError(t, err)
	})
}

func TestDropletDeleteByName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().List().Return(testDropletList, nil)
		tm.droplets.EXPECT().Delete(1).Return(nil)

		config.Args = append(config.Args, testDroplet.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunDropletDelete(config)
		assert.NoError(t, err)
	})
}

func TestDropletDeleteByName_Ambiguous(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		list := do.Droplets{testDroplet, testDroplet}
		tm.droplets.EXPECT().List().Return(list, nil)

		config.Args = append(config.Args, testDroplet.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunDropletDelete(config)
		t.Log(err)
		assert.Error(t, err)
	})
}

func TestDropletDelete_MixedNameAndType(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().List().Return(testDropletList, nil)
		tm.droplets.EXPECT().Delete(1).Return(nil).Times(1)

		id := strconv.Itoa(testDroplet.ID)
		config.Args = append(config.Args, id, testDroplet.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunDropletDelete(config)
		assert.NoError(t, err)
	})

}

func TestDropletGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Get(testDroplet.ID).Return(&testDroplet, nil)

		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID))

		err := RunDropletGet(config)
		assert.NoError(t, err)
	})
}

func TestDropletGet_Template(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Get(testDroplet.ID).Return(&testDroplet, nil)

		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID))
		config.Doit.Set(config.NS, doctl.ArgTemplate, "{{.Name}}")

		err := RunDropletGet(config)
		assert.NoError(t, err)
	})
}

func TestDropletKernelList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Kernels(testDroplet.ID).Return(testKernelList, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletKernels(config)
		assert.NoError(t, err)
	})
}

func TestDropletNeighbors(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Neighbors(testDroplet.ID).Return(testDropletList, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletNeighbors(config)
		assert.NoError(t, err)
	})
}

func TestDropletSnapshotList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Snapshots(testDroplet.ID).Return(testImageList, nil)

		config.Args = append(config.Args, "1")

		err := RunDropletSnapshots(config)
		assert.NoError(t, err)
	})
}

func TestDropletsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().List().Return(testDropletList, nil)

		err := RunDropletList(config)
		assert.NoError(t, err)
	})
}

func TestDropletsListByTag(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().ListByTag("my-tag").Return(testDropletList, nil)

		config.Doit.Set(config.NS, doctl.ArgTagName, "my-tag")

		err := RunDropletList(config)
		assert.NoError(t, err)
	})
}

func TestDropletsTag(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		trr := &godo.TagResourcesRequest{
			Resources: []godo.Resource{
				{ID: "1", Type: godo.DropletResourceType},
			},
		}
		tm.tags.EXPECT().TagResources("my-tag", trr).Return(nil)

		config.Args = append(config.Args, "1")
		config.Doit.Set(config.NS, doctl.ArgTagName, "my-tag")

		err := RunDropletTag(config)
		assert.NoError(t, err)
	})
}

func TestDropletsTagMultiple(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		trr := &godo.TagResourcesRequest{
			Resources: []godo.Resource{
				{ID: "1", Type: godo.DropletResourceType},
				{ID: "2", Type: godo.DropletResourceType},
			},
		}
		tm.tags.EXPECT().TagResources("my-tag", trr).Return(nil)

		config.Args = append(config.Args, "1")
		config.Args = append(config.Args, "2")
		config.Doit.Set(config.NS, doctl.ArgTagName, "my-tag")

		err := RunDropletTag(config)
		assert.NoError(t, err)
	})
}

func TestDropletsTagByName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		trr := &godo.TagResourcesRequest{
			Resources: []godo.Resource{
				{ID: "1", Type: godo.DropletResourceType},
			},
		}
		tm.tags.EXPECT().TagResources("my-tag", trr).Return(nil)
		tm.droplets.EXPECT().List().Return(testDropletList, nil)

		config.Args = append(config.Args, testDroplet.Name)
		config.Doit.Set(config.NS, doctl.ArgTagName, "my-tag")

		err := RunDropletTag(config)
		assert.NoError(t, err)
	})
}

func TestDropletsTagMultipleNameAndID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		trr := &godo.TagResourcesRequest{
			Resources: []godo.Resource{
				{ID: "1", Type: godo.DropletResourceType},
				{ID: "3", Type: godo.DropletResourceType},
			},
		}
		tm.tags.EXPECT().TagResources("my-tag", trr).Return(nil)
		tm.droplets.EXPECT().List().Return(testDropletList, nil)

		config.Args = append(config.Args, testDroplet.Name)
		config.Args = append(config.Args, strconv.Itoa(anotherTestDroplet.ID))
		config.Doit.Set(config.NS, doctl.ArgTagName, "my-tag")

		err := RunDropletTag(config)
		assert.NoError(t, err)
	})
}

func TestDropletsUntag(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		urr := &godo.UntagResourcesRequest{
			Resources: []godo.Resource{
				{ID: "1", Type: godo.DropletResourceType},
			},
		}

		tm.tags.EXPECT().UntagResources("my-tag", urr).Return(nil)
		tm.droplets.EXPECT().List().Return(testDropletList, nil)

		config.Args = []string{testDroplet.Name}
		config.Doit.Set(config.NS, doctl.ArgTagName, "my-tag")

		err := RunDropletUntag(config)
		assert.NoError(t, err)
	})
}

func Test_extractSSHKey(t *testing.T) {
	cases := []struct {
		in       []string
		expected []godo.DropletCreateSSHKey
	}{
		{
			in:       []string{"1"},
			expected: []godo.DropletCreateSSHKey{{ID: 1}},
		},
		{
			in:       []string{"fingerprint"},
			expected: []godo.DropletCreateSSHKey{{Fingerprint: "fingerprint"}},
		},
		{
			in:       []string{"1", "2"},
			expected: []godo.DropletCreateSSHKey{{ID: 1}, {ID: 2}},
		},
		{
			in:       []string{"1", "fingerprint"},
			expected: []godo.DropletCreateSSHKey{{ID: 1}, {Fingerprint: "fingerprint"}},
		},
	}

	for _, c := range cases {
		got := extractSSHKeys(c.in)
		assert.Equal(t, c.expected, got)
	}
}
