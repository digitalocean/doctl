package domains

import "github.com/digitalocean/godo"

// CreateRequest is a struct that contains items describing a domain.
type CreateRequest struct {
	Name      string
	IPAddress string
}

// IsValid tests if the CreateRequest is valid. It is valid if the name
// and ip address contents are not blanks.
func (cr *CreateRequest) IsValid() bool {
	return len(cr.Name) > 0 && len(cr.IPAddress) > 0
}

// Create creates a new domain.
func Create(client *godo.Client, cr *CreateRequest) (*godo.DomainRoot, error) {
	dcr := &godo.DomainCreateRequest{
		Name:      cr.Name,
		IPAddress: cr.IPAddress,
	}

	r, _, err := client.Domains.Create(dcr)

	return r, err
}
