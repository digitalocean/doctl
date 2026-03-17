/*
Copyright 2025 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testId  = "9bf36b9e-1306-4f5d-aa33-d2b73ab1497a"
	testNfs = do.Nfs{
		Nfs: &godo.Nfs{
			ID:        "9bf36b9e-1306-4f5d-aa33-d2b73ab1497a",
			Name:      "sammy-share",
			Region:    "atl1",
			SizeGib:   1024,
			Status:    "ACTIVE",
			VpcIDs:    []string{"a0ca4d02-1c86-4330-a4ff-310ee60c7de0"},
			CreatedAt: "2025-10-17T14:01:01Z",
		},
	}
	testNfsSnapshot = do.NfsSnapshot{
		NfsSnapshot: &godo.NfsSnapshot{
			ID:        "f050990d-4337-4a9d-9c8d-9f759a83936b",
			Name:      "my-snapshot",
			ShareID:   "9bf36b9e-1306-4f5d-aa33-d2b73ab1497a",
			Region:    "atl1",
			SizeGib:   1024,
			Status:    "ACTIVE",
			CreatedAt: "2025-10-17T14:05:01Z",
		},
	}
	testNfsAction = do.NfsAction{
		NfsAction: &godo.NfsAction{
			ID:     123456,
			Status: "in-progress",
			Type:   "snapshot",
		},
	}
)

func TestNfsCommand(t *testing.T) {
	cmd := Nfs()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "get", "delete", "snapshot", "resize", "attach", "detach", "switch-performance-tier")
}

func TestRunNfsCreate(t *testing.T) {
	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name:      "success",
			args:      []string{"sammy-share", "atl1", "1024", "a0ca4d02-1c86-4330-a4ff-310ee60c7de0"},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					req := &godo.NfsCreateRequest{
						Name:            "sammy-share",
						Region:          "atl1",
						SizeGib:         1024,
						VpcIDs:          []string{"a0ca4d02-1c86-4330-a4ff-310ee60c7de0"},
						PerformanceTier: "standard",
					}
					tm.nfs.EXPECT().Create(req).Return(&testNfs, nil)
				}

				config.Doit.Set(config.NS, "name", tc.args[0])
				config.Doit.Set(config.NS, "region", tc.args[1])
				config.Doit.Set(config.NS, "size", tc.args[2])
				config.Doit.Set(config.NS, "vpc-ids", []string{tc.args[3]})
				config.Doit.Set(config.NS, "performance-tier", "standard")
				err := nfsCreate(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestRunNfsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.nfs.EXPECT().List("atl1").Return([]do.Nfs{testNfs}, nil)

		config.Doit.Set(config.NS, "region", "atl1")
		err := nfsList(config)
		require.NoError(t, err)
	})
}

func TestRunNfsGet(t *testing.T) {
	testCases := []struct {
		name      string
		region    string
		id        string
		expectErr bool
	}{
		{
			name:      "success",
			region:    "atl1",
			id:        testId,
			expectErr: false,
		},
		{
			name:      "missing key id",
			region:    "",
			id:        testId,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					tm.nfs.EXPECT().Get(tc.id, tc.region).Return(&testNfs, nil)
				} else {
					tm.nfs.EXPECT().Get(tc.id, tc.region).Return(nil, errors.New("region cannot be empty"))
				}

				config.Doit.Set(config.NS, "id", tc.id)
				config.Doit.Set(config.NS, "region", tc.region)

				err := nfsGet(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}

}

func TestRunNfsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.nfs.EXPECT().Delete(testId, "atl1").Return(nil)

		config.Doit.Set(config.NS, "id", testId)
		config.Doit.Set(config.NS, "region", "atl1")

		err := nfsDelete(config)
		require.NoError(t, err)
	})
}

func TestRunNfsSnapshotList(t *testing.T) {
	testCases := []struct {
		name      string
		shareID   string
		region    string
		expectErr bool
	}{
		{
			name:      "list all snapshots in region",
			shareID:   "",
			region:    "atl1",
			expectErr: false,
		},
		{
			name:      "list snapshots for specific share",
			shareID:   testId,
			region:    "atl1",
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					tm.nfs.EXPECT().ListSnapshots(tc.shareID, tc.region).Return([]do.NfsSnapshot{testNfsSnapshot}, nil)
				}

				config.Doit.Set(config.NS, "region", tc.region)
				if tc.shareID != "" {
					config.Doit.Set(config.NS, "share-id", tc.shareID)
				}

				err := nfsSnapshotList(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestRunNfsSnapshotGet(t *testing.T) {
	testCases := []struct {
		name      string
		id        string
		region    string
		expectErr bool
	}{
		{
			name:      "success",
			id:        testId,
			region:    "atl1",
			expectErr: false,
		},
		{
			name:      "missing region",
			id:        testId,
			region:    "",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					tm.nfs.EXPECT().GetSnapshot(tc.id, tc.region).Return(&testNfsSnapshot, nil)
				} else {
					tm.nfs.EXPECT().GetSnapshot(tc.id, tc.region).Return(nil, errors.New("region cannot be empty"))
				}

				config.Doit.Set(config.NS, "id", tc.id)
				config.Doit.Set(config.NS, "region", tc.region)

				err := nfsSnapshotGet(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestRunNfsSnapshotDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.nfs.EXPECT().DeleteSnapshot(testId, "atl1").Return(nil)

		config.Doit.Set(config.NS, "id", testId)
		config.Doit.Set(config.NS, "region", "atl1")

		err := nfsSnapshotDelete(config)
		require.NoError(t, err)
	})
}

func TestRunNfsSnapshotCreate(t *testing.T) {
	testCases := []struct {
		name      string
		shareID   string
		region    string
		snapName  string
		wait      bool
		expectErr bool
	}{
		{
			name:      "success without wait",
			shareID:   testId,
			region:    "atl1",
			snapName:  "my-snapshot",
			wait:      false,
			expectErr: false,
		},
		{
			name:      "success with wait",
			shareID:   testId,
			region:    "atl1",
			snapName:  "my-snapshot",
			wait:      true,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					tm.nfsActions.EXPECT().Snapshot(tc.shareID, tc.snapName, tc.region).Return(&testNfsAction, nil)
					if tc.wait {
						tm.actions.EXPECT().Get(testNfsAction.ID).Return(&testAction, nil)
					}
				}

				config.Doit.Set(config.NS, "name", tc.snapName)
				config.Doit.Set(config.NS, "share-id", tc.shareID)
				config.Doit.Set(config.NS, "region", tc.region)
				config.Doit.Set(config.NS, "wait", tc.wait)

				err := nfsSnapshotCreate(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestRunNfsResize(t *testing.T) {
	testCases := []struct {
		name      string
		id        string
		region    string
		size      string
		wait      bool
		expectErr bool
	}{
		{
			name:      "success without wait",
			id:        testId,
			region:    "atl1",
			size:      "2048",
			wait:      false,
			expectErr: false,
		},
		{
			name:      "success with wait",
			id:        testId,
			region:    "atl1",
			size:      "2048",
			wait:      true,
			expectErr: false,
		},
		{
			name:      "invalid size",
			id:        testId,
			region:    "atl1",
			size:      "invalid",
			wait:      false,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr && tc.size != "invalid" {
					size := uint64(2048)
					tm.nfsActions.EXPECT().Resize(tc.id, size, tc.region).Return(&testNfsAction, nil)
					if tc.wait {
						tm.actions.EXPECT().Get(testNfsAction.ID).Return(&testAction, nil)
					}
				}

				config.Doit.Set(config.NS, "id", tc.id)
				config.Doit.Set(config.NS, "region", tc.region)
				config.Doit.Set(config.NS, "size", tc.size)
				config.Doit.Set(config.NS, "wait", tc.wait)

				err := nfsResize(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestRunNfsAttach(t *testing.T) {
	testCases := []struct {
		name      string
		id        string
		region    string
		vpcID     string
		wait      bool
		expectErr bool
	}{
		{
			name:      "success without wait",
			id:        testId,
			region:    "atl1",
			vpcID:     "vpc-1234",
			wait:      false,
			expectErr: false,
		},
		{
			name:      "success with wait",
			id:        testId,
			region:    "atl1",
			vpcID:     "vpc-1234",
			wait:      true,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					vpcID := "vpc-1234"
					tm.nfsActions.EXPECT().Attach(tc.id, vpcID, tc.region).Return(&testNfsAction, nil)
					if tc.wait {
						tm.actions.EXPECT().Get(testNfsAction.ID).Return(&testAction, nil)
					}
				}

				config.Doit.Set(config.NS, "id", tc.id)
				config.Doit.Set(config.NS, "region", tc.region)
				config.Doit.Set(config.NS, "vpc_id", tc.vpcID)
				config.Doit.Set(config.NS, "wait", tc.wait)

				err := nfsAttach(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestRunNfsDetach(t *testing.T) {
	testCases := []struct {
		name      string
		id        string
		region    string
		vpcID     string
		wait      bool
		expectErr bool
	}{
		{
			name:      "success without wait",
			id:        testId,
			region:    "atl1",
			vpcID:     "vpc-1234",
			wait:      false,
			expectErr: false,
		},
		{
			name:      "success with wait",
			id:        testId,
			region:    "atl1",
			vpcID:     "vpc-1234",
			wait:      true,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					vpcID := "vpc-1234"
					tm.nfsActions.EXPECT().Detach(tc.id, vpcID, tc.region).Return(&testNfsAction, nil)
					if tc.wait {
						tm.actions.EXPECT().Get(testNfsAction.ID).Return(&testAction, nil)
					}
				}

				config.Doit.Set(config.NS, "id", tc.id)
				config.Doit.Set(config.NS, "region", tc.region)
				config.Doit.Set(config.NS, "vpc_id", tc.vpcID)
				config.Doit.Set(config.NS, "wait", tc.wait)

				err := nfsDetach(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestRunNfsSwitchPerformanceTier(t *testing.T) {
	testCases := []struct {
		name            string
		id              string
		performanceTier string
		wait            bool
		expectErr       bool
	}{
		{
			name:            "success without wait",
			id:              testId,
			performanceTier: "standard",
			wait:            false,
			expectErr:       false,
		},
		{
			name:            "success with wait",
			id:              testId,
			performanceTier: "premium",
			wait:            true,
			expectErr:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					tm.nfsActions.EXPECT().SwitchPerformanceTier(tc.id, tc.performanceTier).Return(&testNfsAction, nil)
					if tc.wait {
						tm.actions.EXPECT().Get(testNfsAction.ID).Return(&testAction, nil)
					}
				}

				config.Doit.Set(config.NS, "id", tc.id)
				config.Doit.Set(config.NS, "performance-tier", tc.performanceTier)
				config.Doit.Set(config.NS, "wait", tc.wait)

				err := nfsSwitchPerformanceTier(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}
