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

// Snapshot is a wrapper for godo.Snapshot
type Snapshot struct {
	*godo.Snapshot
}

// Snapshots is a slice of Snapshot.
type Snapshots []Snapshot

// SnapshotsService is an interface for interacting with DigitalOcean's snapshot api.
type SnapshotsService interface {
	List() (Snapshots, error)
	ListVolume() (Snapshots, error)
	ListDroplet() (Snapshots, error)
	Get(string) (*Snapshot, error)
	Delete(string) error
}

type snapshotsService struct {
	client *godo.Client
}

var _ SnapshotsService = &snapshotsService{}

// NewSnapshotsService builds a SnapshotsService instance.
func NewSnapshotsService(client *godo.Client) SnapshotsService {
	return &snapshotsService{
		client: client,
	}
}

func (ss *snapshotsService) List() (Snapshots, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ss.client.Snapshots.List(context.TODO(), opt)
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

	list := make(Snapshots, len(si))
	for i := range si {
		a := si[i].(godo.Snapshot)
		list[i] = Snapshot{Snapshot: &a}
	}

	return list, nil
}

func (ss *snapshotsService) ListVolume() (Snapshots, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ss.client.Snapshots.ListVolume(context.TODO(), opt)
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

	list := make(Snapshots, len(si))
	for i := range si {
		a := si[i].(godo.Snapshot)
		list[i] = Snapshot{Snapshot: &a}
	}

	return list, nil
}

func (ss *snapshotsService) ListDroplet() (Snapshots, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ss.client.Snapshots.ListDroplet(context.TODO(), opt)
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

	list := make(Snapshots, len(si))
	for i := range si {
		a := si[i].(godo.Snapshot)
		list[i] = Snapshot{Snapshot: &a}
	}

	return list, nil
}

func (ss *snapshotsService) Get(snapshotID string) (*Snapshot, error) {
	s, _, err := ss.client.Snapshots.Get(context.TODO(), snapshotID)
	if err != nil {
		return nil, err
	}

	return &Snapshot{Snapshot: s}, nil
}

func (ss *snapshotsService) Delete(snapshotID string) error {
	_, err := ss.client.Snapshots.Delete(context.TODO(), snapshotID)
	return err
}
