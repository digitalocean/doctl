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
	"fmt"
	"net/http"

	"github.com/digitalocean/godo"
)

const (
	domainRecordsPath = "v2/domains/%s/records"
	domainRecordPath  = "v2/domains/%s/records/%d"
)

// Domain wraps a godo Domain.
type Domain struct {
	*godo.Domain
}

// Domains is a slice of Domain.
type Domains []Domain

// DomainRecord wraps a godo DomainRecord.
type DomainRecord struct {
	*godo.DomainRecord
}

// A DomainRecordEditRequest is used in place of godo's DomainRecordEditRequest
// in order to work around the fact that we cannot send a port value of 0 via
// godo due to Go's JSON encoding logic.
type DomainRecordEditRequest struct {
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	Data     string `json:"data,omitempty"`
	Priority int    `json:"priority"`
	Port     *int   `json:"port,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Weight   int    `json:"weight"`
	Flags    int    `json:"flags"`
	Tag      string `json:"tag,omitempty"`
}

// DomainRecords is a slice of DomainRecord.
type DomainRecords []DomainRecord

// DomainsService is the godo DomainsService interface.
type DomainsService interface {
	List() (Domains, error)
	Get(string) (*Domain, error)
	Create(*godo.DomainCreateRequest) (*Domain, error)
	Delete(string) error

	Records(string) (DomainRecords, error)
	Record(string, int) (*DomainRecord, error)
	DeleteRecord(string, int) error
	EditRecord(string, int, *DomainRecordEditRequest) (*DomainRecord, error)
	CreateRecord(string, *DomainRecordEditRequest) (*DomainRecord, error)
}

type domainsService struct {
	client *godo.Client
}

var _ DomainsService = &domainsService{}

// NewDomainsService builds an instance of DomainsService.
func NewDomainsService(client *godo.Client) DomainsService {
	return &domainsService{
		client: client,
	}
}

func (ds *domainsService) List() (Domains, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Domains.List(context.TODO(), opt)
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

	list := make(Domains, len(si))
	for i := range si {
		d := si[i].(godo.Domain)
		list[i] = Domain{Domain: &d}
	}

	return list, nil
}

func (ds *domainsService) Get(name string) (*Domain, error) {
	d, _, err := ds.client.Domains.Get(context.TODO(), name)
	if err != nil {
		return nil, err
	}

	return &Domain{Domain: d}, nil
}

func (ds *domainsService) Create(dcr *godo.DomainCreateRequest) (*Domain, error) {
	d, _, err := ds.client.Domains.Create(context.TODO(), dcr)
	if err != nil {
		return nil, err
	}

	return &Domain{Domain: d}, nil
}

func (ds *domainsService) Delete(name string) error {
	_, err := ds.client.Domains.Delete(context.TODO(), name)
	return err
}

func (ds *domainsService) Records(name string) (DomainRecords, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Domains.Records(context.TODO(), name, opt)
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

	list := make(DomainRecords, len(si))
	for i := range si {
		dr := si[i].(godo.DomainRecord)
		list[i] = DomainRecord{DomainRecord: &dr}
	}

	return list, nil
}

func (ds *domainsService) Record(domain string, id int) (*DomainRecord, error) {
	dr, _, err := ds.client.Domains.Record(context.TODO(), domain, id)
	if err != nil {
		return nil, err
	}

	return &DomainRecord{DomainRecord: dr}, nil
}

func (ds *domainsService) DeleteRecord(domain string, id int) error {
	_, err := ds.client.Domains.DeleteRecord(context.TODO(), domain, id)
	return err
}

// domainRecordRoot is the root of an individual Domain Record response.
//
// Copied from godo.
type domainRecordRoot struct {
	DomainRecord *DomainRecord `json:"domain_record"`
}

func (ds *domainsService) EditRecord(domain string, id int, drer *DomainRecordEditRequest) (*DomainRecord, error) {
	if len(domain) < 1 {
		return nil, godo.NewArgError("domain", "cannot be an empty string")
	}
	if id < 1 {
		return nil, godo.NewArgError("id", "cannot be less than 1")
	}
	if drer == nil {
		return nil, godo.NewArgError("editRequest", "cannot be nil")
	}

	path := fmt.Sprintf(domainRecordPath, domain, id)
	req, err := ds.client.NewRequest(context.TODO(), http.MethodPatch, path, drer)
	if err != nil {
		return nil, err
	}

	root := new(domainRecordRoot)
	if _, err := ds.client.Do(context.TODO(), req, root); err != nil {
		return nil, err
	}
	return root.DomainRecord, nil
}

func (ds *domainsService) CreateRecord(domain string, drer *DomainRecordEditRequest) (*DomainRecord, error) {
	if len(domain) < 1 {
		return nil, godo.NewArgError("domain", "cannot be empty string")
	}
	if drer == nil {
		return nil, godo.NewArgError("createRequest", "cannot be nil")
	}

	path := fmt.Sprintf(domainRecordsPath, domain)
	req, err := ds.client.NewRequest(context.Background(), http.MethodPost, path, drer)
	if err != nil {
		return nil, err
	}

	root := new(domainRecordRoot)
	if _, err := ds.client.Do(context.Background(), req, root); err != nil {
		return nil, err
	}
	return root.DomainRecord, err
}
