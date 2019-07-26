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
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/stretchr/testify/assert"
)

func TestSnapshotCommand(t *testing.T) {
	cmd := Snapshot()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list", "get", "delete")
}

func TestSnapshotList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.snapshots.EXPECT().List().Return(testSnapshotList, nil)

		err := RunSnapshotList(config)
		assert.NoError(t, err)
	})
}

func TestSnapshotListID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.snapshots.EXPECT().List().Return(testSnapshotList, nil)

		config.Args = append(config.Args, testSnapshot.ID)

		err := RunSnapshotList(config)
		assert.NoError(t, err)
	})
}

func TestSnapshotListName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.snapshots.EXPECT().List().Return(testSnapshotList, nil)

		config.Args = append(config.Args, testSnapshot.Name)

		err := RunSnapshotList(config)
		assert.NoError(t, err)
	})
}

func TestSnapshotListMultiple(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.snapshots.EXPECT().List().Return(testSnapshotList, nil)

		config.Args = append(config.Args, testSnapshot.ID, testSnapshotSecondary.ID)

		err := RunSnapshotList(config)
		assert.NoError(t, err)
	})
}

func TestSnapshotListRegion(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.snapshots.EXPECT().List().Return(testSnapshotList, nil)

		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "dev0")

		err := RunSnapshotList(config)
		assert.NoError(t, err)
	})
}

func TestSnapshotGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.snapshots.EXPECT().Get(testSnapshot.ID).Return(&testSnapshot, nil)

		config.Args = append(config.Args, testSnapshot.ID)

		err := RunSnapshotGet(config)
		assert.NoError(t, err)
	})
}

func TestSnapshotGetMultiple(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.snapshots.EXPECT().Get(testSnapshot.ID).Return(&testSnapshot, nil)
		tm.snapshots.EXPECT().Get(testSnapshotSecondary.ID).Return(&testSnapshotSecondary, nil)

		config.Args = append(config.Args, testSnapshot.ID, testSnapshotSecondary.ID)

		err := RunSnapshotGet(config)
		assert.NoError(t, err)
	})
}

func TestSnapshotDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.snapshots.EXPECT().Delete(testSnapshot.ID).Return(nil)

		config.Args = append(config.Args, testSnapshot.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunSnapshotDelete(config)
		assert.NoError(t, err)

	})
}
