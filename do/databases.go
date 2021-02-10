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

package do

import (
	"context"
	"strings"

	"github.com/digitalocean/godo"
)

// Database is a wrapper for godo.Database
type Database struct {
	*godo.Database
}

// Databases is a slice of Database
type Databases []Database

// DatabaseBackup is a wrapper for godo.DatabaseBackup
type DatabaseBackup struct {
	*godo.DatabaseBackup
}

// DatabaseBackups is a slice of DatabaseBackup
type DatabaseBackups []DatabaseBackup

// DatabaseUser is a wrapper for godo.DatabaseUser
type DatabaseUser struct {
	*godo.DatabaseUser
}

// DatabaseUsers is a slice of DatabaseUser
type DatabaseUsers []DatabaseUser

// DatabaseDB is a wrapper for godo.DatabaseDB
type DatabaseDB struct {
	*godo.DatabaseDB
}

// DatabaseDBs is a slice of DatabaseDB
type DatabaseDBs []DatabaseDB

// DatabasePool is a wrapper for godo.DatabasePool
type DatabasePool struct {
	*godo.DatabasePool
}

// DatabasePools is a slice of DatabasePool
type DatabasePools []DatabasePool

// DatabaseReplica is a wrapper for godo.DatabaseReplica
type DatabaseReplica struct {
	*godo.DatabaseReplica
}

// DatabaseReplicas is a slice of DatabaseReplica
type DatabaseReplicas []DatabaseReplica

// DatabaseConnection is a wrapper for godo.DatabaseConnection
type DatabaseConnection struct {
	*godo.DatabaseConnection
}

// DatabaseMaintenanceWindow is a wrapper for godo.DatabaseMaintenanceWindow
type DatabaseMaintenanceWindow struct {
	*godo.DatabaseMaintenanceWindow
}

// DatabaseFirewallRule is a wrapper for godo.DatabaseFirewallRule
type DatabaseFirewallRule struct {
	*godo.DatabaseFirewallRule
}

// DatabaseFirewallRules is a slice of DatabaseFirewallRule
type DatabaseFirewallRules []DatabaseFirewallRule

// DatabasesService is an interface for interacting with DigitalOcean's Database API
type DatabasesService interface {
	List() (Databases, error)
	Get(string) (*Database, error)
	Create(*godo.DatabaseCreateRequest) (*Database, error)
	Delete(string) error
	GetConnection(string) (*DatabaseConnection, error)
	ListBackups(string) (DatabaseBackups, error)
	Resize(string, *godo.DatabaseResizeRequest) error
	Migrate(string, *godo.DatabaseMigrateRequest) error

	GetMaintenance(string) (*DatabaseMaintenanceWindow, error)
	UpdateMaintenance(string, *godo.DatabaseUpdateMaintenanceRequest) error

	GetUser(string, string) (*DatabaseUser, error)
	ListUsers(string) (DatabaseUsers, error)
	CreateUser(string, *godo.DatabaseCreateUserRequest) (*DatabaseUser, error)
	DeleteUser(string, string) error
	ResetUserAuth(string, string, *godo.DatabaseResetUserAuthRequest) (*DatabaseUser, error)

	ListDBs(string) (DatabaseDBs, error)
	CreateDB(string, *godo.DatabaseCreateDBRequest) (*DatabaseDB, error)
	GetDB(string, string) (*DatabaseDB, error)
	DeleteDB(string, string) error

	ListPools(string) (DatabasePools, error)
	CreatePool(string, *godo.DatabaseCreatePoolRequest) (*DatabasePool, error)
	GetPool(string, string) (*DatabasePool, error)
	DeletePool(string, string) error

	GetReplica(string, string) (*DatabaseReplica, error)
	ListReplicas(string) (DatabaseReplicas, error)
	CreateReplica(string, *godo.DatabaseCreateReplicaRequest) (*DatabaseReplica, error)
	DeleteReplica(string, string) error
	GetReplicaConnection(string, string) (*DatabaseConnection, error)

	GetSQLMode(string) ([]string, error)
	SetSQLMode(string, ...string) error

	GetFirewallRules(string) (DatabaseFirewallRules, error)
	UpdateFirewallRules(databaseID string, req *godo.DatabaseUpdateFirewallRulesRequest) error
}

type databasesService struct {
	client *godo.Client
}

var _ DatabasesService = &databasesService{}

// NewDatabasesService builds a DatabasesService instance.
func NewDatabasesService(client *godo.Client) DatabasesService {
	return &databasesService{
		client: client,
	}
}

func (ds *databasesService) List() (Databases, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Databases.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(Databases, len(si))
	for i := range si {
		d := si[i].(godo.Database)
		list[i] = Database{Database: &d}
	}
	return list, nil
}

func (ds *databasesService) Get(databaseID string) (*Database, error) {
	db, _, err := ds.client.Databases.Get(context.TODO(), databaseID)
	if err != nil {
		return nil, err
	}

	return &Database{Database: db}, nil
}

func (ds *databasesService) Create(req *godo.DatabaseCreateRequest) (*Database, error) {
	db, _, err := ds.client.Databases.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}

	return &Database{Database: db}, nil
}

func (ds *databasesService) Delete(databaseID string) error {
	_, err := ds.client.Databases.Delete(context.TODO(), databaseID)

	return err
}

func (ds *databasesService) GetConnection(databaseID string) (*DatabaseConnection, error) {
	db, err := ds.Get(databaseID)
	if err != nil {
		return nil, err
	}

	return &DatabaseConnection{
		DatabaseConnection: db.Connection,
	}, nil
}

func (ds *databasesService) Resize(databaseID string, req *godo.DatabaseResizeRequest) error {
	_, err := ds.client.Databases.Resize(context.TODO(), databaseID, req)

	return err
}

func (ds *databasesService) Migrate(databaseID string, req *godo.DatabaseMigrateRequest) error {
	_, err := ds.client.Databases.Migrate(context.TODO(), databaseID, req)

	return err
}

func (ds *databasesService) GetMaintenance(databaseID string) (*DatabaseMaintenanceWindow, error) {
	db, err := ds.Get(databaseID)
	if err != nil {
		return nil, err
	}

	return &DatabaseMaintenanceWindow{
		DatabaseMaintenanceWindow: db.MaintenanceWindow,
	}, nil
}

func (ds *databasesService) UpdateMaintenance(databaseID string, req *godo.DatabaseUpdateMaintenanceRequest) error {
	_, err := ds.client.Databases.UpdateMaintenance(context.TODO(), databaseID, req)

	return err
}

func (ds *databasesService) ListBackups(databaseID string) (DatabaseBackups, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Databases.ListBackups(context.TODO(), databaseID, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(DatabaseBackups, len(si))
	for i := range si {
		b := si[i].(godo.DatabaseBackup)
		list[i] = DatabaseBackup{DatabaseBackup: &b}
	}
	return list, nil
}

func (ds *databasesService) GetUser(databaseID, userName string) (*DatabaseUser, error) {
	u, _, err := ds.client.Databases.GetUser(context.TODO(), databaseID, userName)
	if err != nil {
		return nil, err
	}

	return &DatabaseUser{DatabaseUser: u}, nil
}

func (ds *databasesService) ListUsers(databaseID string) (DatabaseUsers, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Databases.ListUsers(context.TODO(), databaseID, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(DatabaseUsers, len(si))
	for i := range si {
		u := si[i].(godo.DatabaseUser)
		list[i] = DatabaseUser{DatabaseUser: &u}
	}
	return list, nil
}

func (ds *databasesService) CreateUser(databaseID string, req *godo.DatabaseCreateUserRequest) (*DatabaseUser, error) {
	u, _, err := ds.client.Databases.CreateUser(context.TODO(), databaseID, req)
	if err != nil {
		return nil, err
	}

	return &DatabaseUser{DatabaseUser: u}, nil
}

func (ds *databasesService) DeleteUser(databaseID, userName string) error {
	_, err := ds.client.Databases.DeleteUser(context.TODO(), databaseID, userName)

	return err
}

func (ds *databasesService) ResetUserAuth(databaseID, userID string, req *godo.DatabaseResetUserAuthRequest) (*DatabaseUser, error) {
	u, _, err := ds.client.Databases.ResetUserAuth(context.TODO(), databaseID, userID, req)
	if err != nil {
		return nil, err
	}
	return &DatabaseUser{DatabaseUser: u}, nil
}

func (ds *databasesService) ListDBs(databaseID string) (DatabaseDBs, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Databases.ListDBs(context.TODO(), databaseID, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(DatabaseDBs, len(si))
	for i := range si {
		db := si[i].(godo.DatabaseDB)
		list[i] = DatabaseDB{DatabaseDB: &db}
	}
	return list, nil
}

func (ds *databasesService) CreateDB(databaseID string, req *godo.DatabaseCreateDBRequest) (*DatabaseDB, error) {
	db, _, err := ds.client.Databases.CreateDB(context.TODO(), databaseID, req)
	if err != nil {
		return nil, err
	}

	return &DatabaseDB{DatabaseDB: db}, nil
}

func (ds *databasesService) GetDB(databaseID, dbID string) (*DatabaseDB, error) {
	db, _, err := ds.client.Databases.GetDB(context.TODO(), databaseID, dbID)
	if err != nil {
		return nil, err
	}

	return &DatabaseDB{DatabaseDB: db}, nil
}

func (ds *databasesService) DeleteDB(databaseID, dbID string) error {
	_, err := ds.client.Databases.DeleteDB(context.TODO(), databaseID, dbID)

	return err
}

func (ds *databasesService) ListPools(databaseID string) (DatabasePools, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Databases.ListPools(context.TODO(), databaseID, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(DatabasePools, len(si))
	for i := range si {
		p := si[i].(godo.DatabasePool)
		list[i] = DatabasePool{DatabasePool: &p}
	}
	return list, nil
}

func (ds *databasesService) CreatePool(databaseID string, req *godo.DatabaseCreatePoolRequest) (*DatabasePool, error) {
	p, _, err := ds.client.Databases.CreatePool(context.TODO(), databaseID, req)
	if err != nil {
		return nil, err
	}

	return &DatabasePool{DatabasePool: p}, nil
}

func (ds *databasesService) GetPool(databaseID, poolName string) (*DatabasePool, error) {
	p, _, err := ds.client.Databases.GetPool(context.TODO(), databaseID, poolName)
	if err != nil {
		return nil, err
	}

	return &DatabasePool{DatabasePool: p}, nil
}

func (ds *databasesService) DeletePool(databaseID, poolName string) error {
	_, err := ds.client.Databases.DeletePool(context.TODO(), databaseID, poolName)

	return err
}

func (ds *databasesService) GetReplica(databaseID, replicaID string) (*DatabaseReplica, error) {
	r, _, err := ds.client.Databases.GetReplica(context.TODO(), databaseID, replicaID)
	if err != nil {
		return nil, err
	}

	return &DatabaseReplica{DatabaseReplica: r}, nil
}

func (ds *databasesService) ListReplicas(databaseID string) (DatabaseReplicas, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Databases.ListReplicas(context.TODO(), databaseID, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(DatabaseReplicas, len(si))
	for i := range si {
		r := si[i].(godo.DatabaseReplica)
		list[i] = DatabaseReplica{DatabaseReplica: &r}
	}
	return list, nil
}

func (ds *databasesService) CreateReplica(databaseID string, req *godo.DatabaseCreateReplicaRequest) (*DatabaseReplica, error) {
	r, _, err := ds.client.Databases.CreateReplica(context.TODO(), databaseID, req)
	if err != nil {
		return nil, err
	}

	return &DatabaseReplica{DatabaseReplica: r}, nil
}

func (ds *databasesService) DeleteReplica(databaseID string, replicaID string) error {
	_, err := ds.client.Databases.DeleteReplica(context.TODO(), databaseID, replicaID)

	return err
}

func (ds *databasesService) GetReplicaConnection(databaseID, replicaID string) (*DatabaseConnection, error) {
	rep, err := ds.GetReplica(databaseID, replicaID)
	if err != nil {
		return nil, err
	}

	return &DatabaseConnection{
		DatabaseConnection: rep.Connection,
	}, nil
}

func (ds *databasesService) GetSQLMode(databaseID string) ([]string, error) {
	sqlModes, _, err := ds.client.Databases.GetSQLMode(context.TODO(), databaseID)
	return strings.Split(sqlModes, ","), err
}

func (ds *databasesService) SetSQLMode(databaseID string, sqlModes ...string) error {
	_, err := ds.client.Databases.SetSQLMode(context.TODO(), databaseID, sqlModes...)
	return err
}

func (ds *databasesService) GetFirewallRules(databaseID string) (DatabaseFirewallRules, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Databases.GetFirewallRules(context.TODO(), databaseID)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(DatabaseFirewallRules, len(si))
	for i := range si {
		r := si[i].(godo.DatabaseFirewallRule)
		list[i] = DatabaseFirewallRule{DatabaseFirewallRule: &r}
	}
	return list, nil
}

func (ds *databasesService) UpdateFirewallRules(databaseID string, req *godo.DatabaseUpdateFirewallRulesRequest) error {
	_, err := ds.client.Databases.UpdateFirewallRules(context.TODO(), databaseID, req)

	return err
}
