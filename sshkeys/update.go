package sshkeys

import "github.com/digitalocean/godo"

// UpdateRequest contains items which can be updated in a public SSH key.
type UpdateRequest struct {
	Name string
}

// UpdateByID updates a public SSH key by id.
func UpdateByID(client *godo.Client, id int, ur *UpdateRequest) (*godo.Key, error) {
	kur := &godo.KeyUpdateRequest{
		Name: ur.Name,
	}

	key, _, err := client.Keys.UpdateByID(id, kur)
	if err != nil {
		return nil, err
	}

	return key, err
}

// UpdateByFingerprint updates a public SSH key by fingerprint.
func UpdateByFingerprint(client *godo.Client, fingerprint string, ur *UpdateRequest) (*godo.Key, error) {
	kur := &godo.KeyUpdateRequest{
		Name: ur.Name,
	}

	key, _, err := client.Keys.UpdateByFingerprint(fingerprint, kur)
	if err != nil {
		return nil, err
	}

	return key, err
}
