package domainrecs

import "github.com/digitalocean/godo"

type CreateRequest struct {
	Type     string
	Name     string
	Data     string
	Priority int
	Port     int
	Weight   int
}

func (cr *CreateRequest) IsValid() bool {
	return true
}

func Create(client *godo.Client, domain string, cr *CreateRequest) (*godo.DomainRecord, error) {
	drcr := &godo.DomainRecordEditRequest{
		Type:     cr.Type,
		Name:     cr.Name,
		Data:     cr.Data,
		Priority: cr.Priority,
		Port:     cr.Port,
		Weight:   cr.Weight,
	}

	r, _, err := client.Domains.CreateRecord(domain, drcr)

	return r, err
}
