package commands

import (
	"errors"
	"fmt"
	"io"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/cobra"
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
func RunSSH(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()
	id, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	name, err := config.GetString(ns, doit.ArgDropletName)
	if err != nil {
		return err
	}

	user, err := config.GetString(ns, doit.ArgSSHUser)
	if err != nil {
		return err
	}

	keyPath, err := config.GetString(ns, doit.ArgsSSHKeyPath)
	if err != nil {
		return err
	}

	port, err := config.GetInt(ns, doit.ArgsSSHPort)
	if err != nil {
		return err
	}

	var droplet *godo.Droplet

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

	user = defaulSSHUser(droplet)
	publicIP := extractDropletPublicIP(droplet)

	if len(publicIP) < 1 {
		return errors.New(sshNoAddress)
	}

	runner := config.SSH(user, publicIP, keyPath, port)
	return runner.Run()

	// return config.SSH(user, publicIP, keyPath, port)
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

func defaulSSHUser(droplet *godo.Droplet) string {
	slug := strings.ToLower(droplet.Image.Slug)
	if strings.Contains(slug, "coreos") {
		return "core"
	}

	return "root"
}
