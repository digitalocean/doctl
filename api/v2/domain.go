package apiv2

import (
	"fmt"
)

const DefaultTTL = 900

// name			string	The name of the domain itself. This should follow the standard domain format of domain.TLD. For instance, example.com is a valid domain name.
// ttl			number	This value is the time to live for the records on this domain, in seconds. This defines the time frame that clients can cache queried information before a refresh should be requested.
// zone_file	string	This attribute contains the complete contents of the zone file for the selected domain. Individual domain record resources should be used to get more granular control over records. However, this attribute can also be used to get information about the SOA record, which is created automatically and is not accessible as an individual record resource.
type Domain struct {
	Name     string `json:"name"`
	TTL      int    `json:"ttl"`
	ZoneFile string `json:"zone_file"`
	client   *Client
}

type DomainListResponse struct {
	Domains []*Domain `json:"domains"`
	Meta    struct {
		Total int `json:"total"`
	} `json:"meta"`
}

type DomainRequest struct {
	Name      string `json:"name"`
	IPAddress string `json:"ip_address"`
}

type DomainResponse struct {
	Domain *Domain `json:"domain"`
}

func (c *Client) NewDomain(name string) *Domain {
	return &Domain{
		Name: name,
		TTL:  DefaultTTL,
	}
}

func (c *Client) NewDomainRequest(name string) *DomainRequest {
	return &DomainRequest{
		Name: name,
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

func (c *Client) CreateDomain(domain *DomainRequest) (*Domain, error) {
	var domainResponse DomainResponse

	apiErr := c.Post("domains", domain, &domainResponse, nil)
	if apiErr != nil {
		return nil, fmt.Errorf("API Error: %s", apiErr.Message)
	}

	return domainResponse.Domain, nil
}

func (c *Client) DestroyDomain(name string) error {
	apiErr := c.Delete(fmt.Sprintf("domains/%s", name), nil, nil)
	if apiErr != nil {
		return fmt.Errorf("API Error: %s", apiErr.Message)
	}

	return nil
}
