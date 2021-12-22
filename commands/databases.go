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
			Long:    "The commands under `doctl databases` are for managing your MySQL, Redis, and PostgreSQL database services.",
		},
	}

	clusterDetails := `

- The database ID, in UUID format
- The name you gave the database cluster
- The database engine (e.g. ` + "`" + `redis` + "`" + `, ` + "`" + `pg` + "`" + `, ` + "`" + `mysql` + "`" + `)
- The engine version (e.g. ` + "`" + `11` + "`" + ` for PostgreSQL version 11)
- The number of nodes in the database cluster
- The region the database cluster resides in (e.g. ` + "`" + `sfo2` + "`" + `, ` + "`" + `nyc1` + "`" + `)
- The current status of the database cluster (e.g. ` + "`" + `online` + "`" + `)
- The size of the machine running the database instance (e.g. ` + "`" + `db-s-1vcpu-1gb` + "`" + `)`

	CmdBuilder(cmd, RunDatabaseList, "list", "List your database clusters", `This command lists the database clusters associated with your account. The following details are provided:`+clusterDetails, Writer, aliasOpt("ls"), displayerType(&displayers.Databases{}))
	CmdBuilder(cmd, RunDatabaseGet, "get <database-id>", "Get details for a database cluster", `This command retrieves the following details about the specified database cluster: `+clusterDetails+`
- A connection string for the database cluster
- The date and time when the database cluster was created`+databaseListDetails, Writer, aliasOpt("g"), displayerType(&displayers.Databases{}))

	nodeSizeDetails := "The size of the nodes in the database cluster, e.g. `db-s-1vcpu-1gb`` for a 1 CPU, 1GB node"
	nodeNumberDetails := "The number of nodes in the database cluster. Valid values are are 1-3. In addition to the primary node, up to two standby nodes may be added for high availability."
	cmdDatabaseCreate := CmdBuilder(cmd, RunDatabaseCreate, "create <name>", "Create a database cluster", `This command creates a database cluster with the specified name.

There are a number of flags that customize the configuration, all of which are optional. Without any flags set, a single-node, single-CPU PostgreSQL database cluster will be created.`, Writer,
		aliasOpt("c"))
	AddIntFlag(cmdDatabaseCreate, doctl.ArgDatabaseNumNodes, "", defaultDatabaseNodeCount, nodeNumberDetails)
	AddStringFlag(cmdDatabaseCreate, doctl.ArgRegionSlug, "", defaultDatabaseRegion, "The region where the database cluster will be created, e.g. `nyc1` or `sfo2`")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgSizeSlug, "", defaultDatabaseNodeSize, nodeSizeDetails)
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseEngine, "", defaultDatabaseEngine, "The database engine to be used for the cluster. Possible values are: `pg` for PostgreSQL, `mysql`, and `redis`.")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgVersion, "", "", "The database engine version, e.g. 11 for PostgreSQL version 11")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgPrivateNetworkUUID, "", "", "The UUID of a VPC to create the database cluster in; the default VPC for the region will be used if excluded")

	cmdDatabaseDelete := CmdBuilder(cmd, RunDatabaseDelete, "delete <database-id>", "Delete a database cluster", `This command deletes the database cluster with the given ID.

To retrieve a list of your database clusters and their IDs, call `+"`"+`doctl databases list`+"`"+`.`, Writer,
		aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the database cluster without a confirmation prompt")

	CmdBuilder(cmd, RunDatabaseConnectionGet, "connection <database-id>", "Retrieve connection details for a database cluster", `This command retrieves the following connection details for a database cluster:

- The connection string for the database cluster
- The default database name
- The fully-qualified domain name of the publicly-connectable host
- The port on which the database is listening for connections
- The default username
- The randomly-generated password for the default username
- A boolean indicating if the connection should be made over SSL

While these connection details will work, you may wish to use different connection details, such as the private hostname, a custom username, or a different database.`, Writer,
		aliasOpt("conn"), displayerType(&displayers.DatabaseConnection{}))

	CmdBuilder(cmd, RunDatabaseBackupsList, "backups <database-id>", "List database cluster backups", `This command retrieves a list of backups created for the specified database cluster.

The list contains the size in GB, and the date and time the backup was taken.`, Writer,
		aliasOpt("bu"), displayerType(&displayers.DatabaseBackups{}))

	cmdDatabaseResize := CmdBuilder(cmd, RunDatabaseResize, "resize <database-id>", "Resize a database cluster", `This command resizes the specified database cluster.

You must specify the desired number of nodes and size of the nodes. For example:

	doctl databases resize ca9f591d-9999-5555-a0ef-1c02d1d1e352 --num-nodes 2 --size db-s-16vcpu-64gb
			
Database nodes cannot be resized to smaller sizes due to the risk of data loss.`, Writer,
		aliasOpt("rs"))
	AddIntFlag(cmdDatabaseResize, doctl.ArgDatabaseNumNodes, "", 0, nodeNumberDetails, requiredOpt())
	AddStringFlag(cmdDatabaseResize, doctl.ArgSizeSlug, "", "", nodeSizeDetails, requiredOpt())

	cmdDatabaseMigrate := CmdBuilder(cmd, RunDatabaseMigrate, "migrate <database-id>", "Migrate a database cluster to a new region", `This command migrates the specified database cluster to a new region`, Writer,
		aliasOpt("m"))
	AddStringFlag(cmdDatabaseMigrate, doctl.ArgRegionSlug, "", "", "The region to which the database cluster should be migrated, e.g. `sfo2` or `nyc3`.", requiredOpt())
	AddStringFlag(cmdDatabaseMigrate, doctl.ArgPrivateNetworkUUID, "", "", "The UUID of a VPC to create the database cluster in; the default VPC for the region will be used if excluded")

	cmd.AddCommand(databaseReplica())
	cmd.AddCommand(databaseMaintenanceWindow())
	cmd.AddCommand(databaseUser())
	cmd.AddCommand(databaseDB())
	cmd.AddCommand(databasePool())
	cmd.AddCommand(sqlMode())
	cmd.AddCommand(databaseFirewalls())

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
			Short:   "Display commands for scheduling automatic maintenance on your database cluster",
			Long: `The ` + "`" + `doctl databases maintenance-window` + "`" + ` commands allow you to schedule, and check the schedule of, maintenance windows for your databases.

Maintenance windows are hour-long blocks of time during which DigitalOcean performs automatic maintenance on databases every week. During this time, health checks, security updates, version upgrades, and more are performed.`,
		},
	}

	CmdBuilder(cmd, RunDatabaseMaintenanceGet, "get <database-id>",
		"Retrieve details about a database cluster's maintenance windows", `This command retrieves the following information on currently-scheduled maintenance windows for the specified database cluster:

- The day of the week the maintenance window occurs
- The hour in UTC when maintenance updates will be applied, in 24 hour format (e.g. "16:00")
- A boolean representing whether maintence updates are currently pending

To see a list of your databases and their IDs, run `+"`"+`doctl databases list`+"`"+`.`, Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseMaintenanceWindow{}))

	cmdDatabaseCreate := CmdBuilder(cmd, RunDatabaseMaintenanceUpdate,
		"update <database-id>", "Update the maintenance window for a database cluster", `This command allows you to update the maintenance window for the specified database cluster.

Maintenance windows are hour-long blocks of time during which DigitalOcean performs automatic maintenance on databases every week. During this time, health checks, security updates, version upgrades, and more are performed.

To change the maintenance window for your database cluster, specify a day of the week and an hour of that day during which you would prefer such maintenance would occur.

	doctl databases maintenance-window ca9f591d-f38h-5555-a0ef-1c02d1d1e35 update --day tuesday --hour 16:00

To see a list of your databases and their IDs, run `+"`"+`doctl databases list`+"`"+`.`, Writer, aliasOpt("u"))
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseMaintenanceDay, "", "",
		"The day of the week the maintenance window occurs (e.g. 'tuesday')", requiredOpt())
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseMaintenanceHour, "", "",
		"The hour in UTC when maintenance updates will be applied, in 24 hour format (e.g. '16:00')", requiredOpt())

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

	userDetailsDesc := `

- The username for the user
- The password for the user
- The user's role. The value will be either "primary" or "normal".

Primary user accounts are created by DigitalOcean at database cluster creation time and can't be deleted. Normal user accounts are created by you. Both have administrative privileges on the database cluster.

To retrieve a list of your databases and their IDs, call ` + "`" + `doctl databases list` + "`" + `.`
	CmdBuilder(cmd, RunDatabaseUserList, "list <database-id>", "Retrieve list of database users",
		`This command retrieves a list of users for the specified database with the following details:`+userDetailsDesc, Writer, aliasOpt("ls"), displayerType(&displayers.DatabaseUsers{}))
	CmdBuilder(cmd, RunDatabaseUserGet, "get <database-id> <user-name>",
		"Retrieve details about a database user", `This command retrieves the following details about the specified user:`+userDetailsDesc+`

To retrieve a list of database users for a database, call `+"`"+`doctl databases user list <database-id>`+"`"+`.`, Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseUsers{}))
	cmdDatabaseUserCreate := CmdBuilder(cmd, RunDatabaseUserCreate, "create <database-id> <user-name>",
		"Create a database user", `This command creates a user with the username you specify, who will be granted access to the database cluster you specify.

The user will be created with the role set to `+"`"+`normal`+"`"+`, and given an automatically-generated password.

To retrieve a list of your databases and their IDs, call `+"`"+`doctl databases list`+"`"+`.`, Writer, aliasOpt("c"))

	AddStringFlag(cmdDatabaseUserCreate, doctl.ArgDatabaseUserMySQLAuthPlugin, "", "",
		"set auth mode for MySQL users")

	CmdBuilder(cmd, RunDatabaseUserResetAuth, "reset <database-id> <user-name> <new-auth-mode>",
		"Resets a user's auth", "This command resets the auth password or the MySQL auth plugin for a given user. It will return the new user credentials. When resetting MySQL auth, valid values for `<new-auth-mode>` are `caching_sha2_password` and `mysql_native_password`.", Writer, aliasOpt("rs"))

	cmdDatabaseUserDelete := CmdBuilder(cmd, RunDatabaseUserDelete,
		"delete <database-id> <user-id>", "Delete a database user", `This command deletes the user with the username you specify, whose account was given access to the database cluster you specify.

To retrieve a list of your databases and their IDs, call `+"`"+`doctl databases list`+"`"+`.`, Writer, aliasOpt("rm"))
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

	user, err := c.Databases().CreateUser(databaseID, req)
	if err != nil {
		return err
	}

	return displayDatabaseUsers(c, *user)
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

func databasePool() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "pool",
			Aliases: []string{"p"},
			Short:   "Display commands for managing connection pools",
			Long: `The subcommands under ` + "`" + `doctl databases pool` + "`" + ` are for managing connection pools for your database cluster.

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
	CmdBuilder(cmd, RunDatabasePoolList, "list <database-id>", "List connection pools for a database cluster", `This command lists the existing connection pools for the specified database. The following information will be returned:`+connectionPoolDetails,
		Writer, aliasOpt("ls"), displayerType(&displayers.DatabasePools{}))
	CmdBuilder(cmd, RunDatabasePoolGet, "get <database-id> <pool-name>",
		"Retrieve information about a database connection pool", `This command retrieves the following information about the specified connection pool for the specified database cluster:`+connectionPoolDetails+getPoolDetails, Writer, aliasOpt("g"),
		displayerType(&displayers.DatabasePools{}))
	cmdDatabasePoolCreate := CmdBuilder(cmd, RunDatabasePoolCreate,
		"create <database-id> <pool-name>", "Create a connection pool for a database", `This command creates a connection pool for the specified database cluster and gives it the specified name.

You must also use flags to specify the target database, pool size, and database user's username that will be used for the pool. An example call would be:

	pool create ca9f591d-fb58-5555-a0ef-1c02d1d1e352 mypool --db defaultdb --size 10 --user doadmin

The pool size is the minimum number of connections the pool can handle. The maximum pool size varies based on the size of the cluster.

There’s no perfect formula to determine how large your pool should be, but there are a few good guidelines to keep in mind:

- A large pool will stress your database at similar levels as that number of clients would alone.
- A pool that’s much smaller than the number of clients communicating with the database can act as a bottleneck, reducing the rate when your database receives and responds to transactions.

We recommend starting with a pool size of about half your available connections and adjusting later based on performance. If you see slow query responses, check the CPU usage on the database’s Overview tab. We recommend decreasing your pool size if CPU usage is high, and increasing your pool size if it’s low.`+getPoolDetails, Writer,
		aliasOpt("c"))
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolMode, "",
		"transaction", "The pool mode for the connection pool, e.g. `session`, `transaction`, and `statement`")
	AddIntFlag(cmdDatabasePoolCreate, doctl.ArgSizeSlug, "", 0, "pool size",
		requiredOpt())
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolUserName, "", "",
		"The username for the database user", requiredOpt())
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolDBName, "", "",
		"The name of the specific database within the database cluster", requiredOpt())

	cmdDatabasePoolDelete := CmdBuilder(cmd, RunDatabasePoolDelete,
		"delete <database-id> <pool-name>", "Delete a connection pool for a database", `This command deletes the specified connection pool for the specified database cluster.`+getPoolDetails, Writer,
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

	doctl databases db list {cluster-id}`
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "db",
			Short: "Display commands for managing individual databases within a cluster",
			Long: `The subcommands under ` + "`" + `doctl databases db` + "`" + ` are for managing specific databases that are served by a database cluster.

You can use these commands to create and delete databases within a cluster, or simply get information about them.` + getClusterList,
		},
	}

	CmdBuilder(cmd, RunDatabaseDBList, "list <database-id>", "Retrieve a list of databases within a cluster", "This command retrieves the names of all databases being hosted in the specified database cluster."+getClusterList, Writer,
		aliasOpt("ls"), displayerType(&displayers.DatabaseDBs{}))
	CmdBuilder(cmd, RunDatabaseDBGet, "get <database-id> <db-name>", "Retrieve the name of a database within a cluster", "This command retrieves the name of the specified database hosted in the specified database cluster."+getClusterList+getDBList,
		Writer, aliasOpt("g"), displayerType(&displayers.DatabaseDBs{}))
	CmdBuilder(cmd, RunDatabaseDBCreate, "create <database-id> <db-name>",
		"Create a database within a cluster", "This command creates a database with the specified name in the specified database cluster."+getClusterList, Writer, aliasOpt("c"))

	cmdDatabaseDBDelete := CmdBuilder(cmd, RunDatabaseDBDelete,
		"delete <database-id> <db-name>", "Delete the specified database from the cluster", "This command deletes the specified database from the specified database cluster."+getClusterList+getDBList, Writer, aliasOpt("rm"))
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
			Long: `The subcommands under ` + "`" + `doctl databases replica` + "`" + ` enable the management of read-only replicas associated with a database cluster.

In addition to primary nodes in a database cluster, you can create up to 2 read-only replica nodes (also referred to as "standby nodes") to maintain high availability.`,
		},
	}
	howToGetReplica := `

This command requires that you pass in the replica's name, which you can retrieve by querying a database ID:

	doctl databases replica list ca9f591d-5555-5555-a0ef-1c02d1d1e352`
	replicaDetails := `

- The name of the replica
- The region where the database cluster is located (e.g. ` + "`" + `nyc3` + "`" + `, ` + "`" + `sfo2` + "`" + `)
- The status of the replica (possible values are ` + "`" + `forking` + "`" + ` and ` + "`" + `active` + "`" + `)
`
	CmdBuilder(cmd, RunDatabaseReplicaList, "list <database-id>", "Retrieve list of read-only database replicas", `Lists the following details for read-only replicas for the specified database cluster.`+replicaDetails+databaseListDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.DatabaseReplicas{}))
	CmdBuilder(cmd, RunDatabaseReplicaGet, "get <database-id> <replica-name>", "Retrieve information about a read-only database replica",
		`Gets the following details for the specified read-only replica for the specified database cluster:

- The name of the replica
- Information required to connect to the read-only replica
- The region where the database cluster is located (e.g. `+"`"+`nyc3`+"`"+`, `+"`"+`sfo2`+"`"+`)
- The status of the replica (possible values are `+"`"+`creating`+"`"+`, `+"`"+`forking`+"`"+`, or `+"`"+`active`+"`"+`)
- A time value given in ISO8601 combined date and time format that represents when the read-only replica was created.`+howToGetReplica+databaseListDetails,
		Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseReplicas{}))

	cmdDatabaseReplicaCreate := CmdBuilder(cmd, RunDatabaseReplicaCreate,
		"create <database-id> <replica-name>", "Create a read-only database replica", `This command creates a read-only database replica for the specified database cluster, giving it the specified name.`+databaseListDetails,
		Writer, aliasOpt("c"))
	AddStringFlag(cmdDatabaseReplicaCreate, doctl.ArgRegionSlug, "",
		defaultDatabaseRegion, "Specifies the region (e.g. nyc3, sfo2) in which to create the replica")
	AddStringFlag(cmdDatabaseReplicaCreate, doctl.ArgSizeSlug, "",
		defaultDatabaseNodeSize, "Specifies the machine size for the replica (e.g. db-s-1vcpu-1gb). Must be the same or equal to the original.")
	AddStringFlag(cmdDatabaseReplicaCreate, doctl.ArgPrivateNetworkUUID, "",
		"", "The UUID of a VPC to create the replica in; the default VPC for the region will be used if excluded")

	cmdDatabaseReplicaDelete := CmdBuilder(cmd, RunDatabaseReplicaDelete,
		"delete <database-id> <replica-name>", "Delete a read-only database replica",
		`Delete the specified read-only replica for the specified database cluster.`+howToGetReplica+databaseListDetails,
		Writer, aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseReplicaDelete, doctl.ArgForce, doctl.ArgShortForce,
		false, "Deletes the replica without a confirmation prompt")

	CmdBuilder(cmd, RunDatabaseReplicaConnectionGet,
		"connection <database-id> <replica-name>",
		"Retrieve information for connecting to a read-only database replica",
		`This command retrieves information for connecting to the specified read-only database replica in the specified database cluster`+howToGetReplica+databaseListDetails, Writer, aliasOpt("conn"))

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
			Long:    "The subcommands of `doctl databases sql-mode` are used to view and configure a MySQL database cluster's global SQL modes.",
		},
	}

	getSqlModeDesc := "This command displays the the configured SQL modes for the specified MySQL database cluster."
	CmdBuilder(cmd, RunDatabaseGetSQLModes, "get <database-id>",
		"Get a MySQL database cluster's SQL modes", getSqlModeDesc, Writer,
		displayerType(&displayers.DatabaseSQLModes{}), aliasOpt("g"))
	setSqlModeDesc := `This command configures the SQL modes for the specified MySQL database cluster. The SQL modes should be provided as a space separated list.

This will replace the existing SQL mode configuration completely. Include all of the current values when adding a new one.
`
	CmdBuilder(cmd, RunDatabaseSetSQLModes, "set <database-id> <sql-mode-1> ... <sql-mode-n>",
		"Set a MySQL database cluster's SQL modes", setSqlModeDesc, Writer, aliasOpt("s"))

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

func databaseFirewalls() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "firewalls",
			Aliases: []string{"fw"},
			Short:   `Display commands to manage firewall rules (called` + "`" + `trusted sources` + "`" + ` in the control panel) for database clusters`,
			Long:    `The subcommands under ` + "`" + `doctl databases firewalls` + "`" + ` enable the management of firewalls for database clusters`,
		},
	}

	firewallRuleDetails := `
This command lists the following details for each firewall rule in a given database:

	- The UUID of the firewall rule.
	- The Cluster UUID for the database cluster to which the rule is applied.
	- The Type of resource that the firewall rule allows to access the database cluster. The possible values are: "droplet", "k8s", "ip_addr", "tag", or "app".
	- The Value, which is either the ID of the specific resource, the name of a tag applied to a group of resources, or the IP address that the firewall rule allows to access the database cluster.
	- The Time value given in ISO8601 combined date and time format that represents when the firewall rule was created.
	`
	databaseFirewallRuleDetails := `

This command requires the ID of a database cluster, which you can retrieve by calling:

	doctl databases list`

	databaseFirewallRulesTxt := "A comma-separated list of firewall rules of format type:value, e.g.: `type:value`"

	databaseFirewallUpdateDetails := `
Use this command to replace the firewall rules of a given database. This command requires the ID of a database cluster, which you can retrieve by calling:

	doctl databases list 
	
This command also requires a --rule flag. You can pass in multiple --rule flags. Each rule passed in to the --rule flag must be of format type:value
	- "type" is the type of resource that the firewall rule allows to access the database cluster. The possible values for type are:  "droplet", "k8s", "ip_addr", "tag", or "app"
	- "value" is either the ID of the specific resource, the name of a tag applied to a group of resources, or the IP address that the firewall rule allows to access the database cluster

For example:
	
	doctl databases firewalls replace d1234-1c12-1234-b123-12345c4789 --rule tag:backend --rule ip_addr:0.0.0.0

	or

	databases firewalls replace d1234-1c12-1234-b123-12345c4789 --rule tag:backend,ip_addr:0.0.0.0

This would replace the firewall rules for database of id d1234-1c12-1234-b123-12345c4789 with the two rules passed above (tag:backend, ip_addr:0.0.0.0)
	`

	databaseFirewallAddDetails :=
		`
Use this command to append a single rule to the existing firewall rules of a given database. This command requires the ID of a database cluster, which you can retrieve by calling:

	doctl databases list

This command also requires a --rule flag. Each rule passed in to the --rule flag must be of format type:value
	- "type" is the type of resource that the firewall rule allows to access the database cluster. The possible values for type are:  "droplet", "k8s", "ip_addr", "tag", or "app"
	- "value" is either the ID of the specific resource, the name of a tag applied to a group of resources, or the IP address that the firewall rule allows to access the database cluster

For example:

	doctl databases firewalls append d1234-1c12-1234-b123-12345c4789 --rule tag:backend

This would append the firewall rule "tag:backend" for database of id d1234-1c12-1234-b123-12345c4789`

	databaseFirewallRemoveDetails :=
		`
Use this command to remove an existing, single rule from the list of firewall rules for a given database. This command requires the ID of a database cluster, which you can retrieve by calling:

	doctl databases list

This command also requires a --uuid flag. You must pass in the UUID of the firewall rule you'd like to remove. You can retrieve the firewall rule's UUIDs by calling:

	doctl database firewalls list <db-id>

For example:

	doctl databases firewalls remove d1234-1c12-1234-b123-12345c4789 --uuid 12345d-1234-123d-123x-123eee456e

This would remove the firewall rule of uuid 12345d-1234-123d-123x-123eee456e for database of id d1234-1c12-1234-b123-12345c4789
			`

	CmdBuilder(cmd, RunDatabaseFirewallRulesList, "list <database-id>", "Retrieve a list of firewall rules for a given database", firewallRuleDetails+databaseFirewallRuleDetails,
		Writer, aliasOpt("ls"))

	cmdDatabaseFirewallUpdate := CmdBuilder(cmd, RunDatabaseFirewallRulesUpdate, "replace <db-id> --rules type:value [--rule type:value]", "Replaces the firewall rules for a given database. The rules passed in to the --rules flag will replace the firewall rules previously assigned to the database,", databaseFirewallUpdateDetails,
		Writer, aliasOpt("r"))
	AddStringSliceFlag(cmdDatabaseFirewallUpdate, doctl.ArgDatabaseFirewallRule, "", []string{}, databaseFirewallRulesTxt, requiredOpt())

	cmdDatabaseFirewallCreate := CmdBuilder(cmd, RunDatabaseFirewallRulesAppend, "append <db-id> --rule type:value", "Add a database firewall rule to a given database", databaseFirewallAddDetails,
		Writer, aliasOpt("a"))
	AddStringFlag(cmdDatabaseFirewallCreate, doctl.ArgDatabaseFirewallRule, "", "", "", requiredOpt())

	cmdDatabaseFirewallRemove := CmdBuilder(cmd, RunDatabaseFirewallRulesRemove, "remove <firerule-uuid>", "Remove a firewall rule for a given database", databaseFirewallRemoveDetails,
		Writer, aliasOpt("rm"))
	AddStringFlag(cmdDatabaseFirewallRemove, doctl.ArgDatabaseFirewallRuleUUID, "", "", "", requiredOpt())

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
