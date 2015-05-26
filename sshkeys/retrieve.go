package sshkeys

import (
	"fmt"

	"github.com/digitalocean/godo"
)

func RetrieveByID(client *godo.Client, id int) (*godo.Key, error) {
	key, _, err := client.Keys.GetByID(id)
	if err != nil {
		return nil, err
	}

	return key, err
}

func RetrieveByFingerprint(client *godo.Client, fingerprint string) (*godo.Key, error) {
	key, _, err := client.Keys.GetByFingerprint(fingerprint)
	if err != nil {
		return nil, err
	}

	return key, err
}

func IsValidGetArgs(id int, fingerprint string) error {
	if id > 0 && len(fingerprint) > 0 {
		return fmt.Errorf("id and fingerprint were specified")
	}

	if id == 0 && len(fingerprint) == 0 {
		return fmt.Errorf("no id or fingerprint was specified")
	}

	return nil
}
