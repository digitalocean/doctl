package commands

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	// godo mocks

	testGODOConnection = &godo.DatabaseConnection{
		URI:      "postgres://doadmin:foobaz@foo-foobar-do-user-1-0.db.ondigitalocean.com:25060/defaultdb?sslmode=require",
		Database: "defaultdb",
		Host:     "foo-foobar-do-user-1-0.db.ondigitalocean.com",
		Port:     25060,
		Password: "foobaz",
		User:     "doadmin",
		SSL:      true,
	}

	testGODOUser = &godo.DatabaseUser{
		Name:     "doadmin",
		Role:     "primary",
		Password: "foobaz",
	}

	testGODOMainWindow = &godo.DatabaseMaintenanceWindow{
		Day:  "monday",
		Hour: "10:00",
	}

	// doctl mocks

	testDBCluster = do.Database{
		Database: &godo.Database{
			ID:                "ea4652de-4fe0-11e9-b7ab-df1ef30eab9e",
			Name:              "sunny-db-cluster",
			RegionSlug:        "nyc1",
			EngineSlug:        "pg",
			VersionSlug:       "11",
			NumNodes:          3,
			SizeSlug:          "db-s-1vcpu-2gb",
			DBNames:           []string{"defaultdb"},
			CreatedAt:         time.Now(),
			Status:            "online",
			Connection:        testGODOConnection,
			MaintenanceWindow: testGODOMainWindow,
			Users: []godo.DatabaseUser{
				*testGODOUser,
			},
		},
	}

	testDBClusters = do.Databases{
		testDBCluster,
	}

	testDBConnection = do.DatabaseConnection{
		DatabaseConnection: testGODOConnection,
	}

	testDBUser = do.DatabaseUser{
		DatabaseUser: testGODOUser,
	}

	testDBUsers = do.DatabaseUsers{
		testDBUser,
	}

	testDBMainWindow = do.DatabaseMaintenanceWindow{
		DatabaseMaintenanceWindow: testGODOMainWindow,
	}

	testDBBackup = do.DatabaseBackup{
		DatabaseBackup: &godo.DatabaseBackup{
			CreatedAt:     time.Now(),
			SizeGigabytes: 2.03464192,
		},
	}

	testDBBackups = do.DatabaseBackups{
		testDBBackup,
	}

	testDBReplica = do.DatabaseReplica{
		DatabaseReplica: &godo.DatabaseReplica{
			Name:       "sunny-db-replica",
			Connection: testGODOConnection,
			Region:     "nyc1",
			Status:     "online",
			CreatedAt:  time.Now(),
		},
	}

	testDBReplicas = do.DatabaseReplicas{
		testDBReplica,
	}

	testDB = do.DatabaseDB{
		DatabaseDB: &godo.DatabaseDB{
			Name: "defaultdb",
		},
	}

	testDBs = do.DatabaseDBs{
		testDB,
	}

	testDBPool = do.DatabasePool{
		DatabasePool: &godo.DatabasePool{
			User:       testDBUser.Name,
			Name:       "sunny-db-pool",
			Database:   testDB.Name,
			Size:       10,
			Mode:       "transaction",
			Connection: testGODOConnection,
		},
	}

	testDBPools = do.DatabasePools{
		testDBPool,
	}

	errTest = errors.New("error")
)

func TestDatabasesCommand(t *testing.T) {
	cmd := Databases()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"list",
		"get",
		"create",
		"delete",
		"connection",
		"migrate",
		"resize",
		"backups",
		"replica",
		"maintenance-window",
		"user",
		"pool",
		"db",
	)
}

func TestDatabaseMaintenanceWindowCommand(t *testing.T) {
	cmd := databaseMaintenanceWindow()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"update",
		"get",
	)
}

func TestDatabaseUserCommand(t *testing.T) {
	cmd := databaseUser()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"list",
		"get",
		"create",
		"delete",
	)
}

func TestDatabasePoolCommand(t *testing.T) {
	cmd := databasePool()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"list",
		"get",
		"create",
		"delete",
	)
}

func TestDatabaseDBCommand(t *testing.T) {
	cmd := databaseDB()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"list",
		"get",
		"create",
		"delete",
	)
}

func TestDatabaseReplicaCommand(t *testing.T) {
	cmd := databaseReplica()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"list",
		"get",
		"create",
		"delete",
		"connection",
	)
}

func TestDatabasesGet(t *testing.T) {
	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("Get", testDBCluster.ID).Return(&testDBCluster, nil)
		config.Args = append(config.Args, testDBCluster.ID)
		err := RunDatabaseGet(config)
		assert.NoError(t, err)
	})

	// Error
	notFound := "not-found"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("Get", notFound).Return(nil, errTest)
		config.Args = append(config.Args, notFound)
		err := RunDatabaseGet(config)
		assert.Error(t, err)
	})

	// ID not provided
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDatabaseGet(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestDatabasesList(t *testing.T) {
	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("List").Return(testDBClusters, nil)
		err := RunDatabaseList(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("List").Return(nil, errTest)
		err := RunDatabaseList(config)
		assert.Error(t, err)
	})
}

func TestDatabasesCreate(t *testing.T) {
	r := &godo.DatabaseCreateRequest{
		Name:       testDBCluster.Name,
		Region:     testDBCluster.RegionSlug,
		Version:    testDBCluster.VersionSlug,
		EngineSlug: testDBCluster.EngineSlug,
		NumNodes:   testDBCluster.NumNodes,
		SizeSlug:   testDBCluster.SizeSlug,
	}

	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("Create", r).Return(&testDBCluster, nil)

		config.Args = append(config.Args, testDBCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testDBCluster.RegionSlug)
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, testDBCluster.SizeSlug)
		config.Doit.Set(config.NS, doctl.ArgVersion, testDBCluster.VersionSlug)
		config.Doit.Set(config.NS, doctl.ArgDatabaseEngine, testDBCluster.EngineSlug)
		config.Doit.Set(config.NS, doctl.ArgDatabaseNumNodes, testDBCluster.NumNodes)

		err := RunDatabaseCreate(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On(
			"Create",
			mock.AnythingOfType("*godo.DatabaseCreateRequest"),
		).Return(nil, errTest)

		config.Args = append(config.Args, testDBCluster.Name)
		err := RunDatabaseCreate(config)
		assert.EqualError(t, err, "error")
	})
}

func TestDatabasesDelete(t *testing.T) {
	// Successful
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("Delete", testDBCluster.ID).Return(nil)

		config.Args = append(config.Args, testDBCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunDatabaseDelete(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("Delete", testDBCluster.ID).Return(errTest)

		config.Args = append(config.Args, testDBCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunDatabaseDelete(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabaseMigrate(t *testing.T) {
	r := &godo.DatabaseMigrateRequest{
		Region: testDBCluster.RegionSlug,
	}

	// Success
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("Migrate", testDBCluster.ID, r).Return(nil)
		config.Args = append(config.Args, testDBCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testDBCluster.RegionSlug)

		err := RunDatabaseMigrate(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("Migrate", testDBCluster.ID, r).Return(errTest)
		config.Args = append(config.Args, testDBCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testDBCluster.RegionSlug)

		err := RunDatabaseMigrate(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabaseResize(t *testing.T) {
	r := &godo.DatabaseResizeRequest{
		SizeSlug: testDBCluster.SizeSlug,
		NumNodes: testDBCluster.NumNodes,
	}

	// Success
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("Resize", testDBCluster.ID, r).Return(nil)
		config.Args = append(config.Args, testDBCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, testDBCluster.SizeSlug)
		config.Doit.Set(config.NS, doctl.ArgDatabaseNumNodes, testDBCluster.NumNodes)

		err := RunDatabaseResize(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("Resize", testDBCluster.ID, r).Return(errTest)
		config.Args = append(config.Args, testDBCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, testDBCluster.SizeSlug)
		config.Doit.Set(config.NS, doctl.ArgDatabaseNumNodes, testDBCluster.NumNodes)

		err := RunDatabaseResize(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabaseListBackups(t *testing.T) {
	// Success
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("ListBackups", testDBCluster.ID).Return(testDBBackups, nil)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseBackupsList(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("ListBackups", testDBCluster.ID).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseBackupsList(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabaseConnectionGet(t *testing.T) {
	// Success
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetConnection", testDBCluster.ID).Return(&testDBConnection, nil)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseConnectionGet(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetConnection", testDBCluster.ID).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseConnectionGet(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabaseGetMaintenance(t *testing.T) {
	// Success
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetMaintenance", testDBCluster.ID).Return(&testDBMainWindow, nil)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseMaintenanceGet(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetMaintenance", testDBCluster.ID).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseMaintenanceGet(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabaseUpdateMaintenance(t *testing.T) {
	r := &godo.DatabaseUpdateMaintenanceRequest{
		Hour: testDBMainWindow.Hour,
		Day:  testDBMainWindow.Day,
	}

	// Success
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("UpdateMaintenance", testDBCluster.ID, r).Return(nil)
		config.Args = append(config.Args, testDBCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgDatabaseMaintenanceDay, testDBMainWindow.Day)
		config.Doit.Set(config.NS, doctl.ArgDatabaseMaintenanceHour, testDBMainWindow.Hour)

		err := RunDatabaseMaintenanceUpdate(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("UpdateMaintenance", testDBCluster.ID, r).Return(errTest)
		config.Args = append(config.Args, testDBCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgDatabaseMaintenanceDay, testDBMainWindow.Day)
		config.Doit.Set(config.NS, doctl.ArgDatabaseMaintenanceHour, testDBMainWindow.Hour)

		err := RunDatabaseMaintenanceUpdate(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabasesUserGet(t *testing.T) {
	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetUser", testDBCluster.ID, testDBUser.Name).Return(&testDBUser, nil)
		config.Args = append(config.Args, testDBCluster.ID, testDBUser.Name)
		err := RunDatabaseUserGet(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetUser", testDBCluster.ID, testDBUser.Name).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID, testDBUser.Name)
		err := RunDatabaseUserGet(config)
		assert.Error(t, err)
	})

	// ID not provided
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDatabaseUserGet(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestDatabasesListUsers(t *testing.T) {
	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("ListUsers", testDBCluster.ID).Return(testDBUsers, nil)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseUserList(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("ListUsers", testDBCluster.ID).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseUserList(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabaseUserCreate(t *testing.T) {
	r := &godo.DatabaseCreateUserRequest{
		Name: testDBUser.Name,
	}

	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("CreateUser", testDBCluster.ID, r).Return(&testDBUser, nil)

		config.Args = append(config.Args, testDBCluster.ID, testDBUser.Name)

		err := RunDatabaseUserCreate(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On(
			"CreateUser",
			testDBCluster.ID,
			mock.AnythingOfType("*godo.DatabaseCreateUserRequest"),
		).Return(nil, errTest)

		config.Args = append(config.Args, testDBCluster.ID, testDBUser.Name)
		err := RunDatabaseUserCreate(config)
		assert.EqualError(t, err, "error")
	})
}

func TestDatabasesUserDelete(t *testing.T) {
	// Successful
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("DeleteUser", testDBCluster.ID, testDBUser.Name).Return(nil)

		config.Args = append(config.Args, testDBCluster.ID, testDBUser.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunDatabaseUserDelete(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("DeleteUser", testDBCluster.ID, testDBUser.Name).Return(errTest)

		config.Args = append(config.Args, testDBCluster.ID, testDBUser.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunDatabaseUserDelete(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabasesPoolGet(t *testing.T) {
	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetPool", testDBCluster.ID, testDBPool.Name).Return(&testDBPool, nil)
		config.Args = append(config.Args, testDBCluster.ID, testDBPool.Name)
		err := RunDatabasePoolGet(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetPool", testDBCluster.ID, testDBPool.Name).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID, testDBPool.Name)
		err := RunDatabasePoolGet(config)
		assert.Error(t, err)
	})

	// ID not provided
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDatabasePoolGet(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestDatabasesListPools(t *testing.T) {
	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("ListPools", testDBCluster.ID).Return(testDBPools, nil)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabasePoolList(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("ListPools", testDBCluster.ID).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabasePoolList(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabasePoolCreate(t *testing.T) {
	pool := *(testDBPool.DatabasePool)
	pool.Connection = nil

	r := &godo.DatabaseCreatePoolRequest{
		Name:     pool.Name,
		User:     pool.User,
		Mode:     pool.Mode,
		Size:     pool.Size,
		Database: pool.Database,
	}

	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("CreatePool", testDBCluster.ID, r).Return(&testDBPool, nil)

		config.Args = append(config.Args, testDBCluster.ID, testDBPool.Name)
		config.Doit.Set(config.NS, doctl.ArgDatabasePoolDBName, testDB.Name)
		config.Doit.Set(config.NS, doctl.ArgDatabasePoolMode, testDBPool.Mode)
		config.Doit.Set(config.NS, doctl.ArgDatabasePoolSize, testDBPool.Size)
		config.Doit.Set(config.NS, doctl.ArgDatabasePoolUserName, testDBUser.Name)

		err := RunDatabasePoolCreate(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On(
			"CreatePool",
			testDBCluster.ID,
			mock.AnythingOfType("*godo.DatabaseCreatePoolRequest"),
		).Return(nil, errTest)

		config.Args = append(config.Args, testDBCluster.ID, testDBPool.Name)
		err := RunDatabasePoolCreate(config)
		assert.EqualError(t, err, "error")
	})
}

func TestDatabasesPoolDelete(t *testing.T) {
	// Successful
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("DeletePool", testDBCluster.ID, testDBPool.Name).Return(nil)

		config.Args = append(config.Args, testDBCluster.ID, testDBPool.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunDatabasePoolDelete(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("DeletePool", testDBCluster.ID, testDBPool.Name).Return(errTest)

		config.Args = append(config.Args, testDBCluster.ID, testDBPool.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunDatabasePoolDelete(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabasesDBGet(t *testing.T) {
	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetDB", testDBCluster.ID, testDB.Name).Return(&testDB, nil)
		config.Args = append(config.Args, testDBCluster.ID, testDB.Name)
		err := RunDatabaseDBGet(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetDB", testDBCluster.ID, testDB.Name).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID, testDB.Name)
		err := RunDatabaseDBGet(config)
		assert.Error(t, err)
	})

	// ID not provided
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDatabaseDBGet(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestDatabasesListDBs(t *testing.T) {
	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("ListDBs", testDBCluster.ID).Return(testDBs, nil)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseDBList(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("ListDBs", testDBCluster.ID).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseDBList(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabaseDBCreate(t *testing.T) {
	r := &godo.DatabaseCreateDBRequest{
		Name: testDB.Name,
	}

	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("CreateDB", testDBCluster.ID, r).Return(&testDB, nil)

		config.Args = append(config.Args, testDBCluster.ID, testDB.Name)

		err := RunDatabaseDBCreate(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On(
			"CreateDB",
			testDBCluster.ID,
			mock.AnythingOfType("*godo.DatabaseCreateDBRequest"),
		).Return(nil, errTest)

		config.Args = append(config.Args, testDBCluster.ID, testDB.Name)
		err := RunDatabaseDBCreate(config)
		assert.EqualError(t, err, "error")
	})
}

func TestDatabasesDBDelete(t *testing.T) {
	// Successful
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("DeleteDB", testDBCluster.ID, testDB.Name).Return(nil)

		config.Args = append(config.Args, testDBCluster.ID, testDB.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunDatabaseDBDelete(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("DeleteDB", testDBCluster.ID, testDB.Name).Return(errTest)

		config.Args = append(config.Args, testDBCluster.ID, testDB.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunDatabaseDBDelete(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabasesReplicaGet(t *testing.T) {
	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetReplica", testDBCluster.ID, testDBReplica.Name).Return(&testDBReplica, nil)
		config.Args = append(config.Args, testDBCluster.ID, testDBReplica.Name)
		err := RunDatabaseReplicaGet(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("GetReplica", testDBCluster.ID, testDBReplica.Name).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID, testDBReplica.Name)
		err := RunDatabaseReplicaGet(config)
		assert.Error(t, err)
	})

	// ID not provided
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDatabaseReplicaGet(config)
		assert.EqualError(t, doctl.NewMissingArgsErr(config.NS), err.Error())
	})
}

func TestDatabasesListReplicas(t *testing.T) {
	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("ListReplicas", testDBCluster.ID).Return(testDBReplicas, nil)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseReplicaList(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("ListReplicas", testDBCluster.ID).Return(nil, errTest)
		config.Args = append(config.Args, testDBCluster.ID)

		err := RunDatabaseReplicaList(config)
		assert.EqualError(t, err, errTest.Error())
	})
}

func TestDatabaseReplicaCreate(t *testing.T) {
	r := &godo.DatabaseCreateReplicaRequest{
		Name:   testDBReplica.Name,
		Region: testDBReplica.Region,
		Size:   testDBCluster.SizeSlug,
	}

	// Successful call
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("CreateReplica", testDBCluster.ID, r).Return(&testDBReplica, nil)

		config.Args = append(config.Args, testDBCluster.ID, testDBReplica.Name)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testDBReplica.Region)
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, testDBCluster.SizeSlug)

		err := RunDatabaseReplicaCreate(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On(
			"CreateReplica",
			testDBCluster.ID,
			mock.AnythingOfType("*godo.DatabaseCreateReplicaRequest"),
		).Return(nil, errTest)

		config.Args = append(config.Args, testDBCluster.ID, testDBReplica.Name)
		err := RunDatabaseReplicaCreate(config)
		assert.EqualError(t, err, "error")
	})
}

func TestDatabasesReplicaDelete(t *testing.T) {
	// Successful
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("DeleteReplica", testDBCluster.ID, testDBReplica.Name).Return(nil)

		config.Args = append(config.Args, testDBCluster.ID, testDBReplica.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunDatabaseReplicaDelete(config)
		assert.NoError(t, err)
	})

	// Error
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.databases.On("DeleteReplica", testDBCluster.ID, testDBReplica.Name).Return(errTest)

		config.Args = append(config.Args, testDBCluster.ID, testDBReplica.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunDatabaseReplicaDelete(config)
		assert.EqualError(t, err, errTest.Error())
	})
}
