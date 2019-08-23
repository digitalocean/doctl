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
			Short:   "database commands",
			Long:    "database is used to access managed databases commands",
		},
	}

	CmdBuilder(cmd, RunDatabaseList, "list", "list database clusters", Writer, aliasOpt("ls"), displayerType(&displayers.Databases{}))
	CmdBuilder(cmd, RunDatabaseGet, "get <database-id>", "get a database cluster", Writer, aliasOpt("g"), displayerType(&displayers.Databases{}))

	createLongDesc := `create a database cluster

When creating a new database cluster, use the '--engine' flag to specify the
type. Use 'pg' for PostgreSQL, 'mysql' for MySQL, or 'redis' for Redis.
`

	cmdDatabaseCreate := CmdBuilder(cmd, RunDatabaseCreate, "create <name>", createLongDesc, Writer,
		aliasOpt("c"))
	AddIntFlag(cmdDatabaseCreate, doctl.ArgDatabaseNumNodes, "", defaultDatabaseNodeCount, "number of nodes in database cluster")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgRegionSlug, "", defaultDatabaseRegion, "database region")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgSizeSlug, "", defaultDatabaseNodeSize, "database size")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseEngine, "", defaultDatabaseEngine, "database engine")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgVersion, "", "", "database engine version")
	AddStringFlag(cmdDatabaseCreate, doctl.ArgPrivateNetworkUUID, "", "", "private network uuid")

	cmdDatabaseDelete := CmdBuilder(cmd, RunDatabaseDelete, "delete <database-id>", "delete database cluster", Writer,
		aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseDelete, doctl.ArgForce, doctl.ArgShortForce, false, "force database delete")

	CmdBuilder(cmd, RunDatabaseConnectionGet, "connection <database-id>", "get database cluster connection info", Writer,
		aliasOpt("conn"), displayerType(&displayers.DatabaseConnection{}))

	CmdBuilder(cmd, RunDatabaseBackupsList, "backups <database-id>", "list database cluster backups", Writer,
		aliasOpt("bu"), displayerType(&displayers.DatabaseBackups{}))

	cmdDatabaseResize := CmdBuilder(cmd, RunDatabaseResize, "resize <database-id>", "resize a database cluster", Writer,
		aliasOpt("rs"))
	AddIntFlag(cmdDatabaseResize, doctl.ArgDatabaseNumNodes, "", 0, "number of nodes in database cluster", requiredOpt())
	AddStringFlag(cmdDatabaseResize, doctl.ArgSizeSlug, "", "", "database size", requiredOpt())

	cmdDatabaseMigrate := CmdBuilder(cmd, RunDatabaseMigrate, "migrate <database-id", "migrate a database cluster", Writer,
		aliasOpt("m"))
	AddStringFlag(cmdDatabaseMigrate, doctl.ArgRegionSlug, "", "", "new database region", requiredOpt())
	AddStringFlag(cmdDatabaseMigrate, doctl.ArgPrivateNetworkUUID, "", "", "private network uuid")

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

	if force || AskForConfirm("delete this database cluster") == nil {
		id := c.Args[0]
		return c.Databases().Delete(id)
	}

	return fmt.Errorf("operation aborted")
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
			Short:   "maintenance window commands",
			Long:    "maintenance is used to access maintenance window commands for a database cluster",
		},
	}

	CmdBuilder(cmd, RunDatabaseMaintenanceGet, "get <database-id>",
		"get maintenance window info", Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseMaintenanceWindow{}))

	cmdDatabaseCreate := CmdBuilder(cmd, RunDatabaseMaintenanceUpdate,
		"update <database-id>", "update maintenance window", Writer, aliasOpt("u"))
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseMaintenanceDay, "", "",
		"new maintenance window day", requiredOpt())
	AddStringFlag(cmdDatabaseCreate, doctl.ArgDatabaseMaintenanceHour, "", "",
		"new maintenance window hour", requiredOpt())

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
			Short:   "database user commands",
			Long:    "database is used to access database user commands",
		},
	}

	CmdBuilder(cmd, RunDatabaseUserList, "list <database-id>", "list database users",
		Writer, aliasOpt("ls"), displayerType(&displayers.DatabaseUsers{}))
	CmdBuilder(cmd, RunDatabaseUserGet, "get <database-id> <user-id>",
		"get a database user", Writer, aliasOpt("g"),
		displayerType(&displayers.DatabaseUsers{}))
	CmdBuilder(cmd, RunDatabaseUserCreate, "create <database-id> <user-name>",
		"create a database user", Writer, aliasOpt("c"))

	cmdDatabaseUserDelete := CmdBuilder(cmd, RunDatabaseUserDelete,
		"delete <database-id> <user-id>", "delete database cluster",
		Writer, aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseUserDelete, doctl.ArgForce, doctl.ArgShortForce, false, "force database delete")

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

	if force || AskForConfirm("delete this database user") == nil {
		databaseID := c.Args[0]
		userID := c.Args[1]
		return c.Databases().DeleteUser(databaseID, userID)
	}

	return fmt.Errorf("operation aborted")
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
			Short:   "database pool commands",
			Long:    "database is used to access database pool commands",
		},
	}

	CmdBuilder(cmd, RunDatabasePoolList, "list <database-id>", "list database pools",
		Writer, aliasOpt("ls"), displayerType(&displayers.DatabasePools{}))
	CmdBuilder(cmd, RunDatabasePoolGet, "get <database-id> <pool-name>",
		"get a database pool", Writer, aliasOpt("g"),
		displayerType(&displayers.DatabasePools{}))
	cmdDatabasePoolCreate := CmdBuilder(cmd, RunDatabasePoolCreate,
		"create <database-id> <pool-name>", "create a database pool", Writer,
		aliasOpt("c"))
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolMode, "",
		"transaction", "pool mode")
	AddIntFlag(cmdDatabasePoolCreate, doctl.ArgSizeSlug, "", 0, "pool size",
		requiredOpt())
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolUserName, "", "",
		"database user name", requiredOpt())
	AddStringFlag(cmdDatabasePoolCreate, doctl.ArgDatabasePoolDBName, "", "",
		"database db name", requiredOpt())

	cmdDatabasePoolDelete := CmdBuilder(cmd, RunDatabasePoolDelete,
		"delete <database-id> <pool-name>", "delete database cluster", Writer,
		aliasOpt("rm"))
	AddBoolFlag(cmdDatabasePoolDelete, doctl.ArgForce, doctl.ArgShortForce,
		false, "force database delete")

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

	if force || AskForConfirm("delete this database pool") == nil {
		databaseID := c.Args[0]
		poolID := c.Args[1]
		return c.Databases().DeletePool(databaseID, poolID)
	}

	return fmt.Errorf("operation aborted")
}

func displayDatabasePools(c *CmdConfig, pools ...do.DatabasePool) error {
	item := &displayers.DatabasePools{DatabasePools: pools}
	return c.Display(item)
}

func databaseDB() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "db",
			Short: "database db commands",
			Long:  "database is used to access database db commands",
		},
	}

	CmdBuilder(cmd, RunDatabaseDBList, "list <database-id>", "list dbs", Writer,
		aliasOpt("ls"), displayerType(&displayers.DatabaseDBs{}))
	CmdBuilder(cmd, RunDatabaseDBGet, "get <database-id> <db-name>", "get a db",
		Writer, aliasOpt("g"), displayerType(&displayers.DatabaseDBs{}))
	CmdBuilder(cmd, RunDatabaseDBCreate, "create <database-id> <db-name>",
		"create a db", Writer, aliasOpt("c"))

	cmdDatabaseDBDelete := CmdBuilder(cmd, RunDatabaseDBDelete,
		"delete <database-id> <db-name>", "delete db", Writer, aliasOpt("rm"))
	AddBoolFlag(cmdDatabaseDBDelete, doctl.ArgForce, doctl.ArgShortForce,
		false, "force database delete")

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

	if force || AskForConfirm("delete this database db") == nil {
		databaseID := c.Args[0]
		dbID := c.Args[1]
		return c.Databases().DeleteDB(databaseID, dbID)
	}

	return fmt.Errorf("operation aborted")
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

	if force || AskForConfirm("delete this database replica") == nil {
		databaseID := c.Args[0]
		replicaID := c.Args[1]
		return c.Databases().DeleteReplica(databaseID, replicaID)
	}

	return fmt.Errorf("operation aborted")
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
