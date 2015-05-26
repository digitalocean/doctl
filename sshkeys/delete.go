package sshkeys

import "github.com/digitalocean/godo"

// DeleteByID deletes a SSH key by id.
func DeleteByID(client *godo.Client, id int) error {
	_, err := client.Keys.DeleteByID(id)
	if err != nil {
		return err
	}

	return err
}

// DeleteByFingerprint deletes a SSH key by fingerprint.
func DeleteByFingerprint(client *godo.Client, fingerprint string) error {
	_, err := client.Keys.DeleteByFingerprint(fingerprint)
	return err
}
