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

	"github.com/digitalocean/godo"
)

// VectorDB is a wrapper for godo.VectorDB
type VectorDB struct {
	*godo.VectorDB
}

// VectorDBs is a slice of VectorDB
type VectorDBs []VectorDB

// VectorDBBackup is a wrapper for godo.VectorDBBackup
type VectorDBBackup struct {
	*godo.VectorDBBackup
}

// VectorDBBackups is a slice of VectorDBBackup
type VectorDBBackups []VectorDBBackup

// VectorDBAdminCredentials is a wrapper for godo.VectorDBAdminCredentials
type VectorDBAdminCredentials struct {
	*godo.VectorDBAdminCredentials
}

// VectorDBRestoreBackupResponse is a wrapper for godo.VectorDBRestoreBackupResponse
type VectorDBRestoreBackupResponse struct {
	*godo.VectorDBRestoreBackupResponse
}

// VectorDBRestoreStatus is a wrapper for godo.VectorDBRestoreStatus
type VectorDBRestoreStatus struct {
	*godo.VectorDBRestoreStatus
}

// VectorDBsService is an interface for interacting with DigitalOcean's Vector DB API
type VectorDBsService interface {
	List() (VectorDBs, error)
	Get(string) (*VectorDB, error)
	Create(*godo.VectorDBCreateRequest) (*VectorDB, error)
	Update(string, *godo.VectorDBUpdateRequest) (*VectorDB, error)
	Delete(string) error
	Resize(string, *godo.VectorDBResizeRequest) (*VectorDB, error)
	UpdateTags(string, *godo.VectorDBUpdateTagsRequest) (*VectorDB, error)
	GetCredentials(string) (*VectorDBAdminCredentials, error)
	ListBackups(string) (VectorDBBackups, error)
	RestoreBackup(string, string, *godo.VectorDBRestoreBackupRequest) (*VectorDBRestoreBackupResponse, error)
	GetRestoreStatus(string, string) (*VectorDBRestoreStatus, error)
}

type vectorDBsService struct {
	client *godo.Client
}

var _ VectorDBsService = &vectorDBsService{}

// NewVectorDBsService builds a VectorDBsService instance.
func NewVectorDBsService(client *godo.Client) VectorDBsService {
	return &vectorDBsService{
		client: client,
	}
}

func (v *vectorDBsService) List() (VectorDBs, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := v.client.VectorDBs.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(VectorDBs, len(si))
	for i := range si {
		db := si[i].(godo.VectorDB)
		list[i] = VectorDB{VectorDB: &db}
	}
	return list, nil
}

func (v *vectorDBsService) Get(id string) (*VectorDB, error) {
	db, _, err := v.client.VectorDBs.Get(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	return &VectorDB{VectorDB: db}, nil
}

func (v *vectorDBsService) Create(req *godo.VectorDBCreateRequest) (*VectorDB, error) {
	db, _, err := v.client.VectorDBs.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}

	return &VectorDB{VectorDB: db}, nil
}

func (v *vectorDBsService) Update(id string, req *godo.VectorDBUpdateRequest) (*VectorDB, error) {
	db, _, err := v.client.VectorDBs.Update(context.TODO(), id, req)
	if err != nil {
		return nil, err
	}

	return &VectorDB{VectorDB: db}, nil
}

func (v *vectorDBsService) Delete(id string) error {
	_, err := v.client.VectorDBs.Delete(context.TODO(), id)

	return err
}

func (v *vectorDBsService) Resize(id string, req *godo.VectorDBResizeRequest) (*VectorDB, error) {
	db, _, err := v.client.VectorDBs.Resize(context.TODO(), id, req)
	if err != nil {
		return nil, err
	}

	return &VectorDB{VectorDB: db}, nil
}

func (v *vectorDBsService) UpdateTags(id string, req *godo.VectorDBUpdateTagsRequest) (*VectorDB, error) {
	db, _, err := v.client.VectorDBs.UpdateTags(context.TODO(), id, req)
	if err != nil {
		return nil, err
	}

	return &VectorDB{VectorDB: db}, nil
}

func (v *vectorDBsService) GetCredentials(id string) (*VectorDBAdminCredentials, error) {
	creds, _, err := v.client.VectorDBs.GetCredentials(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	return &VectorDBAdminCredentials{VectorDBAdminCredentials: creds}, nil
}

func (v *vectorDBsService) ListBackups(id string) (VectorDBBackups, error) {
	list, _, err := v.client.VectorDBs.ListBackups(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	backups := make(VectorDBBackups, len(list))
	for i := range list {
		b := list[i]
		backups[i] = VectorDBBackup{VectorDBBackup: &b}
	}
	return backups, nil
}

func (v *vectorDBsService) RestoreBackup(id, backupID string, req *godo.VectorDBRestoreBackupRequest) (*VectorDBRestoreBackupResponse, error) {
	resp, _, err := v.client.VectorDBs.RestoreBackup(context.TODO(), id, backupID, req)
	if err != nil {
		return nil, err
	}

	return &VectorDBRestoreBackupResponse{VectorDBRestoreBackupResponse: resp}, nil
}

func (v *vectorDBsService) GetRestoreStatus(id, backupID string) (*VectorDBRestoreStatus, error) {
	status, _, err := v.client.VectorDBs.GetRestoreStatus(context.TODO(), id, backupID)
	if err != nil {
		return nil, err
	}

	return &VectorDBRestoreStatus{VectorDBRestoreStatus: status}, nil
}
