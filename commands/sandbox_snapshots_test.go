/*
Copyright 2026 The Doctl Authors All rights reserved.
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
	"errors"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testSandboxSnapshotID = "33333333-4444-4555-8666-777777777777"
	testSnapSboxID        = "01a0ef1a-d873-793d-ae6a-5dd6239be62a"
)

func testSandboxSnapshot(state godo.SandboxSnapshotState) do.SandboxSnapshot {
	return do.SandboxSnapshot{SandboxSnapshot: &godo.SandboxSnapshot{
		ID:        testSandboxSnapshotID,
		SandboxID: testSnapSboxID,
		Name:      "my-snapshot",
		State:     state,
		SizeBytes: "1073741824",
		TTL:       "168h",
		CreatedAt: godo.Timestamp{Time: time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)},
	}}
}

func TestSandboxSnapshotCommandNames(t *testing.T) {
	cmd := Sandbox()
	var snap *Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "snapshot" {
			snap = &Command{Command: sub}
		}
	}
	require.NotNil(t, snap, "doctl sandbox snapshot")
	assert.Equal(t, []string{"snapshots"}, snap.Aliases)
	assertCommandNames(t, snap, "create", "delete", "get", "list")
}

func TestSandboxSnapshotCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testSnapSboxID}
		config.Doit.Set(config.NS, doctl.ArgSandboxSnapshotName, "my-snapshot")
		config.Doit.Set(config.NS, doctl.ArgSandboxSnapshotTTL, "168h")
		config.Doit.Set(config.NS, doctl.ArgCommandWait, false)

		creating := testSandboxSnapshot(godo.SandboxSnapshotStateCreating)
		tm.sandboxes.EXPECT().CreateSnapshot(testSnapSboxID, &godo.SandboxSnapshotCreateRequest{Name: "my-snapshot", TTL: "168h"}).
			Return(&creating, nil)
		require.NoError(t, RunSandboxSnapshotCreate(config))
	})
}

func TestSandboxSnapshotCreateWithNoOptionsAndWait(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testSnapSboxID}
		config.Doit.Set(config.NS, doctl.ArgSandboxSnapshotName, "")
		config.Doit.Set(config.NS, doctl.ArgSandboxSnapshotTTL, "")
		config.Doit.Set(config.NS, doctl.ArgCommandWait, true)
		config.Doit.Set(config.NS, doctl.ArgWaitTimeout, time.Minute)

		creating := testSandboxSnapshot(godo.SandboxSnapshotStateCreating)
		uploading := testSandboxSnapshot(godo.SandboxSnapshotStateUploading)
		complete := testSandboxSnapshot(godo.SandboxSnapshotStateComplete)
		tm.sandboxes.EXPECT().CreateSnapshot(testSnapSboxID, &godo.SandboxSnapshotCreateRequest{}).Return(&creating, nil)
		gomock.InOrder(
			tm.sandboxes.EXPECT().GetSnapshot(testSandboxSnapshotID).Return(&uploading, nil),
			tm.sandboxes.EXPECT().GetSnapshot(testSandboxSnapshotID).Return(&complete, nil),
		)
		require.NoError(t, RunSandboxSnapshotCreate(config))
	})
}

func TestSandboxSnapshotCreateValidatesBeforeCalling(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testSnapSboxID}
		config.Doit.Set(config.NS, doctl.ArgSandboxSnapshotTTL, "")
		for _, name := range []string{"Bad_Name", "UPPER", "a b"} {
			config.Doit.Set(config.NS, doctl.ArgSandboxSnapshotName, name)
			assert.ErrorContains(t, RunSandboxSnapshotCreate(config), "lowercase letters", name)
		}

		config.Doit.Set(config.NS, doctl.ArgSandboxSnapshotName, "")
		for _, ttl := range []string{"soon", "-1h", "0s"} {
			config.Doit.Set(config.NS, doctl.ArgSandboxSnapshotTTL, ttl)
			assert.ErrorContains(t, RunSandboxSnapshotCreate(config), "duration", ttl)
		}

		config.Args = nil
		assert.Error(t, RunSandboxSnapshotCreate(config))
		config.Args = []string{"a", "b"}
		assert.Error(t, RunSandboxSnapshotCreate(config))
	})
}

func TestSandboxSnapshotListGetAndDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		complete := testSandboxSnapshot(godo.SandboxSnapshotStateComplete)

		config.Doit.Set(config.NS, doctl.ArgSandboxID, "")
		tm.sandboxes.EXPECT().ListSnapshots("").Return(do.SandboxSnapshots{complete}, nil)
		require.NoError(t, RunSandboxSnapshotList(config))

		config.Doit.Set(config.NS, doctl.ArgSandboxID, testSnapSboxID)
		tm.sandboxes.EXPECT().ListSnapshots(testSnapSboxID).Return(do.SandboxSnapshots{complete}, nil)
		require.NoError(t, RunSandboxSnapshotList(config))

		config.Args = []string{testSandboxSnapshotID}
		tm.sandboxes.EXPECT().GetSnapshot(testSandboxSnapshotID).Return(&complete, nil)
		require.NoError(t, RunSandboxSnapshotGet(config))

		config.Args = nil
		assert.Error(t, RunSandboxSnapshotGet(config))
		config.Args = []string{"x"}
		assert.Error(t, RunSandboxSnapshotList(config))
	})
}

func TestSandboxSnapshotDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"s-1", "s-2", "s-3"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		tm.sandboxes.EXPECT().DeleteSnapshot("s-1").Return(nil)
		tm.sandboxes.EXPECT().DeleteSnapshot("s-2").Return(errors.New("boom"))

		err := RunSandboxSnapshotDelete(config)
		assert.ErrorContains(t, err, "s-2")
		assert.ErrorContains(t, err, "boom")

		config.Args = nil
		assert.Error(t, RunSandboxSnapshotDelete(config))
	})
}

func TestWaitForSnapshotReportsAFailureWithItsMessage(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		start := testSandboxSnapshot(godo.SandboxSnapshotStateCreating)
		failed := testSandboxSnapshot(godo.SandboxSnapshotStateFailed)
		failed.ErrorMessage = "disk full"
		tm.sandboxes.EXPECT().GetSnapshot(testSandboxSnapshotID).Return(&failed, nil)

		_, err := waitForSnapshot(newTestWaiter(), config.Sandboxes(), &start, time.Millisecond)
		assert.ErrorContains(t, err, "failed: disk full")
	})
}

func TestSandboxSnapshotDisplay(t *testing.T) {
	d := displayers.SandboxSnapshot{Snapshots: do.SandboxSnapshots{testSandboxSnapshot(godo.SandboxSnapshotStateUploading)}}
	assert.Equal(t, []map[string]any{{
		"ID":        testSandboxSnapshotID,
		"SandboxID": testSnapSboxID,
		"Name":      "my-snapshot",
		"State":     "uploading",
		"Size":      "1.0 GiB",
		"Created":   "2026-11-20T12:00:00Z",
		"TTL":       "168h",
		"Error":     "",
	}}, d.KV())
	assert.Equal(t, []string{"ID", "SandboxID", "Name", "State", "Size", "Created", "TTL"}, d.Cols())

	for in, want := range map[uint64]string{0: "", 512: "512 B", 1024: "1.0 KiB", 1536: "1.5 KiB", 5 << 20: "5.0 MiB", 1 << 30: "1.0 GiB", 3 << 40: "3.0 TiB"} {
		assert.Equal(t, want, displayers.FormatSandboxSnapshotSize(in), "size %d", in)
	}
}
