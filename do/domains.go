
/*
Copyright 2016 The Doctl Authors All rights reserved.
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

import "github.com/digitalocean/godo"

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

// DomainRecords is a slice of DomainRecord.
type DomainRecords []DomainRecord

// DomainsService is the godo DOmainsService interface.
type DomainsService interface {
	List() (Domains, error)
	Get(string) (*Domain, error)
	Create(*godo.DomainCreateRequest) (*Domain, error)
	Delete(string) error

	Records(string) (DomainRecords, error)
	Record(string, int) (*DomainRecord, error)
	DeleteRecord(string, int) error
	EditRecord(string, int, *godo.DomainRecordEditRequest) (*DomainRecord, error)
	CreateRecord(string, *godo.DomainRecordEditRequest) (*DomainRecord, error)
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
		list, resp, err := ds.client.Domains.List(opt)
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
	d, _, err := ds.client.Domains.Get(name)
	if err != nil {
		return nil, err
	}

	return &Domain{Domain: d}, nil
}

func (ds *domainsService) Create(dcr *godo.DomainCreateRequest) (*Domain, error) {
	d, _, err := ds.client.Domains.Create(dcr)
	if err != nil {
		return nil, err
	}

	return &Domain{Domain: d}, nil
}

func (ds *domainsService) Delete(name string) error {
	_, err := ds.client.Domains.Delete(name)
	return err
}

func (ds *domainsService) Records(name string) (DomainRecords, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := ds.client.Domains.Records(name, opt)
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
	dr, _, err := ds.client.Domains.Record(domain, id)
	if err != nil {
		return nil, err
	}

	return &DomainRecord{DomainRecord: dr}, nil
}

func (ds *domainsService) DeleteRecord(domain string, id int) error {
	_, err := ds.client.Domains.DeleteRecord(domain, id)
	return err
}

func (ds *domainsService) EditRecord(domain string, id int, drer *godo.DomainRecordEditRequest) (*DomainRecord, error) {
	dr, _, err := ds.client.Domains.EditRecord(domain, id, drer)
	if err != nil {
		return nil, err
	}

	return &DomainRecord{DomainRecord: dr}, nil
}

func (ds *domainsService) CreateRecord(domain string, drer *godo.DomainRecordEditRequest) (*DomainRecord, error) {
	dr, _, err := ds.client.Domains.CreateRecord(domain, drer)
	if err != nil {
		return nil, err
	}

	return &DomainRecord{DomainRecord: dr}, nil
}
