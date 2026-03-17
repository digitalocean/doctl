/*
Copyright 2026 The Doctl Authors All rights reserved.
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
	"errors"

	"github.com/digitalocean/godo"
)

// Scan wraps a godo.Scan.
type Scan struct {
	*godo.Scan
}

// Scans is a slice of Scan.
type Scans []Scan

// AffectedResource wraps a godo.AffectedResource.
type AffectedResource struct {
	*godo.AffectedResource
}

// AffectedResources is a slice of AffectedResource.
type AffectedResources []AffectedResource

// SecurityService is an interface for interacting with DigitalOcean's CSPM API.
type SecurityService interface {
	CreateScan(*godo.CreateScanRequest) (*Scan, error)
	ListScans() (Scans, error)
	GetScan(string, *godo.ScanFindingsOptions) (*Scan, error)
	GetLatestScan(*godo.ScanFindingsOptions) (*Scan, error)
	ListFindingAffectedResources(string, string) (AffectedResources, error)
}

type securityService struct {
	client *godo.Client
}

var _ SecurityService = (*securityService)(nil)

// NewSecurityService builds a SecurityService instance.
func NewSecurityService(godoClient *godo.Client) SecurityService {
	return &securityService{
		client: godoClient,
	}
}

func (ss *securityService) CreateScan(request *godo.CreateScanRequest) (*Scan, error) {
	scan, _, err := ss.client.Security.CreateScan(context.TODO(), request)
	if err != nil {
		return nil, err
	}

	return &Scan{Scan: scan}, nil
}

func (ss *securityService) ListScans() (Scans, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := ss.client.Security.ListScans(context.TODO(), opt)
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

	list := make(Scans, len(si))
	for i := range si {
		scan, ok := si[i].(*godo.Scan)
		if !ok {
			return nil, errors.New("unexpected value in response")
		}
		list[i] = Scan{Scan: scan}
	}

	return list, nil
}

func (ss *securityService) GetScan(scanUUID string, opts *godo.ScanFindingsOptions) (*Scan, error) {
	scan, _, err := ss.client.Security.GetScan(context.TODO(), scanUUID, opts)
	if err != nil {
		return nil, err
	}

	return &Scan{Scan: scan}, nil
}

func (ss *securityService) GetLatestScan(opts *godo.ScanFindingsOptions) (*Scan, error) {
	scan, _, err := ss.client.Security.GetLatestScan(context.TODO(), opts)
	if err != nil {
		return nil, err
	}

	return &Scan{Scan: scan}, nil
}

func (ss *securityService) ListFindingAffectedResources(scanUUID string, findingUUID string) (AffectedResources, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := ss.client.Security.ListFindingAffectedResources(
			context.TODO(),
			&godo.ListFindingAffectedResourcesRequest{
				ScanUUID:    scanUUID,
				FindingUUID: findingUUID,
			},
			opt,
		)
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

	list := make(AffectedResources, len(si))
	for i := range si {
		resource, ok := si[i].(*godo.AffectedResource)
		if !ok {
			return nil, errors.New("unexpected value in response")
		}
		list[i] = AffectedResource{AffectedResource: resource}
	}

	return list, nil
}
