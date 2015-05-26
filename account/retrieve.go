package account

import "github.com/digitalocean/godo"

// Get retrieve the current account.
func Get(client *godo.Client) (*godo.Account, error) {
	root, _, err := client.Account.Get()
	if err != nil {
		return nil, err
	}

	return root.Account, nil
}
