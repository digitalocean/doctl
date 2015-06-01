package domains

import "github.com/digitalocean/godo"

// Retrieve retrieves a domain.
func Retrieve(client *godo.Client, name string) (*godo.Domain, error) {
	d, _, err := client.Domains.Get(name)
	if err != nil {
		return nil, err
	}

	return d, nil
}
