package imageactions

import "github.com/digitalocean/godo"

// Get retrieves an action for an image.
func Get(client *godo.Client, imageID, actionID int) (*godo.Action, error) {
	action, _, err := client.ImageActions.Get(imageID, actionID)
	if err != nil {
		return nil, err
	}

	return action, err
}
