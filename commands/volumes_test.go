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
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	testVolume = do.Volume{
		Volume: &godo.Volume{
			ID:            uuid.New(),
			SizeGigaBytes: 100,
			Name:          "test-volume",
			Description:   "test description",
			Region:        &godo.Region{Slug: "atlantis"},
		},
	}
	testVolumeList = []do.Volume{
		testVolume,
	}
)

func TestVolumeCommand(t *testing.T) {
	cmd := Volume()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list", "snapshot")
}

func TestVolumesGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumes.On("Get", "test-volume").Return(&testVolume, nil)

		config.Args = append(config.Args, "test-volume")

		err := RunVolumeGet(config)
		assert.NoError(t, err)
	})
}

func TestVolumesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumes.On("List").Return(testVolumeList, nil)

		err := RunVolumeList(config)
		assert.NoError(t, err)
	})
}

func TestVolumesListID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumes.On("List").Return(testVolumeList, nil)

		config.Args = append(config.Args, testVolume.ID)

		err := RunVolumeList(config)
		assert.NoError(t, err)
	})
}

func TestVolumesListName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumes.On("List").Return(testVolumeList, nil)

		config.Args = append(config.Args, "test-volume")

		err := RunVolumeList(config)
		assert.NoError(t, err)
	})
}

func TestVolumeCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tcr := godo.VolumeCreateRequest{
			Name:          "test-volume",
			SizeGigaBytes: 100,
			Region:        "atlantis",
			Description:   "test description",
		}
		tm.volumes.On("CreateVolume", &tcr).Return(&testVolume, nil)

		config.Args = append(config.Args, "test-volume")

		config.Doit.Set(config.NS, doctl.ArgVolumeRegion, "atlantis")
		config.Doit.Set(config.NS, doctl.ArgVolumeSize, "100GiB")
		config.Doit.Set(config.NS, doctl.ArgVolumeDesc, "test description")

		err := RunVolumeCreate(config)
		assert.NoError(t, err)
	})
}

func TestVolumesDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.volumes.On("DeleteVolume", "test-volume").Return(nil)

		config.Args = append(config.Args, "test-volume")

		err := RunVolumeDelete(config)
		assert.NoError(t, err)
	})
}

func TestVolumesSnapshot(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tcr := godo.SnapshotCreateRequest{
			VolumeID:    testVolume.ID,
			Name:        "test-volume-snapshot",
			Description: "test description",
		}
		tm.volumes.On("CreateSnapshot", &tcr).Return(nil, nil)

		config.Args = append(config.Args, testVolume.ID)
		config.Doit.Set(config.NS, doctl.ArgSnapshotName, "test-volume-snapshot")
		config.Doit.Set(config.NS, doctl.ArgSnapshotDesc, "test description")

		err := RunVolumeSnapshot(config)
		assert.NoError(t, err)
	})
}
