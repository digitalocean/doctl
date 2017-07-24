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

var r1 = regexp.MustCompile("([^:]+):(.+)")
var r2 = regexp.MustCompile("(([^@]+)@)?(.+)")

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
	host1, err := parseArg(arg1)
	if err != nil {
		return err
	}
	arg2 := c.Args[1]
	host2, err := parseArg(arg2)
	if err != nil {
		return err
	}

	var droplet1 *do.Droplet
	var droplet2 *do.Droplet

	ds := c.Droplets()

	// Parse first argument
	if id, err := strconv.Atoi(host1.host); err == nil {
		// dropletID is an integer

		doDroplet, err := ds.Get(id)
		if err != nil {
			return err
		}

		droplet1 = doDroplet
	} else {
		// dropletID is a string
		droplets, err := ds.List()
		if err != nil {
			return err
		}

		for _, d := range droplets {
			if d.Name == host1.host {
				droplet1 = &d
				break
			}
		}

		if droplet1 == nil {
			return errors.New("could not find droplet")
		}
	}

	// Parse second argument
	if id, err := strconv.Atoi(host2.host); err == nil {
		// dropletID is an integer

		doDroplet, err := ds.Get(id)
		if err != nil {
			return err
		}

		droplet2 = doDroplet
	} else {
		// dropletID is a string
		droplets, err := ds.List()
		if err != nil {
			return err
		}

		for _, d := range droplets {
			if d.Name == host1.host {
				droplet2 = &d
				break
			}
		}

		if droplet2 == nil {
			return errors.New("could not find droplet")
		}
	}

	host1.host, err = droplet1.PublicIPv4()
	if err != nil {
		return err
	}
	host2.host, err = droplet2.PublicIPv4()
	if err != nil {
		return err
	}

	fmt.Println(host1.host)
	fmt.Println(host2.host)

	file1 := host1.username + "@" + host1.host + ":" + host1.file
	file2 := host2.username + "@" + host2.host + ":" + host2.file

	runner := c.Doit.SCP(file1, file2, keyPath, port)
	return runner.Run()
}

type hostInfo struct {
	username string
	host     string
	file     string
}

func parseArg(arg string) (*hostInfo, error) {
	m := r1.FindStringSubmatch(arg)
	if len(m) != 3 {
		return nil, fmt.Errorf("incorrect argument format")
	}
	hostData := m[1]
	file := m[2]
	m = r2.FindStringSubmatch(hostData)
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

	h := &hostInfo{
		username: user,
		host:     host,
		file:     file,
	}
	return h, nil
}
