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

package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// Nfs wraps a godo.Nfs.
type Nfs struct {
	*godo.Nfs
}

// NfsSnapshot wraps a godo.NfsSnapshot.
type NfsSnapshot struct {
	*godo.NfsSnapshot
}

// NfsService is an interface for interacting with DigitalOcean's NFS API.
type NfsService interface {
	List(region string) ([]Nfs, error)
	Create(*godo.NfsCreateRequest) (*Nfs, error)
	Delete(id, region string) error
	Get(id, region string) (*Nfs, error)
	ListSnapshots(shareID, region string) ([]NfsSnapshot, error)
	GetSnapshot(snapshotID, region string) (*NfsSnapshot, error)
	DeleteSnapshot(snapshotID, region string) error
}

type nfsService struct {
	client *godo.Client
}

var _ NfsService = &nfsService{}

// NewNfsService builds a NewNfsService instance.
func NewNfsService(godoClient *godo.Client) NfsService {
	return &nfsService{
		client: godoClient,
	}
}

func (s *nfsService) List(region string) ([]Nfs, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.Nfs.List(context.TODO(), opt, region)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = *list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]Nfs, len(si))
	for i := range si {
		nfs := si[i].(godo.Nfs)
		list[i] = Nfs{Nfs: &nfs}
	}
	return list, nil
}

func (s *nfsService) Create(r *godo.NfsCreateRequest) (*Nfs, error) {
	nfs, _, err := s.client.Nfs.Create(context.TODO(), r)
	if err != nil {
		return nil, err
	}
	return &Nfs{Nfs: nfs}, nil
}

func (s *nfsService) Delete(id, region string) error {
	_, err := s.client.Nfs.Delete(context.TODO(), id, region)
	return err
}

func (s *nfsService) Get(id, region string) (*Nfs, error) {
	nfs, _, err := s.client.Nfs.Get(context.TODO(), id, region)
	if err != nil {
		return nil, err
	}

	return &Nfs{Nfs: nfs}, nil
}

func (s *nfsService) ListSnapshots(shareID, region string) ([]NfsSnapshot, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.Nfs.ListSnapshots(context.TODO(), opt, shareID, region)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = *list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]NfsSnapshot, len(si))
	for i := range si {
		snapshot := si[i].(godo.NfsSnapshot)
		list[i] = NfsSnapshot{NfsSnapshot: &snapshot}
	}
	return list, nil
}

func (s *nfsService) GetSnapshot(snapshotID, region string) (*NfsSnapshot, error) {
	snapshot, _, err := s.client.Nfs.GetSnapshot(context.TODO(), snapshotID, region)
	if err != nil {
		return nil, err
	}

	return &NfsSnapshot{NfsSnapshot: snapshot}, nil
}

func (s *nfsService) DeleteSnapshot(snapshotID, region string) error {
	_, err := s.client.Nfs.DeleteSnapshot(context.TODO(), snapshotID, region)
	return err
}
