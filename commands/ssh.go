package commands

import (
	"errors"
	"fmt"
	"io"
	"os/user"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	sshNoAddress = "could not find droplet address"
)

var (
	errSSHInvalidOptions = fmt.Errorf("neither id or name were supplied")
)

// SSH creates the ssh commands heirarchy
func SSH() *cobra.Command {
	usr, err := user.Current()
	if err != nil {
		logrus.Fatal(err.Error())
	}
	path := filepath.Join(usr.HomeDir, ".ssh", "id_rsa")

	cmdSSH := cmdBuilder(RunSSH, "ssh", "ssh to droplet", writer)
	addIntFlag(cmdSSH, doit.ArgDropletID, 0, "droplet id")
	addStringFlag(cmdSSH, doit.ArgDropletName, "", "droplet name")
	addStringFlag(cmdSSH, doit.ArgSSHUser, "root", "ssh user")
	addStringFlag(cmdSSH, doit.ArgsSSHKeyPath, path, "path to private ssh key")
	addIntFlag(cmdSSH, doit.ArgsSSHPort, 22, "port sshd is running on")

	return cmdSSH
}

// RunSSH finds a droplet to ssh to given input parameters (name or id).
func RunSSH(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)
	name := doit.DoitConfig.GetString(ns, doit.ArgDropletName)
	user := doit.DoitConfig.GetString(ns, doit.ArgSSHUser)
	keyPath := doit.DoitConfig.GetString(ns, doit.ArgsSSHKeyPath)
	port := doit.DoitConfig.GetInt(ns, doit.ArgsSSHPort)

	var droplet *godo.Droplet
	var err error

	switch {
	case id > 0 && len(name) < 1:
		droplet, err = getDropletByID(client, id)
		if err != nil {
			return err
		}
	case len(name) > 0 && id < 1:
		var droplets []godo.Droplet
		droplets, err = listDroplets(client)
		for _, d := range droplets {
			if d.Name == name {
				droplet = &d
				break
			}
		}

		if droplet == nil {
			return errors.New("could not find droplet by name")
		}

	default:
		return errSSHInvalidOptions
	}

	// CoreOS has no root user
	if droplet.Image.Distribution == "CoreOS" {
		user = "core"
	}

	publicIP := extractDropletPublicIP(droplet)

	if len(publicIP) < 1 {
		return errors.New(sshNoAddress)
	}

	return doit.DoitConfig.SSH(user, publicIP, keyPath, port)
}

func removeEmptyOptions(in []string) []string {
	var out []string
	if len(in) == 1 && in[0] == "[]" {
		return out
	}

	for _, s := range in {
		if len(s) > 0 {
			out = append(out, s)
		}
	}

	return out
}
