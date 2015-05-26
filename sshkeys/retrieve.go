package sshkeys

import (
	"fmt"

	"github.com/digitalocean/godo"
)

// RetrieveByID retrieves a public SSH key by ID.
func RetrieveByID(client *godo.Client, id int) (*godo.Key, error) {
	key, _, err := client.Keys.GetByID(id)
	if err != nil {
		return nil, err
	}

	return key, err
}

// RetrieveByFingerprint retrieves a public SSH key by fingerprint.
func RetrieveByFingerprint(client *godo.Client, fingerprint string) (*godo.Key, error) {
	key, _, err := client.Keys.GetByFingerprint(fingerprint)
	if err != nil {
		return nil, err
	}

	return key, err
}

// IsValidGetArgs returns true if arguments for retrieving a public SSH key are correct.
// Arguments are correct if an id or a fingerprint has been supplied.
func IsValidGetArgs(id int, fingerprint string) error {
	if id > 0 && len(fingerprint) > 0 {
		return fmt.Errorf("id and fingerprint were specified")
	}

	if id == 0 && len(fingerprint) == 0 {
		return fmt.Errorf("no id or fingerprint was specified")
	}

	return nil
}
