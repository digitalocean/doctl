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
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testVectorDB = do.VectorDB{
		VectorDB: &godo.VectorDB{
			ID:        "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
			Name:      "example-vdb",
			Region:    "nyc3",
			Status:    "active",
			Size:      "small",
			Tags:      []string{"production"},
			CreatedAt: time.Now(),
			Endpoints: &godo.VectorDBEndpoints{
				HTTP: "https://example.db.ondigitalocean.com",
				GRPC: "example.db.ondigitalocean.com:50051",
			},
		},
	}

	testVectorDBList = do.VectorDBs{testVectorDB}

	testVectorDBBackup = do.VectorDBBackup{
		VectorDBBackup: &godo.VectorDBBackup{
			BackupID:  "backup-1",
			Status:    "completed",
			StartedAt: time.Now(),
		},
	}

	testVectorDBBackups = do.VectorDBBackups{testVectorDBBackup}

	testVectorDBCredentials = do.VectorDBAdminCredentials{
		VectorDBAdminCredentials: &godo.VectorDBAdminCredentials{
			UserID:   "user-1",
			APIToken: "token-1",
		},
	}

	testVectorDBRestoreResponse = do.VectorDBRestoreBackupResponse{
		VectorDBRestoreBackupResponse: &godo.VectorDBRestoreBackupResponse{
			BackupID: "backup-1",
			Status:   "in_progress",
		},
	}

	testVectorDBRestoreStatus = do.VectorDBRestoreStatus{
		VectorDBRestoreStatus: &godo.VectorDBRestoreStatus{
			BackupID: "backup-1",
			Status:   "completed",
		},
	}
)

func TestVectorDatabasesCommand(t *testing.T) {
	cmd := VectorDatabases()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"list",
		"get",
		"create",
		"delete",
		"update",
		"resize",
		"tags",
		"credentials",
		"backups",
		"restore",
		"restore-status",
	)
}

func TestVectorDBList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.vectorDBs.EXPECT().List().Return(testVectorDBList, nil)

		err := RunVectorDBList(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.vectorDBs.EXPECT().List().Return(nil, errTest)

		err := RunVectorDBList(config)
		assert.Error(t, err)
	})
}

func TestVectorDBGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.vectorDBs.EXPECT().Get(testVectorDB.ID).Return(&testVectorDB, nil)
		config.Args = append(config.Args, testVectorDB.ID)

		err := RunVectorDBGet(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVectorDBGet(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestVectorDBCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := &godo.VectorDBCreateRequest{
			Name:      testVectorDB.Name,
			Region:    "nyc3",
			Size:      "small",
			Tags:      []string{"production"},
			ProjectID: "proj-1",
		}
		tm.vectorDBs.EXPECT().Create(r).Return(&testVectorDB, nil)

		config.Args = append(config.Args, testVectorDB.Name)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "small")
		config.Doit.Set(config.NS, doctl.ArgTag, []string{"production"})
		config.Doit.Set(config.NS, doctl.ArgProjectID, "proj-1")

		err := RunVectorDBCreate(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVectorDBCreate(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestVectorDBUpdate(t *testing.T) {
	// updateCmd resolves the "update" subcommand so c.Command.Flags().Changed
	// works the same way it does during a real invocation.
	updateCmd := func(config *CmdConfig) {
		for _, child := range VectorDatabases().Commands() {
			if child.Name() == "update" {
				config.Command = child
				break
			}
		}
		assert.NotNil(t, config.Command)
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := &godo.VectorDBUpdateRequest{
			Config: &godo.VectorDBConfig{
				DefaultQuantization: "scalar",
				EnableAutoSchema:    true,
				WeaviateVersion:     "1.25.0",
			},
		}
		tm.vectorDBs.EXPECT().Update(testVectorDB.ID, r).Return(&testVectorDB, nil)

		config.Args = append(config.Args, testVectorDB.ID)
		updateCmd(config)
		assert.NoError(t, config.Command.Flags().Set(doctl.ArgVectorDBEnableAutoSchema, "true"))
		config.Doit.Set(config.NS, doctl.ArgVectorDBDefaultQuantization, "scalar")
		config.Doit.Set(config.NS, doctl.ArgVectorDBEnableAutoSchema, true)
		config.Doit.Set(config.NS, doctl.ArgVectorDBWeaviateVersion, "1.25.0")

		err := RunVectorDBUpdate(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVectorDBUpdate(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestVectorDBDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.vectorDBs.EXPECT().Delete(testVectorDB.ID).Return(nil)

		config.Args = append(config.Args, testVectorDB.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunVectorDBDelete(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVectorDBDelete(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestVectorDBResize(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := &godo.VectorDBResizeRequest{Size: "medium"}
		tm.vectorDBs.EXPECT().Resize(testVectorDB.ID, r).Return(&testVectorDB, nil)

		config.Args = append(config.Args, testVectorDB.ID)
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "medium")

		err := RunVectorDBResize(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVectorDBResize(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestVectorDBTags(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := &godo.VectorDBUpdateTagsRequest{Tags: []string{"production", "ml"}}
		tm.vectorDBs.EXPECT().UpdateTags(testVectorDB.ID, r).Return(&testVectorDB, nil)

		config.Args = append(config.Args, testVectorDB.ID)
		config.Doit.Set(config.NS, doctl.ArgTag, []string{"production", "ml"})

		err := RunVectorDBTags(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVectorDBTags(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestVectorDBCredentials(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.vectorDBs.EXPECT().GetCredentials(testVectorDB.ID).Return(&testVectorDBCredentials, nil)
		config.Args = append(config.Args, testVectorDB.ID)

		err := RunVectorDBCredentials(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVectorDBCredentials(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestVectorDBBackupsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.vectorDBs.EXPECT().ListBackups(testVectorDB.ID).Return(testVectorDBBackups, nil)
		config.Args = append(config.Args, testVectorDB.ID)

		err := RunVectorDBBackupsList(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVectorDBBackupsList(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestVectorDBRestore(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := &godo.VectorDBRestoreBackupRequest{}
		tm.vectorDBs.EXPECT().RestoreBackup(testVectorDB.ID, "backup-1", r).Return(&testVectorDBRestoreResponse, nil)
		config.Args = append(config.Args, testVectorDB.ID, "backup-1")

		err := RunVectorDBRestore(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testVectorDB.ID)

		err := RunVectorDBRestore(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestVectorDBRestoreStatus(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.vectorDBs.EXPECT().GetRestoreStatus(testVectorDB.ID, "backup-1").Return(&testVectorDBRestoreStatus, nil)
		config.Args = append(config.Args, testVectorDB.ID, "backup-1")

		err := RunVectorDBRestoreStatus(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testVectorDB.ID)

		err := RunVectorDBRestoreStatus(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}
