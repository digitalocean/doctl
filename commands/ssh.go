package commands

import (
	"errors"
	"fmt"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/spf13/cobra"
)

const (
	sshNoAddress = "could not find droplet address"
)

var (
	errSSHInvalidOptions = fmt.Errorf("neither id or name were supplied")
	sshHostRE            = regexp.MustCompile("^((?P<m1>\\w+)@)?(?P<m2>.*?)(:(?P<m3>\\d+))?$")
)

// SSH creates the ssh commands heirarchy
func SSH() *cobra.Command {
	usr, err := user.Current()
	checkErr(err)

	path := filepath.Join(usr.HomeDir, ".ssh", "id_rsa")

	cmdSSH := cmdBuilder(nil, RunSSH, "ssh <droplet-id | host>", "ssh to droplet", writer)
	addStringFlag(cmdSSH, doit.ArgSSHUser, "root", "ssh user")
	addStringFlag(cmdSSH, doit.ArgsSSHKeyPath, path, "path to private ssh key")
	addIntFlag(cmdSSH, doit.ArgsSSHPort, 22, "port sshd is running on")

	return cmdSSH.Command
}

// RunSSH finds a droplet to ssh to given input parameters (name or id).
func RunSSH(c *cmdConfig) error {
	if len(c.args) == 0 {
		return doit.NewMissingArgsErr(c.ns)
	}

	dropletID := c.args[0]

	if dropletID == "" {
		return doit.NewMissingArgsErr(c.ns)
	}

	user, err := c.doitConfig.GetString(c.ns, doit.ArgSSHUser)
	if err != nil {
		return err
	}

	keyPath, err := c.doitConfig.GetString(c.ns, doit.ArgsSSHKeyPath)
	if err != nil {
		return err
	}

	port, err := c.doitConfig.GetInt(c.ns, doit.ArgsSSHPort)
	if err != nil {
		return err
	}

	var droplet *do.Droplet

	ds := c.dropletsService()
	if id, err := strconv.Atoi(dropletID); err == nil {
		// dropletID is an integer

		doDroplet, err := ds.Get(id)
		if err != nil {
			return err
		}

		droplet = doDroplet
	} else {
		// dropletID is a string
		droplets, err := ds.List()
		if err != nil {
			return err
		}

		shi := extractHostInfo(dropletID)

		user = shi.user
		if i, err := strconv.Atoi(shi.port); shi.port != "" && err != nil {
			port = i
		}

		for _, d := range droplets {
			if d.Name == shi.host {
				droplet = &d
				break
			}
			if strconv.Itoa(d.ID) == shi.host {
				droplet = &d
				break
			}
		}

		if droplet == nil {
			return errors.New("could not find droplet")
		}

	}

	if user == "" {
		user = defaultSSHUser(droplet)
	}

	ip, err := droplet.PublicIPv4()
	if err != nil {
		return err
	}

	if ip == "" {
		return errors.New("could not find droplet address")
	}

	runner := c.doitConfig.SSH(user, ip, keyPath, port)
	return runner.Run()
}

func defaultSSHUser(droplet *do.Droplet) string {
	slug := strings.ToLower(droplet.Image.Slug)
	if strings.Contains(slug, "coreos") {
		return "core"
	}

	return "root"
}

type sshHostInfo struct {
	user string
	host string
	port string
}

func extractHostInfo(in string) sshHostInfo {
	m := sshHostRE.FindStringSubmatch(in)
	r := map[string]string{}
	for i, n := range sshHostRE.SubexpNames() {
		r[n] = m[i]
	}

	return sshHostInfo{
		user: r["m1"],
		host: r["m2"],
		port: r["m3"],
	}
}
