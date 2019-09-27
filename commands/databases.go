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
	"strings"

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
)

// Databases creates the databases command
func Databases() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "databases",
			Aliases: []string{"db", "dbs", "d", "database"},
			Short:   "Provides commands that manage databases",
			Long:   `The commands under 'doctl databases' are for managing your MySQL, Redis, and PostgreSQL databases.`,
		},
	}

	clusterDetails := `

- The database ID, in UUID format
- The name you gave the database cluster
- The database engine (redis, pg, mysql, etc)
- The engine version (e.g. '11' for PostgreSQL version 11)
- The number of nodes in the database cluster
- The region the database cluster resides in (sfo2, nyc1, etc)
- The current status of the database cluster (online, etc)
- The size of the machine running the database instance (db-s-1vcpu-1gb)`

	CmdBuilderWithDocs(cmd, RunDatabaseList, "list", "Lists your database clusters", `This command lists the database clusters associated with your account. The following details are provided:` + clusterDetails, Writer, aliasOpt("ls"), displayerType(&displayers.Databases{}))
	CmdBuilderWithDocs(cmd, RunDatabaseGet, "get <database-id>", "Get details for a database cluster", `This command retrieves the following details about the specified database cluster: ` + clusterDetails + `
- A connection string for the database cluster
- The date and time at which the database cluster was created

This command requires the ID of a database cluster, which you can retrieve by calling 'doctl databases list'.`, Writer, aliasOpt("g"), displayerType(&displayers.Databases{}))

	nodeSizeDetails := "The size of the nodes in the database cluster, e.g. 'db-s-1vcpu-1gb' for a 1 CPU, 1GB node"
	nodeNumberDetails := "The number of nodes in the database cluster. Valid values are are 1-3. In addition to the primary node, up to two standby nodes may be added for high availability."
	cmdDatabaseCreate := CmdBuilderWithDocs(cmd, RunDatabaseCreate, "create <name>", "Creates a database cluster",`This command creates a database cluster with the specified name.

There are a number of flags that customize the configuration, all of which are optional. Without any flags set, a single-node, single-CPU PostgreSQL database cluster will be created.`, Writer,
		aliasOpt("c"))
	AddIntFlag(cmdDatabaseCreate, doctl.ArgDatabaseNumNodes, "", defaultDatabaseNodeCount, nodeNumberDetails)
	AddStringFlag(cmdDatabaseCreate, doctl.ArgRegionSlug, "", defaultDatabaseRegion, "The region where the database cluster will be created, e.g. 'nyc1' or 'sfo2'")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgSizeSlug, "", defaultDatabaseNodeSize, nodeSizeDetails)
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseEngine, "", defaultDatabaseEngine, "The database engine to be used for the cluster. Possible values are:'' pg' for PostgreSQL, 'mysql', and 'redis'.")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgVersion, "", "", "The database engine version, e.g. 11 for PostgreSQL version 11")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgPrivateNetworkUUID, "", "", "A UUID to use for private network connections")

	cmdDatabaseDelete := CmdBuilderWithDocs(cmd, RunDatabaseDelete, "delete <database-id>", "Deletes a database cluster", `This command deletes the database cluster with the given ID.

To retrieve a list of your database clusters and their IDs, call 'doctl databases list'.`, Writer,
		aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the database cluster without a confirmation prompt")

	CmdBuilderWithDocs(cmd, RunDatabaseConnectionGet, "connection <database-id>", "Retrieves connection details for a database cluster", `This command retrieves the following connection details for a database cluster:

- The connection string for the database cluster
- The default database name
- The fully-qualified domain name of the publicly-connectable host
- The port on which the database is listening for connections
- The default username
- The randomly-generated password for the default username
- A boolean indicating if the connection should be made over SSL

While these connection details will work, you may wish to use different connection details, such as the private hostname, a custom username, or a different database.`, Writer,
		aliasOpt("conn"), displayerType(&displayers.DatabaseConnection{}))

	CmdBuilderWithDocs(cmd, RunDatabaseBackupsList, "backups <database-id>", "list database cluster backups", `This command retrieves a list of backups created for the specified database cluster.

The list contains the size in GB, and the date and time the backup was taken.`, Writer,
		aliasOpt("bu"), displayerType(&displayers.DatabaseBackups{}))

	cmdDatabaseResize := CmdBuilderWithDocs(cmd, RunDatabaseResize, "resize <database-id>", "Resizes a database cluster", `This command resizes the specified database cluster.

You must specify the size of the machines you wish to use as nodes as well as how many nodes you would like. For example:

	doctl databases resize ca9f591d-9999-5555-a0ef-1c02d1d1e352 --num-nodes 2 --size db-s-16vcpu-64gb`, Writer,
		aliasOpt("rs"))
	AddIntFlag(cmdDatabaseResize, doctl.ArgDatabaseNumNodes, "", 0, nodeNumberDetails, requiredOpt())
	AddStringFlag(cmdDatabaseResize, doctl.ArgSizeSlug, "", "", nodeSizeDetails, requiredOpt())

	cmdDatabaseMigrate := CmdBuilderWithDocs(cmd, RunDatabaseMigrate, "migrate <database-id", "Migrates a database cluster to a new region", `This command migrates the specified database cluster to a new region`, Writer,
		aliasOpt("m"))
	AddStringFlag(cmdDatabaseMigrate, doctl.ArgRegionSlug, "", "", "The region to which the database cluster should be migrated, e.g. sfo2 or nyc3.", requiredOpt())
	AddStringFlag(cmdDatabaseMigrate, doctl.ArgPrivateNetworkUUID, "", "", "A UUID to use for private network connections")

	cmd.AddCommand(databaseReplica())
	cmd.AddCommand(databaseMaintenanceWindow())
	cmd.AddCommand(databaseUser())
	cmd.AddCommand(databaseDB())
	cmd.AddCommand(databasePool())

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

	db, err := c.Databases().Create(r)
	if err != nil {
		return err
	}

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

	return r, nil
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

	if force || AskForConfirm("Delete this database cluster?") == nil {
		id := c.Args[0]
		return c.Databases().Delete(id)
	}

	return fmt.Errorf("Operation aborted.")
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
	connInfo, err := c.Databases().GetConnection(id)
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
			Short:   "Provides commands for scheduling automatic maintenance on your database cluster",
			Long:    `The 'doctl databases maintenance-window' commands allow you to schedule, and check the schedule of, maintenance windows for your databases.

Maintenance windows are hour-long blocks of time during which DigitalOcean performs automatic maintenance on databases every week. During this time, health checks, security updates, version upgrades, and more are performed.`,
		},
	}

	CmdBuilderWithDocs(cmd, RunDatabaseMaintenanceGet, "get <database-id>",
		"Retrieves details about a database cluster's maintenance windows", `This command retrieves the following information on currently-scheduled maintenance windows for the specified database cluster:

- The day of the week the maintenance window occurs
- The hour in UTC at which maintenance updates will be applied, in 24 hour format (e.g. "16:00")
- A boolean representing whether maintence updates are currently pending

To see a list of your databases and their IDs, run 'doctl databases list'.`, Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseMaintenanceWindow{}))

	cmdDatabaseCreate := CmdBuilderWithDocs(cmd, RunDatabaseMaintenanceUpdate,
		"update <database-id>", "Update the maintenance window for a database cluster", `This command allows you to update the maintenance window for the specified database cluster.

Maintenance windows are hour-long blocks of time during which DigitalOcean performs automatic maintenance on databases every week. During this time, health checks, security updates, version upgrades, and more are performed.

To change the maintenance window for your database cluster, specify a day of the week and an hour of that day during which you would prefer such maintenance would occur.

	doctl databases maintenance-window ca9f591d-f38h-5555-a0ef-1c02d1d1e35 update --day tuesday --hour 16:00

To see a list of your databases and their IDs, run 'doctl databases list'.`, Writer, aliasOpt("u"))
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseMaintenanceDay, "", "",
		"The day of the week the maintenance window occurs (e.g. 'tuesday')", requiredOpt())
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseMaintenanceHour, "", "",
		"The hour in UTC at which maintenance updates will be applied, in 24 hour format (e.g. '16:00')", requiredOpt())

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
			Short:   "Provides commands for managing database users",
			Long:    `The commands under 'doctl databases user' allow you to view details for, and create, database users.

Database user accounts are scoped to one database cluster, to which they have full admin access, and are given an automatically-generated password.`,
		},
	}

	userDetailsDesc := `

- The username for the user
- The password for the user
- The user's role. The value will be either "primary" or "normal".

Primary user accounts are created by DigitalOcean at database cluster creation time and can't be deleted. Normal user accounts are created by you. Both have administrative privileges on the database cluster.

To retrieve a list of your databases and their IDs, call 'doctl databases list'.`
	CmdBuilderWithDocs(cmd, RunDatabaseUserList, "list <database-id>", "Retrieves list of database users",
		`This command retrieves a list of users for the specified database with the following details:` + userDetailsDesc, Writer, aliasOpt("ls"), displayerType(&displayers.DatabaseUsers{}))
	CmdBuilderWithDocs(cmd, RunDatabaseUserGet, "get <database-id> <user-name>",
		"Retrieves details about a database user", `This command retrieves the following details about the specified user:` + userDetailsDesc + `

To retrieve a list of database users for a database, call 'doctl databases user list {database-id}'`, Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseUsers{}))
	CmdBuilderWithDocs(cmd, RunDatabaseUserCreate, "create <database-id> <user-name>",
		"Creates a database user", `This command creates a user with the username you specify, who will be granted access to the database cluster you specify.

The user will be created with the role set to 'normal', and given an automatically-generated password.

To retrieve a list of your databases and their IDs, call 'doctl databases list'.`, Writer, aliasOpt("c"))

	cmdDatabaseUserDelete := CmdBuilderWithDocs(cmd, RunDatabaseUserDelete,
		"delete <database-id> <user-id>", "Deletes a database user", `This command deletes the user with the username you specify, whose account was given access to the database cluster you specify.

To retrieve a list of your databases and their IDs, call 'doctl databases list'.`, Writer, aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseUserDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the user without a confirmation prompt")

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

	databaseID := c.Args[0]
	req := &godo.DatabaseCreateUserRequest{Name: c.Args[1]}

	user, err := c.Databases().CreateUser(databaseID, req)
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

	if force || AskForConfirm("Delete this database user?") == nil {
		databaseID := c.Args[0]
		userID := c.Args[1]
		return c.Databases().DeleteUser(databaseID, userID)
	}

	return fmt.Errorf("Operation aborted.")
}

func displayDatabaseUsers(c *CmdConfig, users ...do.DatabaseUser) error {
	item := &displayers.DatabaseUsers{DatabaseUsers: users}
	return c.Display(item)
}

func databasePool() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "pool",
			Aliases: []string{"p"},
			Short:   "Provides commands for managing connection pools",
			Long:    `The subcommands under 'doctl databases pool' are for managing connection pools for your database cluster.

A connection pool may be useful if your database:

- Typically handles a large number of idle connections,
- Has wide variability in the possible number of connections at any given time,
- Drops connections due to max connection limits, or
- Experiences performance issues due to high CPU usage.

Connection pools can be created and deleted with these commands, or you can simply retrieve information about them.`,
		},
	}

	connectionPoolDetails := `

- The username of the database user account that the connection pool uses
- The name of the connection pool
- The size of the connection pool, i.e. the number of connections that will be allocated
- The database within the cluster for which the connection pool is used
- The pool mode for the connection pool, which can be 'session', 'transaction', or 'statement'
- A connection string for the connection pool`
  getPoolDetails := `

You can get a list of existing connection pools by calling:

	doctl databases pool list

You can get a list of existing database clusters and their IDs by calling:

	doctl databases list`
	CmdBuilderWithDocs(cmd, RunDatabasePoolList, "list <database-id>", "Lists connection pools for a database cluster", `This command lists the existing connection pools for the specified database. The following information will be returned:` + connectionPoolDetails,
		Writer, aliasOpt("ls"), displayerType(&displayers.DatabasePools{}))
	CmdBuilderWithDocs(cmd, RunDatabasePoolGet, "get <database-id> <pool-name>",
		"Retrieves information about a database connection pool", `This command retrieves the following information about the specified connection pool for the specified database cluster:` + connectionPoolDetails + getPoolDetails, Writer, aliasOpt("g"),
		displayerType(&displayers.DatabasePools{}))
	cmdDatabasePoolCreate := CmdBuilderWithDocs(cmd, RunDatabasePoolCreate,
		"create <database-id> <pool-name>", "Creates a connection pool for a database", `This command creates a connection pool for the specified database cluster and gives it the specified name.

You must also use flags to specify the target database, pool size, and database user's username that will be used for the pool. An example call would be:

	pool create ca9f591d-fb58-5555-a0ef-1c02d1d1e352 mypool --db defaultdb --size 10 --user doadmin

The pool size is the minimum number of connections the pool can handle. The maximum pool size varies based on the size of the cluster.

There’s no perfect formula to determine how large your pool should be, but there are a few good guidelines to keep in mind:

- A large pool will stress your database at similar levels as that number of clients would alone.
- A pool that’s much smaller than the number of clients communicating with the database can act as a bottleneck, reducing the rate at which your database receives and responds to transactions.

We recommend starting with a pool size of about half your available connections and adjusting later based on performance. If you see slow query responses, check the CPU usage on the database’s Overview tab. We recommend decreasing your pool size if CPU usage is high, and increasing your pool size if it’s low.` + getPoolDetails, Writer,
		aliasOpt("c"))
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolMode, "",
		"transaction", "The pool mode for the connection pool, e.g. 'session', 'transaction', and 'statement'")
	AddIntFlag(cmdDatabasePoolCreate, doctl.ArgSizeSlug, "", 0, "pool size",
		requiredOpt())
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolUserName, "", "",
		"The username for the database user", requiredOpt())
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolDBName, "", "",
		"The name of the specific database within the database cluster", requiredOpt())

	cmdDatabasePoolDelete := CmdBuilderWithDocs(cmd, RunDatabasePoolDelete,
		"delete <database-id> <pool-name>", "Deletes a connection pool for a database", `This command deletes the specified connection pool for the specified database cluster.` + getPoolDetails, Writer,
		aliasOpt("rm"))
	AddBoolFlag(cmdDatabasePoolDelete, doctl.ArgForce, doctl.ArgShortForce,
		false, "Delete connection pool without confirmation prompt")

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

	if force || AskForConfirm("Delete this database pool?") == nil {
		databaseID := c.Args[0]
		poolID := c.Args[1]
		return c.Databases().DeletePool(databaseID, poolID)
	}

	return fmt.Errorf("Operation aborted.")
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

	doctl databases db list {cluster-id}`
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "db",
			Short: "Provides commands for managing individual databases within a cluster",
			Long:  `The subcommands under 'doctl databases db' are for managing specific databases that are served by a database cluster.

	You can use these commands to create and delete databases within a cluster, or simply get information about them.` + getClusterList,
		},
	}

	CmdBuilderWithDocs(cmd, RunDatabaseDBList, "list <database-id>", "Retrieves list of databases within a cluster", "This command retrieves the names of all databases being hosted in the specified database cluster." + getClusterList, Writer,
		aliasOpt("ls"), displayerType(&displayers.DatabaseDBs{}))
	CmdBuilderWithDocs(cmd, RunDatabaseDBGet, "get <database-id> <db-name>", "Retrieves the name of a database within a cluster", "This command retrieves name of the specified database hosted in the specified database cluster." + getClusterList  + getDBList,
		Writer, aliasOpt("g"), displayerType(&displayers.DatabaseDBs{}))
	CmdBuilderWithDocs(cmd, RunDatabaseDBCreate, "create <database-id> <db-name>",
		"Creates a database within a cluster", "This command creates a database with the specified name in the specified database cluster." + getClusterList, Writer, aliasOpt("c"))

	cmdDatabaseDBDelete := CmdBuilderWithDocs(cmd, RunDatabaseDBDelete,
		"delete <database-id> <db-name>", "Deletes the specified database from the cluster", "This command deletes the specified database from the specified database cluster." + getClusterList + getDBList, Writer, aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseDBDelete, doctl.ArgForce, doctl.ArgShortForce,
		false, "Delete the database without a confirmation prompt")

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

	if force || AskForConfirm("Delete this database?") == nil {
		databaseID := c.Args[0]
		dbID := c.Args[1]
		return c.Databases().DeleteDB(databaseID, dbID)
	}

	return fmt.Errorf("Operation aborted.")
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
			Short:   "database replica commands",
			Long:    "database is used to access database replica commands",
		},
	}

	CmdBuilder(cmd, RunDatabaseReplicaList, "list <database-id>",
		"list database replicas", Writer, aliasOpt("ls"),
		displayerType(&displayers.DatabaseReplicas{}))
	CmdBuilder(cmd, RunDatabaseReplicaGet, "get <database-id> <replica-name>",
		"get a database replica", Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseReplicas{}))

	cmdDatabaseReplicaCreate := CmdBuilder(cmd, RunDatabaseReplicaCreate,
		"create <database-id> <replica-name>", "create a database replica",
		Writer, aliasOpt("c"))
	AddStringFlag(cmdDatabaseReplicaCreate, doctl.ArgRegionSlug, "",
		defaultDatabaseRegion, "database replica region")
	AddStringFlag(cmdDatabaseReplicaCreate, doctl.ArgSizeSlug, "",
		defaultDatabaseNodeSize, "database replica size")
	AddStringFlag(cmdDatabaseReplicaCreate, doctl.ArgPrivateNetworkUUID, "",
		"", "private network uuid")

	cmdDatabaseReplicaDelete := CmdBuilder(cmd, RunDatabaseReplicaDelete,
		"delete <database-id> <replica-name>", "delete database replica",
		Writer, aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseReplicaDelete, doctl.ArgForce, doctl.ArgShortForce,
		false, "force database delete")

	CmdBuilder(cmd, RunDatabaseReplicaConnectionGet,
		"connection <database-id> <replica-name>",
		"get database replica connection info", Writer, aliasOpt("conn"))

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

	if force || AskForConfirm("Delete this database replica?") == nil {
		databaseID := c.Args[0]
		replicaID := c.Args[1]
		return c.Databases().DeleteReplica(databaseID, replicaID)
	}

	return fmt.Errorf("Operation aborted.")
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
