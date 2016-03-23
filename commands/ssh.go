/*
Copyright 2016 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"errors"
	"fmt"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
)

const (
	sshNoAddress = "could not find droplet address"
)

var (
	errSSHInvalidOptions = fmt.Errorf("neither id or name were supplied")
	sshHostRE            = regexp.MustCompile("^((?P<m1>\\w+)@)?(?P<m2>.*?)(:(?P<m3>\\d+))?$")
)

// SSH creates the ssh commands heirarchy
func SSH() *Command {
	usr, err := user.Current()
	checkErr(err)

	path := filepath.Join(usr.HomeDir, ".ssh", "id_rsa")

	cmdSSH := CmdBuilder(nil, RunSSH, "ssh <droplet-id | host>", "ssh to droplet", Writer,
		docCategories("droplet"))
	AddStringFlag(cmdSSH, doit.ArgSSHUser, "root", "ssh user")
	AddStringFlag(cmdSSH, doit.ArgsSSHKeyPath, path, "path to private ssh key")
	AddIntFlag(cmdSSH, doit.ArgsSSHPort, 22, "port sshd is running on")

	return cmdSSH
}

// RunSSH finds a droplet to ssh to given input parameters (name or id).
func RunSSH(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doit.NewMissingArgsErr(c.NS)
	}

	dropletID := c.Args[0]

	if dropletID == "" {
		return doit.NewMissingArgsErr(c.NS)
	}

	user, err := c.Doit.GetString(c.NS, doit.ArgSSHUser)
	if err != nil {
		return err
	}

	keyPath, err := c.Doit.GetString(c.NS, doit.ArgsSSHKeyPath)
	if err != nil {
		return err
	}

	port, err := c.Doit.GetInt(c.NS, doit.ArgsSSHPort)
	if err != nil {
		return err
	}

	var droplet *do.Droplet

	ds := c.Droplets()
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

	runner := c.Doit.SSH(user, ip, keyPath, port)
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
