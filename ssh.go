package doit

import (
	"fmt"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

const (
	sshNoAddress = "could not find droplet address"
)

var (
	errSSHInvalidOptions = fmt.Errorf("neither id or name were supplied")
)

// SSH finds a droplet to ssh to given input parameters (name or id).
func SSH(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgDropletID)
	name := c.String(ArgDropletName)
	user := c.String(ArgSSHUser)

	if len(user) < 1 {
		user = "root"
	}

	var droplet *godo.Droplet
	var err error

	switch {
	case id > 0 && len(name) < 1:
		droplet, err = getDropletByID(client, id)
		if err != nil {
			Bail(err, sshNoAddress)
			return
		}
	case len(name) > 0 && id < 1:
		var droplets []godo.Droplet
		droplets, err = listDroplets(client, LoadOpts(c))
		for _, d := range droplets {
			if d.Name == name {
				droplet = &d
				break
			}
		}

		if droplet == nil {
			Bail(fmt.Errorf("could not find droplet by name"), sshNoAddress)
			return
		}

	default:
		Bail(errSSHInvalidOptions, sshNoAddress)
		return
	}

	publicIP := extractDropletPublicIP(droplet)

	if len(publicIP) < 1 {
		Bail(fmt.Errorf("no public interface for droplet"), sshNoAddress)
	}

	err = DefaultConfig.SSH(user, publicIP)
	if err != nil {
		Bail(err, "unable to ssh to host")
	}
}
