package dropletactions

import "github.com/digitalocean/godo"

// Get returns a droplet action by id.
func Get(client *godo.Client, dropletID, actionID int) (*godo.Action, error) {
	r, _, err := client.DropletActions.Get(dropletID, actionID)
	if err != nil {
		return nil, err
	}

	return r, err
}
