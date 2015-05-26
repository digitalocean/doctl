package sshkeys

import "github.com/digitalocean/godo"

func DeleteByID(client *godo.Client, id int) error {
	_, err := client.Keys.DeleteByID(id)
	if err != nil {
		return err
	}

	return err
}

func DeleteByFingerprint(client *godo.Client, fingerprint string) error {
	_, err := client.Keys.DeleteByFingerprint(fingerprint)
	if err != nil {
		return err
	}

	return err
}
