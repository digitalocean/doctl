package sshkeys

import "github.com/digitalocean/godo"

type CreateRequest struct {
	Name      string
	PublicKey string
}

func (cr *CreateRequest) IsValid() bool {
	return len(cr.Name) > 0 && len(cr.PublicKey) > 0
}

func Create(client *godo.Client, cr *CreateRequest) (*godo.Key, error) {
	kcr := &godo.KeyCreateRequest{
		Name:      cr.Name,
		PublicKey: cr.PublicKey,
	}

	r, _, err := client.Keys.Create(kcr)

	return r, err
}
