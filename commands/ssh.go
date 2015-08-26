package commands

import (
	"errors"
	"fmt"
	"io"

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
	cmdSSH := cmdBuilder(RunSSH, "ssh", "ssh to droplet", writer)
	addIntFlag(cmdSSH, doit.ArgDropletID, 0, "droplet id")
	addStringFlag(cmdSSH, doit.ArgDropletName, "", "droplet name")
	addStringFlag(cmdSSH, doit.ArgSSHUser, "root", "ssh user")
	addStringSliceFlag(cmdSSH, doit.ArgSSHOption, nil, "ssh flag")

	return cmdSSH
}

// RunSSH finds a droplet to ssh to given input parameters (name or id).
func RunSSH(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetInt(ns, doit.ArgDropletID)
	name := doit.VConfig.GetString(ns, doit.ArgDropletName)
	user := doit.VConfig.GetString(ns, doit.ArgSSHUser)
	options := doit.VConfig.GetStringSlice(ns, doit.ArgSSHOption)

	options = removeEmptyOptions(options)

	if len(user) < 1 {
		user = "root"
	}

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

	publicIP := extractDropletPublicIP(droplet)

	if len(publicIP) < 1 {
		return errors.New(sshNoAddress)
	}

	runner := doit.VConfig.SSH(user, publicIP, options)
	return runner.Run()
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
