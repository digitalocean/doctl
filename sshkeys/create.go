package sshkeys

import "github.com/digitalocean/godo"

// CreateRequest is a struct that contains items describing a ssh key.
type CreateRequest struct {
	Name      string
	PublicKey string
}

// IsValid tests if the CreateRequest is valid. It is valid if the name
// and public key contents are not blank.
func (cr *CreateRequest) IsValid() bool {
	return len(cr.Name) > 0 && len(cr.PublicKey) > 0
}

// Create uploads a SSH key.
func Create(client *godo.Client, cr *CreateRequest) (*godo.Key, error) {
	kcr := &godo.KeyCreateRequest{
		Name:      cr.Name,
		PublicKey: cr.PublicKey,
	}

	r, _, err := client.Keys.Create(kcr)

	return r, err
}
