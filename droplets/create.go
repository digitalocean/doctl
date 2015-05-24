package droplets

import (
	"strconv"

	"github.com/digitalocean/godo"
)

type CreateRequest struct {
	Name              string
	Region            string
	Size              string
	Image             string
	SSHKeys           []string
	Backups           bool
	IPv6              bool
	PrivateNetworking bool
	UserData          string
}

func Create(client *godo.Client, cr *CreateRequest) (*godo.DropletRoot, error) {

	image := godo.DropletCreateImage{}

	if i, err := strconv.Atoi(cr.Image); err == nil {
		image.ID = i
	} else {
		image.Slug = cr.Image
	}

	sshKeys := []godo.DropletCreateSSHKey{}
	for _, rawKey := range cr.SSHKeys {
		if i, err := strconv.Atoi(rawKey); err == nil {
			sshKeys = append(sshKeys, godo.DropletCreateSSHKey{ID: i})
			continue
		}

		sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: rawKey})
	}

	dcr := &godo.DropletCreateRequest{
		Name:              cr.Name,
		Region:            cr.Region,
		Size:              cr.Size,
		Image:             image,
		SSHKeys:           sshKeys,
		Backups:           cr.Backups,
		IPv6:              cr.IPv6,
		PrivateNetworking: cr.PrivateNetworking,
		UserData:          cr.UserData,
	}

	r, _, err := client.Droplets.Create(dcr)

	return r, err
}
