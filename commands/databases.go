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
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	defaultDatabaseNodeSize  = "db-s-1vcpu-1gb"
	defaultDatabaseNodeCount = 1
	defaultDatabaseRegion    = "nyc1"
	defaultDatabaseEngine    = "pg"
	databaseListDetails      = `

This command requires the ID of a database cluster, which you can retrieve by calling:

	doctl databases list`
)

// Databases creates the databases command
func Databases() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "databases",
			Aliases: []string{"db", "dbs", "d", "database"},
			Short:   "Display commands that manage databases",
			Long:    "The commands under `doctl databases` are for managing your MySQL, Redis, PostgreSQL, MongoDB, Kafka and Opensearch database services.",
			GroupID: manageResourcesGroup,
		},
	}

	clusterDetails := `

- The database ID, in UUID format
- The name you gave the database cluster
- The database engine. Possible values: ` + "`redis`, `pg`, `mysql` , `mongodb`, `kafka`, `opensearch`" + `
- The engine version, such as ` + "`14`" + ` for PostgreSQL version 14
- The number of nodes in the database cluster
- The region the database cluster resides in, such as ` + "`sfo2`, " + "`nyc1`" + `
- The current status of the database cluster, such as ` + "`online`" + `
- The size of the machine running the database instance, such as ` + "`db-s-1vcpu-1gb`" + `)`

	cmdDatabaseList := CmdBuilder(cmd, RunDatabaseList, "list", "List your database clusters", `Retrieves a list of database clusters and their following details:`+clusterDetails, Writer, aliasOpt("ls"), displayerType(&displayers.Databases{}))
	cmdDatabaseList.Example = `The following example lists all database associated with your account and uses the ` + "`" + `--format` + "`" + ` flag to return only the ID, engine, and engine version of each database: doctl databases list --format ID,Engine,Version`
	cmdDatabaseGet := CmdBuilder(cmd, RunDatabaseGet, "get <database-cluster-id>", "Get details for a database cluster", `Retrieves the following details about the specified database cluster: `+clusterDetails+`
- A connection string for the database cluster
- The date and time when the database cluster was created`+databaseListDetails, Writer, aliasOpt("g"), displayerType(&displayers.Databases{}))
	cmdDatabaseGet.Example = `The following example retrieves the details for a database cluster with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` and uses the ` + "`" + `--format` + "`" + ` flag to return only the database's ID, engine, and engine version: doctl databases get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	nodeSizeDetails := "The size of the nodes in the database cluster, for example `db-s-1vcpu-1gb` indicates a 1 CPU, 1GB node. For a list of available size slugs, visit: https://docs.digitalocean.com/reference/api/api-reference/#tag/Databases"
	nodeNumberDetails := "The number of nodes in the database cluster. Valid values are 1-3. In addition to the primary node, up to two standby nodes may be added for high availability."
	storageSizeMiBDetails := "The amount of disk space allocated to the cluster. Applicable for PostgreSQL and MySQL clusters. Each plan size has a default value but can be increased in increments up to a maximum amount. For ranges, visit: https://www.digitalocean.com/pricing/managed-databases"
	cmdDatabaseCreate := CmdBuilder(cmd, RunDatabaseCreate, "create <name>", "Create a database cluster", `Creates a database cluster with the specified name.

You can customize the configuration using the listed flags, all of which are optional. Without any flags set, the command creates a single-node, single-CPU PostgreSQL database cluster.`, Writer,
		aliasOpt("c"))
	AddIntFlag(cmdDatabaseCreate, doctl.ArgDatabaseNumNodes, "", defaultDatabaseNodeCount, nodeNumberDetails)
	AddStringFlag(cmdDatabaseCreate, doctl.ArgRegionSlug, "", defaultDatabaseRegion, "The data center region where the database cluster resides, such as `nyc1` or `sfo2`.")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgSizeSlug, "", defaultDatabaseNodeSize, nodeSizeDetails)
	AddIntFlag(cmdDatabaseCreate, doctl.ArgDatabaseStorageSizeMib, "", 0, storageSizeMiBDetails)
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseEngine, "", defaultDatabaseEngine, "The database's engine. Possible values are: `pg`, `mysql`, `redis`, `mongodb`, `kafka` and `opensearch`.")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgVersion, "", "", "The database engine's version, such as 14 for PostgreSQL version 14.")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgPrivateNetworkUUID, "", "", "The UUID of a VPC to create the database cluster in. The command uses the region's default VPC if excluded.")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseRestoreFromClusterName, "", "", "The name of an existing database cluster to restore from.")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseRestoreFromTimestamp, "", "", "The timestamp of an existing database cluster backup in UTC combined date and time format (2006-01-02 15:04:05 +0000 UTC). The most recent backup is used if excluded.")
	AddBoolFlag(cmdDatabaseCreate, doctl.ArgCommandWait, "", false, "A boolean value that specifies whether to wait for the database cluster to be provisioned before returning control to the terminal.")
	AddStringSliceFlag(cmdDatabaseCreate, doctl.ArgTag, "", nil, "A comma-separated list of tags to apply to the database cluster.")
	cmdDatabaseCreate.Example = `The following example creates a database cluster named ` + "`" + `example-database` + "`" + ` in the ` + "`" + `nyc1` + "`" + ` region with a single  1 GB node: doctl databases create example-database --region nyc1 --size db-s-1vcpu-1gb --num-nodes 1`

	cmdDatabaseDelete := CmdBuilder(cmd, RunDatabaseDelete, "delete <database-cluster-id>", "Delete a database cluster", `Deletes the database cluster with the specified ID.

To retrieve a list of your database clusters and their IDs, use `+"`"+`doctl databases list`+"`"+`.`, Writer,
		aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the database cluster without a confirmation prompt")
	cmdDatabaseDelete.Example = `The following example deletes the database cluster with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl databases delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdDatabaseGetConn := CmdBuilder(cmd, RunDatabaseConnectionGet, "connection <database-cluster-id>", "Retrieve connection details for a database cluster", `Retrieves the following connection details for a database cluster:

- A connection string for the database cluster
- The default database name
- The fully-qualified domain name of the publicly-connectable host
- The port on which the database is listening for connections
- The default username
- The randomly-generated password for the default username
- A boolean value indicating if the connection should be made over SSL

While you can use these connection details, you can manually update the connection string's parameters to change how you connect to the database, such using a private hostname, custom username, or a different database.`, Writer,
		aliasOpt("conn"), displayerType(&displayers.DatabaseConnection{}))
	AddBoolFlag(cmdDatabaseGetConn, doctl.ArgDatabasePrivateConnectionBool, "", false, "Returns connection details that use the database's VPC network connection.")
	cmdDatabaseGetConn.Example = `The following example retrieves the connection details for a database cluster with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl databases connection f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdDatabaseListBackups := CmdBuilder(cmd, RunDatabaseBackupsList, "backups <database-cluster-id>", "List database cluster backups", `Retrieves a list of backups created for the specified database cluster.

The list contains the size in GB, and the date and time the backup was created.`, Writer,
		aliasOpt("bu"), displayerType(&displayers.DatabaseBackups{}))
	cmdDatabaseListBackups.Example = `The following example retrieves a list of backups for a database cluster with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl databases backups f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdDatabaseResize := CmdBuilder(cmd, RunDatabaseResize, "resize <database-cluster-id>", "Resize a database cluster", `Resizes the specified database cluster.

You must specify the desired number of nodes and size of the nodes. For example:

	doctl databases resize ca9f591d-9999-5555-a0ef-1c02d1d1e352 --num-nodes 2 --size db-s-16vcpu-64gb

Database nodes cannot be resized to smaller sizes due to the risk of data loss.

For PostgreSQL and MySQL clusters, you can also provide a disk size in MiB to scale the storage up to 15 TB, depending on your plan. You cannot reduce the storage size of a cluster.`, Writer,
		aliasOpt("rs"))
	AddIntFlag(cmdDatabaseResize, doctl.ArgDatabaseNumNodes, "", 0, nodeNumberDetails, requiredOpt())
	AddStringFlag(cmdDatabaseResize, doctl.ArgSizeSlug, "", "", nodeSizeDetails, requiredOpt())
	AddIntFlag(cmdDatabaseResize, doctl.ArgDatabaseStorageSizeMib, "", 0, storageSizeMiBDetails)
	cmdDatabaseResize.Example = `The following example resizes a PostgreSQL or MySQL database to have two nodes, 16 vCPUs, 64 GB of memory, and 2048 GiB of storage space: doctl databases resize ca9f591d-9999-5555-a0ef-1c02d1d1e352 --num-nodes 2 --size db-s-16vcpu-64gb --storage-size-mib 2048000`

	cmdDatabaseMigrate := CmdBuilder(cmd, RunDatabaseMigrate, "migrate <database-cluster-id>", "Migrate a database cluster to a new region", `Migrates the specified database cluster to a new region.`, Writer,
		aliasOpt("m"))
	AddStringFlag(cmdDatabaseMigrate, doctl.ArgRegionSlug, "", "", "The region to which the database cluster should be migrated, such as `sfo2` or `nyc3`.", requiredOpt())
	AddStringFlag(cmdDatabaseMigrate, doctl.ArgPrivateNetworkUUID, "", "", "The UUID of a VPC network to create the database cluster in. The command uses the region's default VPC network if not specified.")

	cmdDatabaseFork := CmdBuilder(cmd, RunDatabaseFork, "fork <name>", "Create a new database cluster by forking an existing database cluster.", `Creates a new database cluster from an existing cluster. The forked database contains all of the data from the original database at the time the fork is created.`, Writer, aliasOpt("f"))
	AddStringFlag(cmdDatabaseFork, doctl.ArgDatabaseRestoreFromClusterID, "", "", "The ID of an existing database cluster from which the new database will be forked from", requiredOpt())
	AddStringFlag(cmdDatabaseFork, doctl.ArgDatabaseRestoreFromTimestamp, "", "", "The timestamp of an existing database cluster backup in UTC combined date and time format (2006-01-02 15:04:05 +0000 UTC). The most recent backup is used if excluded.")
	AddBoolFlag(cmdDatabaseFork, doctl.ArgCommandWait, "", false, "A boolean that specifies whether to wait for a database to complete before returning control to the terminal")

	cmdDatabaseFork.Example = `The following example forks a database cluster with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to create a new database cluster. The command also uses the ` + "`" + `--restore-from-timestamp` + "`" + ` flag to specifically fork the database from a cluster backup that was created on 2023 November 7: doctl databases fork new-db-cluster --restore-from-cluster-id f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --restore-from-timestamp 2023-11-07 12:34:56 +0000 UTC`

	cmd.AddCommand(databaseReplica())
	cmd.AddCommand(databaseMaintenanceWindow())
	cmd.AddCommand(databaseUser())
	cmd.AddCommand(databaseDB())
	cmd.AddCommand(databasePool())
	cmd.AddCommand(sqlMode())
	cmd.AddCommand(databaseFirewalls())
	cmd.AddCommand(databaseOptions())
	cmd.AddCommand(databaseConfiguration())
	cmd.AddCommand(databaseTopic())
	cmd.AddCommand(databaseEvents())

	return cmd
}

// Clusters

// RunDatabaseList returns a list of database clusters.
func RunDatabaseList(c *CmdConfig) error {
	dbs, err := c.Databases().List()
	if err != nil {
		return err
	}

	return displayDatabases(c, true, dbs...)
}

// RunDatabaseGet returns an individual database cluster
func RunDatabaseGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]
	db, err := c.Databases().Get(id)
	if err != nil {
		return err
	}

	return displayDatabases(c, false, *db)
}

// RunDatabaseCreate creates a database cluster
func RunDatabaseCreate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	r, err := buildDatabaseCreateRequestFromArgs(c)
	if err != nil {
		return err
	}

	dbs := c.Databases()

	db, err := dbs.Create(r)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		connection := db.Connection
		dbs := c.Databases()
		notice("Database creation is in progress, waiting for database to be online")

		err := waitForDatabaseReady(dbs, db.ID)
		if err != nil {
			return fmt.Errorf(
				"database couldn't enter the `online` state: %v",
				err,
			)
		}

		db, err = dbs.Get(db.ID)
		if err != nil {
			return fmt.Errorf(
				"failed to retrieve the new database: %v",
				err,
			)
		}
		db.Connection = connection
	}

	notice("Database created")

	return displayDatabases(c, false, *db)
}

func buildDatabaseCreateRequestFromArgs(c *CmdConfig) (*godo.DatabaseCreateRequest, error) {
	r := &godo.DatabaseCreateRequest{Name: c.Args[0]}

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return nil, err
	}
	r.Region = region

	numNodes, err := c.Doit.GetInt(c.NS, doctl.ArgDatabaseNumNodes)
	if err != nil {
		return nil, err
	}
	r.NumNodes = numNodes

	size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return nil, err
	}
	r.SizeSlug = size

	engine, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseEngine)
	if err != nil {
		return nil, err
	}
	r.EngineSlug = engine

	version, err := c.Doit.GetString(c.NS, doctl.ArgVersion)
	if err != nil {
		return nil, err
	}
	r.Version = version

	privateNetworkUUID, err := c.Doit.GetString(c.NS, doctl.ArgPrivateNetworkUUID)
	if err != nil {
		return nil, err
	}
	r.PrivateNetworkUUID = privateNetworkUUID

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return nil, err
	}
	r.Tags = tags

	restoreFromCluster, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseRestoreFromClusterName)
	if err != nil {
		return nil, err
	}
	if restoreFromCluster != "" {
		backUpRestore := &godo.DatabaseBackupRestore{}
		backUpRestore.DatabaseName = restoreFromCluster
		// only set the restore-from-timestamp if restore-from-cluster is set.
		restoreFromTimestamp, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseRestoreFromTimestamp)
		if err != nil {
			return nil, err
		}
		if restoreFromTimestamp != "" {
			dateFormatted, err := convertUTCtoISO8601(restoreFromTimestamp)
			if err != nil {
				return nil, err
			}
			backUpRestore.BackupCreatedAt = dateFormatted
		}
		r.BackupRestore = backUpRestore
	}

	r.PrivateNetworkUUID = privateNetworkUUID

	storageSizeMibInt, err := c.Doit.GetInt(c.NS, doctl.ArgDatabaseStorageSizeMib)
	if err != nil {
		return nil, err
	}
	r.StorageSizeMib = uint64(storageSizeMibInt)

	return r, nil
}

// RunDatabaseFork creates a database cluster by forking an existing cluster.
func RunDatabaseFork(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	r, err := buildDatabaseForkRequest(c)
	if err != nil {
		return err
	}

	dbs := c.Databases()

	db, err := dbs.Create(r)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		connection := db.Connection
		dbs := c.Databases()
		notice("Database forking is in progress, waiting for database to be online")

		err := waitForDatabaseReady(dbs, db.ID)
		if err != nil {
			return fmt.Errorf(
				"database couldn't enter the `online` state: %v",
				err,
			)
		}

		db, _ = dbs.Get(db.ID)
		db.Connection = connection
	}

	notice("Database created")

	return displayDatabases(c, false, *db)
}

func buildDatabaseForkRequest(c *CmdConfig) (*godo.DatabaseCreateRequest, error) {
	r := &godo.DatabaseCreateRequest{Name: c.Args[0]}

	existingDatabaseID, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseRestoreFromClusterID)
	if err != nil {
		return nil, err
	}

	existingDatabase, err := c.Databases().Get(existingDatabaseID)
	if err != nil {
		return nil, err
	}

	backUpRestore := &godo.DatabaseBackupRestore{}
	backUpRestore.DatabaseName = existingDatabase.Name
	restoreFromTimestamp, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseRestoreFromTimestamp)
	if err != nil {
		return nil, err
	}
	if restoreFromTimestamp != "" {
		dateFormatted, err := convertUTCtoISO8601(restoreFromTimestamp)
		if err != nil {
			return nil, err
		}
		backUpRestore.BackupCreatedAt = dateFormatted
	}

	r.BackupRestore = backUpRestore
	r.EngineSlug = existingDatabase.EngineSlug
	r.NumNodes = existingDatabase.NumNodes
	r.SizeSlug = existingDatabase.SizeSlug
	r.Region = existingDatabase.RegionSlug
	r.Version = existingDatabase.VersionSlug
	r.PrivateNetworkUUID = existingDatabase.PrivateNetworkUUID
	r.Tags = existingDatabase.Tags
	r.ProjectID = existingDatabase.ProjectID

	return r, nil
}

func convertUTCtoISO8601(restoreFromTimestamp string) (string, error) {
	// accepts UTC time format from user (to match db list output) and converts it to ISO8601 for api parity.
	date, error := time.Parse("2006-01-02 15:04:05 +0000 UTC", restoreFromTimestamp)
	if error != nil {
		return "", fmt.Errorf("Invalid format for --restore-from-timestamp. Must be in UTC format: 2006-01-02 15:04:05 +0000 UTC")
	}
	dateFormatted := date.Format(time.RFC3339)

	return dateFormatted, nil
}

// RunDatabaseDelete deletes a database cluster
func RunDatabaseDelete(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("database cluster", 1) == nil {
		id := c.Args[0]
		return c.Databases().Delete(id)
	}

	return errOperationAborted
}

func displayDatabases(c *CmdConfig, short bool, dbs ...do.Database) error {
	item := &displayers.Databases{
		Databases: do.Databases(dbs),
		Short:     short,
	}
	return c.Display(item)
}

// RunDatabaseConnectionGet gets database connection info
func RunDatabaseConnectionGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]
	private, err := c.Doit.GetBool(c.NS, doctl.ArgDatabasePrivateConnectionBool)
	if err != nil {
		return err
	}

	connInfo, err := c.Databases().GetConnection(id, private)
	if err != nil {
		return err
	}

	return displayDatabaseConnection(c, *connInfo)
}

func displayDatabaseConnection(c *CmdConfig, conn do.DatabaseConnection) error {
	item := &displayers.DatabaseConnection{DatabaseConnection: conn}
	return c.Display(item)
}

// RunDatabaseBackupsList lists all the backups for a database cluster
func RunDatabaseBackupsList(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]
	backups, err := c.Databases().ListBackups(id)
	if err != nil {
		return err
	}

	return displayDatabaseBackups(c, backups)
}

func displayDatabaseBackups(c *CmdConfig, bu do.DatabaseBackups) error {
	item := &displayers.DatabaseBackups{DatabaseBackups: bu}
	return c.Display(item)
}

// RunDatabaseResize resizes a database cluster
func RunDatabaseResize(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	r, err := buildDatabaseResizeRequestFromArgs(c)
	if err != nil {
		return err
	}

	return c.Databases().Resize(id, r)
}

func buildDatabaseResizeRequestFromArgs(c *CmdConfig) (*godo.DatabaseResizeRequest, error) {
	r := &godo.DatabaseResizeRequest{}

	numNodes, err := c.Doit.GetInt(c.NS, doctl.ArgDatabaseNumNodes)
	if err != nil {
		return nil, err
	}
	r.NumNodes = numNodes

	size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return nil, err
	}
	r.SizeSlug = size

	storageSizeMibInt, err := c.Doit.GetInt(c.NS, doctl.ArgDatabaseStorageSizeMib)
	if err != nil {
		return nil, err
	}
	r.StorageSizeMib = uint64(storageSizeMibInt)

	return r, nil
}

// RunDatabaseMigrate migrates a database cluster to a new region
func RunDatabaseMigrate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	r, err := buildDatabaseMigrateRequestFromArgs(c)
	if err != nil {
		return err
	}

	return c.Databases().Migrate(id, r)
}

func buildDatabaseMigrateRequestFromArgs(c *CmdConfig) (*godo.DatabaseMigrateRequest, error) {
	r := &godo.DatabaseMigrateRequest{}

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return nil, err
	}
	r.Region = region

	privateNetworkUUID, err := c.Doit.GetString(c.NS, doctl.ArgPrivateNetworkUUID)
	if err != nil {
		return nil, err
	}
	r.PrivateNetworkUUID = privateNetworkUUID

	return r, nil
}

func databaseMaintenanceWindow() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "maintenance-window",
			Aliases: []string{"maintenance", "mw", "main"},
			Short:   "Display commands for scheduling automatic maintenance on your database cluster",
			Long: `The ` + "`" + `doctl databases maintenance-window` + "`" + ` commands allow you to schedule, and check the schedule of, maintenance windows for your databases.

Maintenance windows are hour-long blocks of time during which DigitalOcean performs automatic maintenance on databases every week. During this time, health checks, security updates, version upgrades, and more are performed.`,
		},
	}

	cmdMaintenanceGet := CmdBuilder(cmd, RunDatabaseMaintenanceGet, "get <database-cluster-id>",
		"Retrieve details about a database cluster's maintenance windows", `Retrieves the following information on currently-scheduled maintenance windows for the specified database cluster:

- The day of the week the maintenance window occurs
- The hour in UTC when maintenance updates will be applied, in 24 hour format, such as "16:00"
- A boolean representing whether maintenance updates are currently pending

To see a list of your databases and their IDs, run `+"`"+`doctl databases list`+"`"+`.`, Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseMaintenanceWindow{}))
	cmdMaintenanceGet.Example = `The following example retrieves the maintenance window for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases maintenance-window ca9f591d-f38h-5555-a0ef-1c02d1d1e35`

	cmdDatabaseCreate := CmdBuilder(cmd, RunDatabaseMaintenanceUpdate,
		"update <database-cluster-id>", "Update the maintenance window for a database cluster", `Updates the maintenance window for the specified database cluster.

Maintenance windows are hour-long blocks of time during which DigitalOcean performs automatic maintenance on databases every week. During this time, health checks, security updates, version upgrades, and more are performed.

To change the maintenance window for your database cluster, specify a day of the week and an hour of that day during which you would prefer such maintenance would occur.

To see a list of your databases and their IDs, run `+"`"+`doctl databases list`+"`"+`.`, Writer, aliasOpt("u"))
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseMaintenanceDay, "", "",
		"The day of the week the maintenance window occurs, for example: 'tuesday')", requiredOpt())
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseMaintenanceHour, "", "",
		"The hour when maintenance updates are applied, in UTC 24-hour format. Example: '16:00')", requiredOpt())
	cmdDatabaseCreate.Example = `The following example updates the maintenance window for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases maintenance-window update ca9f591d-f38h-5555-a0ef-1c02d1d1e35 --day tuesday --hour 16:00`

	return cmd
}

// Database Maintenance Window

// RunDatabaseMaintenanceGet retrieves the maintenance window info for a database cluster
func RunDatabaseMaintenanceGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	window, err := c.Databases().GetMaintenance(id)
	if err != nil {
		return err
	}

	return displayDatabaseMaintenanceWindow(c, *window)
}

func displayDatabaseMaintenanceWindow(c *CmdConfig, mw do.DatabaseMaintenanceWindow) error {
	item := &displayers.DatabaseMaintenanceWindow{DatabaseMaintenanceWindow: mw}
	return c.Display(item)
}

// RunDatabaseMaintenanceUpdate updates the maintenance window info for a database cluster
func RunDatabaseMaintenanceUpdate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]
	r, err := buildDatabaseUpdateMaintenanceRequestFromArgs(c)
	if err != nil {
		return err
	}

	return c.Databases().UpdateMaintenance(id, r)
}

func buildDatabaseUpdateMaintenanceRequestFromArgs(c *CmdConfig) (*godo.DatabaseUpdateMaintenanceRequest, error) {
	r := &godo.DatabaseUpdateMaintenanceRequest{}

	day, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseMaintenanceDay)
	if err != nil {
		return nil, err
	}
	r.Day = strings.ToLower(day)

	hour, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseMaintenanceHour)
	if err != nil {
		return nil, err
	}
	r.Hour = hour

	return r, nil
}

func databaseUser() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "user",
			Aliases: []string{"u"},
			Short:   "Display commands for managing database users",
			Long: `The commands under ` + "`" + `doctl databases user` + "`" + ` allow you to view details for, and create, database users.

Database user accounts are scoped to one database cluster, to which they have full admin access, and are given an automatically-generated password.`,
		},
	}
	databaseKafkaACLsTxt := `A comma-separated list of kafka ACL rules, in ` + "`" + `topic:permission` + "`" + ` format.`
	userDetailsDesc := `

- The username for the user
- The password for the user
- The user's role, either "primary" or "normal"

Primary user accounts are created by DigitalOcean at database cluster creation time and can't be deleted. You can create additional users with a "normal" role. Both have administrative privileges on the database cluster.

To retrieve a list of your databases and their IDs, call ` + "`" + `doctl databases list` + "`" + `.`
	cmdDatabaseUserList := CmdBuilder(cmd, RunDatabaseUserList, "list <database-cluster-id>", "Retrieve list of database users",
		`Retrieves a list of users for the specified database with the following details:`+userDetailsDesc, Writer, aliasOpt("ls"), displayerType(&displayers.DatabaseUsers{}))
	cmdDatabaseUserList.Example = `The following example retrieves a list of users for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + ` and uses the ` + "`" + `--format flag` + "`" + ` to return only the name and role for each each user: doctl databases user list ca9f591d-f38h-5555-a0ef-1c02d1d1e35 --format Name,Role`

	cmdDatabaseUserGet := CmdBuilder(cmd, RunDatabaseUserGet, "get <database-cluster-id> <user-name>",
		"Retrieve details about a database user", `Retrieves the following details about the specified user:`+userDetailsDesc+`

To retrieve a list of database users for a database cluster, call `+"`"+`doctl databases user list <database-cluster-id>`+"`"+`.`, Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseUsers{}))
	cmdDatabaseUserGet.Example = `The following example retrieves the details for the user with the username ` + "`" + `example-user` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + ` and uses the ` + "`" + `--format` + "`" + ` flag to return only the user's name and role: doctl databases user get ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-user --format Name,Role`

	cmdDatabaseUserCreate := CmdBuilder(cmd, RunDatabaseUserCreate, "create <database-cluster-id> <user-name>",
		"Create a database user", `Creates a new user for a database. New users are given a role of `+"`"+`normal`+"`"+` and are given an automatically-generated password.

To retrieve a list of your databases and their IDs, call `+"`"+`doctl databases list`+"`"+`.`, Writer, aliasOpt("c"))

	AddStringFlag(cmdDatabaseUserCreate, doctl.ArgDatabaseUserMySQLAuthPlugin, "", "",
		"Sets authorization plugin for a MySQL user. Possible values: `caching_sha2_password` or `mysql_native_password`")
	AddStringSliceFlag(cmdDatabaseUserCreate, doctl.ArgDatabaseUserKafkaACLs, "", []string{}, databaseKafkaACLsTxt)
	cmdDatabaseUserCreate.Example = `The following example creates a new user with the username ` + "`" + `example-user` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases user create ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-user`

	cmdDatabaseUserResetAuth := CmdBuilder(cmd, RunDatabaseUserResetAuth, "reset <database-cluster-id> <user-name> <new-auth-mode>",
		"Resets a user's auth", "Resets the auth password or the MySQL authorization plugin for a given user and returns the user's new credentials. When resetting MySQL auth, valid values for `<new-auth-mode>` are `caching_sha2_password` and `mysql_native_password`.", Writer, aliasOpt("rs"))
	cmdDatabaseUserResetAuth.Example = `The following example resets the auth plugin for the user with the username ` + "`" + `example-user` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + ` to ` + "`" + `mysql_native_password` + "`" + `: doctl databases user reset ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-user mysql_native_password`

	cmdDatabaseUserDelete := CmdBuilder(cmd, RunDatabaseUserDelete,
		"delete <database-cluster-id> <user-id>", "Delete a database user", `Deletes the specified database user.

To retrieve a list of your databases and their IDs, call `+"`"+`doctl databases list`+"`"+`.`, Writer, aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseUserDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the user without a confirmation prompt")
	cmdDatabaseUserDelete.Example = `The following example deletes the user with the username ` + "`" + `example-user` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases user delete ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-user`

	return cmd
}

// Database Users

// RunDatabaseUserList retrieves a list of users for specific database cluster
func RunDatabaseUserList(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	users, err := c.Databases().ListUsers(id)
	if err != nil {
		return err
	}

	return displayDatabaseUsers(c, users...)
}

// RunDatabaseUserGet retrieves a database user for a specific database cluster
func RunDatabaseUserGet(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	userID := c.Args[1]

	user, err := c.Databases().GetUser(databaseID, userID)
	if err != nil {
		return err
	}

	return displayDatabaseUsers(c, *user)
}

// RunDatabaseUserCreate creates a database user for a database cluster
func RunDatabaseUserCreate(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	var (
		databaseID = c.Args[0]
		userName   = c.Args[1]
	)

	req := &godo.DatabaseCreateUserRequest{Name: userName}

	authMode, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseUserMySQLAuthPlugin)
	if err != nil {
		return err
	}

	if authMode != "" {
		req.MySQLSettings = &godo.DatabaseMySQLUserSettings{
			AuthPlugin: authMode,
		}
	}

	kafkaAcls, err := buildDatabaseCreateKafkaUserACls(c)
	if err != nil {
		return err
	}

	if len(kafkaAcls) != 0 {
		req.Settings = &godo.DatabaseUserSettings{
			ACL: kafkaAcls,
		}
	}

	user, err := c.Databases().CreateUser(databaseID, req)
	if err != nil {
		return err
	}

	return displayDatabaseUsers(c, *user)
}

func buildDatabaseCreateKafkaUserACls(c *CmdConfig) (kafkaACls []*godo.KafkaACL, err error) {
	acls, err := c.Doit.GetStringSlice(c.NS, doctl.ArgDatabaseUserKafkaACLs)
	if err != nil {
		return nil, err
	}
	for _, acl := range acls {
		pair := strings.SplitN(acl, ":", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("Unexpected input value [%v], must be a topic:permission pair", pair)
		}

		kafkaACl := new(godo.KafkaACL)
		kafkaACl.Topic = pair[0]
		kafkaACl.Permission = pair[1]

		kafkaACls = append(kafkaACls, kafkaACl)
	}
	return kafkaACls, nil
}

func RunDatabaseUserResetAuth(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	var (
		databaseID = c.Args[0]
		userName   = c.Args[1]
	)

	database, err := c.Databases().Get(databaseID)

	if err != nil {
		return err
	}

	var req *godo.DatabaseResetUserAuthRequest
	if strings.ToLower(database.EngineSlug) == "mysql" {
		if len(c.Args) < 3 {
			return doctl.NewMissingArgsErr(c.NS)
		}
		authMode := c.Args[2]
		req = &godo.DatabaseResetUserAuthRequest{
			MySQLSettings: &godo.DatabaseMySQLUserSettings{
				AuthPlugin: authMode,
			},
		}
	} else {
		req = &godo.DatabaseResetUserAuthRequest{}
	}

	user, err := c.Databases().ResetUserAuth(databaseID, userName, req)
	if err != nil {
		return err
	}

	return displayDatabaseUsers(c, *user)
}

// RunDatabaseUserDelete deletes a database user
func RunDatabaseUserDelete(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("database user", 1) == nil {
		databaseID := c.Args[0]
		userID := c.Args[1]
		return c.Databases().DeleteUser(databaseID, userID)
	}

	return errOperationAborted
}

func displayDatabaseUsers(c *CmdConfig, users ...do.DatabaseUser) error {
	item := &displayers.DatabaseUsers{DatabaseUsers: users}
	return c.Display(item)
}

func databaseOptions() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "options",
			Aliases: []string{"o"},
			Short:   `Display available database options (regions, version, layouts, etc.) for all available database engines`,
			Long:    `The subcommands under ` + "`" + `doctl databases options` + "`" + ` retrieve configuration options for databases, such as available engines, engine versions and their equivalent slugs.`,
		},
	}

	cmdEngineOptions := CmdBuilder(cmd, RunDatabaseEngineOptions, "engines", "Retrieves a list of the available database engines", `Lists the available database engines for DigitalOcean Managed Databases.`,
		Writer, aliasOpt("eng"))
	cmdEngineOptions.Example = `The following example retrieves a list of the available database engines: doctl databases options engines`

	cmdRegionOptions := CmdBuilder(cmd, RunDatabaseRegionOptions, "regions", "Retrieves a list of the available regions for a given database engine", `Lists the available regions for a given database engine. Some engines may not be available in certain regions.`,
		Writer, aliasOpt("r"))
	AddStringFlag(cmdRegionOptions, doctl.ArgDatabaseEngine, "",
		"", `The database engine. Possible values:  `+"`"+`mysql`+"`"+`,  `+"`"+`pg`+"`"+`,  `+"`"+`redis`+"`"+`,  `+"`"+`kafka`+"`"+`, `+"`"+`opensearch`+"`"+`,  `+"`"+`mongodb`+"`"+``)
	cmdRegionOptions.Example = `The following example retrieves a list of the available regions for the PostgreSQL engine: doctl databases options regions --engine pg`

	cmdVersionOptions := CmdBuilder(cmd, RunDatabaseVersionOptions, "versions", "Retrieves a list of the available versions for a given database engine", `Lists the available versions for a given database engine.`,
		Writer, aliasOpt("v"))
	AddStringFlag(cmdVersionOptions, doctl.ArgDatabaseEngine, "",
		"", `The database engine. Possible values:  `+"`"+`mysql`+"`"+`,  `+"`"+`pg`+"`"+`,  `+"`"+`redis`+"`"+`,  `+"`"+`kafka`+"`"+`,  `+"`"+`opensearch`+"`"+`, `+"`"+`mongodb`+"`"+``)
	cmdVersionOptions.Example = `The following example retrieves a list of the available versions for the PostgreSQL engine: doctl databases options versions --engine pg`

	cmdSlugOptions := CmdBuilder(cmd, RunDatabaseSlugOptions, "slugs", "Retrieves a list of the available slugs for a given database engine", `Lists the available slugs for a given database engine.`,
		Writer, aliasOpt("s"))
	AddStringFlag(cmdSlugOptions, doctl.ArgDatabaseEngine, "",
		"", `The database engine. Possible values:  `+"`"+`mysql`+"`"+`,  `+"`"+`pg`+"`"+`,  `+"`"+`redis`+"`"+`,  `+"`"+`kafka`+"`"+`,  `+"`"+`opensearch`+"`"+`, `+"`"+`mongodb`+"`"+``, requiredOpt())
	cmdSlugOptions.Example = `The following example retrieves a list of the available slugs for the PostgreSQL engine: doctl databases options slugs --engine pg`

	return cmd
}

// RunDatabaseEngineOptions retrieves a list of the available database engines
func RunDatabaseEngineOptions(c *CmdConfig) error {
	options, err := c.Databases().ListOptions()
	if err != nil {
		return err
	}

	return displayDatabaseEngineOptions(c, options)
}

func displayDatabaseEngineOptions(c *CmdConfig, options *do.DatabaseOptions) error {
	item := &displayers.DatabaseOptions{DatabaseOptions: *options}
	return c.Display(item)
}

func displayDatabaseRegionOptions(c *CmdConfig, regions map[string][]string) error {
	item := &displayers.DatabaseRegionOptions{RegionMap: regions}
	return c.Display(item)
}

func displayDatabaseVersionOptions(c *CmdConfig, versions map[string][]string) error {
	item := &displayers.DatabaseVersionOptions{VersionMap: versions}
	return c.Display(item)
}

func displayDatabaseLayoutOptions(c *CmdConfig, layouts []godo.DatabaseLayout) error {
	item := &displayers.DatabaseLayoutOptions{Layouts: layouts}
	return c.Display(item)
}

// RunDatabaseRegionOptions retrieves a list of the available regions for a given database engine
func RunDatabaseRegionOptions(c *CmdConfig) error {
	engine, _ := c.Doit.GetString(c.NS, doctl.ArgDatabaseEngine)

	options, err := c.Databases().ListOptions()
	if err != nil {
		return err
	}

	regions := make(map[string][]string, 0)
	switch engine {
	case "mongodb":
		regions["mongodb"] = options.MongoDBOptions.Regions
	case "mysql":
		regions["mysql"] = options.MySQLOptions.Regions
	case "pg":
		regions["pg"] = options.PostgresSQLOptions.Regions
	case "redis":
		regions["redis"] = options.RedisOptions.Regions
	case "kafka":
		regions["kafka"] = options.KafkaOptions.Regions
	case "opensearch":
		regions["opensearch"] = options.OpensearchOptions.Regions
	case "":
		regions["mongodb"] = options.MongoDBOptions.Regions
		regions["mysql"] = options.MySQLOptions.Regions
		regions["pg"] = options.PostgresSQLOptions.Regions
		regions["redis"] = options.RedisOptions.Regions
		regions["kafka"] = options.KafkaOptions.Regions
		regions["opensearch"] = options.OpensearchOptions.Regions
	}

	return displayDatabaseRegionOptions(c, regions)
}

// RunDatabaseVersionOptions retrieves a list of the available versions for a given database engine
func RunDatabaseVersionOptions(c *CmdConfig) error {
	engine, _ := c.Doit.GetString(c.NS, doctl.ArgDatabaseEngine)

	options, err := c.Databases().ListOptions()
	if err != nil {
		return err
	}

	versions := make(map[string][]string, 0)
	switch engine {
	case "mongodb":
		versions["mongodb"] = options.MongoDBOptions.Versions
	case "mysql":
		versions["mysql"] = options.MySQLOptions.Versions
	case "pg":
		versions["pg"] = options.PostgresSQLOptions.Versions
	case "redis":
		versions["redis"] = options.RedisOptions.Versions
	case "kafka":
		versions["kafka"] = options.KafkaOptions.Versions
	case "opensearch":
		versions["opensearch"] = options.OpensearchOptions.Versions
	case "":
		versions["mongodb"] = options.MongoDBOptions.Versions
		versions["mysql"] = options.MySQLOptions.Versions
		versions["pg"] = options.PostgresSQLOptions.Versions
		versions["redis"] = options.RedisOptions.Versions
		versions["kafka"] = options.KafkaOptions.Versions
		versions["opensearch"] = options.OpensearchOptions.Versions
	}

	return displayDatabaseVersionOptions(c, versions)
}

// RunDatabaseSlugOptions retrieves a list of the available slugs for a given database engine
func RunDatabaseSlugOptions(c *CmdConfig) error {
	engine, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseEngine)
	if err != nil {
		return doctl.NewMissingArgsErr(c.NS)
	}

	options, err := c.Databases().ListOptions()
	if err != nil {
		return err
	}

	layouts := make([]godo.DatabaseLayout, 0)
	switch engine {
	case "mongodb":
		layouts = options.MongoDBOptions.Layouts
	case "mysql":
		layouts = options.MySQLOptions.Layouts
	case "pg":
		layouts = options.PostgresSQLOptions.Layouts
	case "redis":
		layouts = options.RedisOptions.Layouts
	case "kafka":
		layouts = options.KafkaOptions.Layouts
	case "opensearch":
		layouts = options.OpensearchOptions.Layouts
	}

	return displayDatabaseLayoutOptions(c, layouts)
}

func databasePool() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "pool",
			Aliases: []string{"p"},
			Short:   "Display commands for managing connection pools",
			Long: `The subcommands under ` + "`" + `doctl databases pool` + "`" + ` manage connection pools for your database cluster.

A connection pool may be useful if your database:

- Typically handles a large number of idle connections
- Has wide variability in the possible number of connections at any given time
- Drops connections due to max connection limits
- Experiences performance issues due to high CPU usage
`,
		},
	}

	connectionPoolDetails := `

- The database user that the connection pool uses. When excluded, all connections to the database use the inbound user.
- The connection pool's name
- The connection pool's size
- The database within the cluster that the connection pool connects to
- The pool mode for the connection pool. Possible values: ` + "`" + `session` + "`" + `, ` + "`" + `transaction` + "`" + `, or ` + "`" + `statement` + "`" + `
- A connection string for the connection pool`
	getPoolDetails := `

You can get a list of existing connection pools by calling:

	doctl databases pool list <database-cluster-id>

You can get a list of existing database clusters and their IDs by calling:

	doctl databases list`

	cmdDatabasePoolList := CmdBuilder(cmd, RunDatabasePoolList, "list <database-cluster-id>", "List connection pools for a database cluster", `Lists the existing connection pools for the specified database. The command returns the following details about each connection pool:`+connectionPoolDetails,
		Writer, aliasOpt("ls"), displayerType(&displayers.DatabasePools{}))
	cmdDatabasePoolList.Example = `The following example lists the connection pools for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + ` and uses the ` + "`" + `--format` + "`" + ` flag to return only each pool's name and connection string: doctl databases pool list ca9f591d-f38h-5555-a0ef-1c02d1d1e35 --format Name,URI`

	cmdDatabasePoolGet := CmdBuilder(cmd, RunDatabasePoolGet, "get <database-cluster-id> <pool-name>",
		"Retrieve information about a database connection pool", `This command retrieves the following information about the specified connection pool for the specified database cluster:`+connectionPoolDetails+getPoolDetails, Writer, aliasOpt("g"),
		displayerType(&displayers.DatabasePools{}))
	cmdDatabasePoolGet.Example = `The following example retrieves the details for a connection pool named ` + "`" + `example-pool` + "`" + ` and uses the ` + "`" + `--format` + "`" + ` flag to return only the pool's name and connection string: doctl databases pool get ca9f591d-fb58-5555-a0ef-1c02d1d1e352 example-pool --format Name,URI`

	cmdDatabasePoolCreate := CmdBuilder(cmd, RunDatabasePoolCreate,
		"create <database-cluster-id> <pool-name>", "Create a connection pool for a database cluster", `Creates a connection pool for the specified database cluster.

In addition to the pool's name, you must also use flags to specify the pool's target database, its size, and a database user that the pool uses to authenticate. If you do not specify a user, the field is set to inbound user. An example call would be:

The pool size is the minimum number of connections the pool can handle. The maximum pool size varies based on the size of the cluster.

There’s no perfect formula to determine how large your pool should be, but there are a few good guidelines to keep in mind:

- A large pool stresses your database at similar levels as that number of clients would alone.
- A pool that’s much smaller than the number of clients communicating with the database can act as a bottleneck, reducing the rate when your database receives and responds to transactions.

We recommend starting with a pool size of about half your available connections and adjusting later based on performance. If you see slow query responses, check the CPU usage on the database’s Overview tab. We recommend decreasing your pool size if CPU usage is high, and increasing your pool size if it’s low.`+getPoolDetails, Writer,
		aliasOpt("c"))
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolMode, "",
		"transaction", "The pool mode for the connection pool, such as `session`, `transaction`, and `statement`")
	AddIntFlag(cmdDatabasePoolCreate, doctl.ArgSizeSlug, "", 0, "pool size",
		requiredOpt())
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolUserName, "", "",
		"The username for the database user")
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolDBName, "", "",
		"The name of the specific database within the database cluster", requiredOpt())
	cmdDatabasePoolCreate.Example = `The following example creates a connection pool named ` + "`" + `example-pool` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `. The command uses the ` + "`" + `--size` + "`" + ` flag to set the pool size to 10 and sets the user to the database's default user: doctl databases pool create ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-pool --size 10`

	cmdDatabasePoolDelete := CmdBuilder(cmd, RunDatabasePoolDelete,
		"delete <database-cluster-id> <pool-name>", "Delete a connection pool for a database", `Deletes the specified connection pool for the specified database cluster.`+getPoolDetails, Writer,
		aliasOpt("rm"))
	AddBoolFlag(cmdDatabasePoolDelete, doctl.ArgForce, doctl.ArgShortForce,
		false, "Delete the connection pool without confirmation prompt")
	cmdDatabasePoolDelete.Example = `The following example deletes a connection pool named ` + "`" + `example-pool` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases pool delete ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-pool`

	return cmd
}

// Database Pools

// RunDatabasePoolList retrieves a list of pools for specific database cluster
func RunDatabasePoolList(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	pools, err := c.Databases().ListPools(id)
	if err != nil {
		return err
	}

	return displayDatabasePools(c, pools...)
}

// RunDatabasePoolGet retrieves a database pool for a specific database cluster
func RunDatabasePoolGet(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	poolID := c.Args[1]

	pool, err := c.Databases().GetPool(databaseID, poolID)
	if err != nil {
		return err
	}

	return displayDatabasePools(c, *pool)
}

// RunDatabasePoolCreate creates a database pool for a database cluster
func RunDatabasePoolCreate(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	r, err := buildDatabaseCreatePoolRequestFromArgs(c)
	if err != nil {
		return err
	}

	pool, err := c.Databases().CreatePool(databaseID, r)
	if err != nil {
		return err
	}

	return displayDatabasePools(c, *pool)
}

func buildDatabaseCreatePoolRequestFromArgs(c *CmdConfig) (*godo.DatabaseCreatePoolRequest, error) {
	req := &godo.DatabaseCreatePoolRequest{Name: c.Args[1]}

	mode, err := c.Doit.GetString(c.NS, doctl.ArgDatabasePoolMode)
	if err != nil {
		return nil, err
	}
	req.Mode = mode

	size, err := c.Doit.GetInt(c.NS, doctl.ArgDatabasePoolSize)
	if err != nil {
		return nil, err
	}
	req.Size = size

	db, err := c.Doit.GetString(c.NS, doctl.ArgDatabasePoolDBName)
	if err != nil {
		return nil, err
	}
	req.Database = db

	user, err := c.Doit.GetString(c.NS, doctl.ArgDatabasePoolUserName)
	if err != nil {
		return nil, err
	}
	req.User = user

	return req, nil
}

// RunDatabasePoolDelete deletes a database pool
func RunDatabasePoolDelete(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("database pool", 1) == nil {
		databaseID := c.Args[0]
		poolID := c.Args[1]
		return c.Databases().DeletePool(databaseID, poolID)
	}

	return errOperationAborted
}

func displayDatabasePools(c *CmdConfig, pools ...do.DatabasePool) error {
	item := &displayers.DatabasePools{DatabasePools: pools}
	return c.Display(item)
}

func databaseDB() *Command {
	getClusterList := `

You can get a list of existing database clusters and their IDs by calling:

	doctl databases list`
	getDBList := `

You can get a list of existing databases that are hosted within a cluster by calling:

	doctl databases db list <cluster-id>`
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "db",
			Short: "Display commands for managing individual databases within a cluster",
			Long:  `The subcommands under ` + "`" + `doctl databases db` + "`" + ` are for managing specific databases that are served by a database cluster.` + getClusterList,
		},
	}

	cmdDatabaseDBList := CmdBuilder(cmd, RunDatabaseDBList, "list <database-cluster-id>", "Retrieve a list of databases within a cluster", "Retrieves a list of databases being hosted in the specified database cluster."+getClusterList, Writer,
		aliasOpt("ls"), displayerType(&displayers.DatabaseDBs{}))
	cmdDatabaseDBList.Example = `The following example retrieves a list of databases in a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases db list ca9f591d-f38h-5555-a0ef-1c02d1d1e35`

	cmdDatabaseDBGet := CmdBuilder(cmd, RunDatabaseDBGet, "get <database-cluster-id> <database-name>", "Retrieve the name of a database within a cluster", "Retrieves the name of the specified database hosted in the specified database cluster."+getClusterList+getDBList,
		Writer, aliasOpt("g"), displayerType(&displayers.DatabaseDBs{}))
	cmdDatabaseDBGet.Example = `The following example retrieves the name of a database in a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + ` and the name ` + "`" + `example-db` + "`" + `: doctl databases db get ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-db`

	cmdDatabaseDBCreate := CmdBuilder(cmd, RunDatabaseDBCreate, "create <database-cluster-id> <database-name>",
		"Create a database within a cluster", "Creates a database with the specified name in the specified database cluster."+getClusterList, Writer, aliasOpt("c"))
	cmdDatabaseDBCreate.Example = `The following example creates a database named ` + "`" + `example-db` + "`" + ` in a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases db create ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-db`

	cmdDatabaseDBDelete := CmdBuilder(cmd, RunDatabaseDBDelete,
		"delete <database-cluster-id> <database-name>", "Delete the specified database from the cluster", "Deletes the specified database from the specified database cluster."+getClusterList+getDBList, Writer, aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseDBDelete, doctl.ArgForce, doctl.ArgShortForce,
		false, "Deletes the database without a confirmation prompt")
	cmdDatabaseDBDelete.Example = `The following example deletes a database named ` + "`" + `example-db` + "`" + ` in a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases db delete ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-db`

	return cmd
}

// Database DBs

// RunDatabaseDBList retrieves a list of databases for specific database cluster
func RunDatabaseDBList(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	dbs, err := c.Databases().ListDBs(id)
	if err != nil {
		return err
	}

	return displayDatabaseDBs(c, dbs...)
}

// RunDatabaseDBGet retrieves a database for a specific database cluster
func RunDatabaseDBGet(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	dbID := c.Args[1]

	db, err := c.Databases().GetDB(databaseID, dbID)
	if err != nil {
		return err
	}

	return displayDatabaseDBs(c, *db)
}

// RunDatabaseDBCreate creates a database for a database cluster
func RunDatabaseDBCreate(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	req := &godo.DatabaseCreateDBRequest{Name: c.Args[1]}

	db, err := c.Databases().CreateDB(databaseID, req)
	if err != nil {
		return err
	}

	return displayDatabaseDBs(c, *db)
}

// RunDatabaseDBDelete deletes a database
func RunDatabaseDBDelete(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("database", 1) == nil {
		databaseID := c.Args[0]
		dbID := c.Args[1]
		return c.Databases().DeleteDB(databaseID, dbID)
	}

	return errOperationAborted
}

func displayDatabaseDBs(c *CmdConfig, dbs ...do.DatabaseDB) error {
	item := &displayers.DatabaseDBs{DatabaseDBs: dbs}
	return c.Display(item)
}

func databaseReplica() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "replica",
			Aliases: []string{"rep", "r"},
			Short:   "Display commands to manage read-only database replicas",
			Long: `The subcommands under ` + "`" + `doctl databases replica` + "`" + ` allow you to manage read-only replicas associated with a database cluster.

In addition to primary nodes in a database cluster, you can create up to 2 read-only replica nodes (also referred to as "standby nodes") to maintain high availability.`,
		},
	}
	howToGetReplica := `

This command requires that you pass in the replica's name, which you can retrieve by querying a database ID:

	doctl databases replica list <database-cluster-id>`
	replicaDetails := `

- The replica's name
- The region where the database cluster is located, such as ` + "`" + `nyc3` + "`" + `, ` + "`" + `sfo2` + "`" + `
- The replica's status. Possible values: ` + "`" + `forking` + "`" + ` and ` + "`" + `active` + "`" + `
`
	cmdDatabaseReplicaList := CmdBuilder(cmd, RunDatabaseReplicaList, "list <database-cluster-id>", "Retrieve list of read-only database replicas", `Lists the following details for read-only replicas for the specified database cluster.`+replicaDetails+databaseListDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.DatabaseReplicas{}))
	cmdDatabaseReplicaList.Example = `The following example retrieves a list of read-only replicas for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + ` and uses the ` + "`" + `--format` + "`" + ` flag to return only the ID and URI for each replica: doctl databases replica list ca9f591d-f38h-5555-a0ef-1c02d1d1e35 --format ID,URI`

	DatabaseReplicaGet := CmdBuilder(cmd, RunDatabaseReplicaGet, "get <database-cluster-id> <replica-name>", "Retrieve information about a read-only database replica",
		`Gets the following details for the specified read-only replica of the specified database cluster:

- The name of the replica
- The information required to connect to the read-only replica
- The region where the database cluster is located, such as `+"`"+`nyc3`+"`"+` or `+"`"+`sfo2`+"`"+`
- The status of the replica. Possible values: `+"`"+`creating`+"`"+`, `+"`"+`forking`+"`"+`, `+"`"+`active`+"`"+`
- When the read-only replica was created, in ISO8601 date/time format`+howToGetReplica+databaseListDetails,
		Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseReplicas{}))
	DatabaseReplicaGet.Example = `The following example retrieves the details for a read-only replica named ` + "`" + `example-replica` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases replica get ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-replica`

	cmdDatabaseReplicaCreate := CmdBuilder(cmd, RunDatabaseReplicaCreate,
		"create <database-cluster-id> <replica-name>", "Create a read-only database replica", `Creates a read-only database replica for the specified database cluster, giving it the specified name.`+databaseListDetails,
		Writer, aliasOpt("c"))
	AddStringFlag(cmdDatabaseReplicaCreate, doctl.ArgRegionSlug, "",
		defaultDatabaseRegion, `Specifies the region in which to create the replica, such as `+"`"+`nyc3`+"`"+` or `+"`"+`sfo2`+"`"+`.`)
	AddStringFlag(cmdDatabaseReplicaCreate, doctl.ArgSizeSlug, "",
		defaultDatabaseNodeSize, `Specifies the machine size for the replica, such as `+"`"+`db-s-1vcpu-1gb`+"`"+`. Must be the same size or larger than the primary database cluster.`)
	AddStringFlag(cmdDatabaseReplicaCreate, doctl.ArgPrivateNetworkUUID, "",
		"", "The UUID of a VPC to create the replica in; the default VPC for the region will be used if excluded.")
	cmdDatabaseReplicaCreate.Example = `The following example creates a read-only replica named ` + "`" + `example-replica` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases replica create ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-replica --size db-s-1vcpu-1gb`

	cmdDatabaseReplicaDelete := CmdBuilder(cmd, RunDatabaseReplicaDelete,
		"delete <database-cluster-id> <replica-name>", "Delete a read-only database replica",
		`Deletes the specified read-only replica for the specified database cluster.`+howToGetReplica+databaseListDetails,
		Writer, aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseReplicaDelete, doctl.ArgForce, doctl.ArgShortForce,
		false, "Deletes the replica without a confirmation prompt.")
	cmdDatabaseReplicaDelete.Example = `The following example deletes a read-only replica named ` + "`" + `example-replica` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases replica delete ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-replica`

	cmdDatabaseReplicaPromote := CmdBuilder(cmd, RunDatabaseReplicaPromote,
		"promote <database-cluster-id> <replica-name>", "Promote a read-only database replica to become a primary cluster",
		`Promotes a read-only database replica to become its own independent primary cluster. Promoted replicas no longer stay in sync with primary cluster they were forked from.`+howToGetReplica+databaseListDetails,
		Writer, aliasOpt("p"))
	cmdDatabaseReplicaPromote.Example = `The following example promotes a read-only replica named ` + "`" + `example-replica` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases replica promote ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-replica`

	cmdDatabaseReplicaConnectionGet := CmdBuilder(cmd, RunDatabaseReplicaConnectionGet,
		"connection <database-cluster-id> <replica-name>",
		"Retrieve information for connecting to a read-only database replica",
		`Retrieves information for connecting to the specified read-only database replica in the specified database cluster`+howToGetReplica+databaseListDetails, Writer, aliasOpt("conn"))
	cmdDatabaseReplicaConnectionGet.Example = `The following example retrieves the connection details for a read-only replica named ` + "`" + `example-replica` + "`" + ` for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases replica connection get ca9f591d-f38h-5555-a0ef-1c02d1d1e35 example-replica`

	return cmd
}

// Database Replicas

// RunDatabaseReplicaList retrieves a list of replicas for specific database cluster
func RunDatabaseReplicaList(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	replicas, err := c.Databases().ListReplicas(id)
	if err != nil {
		return err
	}

	return displayDatabaseReplicas(c, true, replicas...)
}

// RunDatabaseReplicaGet retrieves a read-only replica for a specific database cluster
func RunDatabaseReplicaGet(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	replicaID := c.Args[1]

	replica, err := c.Databases().GetReplica(databaseID, replicaID)
	if err != nil {
		return err
	}

	return displayDatabaseReplicas(c, false, *replica)
}

// RunDatabaseReplicaCreate creates a read-only replica for a database cluster
func RunDatabaseReplicaCreate(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	r, err := buildDatabaseCreateReplicaRequestFromArgs(c)
	if err != nil {
		return err
	}

	replica, err := c.Databases().CreateReplica(databaseID, r)
	if err != nil {
		return err
	}

	return displayDatabaseReplicas(c, false, *replica)
}

func buildDatabaseCreateReplicaRequestFromArgs(c *CmdConfig) (*godo.DatabaseCreateReplicaRequest, error) {
	r := &godo.DatabaseCreateReplicaRequest{Name: c.Args[1]}

	size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return nil, err
	}
	r.Size = size

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return nil, err
	}
	r.Region = region

	privateNetworkUUID, err := c.Doit.GetString(c.NS, doctl.ArgPrivateNetworkUUID)
	if err != nil {
		return nil, err
	}
	r.PrivateNetworkUUID = privateNetworkUUID

	return r, nil
}

// RunDatabaseReplicaDelete deletes a read-only replica
func RunDatabaseReplicaDelete(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("database replica", 1) == nil {
		databaseID := c.Args[0]
		replicaID := c.Args[1]
		return c.Databases().DeleteReplica(databaseID, replicaID)
	}

	return errOperationAborted
}

// RunDatabaseReplicaPromote promotes a read-only replica to become a primary cluster
func RunDatabaseReplicaPromote(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	replicaID := c.Args[1]
	return c.Databases().PromoteReplica(databaseID, replicaID)
}

func displayDatabaseReplicas(c *CmdConfig, short bool, replicas ...do.DatabaseReplica) error {
	item := &displayers.DatabaseReplicas{
		DatabaseReplicas: replicas,
		Short:            short,
	}
	return c.Display(item)
}

// RunDatabaseReplicaConnectionGet gets read-only replica connection info
func RunDatabaseReplicaConnectionGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	replicaID := c.Args[1]
	connInfo, err := c.Databases().GetReplicaConnection(databaseID, replicaID)
	if err != nil {
		return err
	}

	return displayDatabaseReplicaConnection(c, *connInfo)
}

func displayDatabaseReplicaConnection(c *CmdConfig, conn do.DatabaseConnection) error {
	item := &displayers.DatabaseConnection{DatabaseConnection: conn}
	return c.Display(item)
}

func sqlMode() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "sql-mode",
			Aliases: []string{"sm"},
			Short:   "Display commands to configure a MySQL database cluster's SQL modes",
			Long:    "The subcommands of `doctl databases sql-mode` are used to view and configure a MySQL database cluster's global SQL modes. Global SQL modes affect the SQL syntax MySQL supports and the data validation checks it performs.",
		},
	}

	getSqlModeDesc := "Displays the configured SQL modes for the specified MySQL database cluster."
	cmdDatabaseGetSQLModes := CmdBuilder(cmd, RunDatabaseGetSQLModes, "get <database-cluster-id>",
		"Get a MySQL database cluster's SQL modes", getSqlModeDesc, Writer,
		displayerType(&displayers.DatabaseSQLModes{}), aliasOpt("g"))
	cmdDatabaseGetSQLModes.Example = `The following example retrieves the SQL modes for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases sql-mode get ca9f591d-f38h-5555-a0ef-1c02d1d1e35`

	setSqlModeDesc := `This command configures the SQL modes for the specified MySQL database cluster. The SQL modes should be provided as a space separated list.

This command replaces the existing SQL mode configuration completely. Include all of the current values when adding a new one.
`
	cmdDatabaseSetSQLModes := CmdBuilder(cmd, RunDatabaseSetSQLModes, "set <database-cluster-id> <sql-mode-1> ... <sql-mode-n>",
		"Set a MySQL database cluster's SQL modes", setSqlModeDesc, Writer, aliasOpt("s"))
	cmdDatabaseSetSQLModes.Example = `The following example sets the SQL mode ALLOW_INVALID_DATES for an existing database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `. The cluster already has the modes ` + "`" + `NO_ZERO_DATE` + "`" + `, ` + "`" + `NO_ZERO_IN_DATE` + "`" + `, ` + "`" + `STRICT_ALL_TABLES` + "`" + ` set, but they must be included in the command to avoid being overwritten by the additional mode: doctl databases sql-mode set ca9f591d-f38h-5555-a0ef-1c02d1d1e35 NO_ZERO_DATE NO_ZERO_IN_DATE STRICT_ALL_TABLES ALLOW_INVALID_DATES`
	return cmd
}

// RunDatabaseGetSQLModes gets the sql modes set on the database
func RunDatabaseGetSQLModes(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	databaseID := c.Args[0]
	sqlModes, err := c.Databases().GetSQLMode(databaseID)
	if err != nil {
		return err
	}
	return displaySQLModes(c, sqlModes)
}

func displaySQLModes(c *CmdConfig, sqlModes []string) error {
	return c.Display(&displayers.DatabaseSQLModes{
		DatabaseSQLModes: sqlModes,
	})
}

// RunDatabaseSetSQLModes sets the sql modes on the database
func RunDatabaseSetSQLModes(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	sqlModes := c.Args[1:]

	return c.Databases().SetSQLMode(databaseID, sqlModes...)
}

func RunDatabaseTopicList(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	topics, err := c.Databases().ListTopics(databaseID)
	if err != nil {
		return err
	}
	item := &displayers.DatabaseKafkaTopics{DatabaseTopics: topics}
	return c.Display(item)
}

func RunDatabaseTopicGet(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	topicName := c.Args[1]
	topic, err := c.Databases().GetTopic(databaseID, topicName)
	if err != nil {
		return err
	}

	item := &displayers.DatabaseKafkaTopic{DatabaseTopic: *topic}
	return c.Display(item)
}

func RunDatabaseTopicListPartition(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	topicName := c.Args[1]
	topic, err := c.Databases().GetTopic(databaseID, topicName)
	if err != nil {
		return err
	}

	item := &displayers.DatabaseKafkaTopicPartitions{DatabaseTopicPartitions: topic.Partitions}
	return c.Display(item)
}

func RunDatabaseTopicDelete(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("kafka topic", 1) == nil {
		databaseID := c.Args[0]
		topicName := c.Args[1]
		return c.Databases().DeleteTopic(databaseID, topicName)
	}

	return errOperationAborted
}

func RunDatabaseTopicCreate(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	topicName := c.Args[1]

	createReq := &godo.DatabaseCreateTopicRequest{Name: topicName}

	pc, err := c.Doit.GetInt(c.NS, doctl.ArgDatabaseTopicPartitionCount)
	if err == nil && pc != 0 {
		pcUInt32 := uint32(pc)
		createReq.PartitionCount = &pcUInt32
	}
	rf, err := c.Doit.GetInt(c.NS, doctl.ArgDatabaseTopicReplicationFactor)
	if err == nil && rf != 0 {
		rfUInt32 := uint32(rf)
		createReq.ReplicationFactor = &rfUInt32
	}
	createReq.Config = getDatabaseTopicConfigArgs(c)

	_, err = c.Databases().CreateTopic(databaseID, createReq)
	return err
}

func RunDatabaseTopicUpdate(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	databaseID := c.Args[0]
	topicName := c.Args[1]

	updateReq := &godo.DatabaseUpdateTopicRequest{}

	pc, err := c.Doit.GetInt(c.NS, doctl.ArgDatabaseTopicPartitionCount)
	if err == nil && pc != 0 {
		pcUInt32 := uint32(pc)
		updateReq.PartitionCount = &pcUInt32
	}
	rf, err := c.Doit.GetInt(c.NS, doctl.ArgDatabaseTopicReplicationFactor)
	if err == nil && rf != 0 {
		rfUInt32 := uint32(rf)
		updateReq.ReplicationFactor = &rfUInt32
	}
	updateReq.Config = getDatabaseTopicConfigArgs(c)

	err = c.Databases().UpdateTopic(databaseID, topicName, updateReq)
	return err
}

func getDatabaseTopicConfigArgs(c *CmdConfig) *godo.TopicConfig {
	res := &godo.TopicConfig{}
	val, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicCleanupPolicy)
	if err == nil {
		res.CleanupPolicy = val
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicCompressionType)
	if err == nil && val != "" {
		res.CompressionType = val
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicDeleteRetentionMS)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.DeleteRetentionMS = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicFileDeleteDelayMS)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.FileDeleteDelayMS = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicFlushMessages)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.FlushMessages = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicFlushMS)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.FlushMS = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicIntervalIndexBytes)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.IndexIntervalBytes = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicMaxCompactionLagMS)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.MaxCompactionLagMS = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicMaxMessageBytes)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.MaxMessageBytes = &i
		}
	}
	bVal, err := c.Doit.GetBoolPtr(c.NS, doctl.ArgDatabaseTopicMesssageDownConversionEnable)
	if err == nil && bVal != nil {
		res.MessageDownConversionEnable = bVal
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicMessageFormatVersion)
	if err == nil && val != "" {
		res.MessageFormatVersion = val
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicMessageTimestampType)
	if err == nil && val != "" {
		res.MessageTimestampType = val
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicMinCleanableDirtyRatio)
	if err == nil && val != "" {
		i, err := strconv.ParseFloat(val, 32)
		if err == nil {
			iFloat32 := float32(i)
			res.MinCleanableDirtyRatio = &iFloat32
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicMinCompactionLagMS)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.MinCompactionLagMS = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicMinInsyncReplicas)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 32)
		if err == nil {
			iUint32 := uint32(i)
			res.MinInsyncReplicas = &iUint32
		}
	}
	bVal, err = c.Doit.GetBoolPtr(c.NS, doctl.ArgDatabaseTopicPreallocate)
	if err == nil && bVal != nil {
		res.Preallocate = bVal
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicRetentionBytes)
	if err == nil && val != "" {
		i, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			res.RetentionBytes = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicRetentionMS)
	if err == nil && val != "" {
		i, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			res.RetentionMS = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicSegmentBytes)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.SegmentBytes = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicSegmentJitterMS)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.SegmentJitterMS = &i
		}
	}
	val, err = c.Doit.GetString(c.NS, doctl.ArgDatabaseTopicSegmentMS)
	if err == nil && val != "" {
		i, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			res.SegmentMS = &i
		}
	}

	return res
}

func databaseTopic() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "topics",
			Short: `Display commands to manage topics for kafka database clusters`,
			Long:  `The subcommands under ` + "`" + `doctl databases topics` + "`" + ` enable the management of topics for kafka database clusters`,
		},
	}

	topicListDetails := `
This command lists the following details for each topic in a kafka database cluster:

	- The Name of the topic.
	- The State of the topic.
	- The Replication Factor of the topic - number of brokers the topic's partitions are replicated across.
	`

	topicGetDetails := `
This command lists the following details for a given topic in a kafka database cluster:

	- The Name of the topic.
	- The Partitions of the topic - the number of partitions in the topics
	- The Replication Factor of the topic - number of brokers the topic's partitions are replicated across.
	- Additional advanced configuration for the topic.

The details of the topic are listed in key/value pairs
		`
	topicGetPartitionDetails := `
This command lists the following details for each partition of a given topic in a kafka database cluster:

	- The Id - identifier of the topic partition.
	- The Size - size of the topic partition, in bytes.
	- The InSyncReplicas - number of brokers that are in sync with the partition leader.
	- The EarliestOffset - earliest offset read amongst all consumers of the partition.
	`

	CmdBuilder(cmd, RunDatabaseTopicList, "list <database-uuid>", "Retrieve a list of topics for a given kafka database", topicListDetails, Writer, displayerType(&displayers.DatabaseKafkaTopics{}), aliasOpt("ls"))
	CmdBuilder(cmd, RunDatabaseTopicGet, "get <database-uuid> <topic-name>", "Retrieve the configuration for a given kafka topic", topicGetDetails, Writer, displayerType(&displayers.DatabaseKafkaTopic{}), aliasOpt("g"))
	CmdBuilder(cmd, RunDatabaseTopicListPartition, "partitions <database-id> <topic-name>", "Retrieve the partitions for a given kafka topic", topicGetPartitionDetails, Writer, aliasOpt("p"))
	cmdDatabaseTopicDelete := CmdBuilder(cmd, RunDatabaseTopicDelete, "delete <database-uuid> <topic-name>", "Deletes a kafka topic by topic name", "", Writer, aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseTopicDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the kafka topic without a confirmation prompt")
	cmdDatabaseTopicCreate := CmdBuilder(cmd, RunDatabaseTopicCreate, "create <database-uuid> <topic-name>", "Creates a topic for a given kafka database",
		"This command creates a kafka topic for the specified kafka database cluster, giving it the specified name. Example: doctl databases topics create <database-uuid> <topic-name> --replication_factor 2 --partition_count 4", Writer, aliasOpt("c"))
	cmdDatabaseTopicUpdate := CmdBuilder(cmd, RunDatabaseTopicUpdate, "update <database-uuid> <topic-name>", "Updates a topic for a given kafka database",
		"This command updates a kafka topic for the specified kafka database cluster. Example: doctl databases topics update <database-uuid> <topic-name>", Writer, aliasOpt("u"))
	cmdsWithConfig := []*Command{cmdDatabaseTopicCreate, cmdDatabaseTopicUpdate}
	for _, c := range cmdsWithConfig {
		AddIntFlag(c, doctl.ArgDatabaseTopicReplicationFactor, "", 2, "Specifies the number of nodes to replicate data across the kafka cluster")
		AddIntFlag(c, doctl.ArgDatabaseTopicPartitionCount, "", 1, "Specifies the number of partitions available for the topic")
		AddStringFlag(c, doctl.ArgDatabaseTopicCleanupPolicy, "", "delete",
			"Specifies the retention policy to use on log segments: Possible values are 'delete', 'compact_delete', 'compact'")
		AddStringFlag(c, doctl.ArgDatabaseTopicCompressionType, "", "producer",
			"Specifies the compression type for a kafka topic: Possible values are 'producer', 'gzip', 'snappy', 'Iz4', 'zstd', 'uncompressed'")
		AddStringFlag(c, doctl.ArgDatabaseTopicDeleteRetentionMS, "", "",
			"Specifies how long (in ms) to retain delete tombstone markers for topics")
		AddStringFlag(c, doctl.ArgDatabaseTopicFileDeleteDelayMS, "", "",
			"Specifies the minimum time (in ms) to wait before deleting a file from the filesystem")
		AddStringFlag(c, doctl.ArgDatabaseTopicFlushMessages, "", "",
			"Specifies the maximum number of messages to accumulate on a log partition before messages are flushed to disk")
		AddStringFlag(c, doctl.ArgDatabaseTopicFlushMS, "", "",
			"Specifies the maximum time (in ms) that a message is kept in memory before being flushed to disk")
		AddStringFlag(c, doctl.ArgDatabaseTopicIntervalIndexBytes, "", "",
			"Specifies the number of bytes between entries being added into the offset index")
		AddStringFlag(c, doctl.ArgDatabaseTopicMaxCompactionLagMS, "", "",
			"Specifies the maximum time (in ms) that a message will remain uncompacted. This is only applicable if the logs have compaction enabled")
		AddStringFlag(c, doctl.ArgDatabaseTopicMaxMessageBytes, "", "",
			"Specifies the largest record batch (in bytes) that can be sent to the server. This is calculated after compression, if compression is enabled")
		AddBoolFlag(c, doctl.ArgDatabaseTopicMesssageDownConversionEnable, "", true,
			"Specifies whether down-conversion of message formats is enabled to satisfy consumer requests")
		AddStringFlag(c, doctl.ArgDatabaseTopicMessageFormatVersion, "", "",
			"Specifies the message format version used by the broker to append messages to the logs. By setting a format version, all existing messages on disk must be smaller or equal to the specified version")
		AddStringFlag(c, doctl.ArgDatabaseTopicMessageTimestampType, "", "",
			"Specifies whether to use the create time or log append time as the timestamp on a message")
		AddStringFlag(c, doctl.ArgDatabaseTopicMinCleanableDirtyRatio, "", "",
			"Specifies the frequenty of log compaction (if enabled) in relation to duplicates present in the logs. For example, 0.5 would mean at most half of the log could be duplicates before compaction would begin")
		AddStringFlag(c, doctl.ArgDatabaseTopicMinCompactionLagMS, "", "",
			"Specifies the minimum time (in ms) that a message will remain uncompacted. This is only applicable if the logs have compaction enabled")
		AddStringFlag(c, doctl.ArgDatabaseTopicMinInsyncReplicas, "", "",
			"Specifies the minimum number of replicas that must ACK a write for it to be considered successful")
		AddBoolFlag(c, doctl.ArgDatabaseTopicPreallocate, "", false,
			"Specifies whether a file should be preallocated on disk when creating a new log segment")
		AddStringFlag(c, doctl.ArgDatabaseTopicRetentionBytes, "", "",
			"Specifies the maximum size (in bytes) before deleting messages. '-1' indicates that there is no limit")
		AddStringFlag(c, doctl.ArgDatabaseTopicRetentionMS, "", "",
			"Specifies the maximum time (in ms) to store a message before deleting it. '-1' indicates that there is no limit")
		AddStringFlag(c, doctl.ArgDatabaseTopicSegmentBytes, "", "",
			"Specifies the maximum size (in bytes) of a single log file")
		AddStringFlag(c, doctl.ArgDatabaseTopicSegmentJitterMS, "", "",
			"Specifies the maximum time (in ms) for random jitter that is subtracted from the scheduled segment roll time to avoid thundering herd problems")
		AddStringFlag(c, doctl.ArgDatabaseTopicSegmentMS, "", "",
			"Specifies the maximum time (in ms) to wait to force a log roll if the segment file isn't full. After this period, the log will be forced to roll")
	}
	return cmd
}

func databaseFirewalls() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "firewalls",
			Aliases: []string{"fw"},
			Short:   `Display commands to manage firewall rules (called` + "`" + `trusted sources` + "`" + ` in the control panel) for database clusters`,
			Long:    `The subcommands under ` + "`" + `doctl databases firewalls` + "`" + ` enable the management of firewalls for database clusters.`,
		},
	}

	firewallRuleDetails := `
This command lists the following details for each firewall rule in a given database:

	- The UUID of the firewall rule
	- The UUID of the cluster for which the rule is applied
	- The type of resource that the firewall rule allows to access the database cluster. Possible values are: ` + "`" + `droplet` + "`" + `, ` + "`" + `k8s` + "`" + `, ` + "`" + `ip_addr` + "`" + `, ` + "`" + `tag` + "`" + `, ` + "`" + `app` + "`" + `
	- The value, which specifies the resource or resources allowed to access the database cluster. Possible values are either the ID of the specific resource, the name of a tag applied to a group of resources, or an IP address
	- When the firewall rule was created, in ISO8601 date/time format
	`
	databaseFirewallRuleDetails := `

This command requires the ID of a database cluster, which you can retrieve by calling:

	doctl databases list`

	databaseFirewallRulesTxt := `A comma-separated list of firewall rules, in ` + "`" + `type:value` + "`" + ` format.`

	databaseFirewallUpdateDetails := `
Replace the firewall rules for a specified database. This command requires the ` + "`" + `--rule` + "`" + ` flag. 

You can configure multiple rules for the firewall by passing additional arguments in a comma-separated list with the ` + "`" + `--rule` + "`" + ` flag. Each rule passed using the ` + "`" + `--rule` + "`" + ` flag must be in a ` + "`" + `<type>:<value>` + "`" + ` format where:
	 ` + "`" + `type` + "`" + ` is the type of resource that the firewall rule allows to access the database cluster. Possible values are:  ` + "`" + `droplet` + "`" + `, ` + "`" + `k8s` + "`" + `, ` + "`" + `ip_addr` + "`" + `, ` + "`" + `tag` + "`" + `, ` + "`" + `app` + "`" + `
	- ` + "`" + `value` + "`" + ` is either the ID of a specific resource, the name of a tag applied to a group of resources, or the IP address that the firewall rule allows to access the database cluster.
	`

	databaseFirewallAddDetails :=
		`
Appends a single rule to the existing firewall rules of the specified database. 

This command requires the ` + "`" + `--rule` + "`" + ` flag specifying the resource or resources allowed to access the database cluster. The rule passed to the ` + "`" + `--rule` + "`" + ` flag must be in a <type>:<value> format where:
	- ` + "`" + `type` + "`" + ` is the type of resource that the firewall rule allows to access the database cluster. Possible values are:  ` + "`" + `droplet` + "`" + `, ` + "`" + `k8s", ` + "`" + `ip_addr` + "`" + `, ` + "`" + `tag` + "`" + `, ` + "`" + `app` + "`" + `
	- ` + "`" + `value` + "`" + ` is either the ID of a specific resource, the name of a tag applied to a group of resources, or the IP address that the firewall rule allows to access the database cluster.`

	databaseFirewallRemoveDetails :=
		`
Removes single rule from the list of firewall rules for a specified database. You can retrieve a firewall rule's UUIDs by calling:

	doctl database firewalls list <database-cluster-id>`

	cmdDatabaseFirewallRulesList := CmdBuilder(cmd, RunDatabaseFirewallRulesList, "list <database-cluster-id>", "Retrieve a list of firewall rules for a given database", firewallRuleDetails+databaseFirewallRuleDetails,
		Writer, aliasOpt("ls"))
	cmdDatabaseFirewallRulesList.Example = `The following example retrieves a list of firewall rules for a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases firewalls list ca9f591d-f38h-5555-a0ef-1c02d1d1e35`

	cmdDatabaseFirewallUpdate := CmdBuilder(cmd, RunDatabaseFirewallRulesUpdate, "replace <database-cluster-id> --rules type:value [--rule type:value]", `Replaces the firewall rules for a given database. The rules passed to the `+"`"+`--rules`+"`"+` flag replace the firewall rules previously assigned to the database,`, databaseFirewallUpdateDetails,
		Writer, aliasOpt("r"))
	AddStringSliceFlag(cmdDatabaseFirewallUpdate, doctl.ArgDatabaseFirewallRule, "", []string{}, databaseFirewallRulesTxt, requiredOpt())
	cmdDatabaseFirewallUpdate.Example = `The following example replaces the firewall rules for a database cluster, with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `, with rules that allow a specific Droplet, a specific IP address, and any resources with the ` + "`" + `example-tag` + "`" + ` to access the database: doctl databases firewalls replace ca9f591d-f38h-5555-a0ef-1c02d1d1e35 --rules droplet:f81d4fae-7dec-11d0-a765-00a0c91e6bf6,ip_addr:192.168.1.1,tag:example-tag`

	cmdDatabaseFirewallCreate := CmdBuilder(cmd, RunDatabaseFirewallRulesAppend, "append <database-cluster-id> --rule <type>:<value>", "Add a database firewall rule to a given database", databaseFirewallAddDetails,
		Writer, aliasOpt("a"))
	AddStringFlag(cmdDatabaseFirewallCreate, doctl.ArgDatabaseFirewallRule, "", "", "", requiredOpt())
	cmdDatabaseFirewallCreate.Example = `The following example appends a firewall rule to a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + ` that allows any resources with the ` + "`" + `example-tag` + "`" + ` to access the database: doctl databases firewalls append ca9f591d-f38h-5555-a0ef-1c02d1d1e35 --rule tag:example-tag`

	cmdDatabaseFirewallRemove := CmdBuilder(cmd, RunDatabaseFirewallRulesRemove, "remove <database-cluster-id> --uuid <firerule-uuid>", "Remove a firewall rule for a given database", databaseFirewallRemoveDetails,
		Writer, aliasOpt("rm"))
	AddStringFlag(cmdDatabaseFirewallRemove, doctl.ArgDatabaseFirewallRuleUUID, "", "", "", requiredOpt())
	cmdDatabaseFirewallRemove.Example = `The following example removes a firewall rule with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` from a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases firewalls remove ca9f591d-f38h-5555-a0ef-1c02d1d1e35 f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	return cmd
}

// displayDatabaseFirewallRules calls Get Firewall Rules to list all current rules.
func displayDatabaseFirewallRules(c *CmdConfig, short bool, id string) error {
	firewallRules, err := c.Databases().GetFirewallRules(id)
	if err != nil {
		return err
	}

	item := &displayers.DatabaseFirewallRules{
		DatabaseFirewallRules: firewallRules,
	}

	return c.Display(item)
}

// All firewall rules require the databaseID
func firewallRulesArgumentCheck(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if len(c.Args) > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	return nil
}

// RunDatabaseFirewallRulesList retrieves a list of firewalls for specific database cluster
func RunDatabaseFirewallRulesList(c *CmdConfig) error {
	err := firewallRulesArgumentCheck(c)
	if err != nil {
		return err
	}

	id := c.Args[0]

	return displayDatabaseFirewallRules(c, true, id)
}

// RunDatabaseFirewallRulesUpdate replaces previous rules with the rules passed in to --rules
func RunDatabaseFirewallRulesUpdate(c *CmdConfig) error {
	err := firewallRulesArgumentCheck(c)
	if err != nil {
		return err
	}

	id := c.Args[0]
	r, err := buildDatabaseUpdateFirewallRulesRequestFromArgs(c)
	if err != nil {
		return err
	}

	err = c.Databases().UpdateFirewallRules(id, r)
	if err != nil {
		return err
	}

	return displayDatabaseFirewallRules(c, true, id)
}

// buildDatabaseUpdateFirewallRulesRequestFromArgs will ingest the --rules arguments into a DatabaseUpdateFirewallRulesRequest object.
func buildDatabaseUpdateFirewallRulesRequestFromArgs(c *CmdConfig) (*godo.DatabaseUpdateFirewallRulesRequest, error) {
	r := &godo.DatabaseUpdateFirewallRulesRequest{}

	firewallRules, err := c.Doit.GetStringSlice(c.NS, doctl.ArgDatabaseFirewallRule)
	if err != nil {
		return nil, err
	}

	if len(firewallRules) == 0 {
		return nil, errors.New("Must pass in a key:value pair for the --rule flag")
	}

	firewallRulesList, err := extractFirewallRules(firewallRules)
	if err != nil {
		return nil, err
	}
	r.Rules = firewallRulesList

	return r, nil
}

// extractFirewallRules will ingest the --rules arguments into a list of DatabaseFirewallRule objects.
func extractFirewallRules(rulesStringList []string) (rules []*godo.DatabaseFirewallRule, err error) {
	for _, rule := range rulesStringList {
		pair := strings.SplitN(rule, ":", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("Unexpected input value [%v], must be a key:value pair", pair)
		}

		firewallRule := new(godo.DatabaseFirewallRule)
		firewallRule.Type = pair[0]
		firewallRule.Value = pair[1]

		rules = append(rules, firewallRule)
	}

	return rules, nil
}

// RunDatabaseFirewallRulesAppend creates a firewall rule for a database cluster.
//
// Any new rules will be appended to the existing rules. If you want to replace
// rules, use RunDatabaseFirewallRulesUpdate.
func RunDatabaseFirewallRulesAppend(c *CmdConfig) error {
	err := firewallRulesArgumentCheck(c)
	if err != nil {
		return err
	}

	databaseID := c.Args[0]
	firewallRuleArg, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseFirewallRule)
	if err != nil {
		return err
	}

	pair := strings.SplitN(firewallRuleArg, ":", 2)
	if len(pair) != 2 {
		return fmt.Errorf("Unexpected input value [%v], must be a key:value pair", pair)
	}

	// Slice will house old rules and new rule
	allRules := []*godo.DatabaseFirewallRule{}

	// Adding new rule to slice.
	allRules = append(allRules, &godo.DatabaseFirewallRule{
		Type:        pair[0],
		Value:       pair[1],
		ClusterUUID: databaseID,
	})

	// Retrieve any existing firewall rules so that we don't destroy existing
	// rules in the create request.
	oldRules, err := c.Databases().GetFirewallRules(databaseID)
	if err != nil {
		return err
	}

	// Add old rules to allRules slice.
	for _, rule := range oldRules {

		firewallRule := new(godo.DatabaseFirewallRule)
		firewallRule.Type = rule.Type
		firewallRule.Value = rule.Value
		firewallRule.ClusterUUID = rule.ClusterUUID
		firewallRule.UUID = rule.UUID

		allRules = append(allRules, firewallRule)
	}

	// Run update firewall rules with old rules + new rule
	if err := c.Databases().UpdateFirewallRules(databaseID, &godo.DatabaseUpdateFirewallRulesRequest{
		Rules: allRules,
	}); err != nil {
		return err
	}

	return displayDatabaseFirewallRules(c, true, databaseID)
}

// RunDatabaseFirewallRulesRemove removes a firewall rule for a database cluster via Firewall rule UUID
func RunDatabaseFirewallRulesRemove(c *CmdConfig) error {
	err := firewallRulesArgumentCheck(c)
	if err != nil {
		return err
	}

	databaseID := c.Args[0]

	firewallRuleUUIDArg, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseFirewallRuleUUID)
	if err != nil {
		return err
	}

	// Retrieve any existing firewall rules so that we don't destroy existing
	// rules in the create request.
	rules, err := c.Databases().GetFirewallRules(databaseID)
	if err != nil {
		return err
	}

	// Create a slice of database firewall rules containing only the new rule.
	firewallRules := []*godo.DatabaseFirewallRule{}

	// only append rules that do not match the firewall rule with uuid to be removed.
	for _, rule := range rules {
		if rule.UUID != firewallRuleUUIDArg {
			firewallRules = append(firewallRules, &godo.DatabaseFirewallRule{
				UUID:        rule.UUID,
				ClusterUUID: rule.ClusterUUID,
				Type:        rule.Type,
				Value:       rule.Value,
			})
		}
	}

	if err := c.Databases().UpdateFirewallRules(databaseID, &godo.DatabaseUpdateFirewallRulesRequest{
		Rules: firewallRules,
	}); err != nil {
		return err
	}

	return displayDatabaseFirewallRules(c, true, databaseID)
}

func waitForDatabaseReady(dbs do.DatabasesService, dbID string) error {
	const (
		maxAttempts = 180
		wantStatus  = "online"
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

		db, err := dbs.Get(dbID)
		if err != nil {
			return err
		}

		if db.Status == wantStatus {
			return nil
		}

		attempts++
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf(
		"timeout waiting for database (%s) to enter `online` state",
		dbID,
	)
}

func databaseConfiguration() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "configuration",
			Aliases: []string{"cfg", "config"},
			Short:   "View the configuration of a database cluster given its ID and Engine",
			Long:    "The subcommands of `doctl databases configuration` are used to view a database cluster's configuration.",
		},
	}
	getConfigurationLongDesc := "Retrieves the configuration for the specified cluster, including its backup settings, temporary file limit, and session timeout values."
	updateConfigurationLongDesc := "Updates the specified database cluster's configuration. Using this command, you can update varioous settings like backup times, temporary file limits, and session timeouts."

	getDatabaseCfgCommand := CmdBuilder(

		cmd,
		RunDatabaseConfigurationGet,
		"get <database-cluster-id>",
		"Get a database cluster's configuration",
		getConfigurationLongDesc,
		Writer,
		aliasOpt("g"),
		displayerType(&displayers.MySQLConfiguration{}),
		displayerType(&displayers.PostgreSQLConfiguration{}),
		displayerType(&displayers.RedisConfiguration{}),
	)
	AddStringFlag(
		getDatabaseCfgCommand,
		doctl.ArgDatabaseEngine,
		"e",
		"",
		"The engine of the database you want to get the configuration for.",
		requiredOpt(),
	)

	updateDatabaseCfgCommand := CmdBuilder(
		cmd,
		RunDatabaseConfigurationUpdate,
		"update <db-id>",
		"Update a database cluster's configuration",
		updateConfigurationLongDesc,
		Writer,
		aliasOpt("u"),
	)
	AddStringFlag(
		updateDatabaseCfgCommand,
		doctl.ArgDatabaseEngine,
		"e",
		"",
		"the engine of the database you want to update the configuration for",
		requiredOpt(),
	)
	AddStringFlag(
		updateDatabaseCfgCommand,
		doctl.ArgDatabaseConfigJson,
		"",
		"{}",
		"the desired configuration of the database cluster you want to update",
		requiredOpt(),
	)

	return cmd
}

func RunDatabaseConfigurationGet(c *CmdConfig) error {
	args := c.Args
	if len(args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if len(args) > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}

	engine, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseEngine)
	if err != nil {
		return doctl.NewMissingArgsErr(c.NS)
	}

	allowedEngines := map[string]any{
		"mysql": nil,
		"pg":    nil,
		"redis": nil,
	}
	if _, ok := allowedEngines[engine]; !ok {
		return fmt.Errorf("(%s) command: engine must be one of: 'pg', 'mysql', 'redis'", c.NS)
	}

	dbId := args[0]
	if engine == "mysql" {
		config, err := c.Databases().GetMySQLConfiguration(dbId)
		if err != nil {
			return err
		}

		displayer := displayers.MySQLConfiguration{
			MySQLConfiguration: *config,
		}
		return c.Display(&displayer)
	} else if engine == "pg" {
		config, err := c.Databases().GetPostgreSQLConfiguration(dbId)
		if err != nil {
			return err
		}

		displayer := displayers.PostgreSQLConfiguration{
			PostgreSQLConfig: *config,
		}
		return c.Display(&displayer)
	} else if engine == "redis" {
		config, err := c.Databases().GetRedisConfiguration(dbId)
		if err != nil {
			return err
		}

		displayer := displayers.RedisConfiguration{
			RedisConfig: *config,
		}
		return c.Display(&displayer)
	}
	return nil
}

func RunDatabaseConfigurationUpdate(c *CmdConfig) error {
	args := c.Args
	if len(args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if len(args) > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}

	engine, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseEngine)
	if err != nil {
		return doctl.NewMissingArgsErr(c.NS)
	}

	allowedEngines := map[string]any{
		"mysql": nil,
		"pg":    nil,
		"redis": nil,
	}
	if _, ok := allowedEngines[engine]; !ok {
		return fmt.Errorf("(%s) command: engine must be one of: 'pg', 'mysql', 'redis'", c.NS)
	}

	configJson, err := c.Doit.GetString(c.NS, doctl.ArgDatabaseConfigJson)
	if err != nil {
		return doctl.NewMissingArgsErr(c.NS)
	}

	dbId := args[0]
	if engine == "mysql" {
		err := c.Databases().UpdateMySQLConfiguration(dbId, configJson)
		if err != nil {
			return err
		}
	} else if engine == "pg" {
		err := c.Databases().UpdatePostgreSQLConfiguration(dbId, configJson)
		if err != nil {
			return err
		}
	} else if engine == "redis" {
		err := c.Databases().UpdateRedisConfiguration(dbId, configJson)
		if err != nil {
			return err
		}
	}
	return nil
}

func databaseEvents() *Command {
	listDatabaseEvents := `

You can get a list of database events by calling:

	doctl databases events list <cluster-id>`
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "events",
			Short: "Display commands for listing database cluster events",
			Long:  `The subcommands under ` + "`" + `doctl databases events` + "`" + ` are for listing database cluster events.` + listDatabaseEvents,
		},
	}
	cmdDatabaseEventsList := CmdBuilder(cmd, RunDatabaseEvents, "list <database-cluster-id>", "List your database cluster events", `Retrieves a list of database clusters events:`+listDatabaseEvents, Writer, aliasOpt("ls"), displayerType(&displayers.DatabaseEvents{}))

	cmdDatabaseEventsList.Example = `The following example retrieves a list of databases events in a database cluster with the ID ` + "`" + `ca9f591d-f38h-5555-a0ef-1c02d1d1e35` + "`" + `: doctl databases events list ca9f591d-f38h-5555-a0ef-1c02d1d1e35`

	return cmd
}

// RunDatabaseDBList retrieves a list of databases for specific database cluster
func RunDatabaseEvents(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	dbEvents, err := c.Databases().ListDatabaseEvents(id)
	if err != nil {
		return err
	}

	item := &displayers.DatabaseEvents{DatabaseEvents: dbEvents}
	return c.Display(item)
}
