package sshkeys

import "github.com/digitalocean/godo"

type UpdateRequest struct {
	Name string
}

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
