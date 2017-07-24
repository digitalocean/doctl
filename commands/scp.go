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
	"github.com/digitalocean/doctl"

	"errors"
	"fmt"
	"github.com/digitalocean/doctl/do"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	scpREPath = regexp.MustCompile("([^:]+):(.+)")
	scpREHost = regexp.MustCompile("(([^@]+)@)?(.+)")
)

func SCP(parent *Command) *Command {
	usr, err := user.Current()
	checkErr(err)

	path := filepath.Join(usr.HomeDir, ".ssh", "id_rsa")

	cmdSCP := CmdBuilder(parent, RunSCP, "scp", "secure copy files", Writer,
		docCategories("droplet"))
	AddStringFlag(cmdSCP, doctl.ArgsSSHKeyPath, "", path, "path to private ssh key")
	AddIntFlag(cmdSCP, doctl.ArgsSSHPort, "", 22, "port sshd is running on")

	return cmdSCP
}

func RunSCP(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	keyPath, err := c.Doit.GetString(c.NS, doctl.ArgsSSHKeyPath)
	if err != nil {
		return err
	}

	port, err := c.Doit.GetInt(c.NS, doctl.ArgsSSHPort)
	if err != nil {
		return err
	}

	arg1 := c.Args[0]
	host1, err := extractArgument(arg1)
	if err != nil {
		return err
	}
	arg2 := c.Args[1]
	host2, err := extractArgument(arg2)
	if err != nil {
		return err
	}

	ds := c.Droplets()
	err = matchSCPDroplet(host1, ds)
	if err != nil {
		return err
	}
	err = matchSCPDroplet(host2, ds)
	if err != nil {
		return err
	}

	runner := c.Doit.SCP(formatSCPArgument(host1), formatSCPArgument(host2), keyPath, port)
	return runner.Run()
}

type scpHostInfo struct {
	username string
	host     string
	file     string
}

func extractArgument(arg string) (*scpHostInfo, error) {
	if !strings.Contains(arg, "@") && !strings.Contains(arg, ":") {
		h := &scpHostInfo{
			username: "",
			host:     "",
			file:     arg,
		}
		return h, nil
	}

	m := scpREPath.FindStringSubmatch(arg)
	if len(m) != 3 {
		return nil, fmt.Errorf("incorrect argument format")
	}
	hostData := m[1]
	file := m[2]
	m = scpREHost.FindStringSubmatch(hostData)
	if len(m) != 4 {
		return nil, fmt.Errorf("incorrect argument format")
	}
	if m[2] == "" {
		// make sure host is in the correct format.
		if strings.Contains(m[3], "@") {
			m[3] = m[3][1:]
		}
	}
	user := m[2]
	host := m[3]

	h := &scpHostInfo{
		username: user,
		host:     host,
		file:     file,
	}
	return h, nil
}

func formatSCPArgument(host *scpHostInfo) string {
	var arg string
	if host.username != "" {
		arg = host.username + "@"
	}
	if host.host != "" {
		arg = arg + host.host + ":"
	}
	return arg + host.file
}

func matchSCPDroplet(h *scpHostInfo, ds do.DropletsService) error {
	if h.host != "" {
		var droplet *do.Droplet
		var err error
		if id, err := strconv.Atoi(h.host); err == nil {
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

			for _, d := range droplets {
				if d.Name == h.host {
					droplet = &d
					break
				}
			}

			if droplet == nil {
				return errors.New("could not find droplet")
			}
		}
		h.host, err = droplet.PublicIPv4()
		if err != nil {
			return err
		}
		if h.username == "" {
			h.username = defaultSSHUser(droplet)
		}
	}

	return nil
}
