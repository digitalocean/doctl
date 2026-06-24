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
	"fmt"
	"os"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	defaultVectorDBRegion = "nyc3"
	defaultVectorDBSize   = "small"
	vectorDBListDetails   = `

This command requires the ID of a vector database, which you can retrieve by calling:

	doctl vector-databases list`
)

// VectorDatabases creates the vector-databases command
func VectorDatabases() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "vector-databases",
			Aliases: []string{"vdb", "vdbs"},
			Short:   "Display commands that manage vector databases",
			Long:    "The commands under `doctl vector-databases` are for managing your DigitalOcean vector database services.",
			GroupID: manageResourcesGroup,
		},
	}

	clusterDetails := `

- The vector database ID, in UUID format
- The name you gave the vector database
- The region the vector database resides in, such as ` + "`sfo2`" + ` or ` + "`nyc3`" + `
- The current status of the vector database, such as ` + "`active`" + `
- The size tier of the vector database, such as ` + "`small`" + `)`

	cmdVectorDBList := CmdBuilder(cmd, RunVectorDBList, "list", "List your vector databases", `Retrieves a list of vector databases and their following details:`+clusterDetails, Writer, aliasOpt("ls"), displayerType(&displayers.VectorDBs{}))
	cmdVectorDBList.Example = `The following example lists all vector databases associated with your account: doctl vector-databases list`

	cmdVectorDBGet := CmdBuilder(cmd, RunVectorDBGet, "get <vector-database-id>", "Get details for a vector database", `Retrieves the following details about the specified vector database:`+clusterDetails+`
- HTTP and gRPC connection endpoints
- Tags applied to the vector database
- The date and time when the vector database was created`+vectorDBListDetails, Writer, aliasOpt("g"), displayerType(&displayers.VectorDBs{}))
	cmdVectorDBGet.Example = `The following example retrieves details for a vector database: doctl vector-databases get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	sizeDetails := "The size tier of the vector database, such as `small`, `medium`, or `large`. For a list of available sizes, visit: https://docs.digitalocean.com/reference/api/api-reference/#tag/Vector-Databases"
	cmdVectorDBCreate := CmdBuilder(cmd, RunVectorDBCreate, "create <name>", "Create a vector database", `Creates a vector database with the specified name.

You can customize the configuration using the listed flags, all of which are optional. Without any flags set, the command creates a vector database in the `+"`"+defaultVectorDBRegion+"`"+` region with the `+"`"+defaultVectorDBSize+"`"+` size tier.`, Writer, aliasOpt("c"))
	AddStringFlag(cmdVectorDBCreate, doctl.ArgRegionSlug, "", defaultVectorDBRegion, "The data center region where the vector database resides, such as `nyc3` or `sfo2`.")
	AddStringFlag(cmdVectorDBCreate, doctl.ArgSizeSlug, "", defaultVectorDBSize, sizeDetails)
	AddStringSliceFlag(cmdVectorDBCreate, doctl.ArgTag, "", nil, "A comma-separated list of tags to apply to the vector database.")
	AddStringFlag(cmdVectorDBCreate, doctl.ArgProjectID, "", "", "The ID of the project to assign the vector database to. If not specified, the default project will be used.")
	AddBoolFlag(cmdVectorDBCreate, doctl.ArgCommandWait, "", false, "A boolean value that specifies whether to wait for the vector database to be active before returning control to the terminal.")
	cmdVectorDBCreate.Example = `The following example creates a vector database named example-vdb in the nyc3 region: doctl vector-databases create example-vdb --region nyc3 --size small`

	cmdVectorDBDelete := CmdBuilder(cmd, RunVectorDBDelete, "delete <vector-database-id>", "Delete a vector database", `Deletes the vector database with the specified ID.

To retrieve a list of your vector databases and their IDs, use `+"`"+`doctl vector-databases list`+"`"+`.`, Writer, aliasOpt("rm"))
	AddBoolFlag(cmdVectorDBDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the vector database without a confirmation prompt")
	cmdVectorDBDelete.Example = `The following example deletes a vector database: doctl vector-databases delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdVectorDBUpdate := CmdBuilder(cmd, RunVectorDBUpdate, "update <vector-database-id>", "Update a vector database configuration", `Updates advanced configuration settings for the specified vector database.`, Writer, aliasOpt("u"))
	AddStringFlag(cmdVectorDBUpdate, doctl.ArgVectorDBDefaultQuantization, "", "", "The default quantization setting for the vector database.")
	AddBoolFlag(cmdVectorDBUpdate, doctl.ArgVectorDBEnableAutoSchema, "", false, "Whether to enable automatic schema creation.")
	AddStringFlag(cmdVectorDBUpdate, doctl.ArgVectorDBWeaviateVersion, "", "", "The Weaviate version to use.")
	cmdVectorDBUpdate.Example = `The following example updates a vector database configuration: doctl vector-databases update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --enable-auto-schema`

	cmdVectorDBResize := CmdBuilder(cmd, RunVectorDBResize, "resize <vector-database-id>", "Resize a vector database", `Resizes the specified vector database to a new size tier.`, Writer, aliasOpt("rs"))
	AddStringFlag(cmdVectorDBResize, doctl.ArgSizeSlug, "", "", sizeDetails, requiredOpt())
	AddBoolFlag(cmdVectorDBResize, doctl.ArgCommandWait, "", false, "A boolean value that specifies whether to wait for the resize to complete before returning control to the terminal.")
	cmdVectorDBResize.Example = `The following example resizes a vector database: doctl vector-databases resize f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --size medium`

	cmdVectorDBTags := CmdBuilder(cmd, RunVectorDBTags, "tags <vector-database-id>", "Update tags on a vector database", `Replaces all tags on the specified vector database with the tags provided.`, Writer)
	AddStringSliceFlag(cmdVectorDBTags, doctl.ArgTag, "", nil, "A comma-separated list of tags to apply to the vector database.", requiredOpt())
	cmdVectorDBTags.Example = `The following example sets tags on a vector database: doctl vector-databases tags f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --tag production --tag ml`

	cmdVectorDBCredentials := CmdBuilder(cmd, RunVectorDBCredentials, "credentials <vector-database-id>", "Retrieve admin credentials for a vector database", `Retrieves admin credentials for the specified vector database.`, Writer, aliasOpt("creds"), displayerType(&displayers.VectorDBAdminCredentials{}))
	cmdVectorDBCredentials.Example = `The following example retrieves credentials for a vector database: doctl vector-databases credentials f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdVectorDBBackups := CmdBuilder(cmd, RunVectorDBBackupsList, "backups <vector-database-id>", "List vector database backups", `Retrieves a list of backups for the specified vector database.`+vectorDBListDetails, Writer, aliasOpt("bu"), displayerType(&displayers.VectorDBBackups{}))
	cmdVectorDBBackups.Example = `The following example lists backups for a vector database: doctl vector-databases backups f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdVectorDBRestore := CmdBuilder(cmd, RunVectorDBRestore, "restore <vector-database-id> <backup-id>", "Restore a vector database from a backup", `Initiates an asynchronous restore of the specified vector database from a backup.`, Writer, displayerType(&displayers.VectorDBRestoreBackupResponse{}))
	cmdVectorDBRestore.Example = `The following example restores a vector database from a backup: doctl vector-databases restore f81d4fae-7dec-11d0-a765-00a0c91e6bf6 backup-id`

	cmdVectorDBRestoreStatus := CmdBuilder(cmd, RunVectorDBRestoreStatus, "restore-status <vector-database-id> <backup-id>", "Get the status of a vector database restore", `Retrieves the current status of a restore operation for the specified vector database and backup.`, Writer, aliasOpt("rs-status"), displayerType(&displayers.VectorDBRestoreStatus{}))
	cmdVectorDBRestoreStatus.Example = `The following example retrieves restore status: doctl vector-databases restore-status f81d4fae-7dec-11d0-a765-00a0c91e6bf6 backup-id`

	return cmd
}

// RunVectorDBList returns a list of vector databases.
func RunVectorDBList(c *CmdConfig) error {
	vdbs, err := c.VectorDBs().List()
	if err != nil {
		return err
	}

	return displayVectorDBs(c, true, vdbs...)
}

// RunVectorDBGet returns an individual vector database.
func RunVectorDBGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	vdb, err := c.VectorDBs().Get(c.Args[0])
	if err != nil {
		return err
	}

	return displayVectorDBs(c, false, *vdb)
}

// RunVectorDBCreate creates a vector database.
func RunVectorDBCreate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	r, err := buildVectorDBCreateRequestFromArgs(c)
	if err != nil {
		return err
	}

	svc := c.VectorDBs()
	vdb, err := svc.Create(r)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		notice("Vector database creation is in progress, waiting for vector database to be active")
		if err := waitForVectorDBReady(svc, vdb.ID); err != nil {
			return fmt.Errorf("vector database couldn't enter the `active` state: %v", err)
		}

		vdb, err = svc.Get(vdb.ID)
		if err != nil {
			return fmt.Errorf("failed to retrieve the new vector database: %v", err)
		}
	}

	notice("Vector database created")
	return displayVectorDBs(c, false, *vdb)
}

func buildVectorDBCreateRequestFromArgs(c *CmdConfig) (*godo.VectorDBCreateRequest, error) {
	r := &godo.VectorDBCreateRequest{Name: c.Args[0]}

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return nil, err
	}
	r.Region = region

	size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return nil, err
	}
	r.Size = size

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return nil, err
	}
	r.Tags = tags

	projectID, err := c.Doit.GetString(c.NS, doctl.ArgProjectID)
	if err != nil {
		return nil, err
	}
	r.ProjectID = projectID

	return r, nil
}

// RunVectorDBDelete deletes a vector database.
func RunVectorDBDelete(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("vector database", 1) == nil {
		return c.VectorDBs().Delete(c.Args[0])
	}

	return errOperationAborted
}

// RunVectorDBUpdate updates a vector database configuration.
func RunVectorDBUpdate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	r, err := buildVectorDBUpdateRequestFromArgs(c)
	if err != nil {
		return err
	}

	vdb, err := c.VectorDBs().Update(c.Args[0], r)
	if err != nil {
		return err
	}

	return displayVectorDBs(c, false, *vdb)
}

func buildVectorDBUpdateRequestFromArgs(c *CmdConfig) (*godo.VectorDBUpdateRequest, error) {
	cfg := &godo.VectorDBConfig{}

	quantization, err := c.Doit.GetString(c.NS, doctl.ArgVectorDBDefaultQuantization)
	if err != nil {
		return nil, err
	}
	if quantization != "" {
		cfg.DefaultQuantization = quantization
	}

	enableAutoSchema, err := c.Doit.GetBool(c.NS, doctl.ArgVectorDBEnableAutoSchema)
	if err != nil {
		return nil, err
	}
	if c.Command.Flags().Changed(doctl.ArgVectorDBEnableAutoSchema) {
		cfg.EnableAutoSchema = enableAutoSchema
	}

	weaviateVersion, err := c.Doit.GetString(c.NS, doctl.ArgVectorDBWeaviateVersion)
	if err != nil {
		return nil, err
	}
	if weaviateVersion != "" {
		cfg.WeaviateVersion = weaviateVersion
	}

	return &godo.VectorDBUpdateRequest{Config: cfg}, nil
}

// RunVectorDBResize resizes a vector database.
func RunVectorDBResize(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]
	svc := c.VectorDBs()

	size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return err
	}

	_, err = svc.Resize(id, &godo.VectorDBResizeRequest{Size: size})
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		notice("Vector database resizing is in progress, waiting for vector database to be active")
		if err := waitForVectorDBReady(svc, id); err != nil {
			return fmt.Errorf("vector database couldn't enter the `active` state after resizing: %v", err)
		}
		notice("Vector database resized successfully")
	}

	return nil
}

// RunVectorDBTags updates tags on a vector database.
func RunVectorDBTags(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return err
	}

	vdb, err := c.VectorDBs().UpdateTags(c.Args[0], &godo.VectorDBUpdateTagsRequest{Tags: tags})
	if err != nil {
		return err
	}

	return displayVectorDBs(c, false, *vdb)
}

// RunVectorDBCredentials retrieves admin credentials for a vector database.
func RunVectorDBCredentials(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	creds, err := c.VectorDBs().GetCredentials(c.Args[0])
	if err != nil {
		return err
	}

	item := &displayers.VectorDBAdminCredentials{VectorDBAdminCredentials: *creds}
	return c.Display(item)
}

// RunVectorDBBackupsList lists backups for a vector database.
func RunVectorDBBackupsList(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	backups, err := c.VectorDBs().ListBackups(c.Args[0])
	if err != nil {
		return err
	}

	item := &displayers.VectorDBBackups{VectorDBBackups: backups}
	return c.Display(item)
}

// RunVectorDBRestore initiates a restore from backup.
func RunVectorDBRestore(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	resp, err := c.VectorDBs().RestoreBackup(c.Args[0], c.Args[1], &godo.VectorDBRestoreBackupRequest{})
	if err != nil {
		return err
	}

	item := &displayers.VectorDBRestoreBackupResponse{VectorDBRestoreBackupResponse: *resp}
	return c.Display(item)
}

// RunVectorDBRestoreStatus retrieves the status of a restore operation.
func RunVectorDBRestoreStatus(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	status, err := c.VectorDBs().GetRestoreStatus(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}

	item := &displayers.VectorDBRestoreStatus{VectorDBRestoreStatus: *status}
	return c.Display(item)
}

func displayVectorDBs(c *CmdConfig, short bool, vdbs ...do.VectorDB) error {
	item := &displayers.VectorDBs{
		VectorDBs: do.VectorDBs(vdbs),
		Short:     short,
	}
	return c.Display(item)
}

func waitForVectorDBReady(svc do.VectorDBsService, id string) error {
	const (
		maxAttempts = 180
		wantStatus  = "active"
	)
	attempts := 0
	printNewLineSet := false

	for i := 0; i < maxAttempts; i++ {
		if attempts != 0 {
			fmt.Fprint(os.Stderr, ".")
			if !printNewLineSet {
				printNewLineSet = true
				defer fmt.Fprintln(os.Stderr)
			}
		}

		vdb, err := svc.Get(id)
		if err != nil {
			return err
		}

		if vdb.Status == wantStatus {
			return nil
		}

		attempts++
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timeout waiting for vector database to become active")
}
