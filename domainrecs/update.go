package domainrecs

import (
	"fmt"

	"github.com/digitalocean/godo"
)

// EditRequest contains domain record items which can be updated.
type EditRequest struct {
	Type     string
	Name     string
	Data     string
	Priority int
	Port     int
	Weight   int
}

// IsValid returns if an edit request is valid or not.
func (ur *EditRequest) IsValid() bool {
	return true
}

// Update updates a domain record.
func Update(client *godo.Client, domain string, id int, ur *EditRequest) (*godo.DomainRecord, error) {
	drur := &godo.DomainRecordEditRequest{
		Type:     ur.Type,
		Name:     ur.Name,
		Data:     ur.Data,
		Priority: ur.Priority,
		Port:     ur.Port,
		Weight:   ur.Weight,
	}

	rec, _, err := client.Domains.EditRecord(domain, id, drur)
	if err != nil {
		return nil, err
	}

	fmt.Printf("rec: %#v\n", rec)

	return rec, err
}
