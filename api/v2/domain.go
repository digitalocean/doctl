package apiv2

import (
	"fmt"
)

// name			string	The name of the domain itself. This should follow the standard domain format of domain.TLD. For instance, example.com is a valid domain name.
// ttl			number	This value is the time to live for the records on this domain, in seconds. This defines the time frame that clients can cache queried information before a refresh should be requested.
// zone_file	string	This attribute contains the complete contents of the zone file for the selected domain. Individual domain record resources should be used to get more granular control over records. However, this attribute can also be used to get information about the SOA record, which is created automatically and is not accessible as an individual record resource.
type Domain struct {
	Name     string `json:"name"`
	TTL      int    `json:"ttl"`
	ZoneFile string `json:"zone_file"`
}

// id			number	The unique id for the individual record.
// type			string	The DNS record type (A, MX, CNAME, etc).
// name			string	The host name, alias, or service being defined by the record. See the [domain record] object to find out more.
// data			string	Variable data depending on record type. See the [domain record] object for more detail on each record type.
// priority		nullable number	The priority of the host (for SRV and MX records. null otherwise).
// port			nullable number	The port that the service is accessible on (for SRV records only. null otherwise).
// weight		nullable number	The weight of records with the same priority (for SRV records only. null otherwise).
type DomainRecord struct {
	ID       int    `json:"id,omitempty"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority int    `json:"priority"`
	Port     int    `json:"port"`
	Weight   int    `json:"weight"`
}

type DomainListResponse struct {
	Domains []*Domain `json:"domains"`
	Meta    struct {
		Total int `json:"total"`
	} `json:"meta"`
}

type DomainResponse struct {
	Domain *Domain `json:"domain"`
}

func NewDomain(name string) *Domain {
	return &Domain{
		Name: name,
		TTL:  900,
	}
}

func (c *Client) LoadDomain(name string) (*Domain, error) {
	var domain DomainResponse

	err := c.Get(fmt.Sprintf("domains/%s", name), nil, &domain, nil)
	if err != nil {
		return nil, fmt.Errorf("API Error: %s", err.Message)
	}

	return domain.Domain, nil
}

func (c *Client) ListAllDomains() (*DomainListResponse, error) {
	var domainList *DomainListResponse

	err := c.Get("domains", nil, &domainList, nil)
	if err != nil {
		return nil, fmt.Errorf("API Error: %s", err.Message)
	}

	return domainList, nil
}
