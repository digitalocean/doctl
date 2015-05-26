package domainrecs

import "github.com/digitalocean/godo"

// Create a domain record.
func Create(client *godo.Client, domain string, cr *EditRequest) (*godo.DomainRecord, error) {
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
