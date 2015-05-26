package domainrecs

import "github.com/digitalocean/godo"

// Delete removes a domain record from a domain.
func Delete(client *godo.Client, domain string, id int) error {
	_, err := client.Domains.DeleteRecord(domain, id)
	return err
}
