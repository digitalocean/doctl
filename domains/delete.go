package domains

import "github.com/digitalocean/godo"

// Delete deletes a domain name.
func Delete(client *godo.Client, name string) error {
	_, err := client.Domains.Delete(name)
	return err

}
