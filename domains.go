package godo

import "fmt"

const domainsBasePath = "v2/domains"

// DomainsService handles communication wit the domain related methods of the
// DigitalOcean API.
type DomainsService struct {
	client *Client
}

type DomainRecordRoot struct {
	DomainRecord *DomainRecord `json:"domain_record"`
}

type DomainRecordsRoot struct {
	DomainRecords []DomainRecord `json:"domain_records"`
}

// DomainRecord represents a DigitalOcean DomainRecord
type DomainRecord struct {
	ID       int    `json:"id,float64,omitempty"`
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	Data     string `json:"data,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Port     int    `json:"port,omitempty"`
	Weight   int    `json:"weight,omitempty"`
}

type DomainRecordsOptions struct {
	ListOptions
}

// Converts a DomainRecord to a string.
func (d DomainRecord) String() string {
	return Stringify(d)
}

// DomainRecordEditRequest represents a request to update a domain record.
type DomainRecordEditRequest struct {
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	Data     string `json:"data,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Port     int    `json:"port,omitempty"`
	Weight   int    `json:"weight,omitempty"`
}

// Converts a DomainRecordEditRequest to a string.
func (d DomainRecordEditRequest) String() string {
	return Stringify(d)
}

// Records returns a slice of DomainRecords for a domain
func (s *DomainsService) Records(domain string, opt *DomainRecordsOptions) ([]DomainRecord, *Response, error) {
	path := fmt.Sprintf("%s/%s/records", domainsBasePath, domain)
	path, err := addOptions(path, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	records := new(DomainRecordsRoot)
	resp, err := s.client.Do(req, records)
	if err != nil {
		return nil, resp, err
	}

	return records.DomainRecords, resp, err
}

// Record returns the record id from a domain
func (s *DomainsService) Record(domain string, id int) (*DomainRecord, *Response, error) {
	path := fmt.Sprintf("%s/%s/records/%d", domainsBasePath, domain, id)

	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	record := new(DomainRecordRoot)
	resp, err := s.client.Do(req, record)
	if err != nil {
		return nil, resp, err
	}

	return record.DomainRecord, resp, err
}

// DeleteRecord deletes a record from a domain identified by id
func (s *DomainsService) DeleteRecord(domain string, id int) (*Response, error) {
	path := fmt.Sprintf("%s/%s/records/%d", domainsBasePath, domain, id)

	req, err := s.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req, nil)

	return resp, err
}

// EditRecord edits a record using a DomainRecordEditRequest
func (s *DomainsService) EditRecord(
	domain string,
	id int,
	editRequest *DomainRecordEditRequest) (*DomainRecord, *Response, error) {
	path := fmt.Sprintf("%s/%s/records/%d", domainsBasePath, domain, id)

	req, err := s.client.NewRequest("PUT", path, editRequest)
	if err != nil {
		return nil, nil, err
	}

	d := new(DomainRecord)
	resp, err := s.client.Do(req, d)
	if err != nil {
		return nil, resp, err
	}

	return d, resp, err
}

// CreateRecord creates a record using a DomainRecordEditRequest
func (s *DomainsService) CreateRecord(
	domain string,
	createRequest *DomainRecordEditRequest) (*DomainRecord, *Response, error) {
	path := fmt.Sprintf("%s/%s/records", domainsBasePath, domain)
	req, err := s.client.NewRequest("POST", path, createRequest)

	if err != nil {
		return nil, nil, err
	}

	d := new(DomainRecordRoot)
	resp, err := s.client.Do(req, d)
	if err != nil {
		return nil, resp, err
	}

	return d.DomainRecord, resp, err
}
