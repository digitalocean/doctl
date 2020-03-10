/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"github.com/digitalocean/doctl/pkg/ssh"
)

var (
	sshHostRE = regexp.MustCompile(`^((?P<m1>\w+)@)?(?P<m2>.*?)(:(?P<m3>\d+))?$`)
)

// SSH creates the ssh commands hierarchy
func SSH(parent *Command) *Command {
	usr, err := user.Current()
	checkErr(err)

	path := filepath.Join(usr.HomeDir, ".ssh", "id_rsa")

	sshDesc := fmt.Sprintf(`Access a Droplet using SSH by providing its ID or name.

You may specify the user to login with by passing the `+"`"+`--%s`+"`"+` flag. To access the Droplet on a non-default port, use the `+"`"+`--%s`+"`"+` flag. By default, the connection will be made to the Droplet's public IP address. In order access it using its private IP address, use the `+"`"+`--%s`+"`"+` flag.
`, doctl.ArgSSHUser, doctl.ArgsSSHPort, doctl.ArgsSSHPrivateIP)

	cmdSSH := CmdBuilder(parent, RunSSH, "ssh <droplet-id|name>", "Access a Droplet using SSH", sshDesc, Writer)
	AddStringFlag(cmdSSH, doctl.ArgSSHUser, "", "root", "SSH user for connection")
	AddStringFlag(cmdSSH, doctl.ArgsSSHKeyPath, "", path, "Path to SSH private key")
	AddIntFlag(cmdSSH, doctl.ArgsSSHPort, "", 22, "The remote port sshd is running on")
	AddBoolFlag(cmdSSH, doctl.ArgsSSHAgentForwarding, "", false, "Enable SSH agent forwarding")
	AddBoolFlag(cmdSSH, doctl.ArgsSSHPrivateIP, "", false, "SSH to Droplet's private IP address")
	AddStringFlag(cmdSSH, doctl.ArgSSHCommand, "", "", "Command to execute on Droplet")

	return cmdSSH
}

// RunSSH finds a droplet to ssh to given input parameters (name or id).
func RunSSH(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	dropletID := c.Args[0]

	if dropletID == "" {
		return doctl.NewMissingArgsErr(c.NS)
	}

	user, err := c.Doit.GetString(c.NS, doctl.ArgSSHUser)
	if err != nil {
		return err
	}

	keyPath, err := c.Doit.GetString(c.NS, doctl.ArgsSSHKeyPath)
	if err != nil {
		return err
	}

	port, err := c.Doit.GetInt(c.NS, doctl.ArgsSSHPort)
	if err != nil {
		return err
	}

	var opts = make(ssh.Options)
	opts[doctl.ArgsSSHAgentForwarding], err = c.Doit.GetBool(c.NS, doctl.ArgsSSHAgentForwarding)
	if err != nil {
		return err
	}

	opts[doctl.ArgSSHCommand], err = c.Doit.GetString(c.NS, doctl.ArgSSHCommand)
	if err != nil {
		return nil
	}

	privateIPChoice, err := c.Doit.GetBool(c.NS, doctl.ArgsSSHPrivateIP)
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

		if shi.user != "" {
			user = shi.user
		}

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
			return errors.New("Could not find Droplet")
		}

	}

	if user == "" {
		user = defaultSSHUser(droplet)
	}

	ip, err := privateIPElsePub(droplet, privateIPChoice)
	if err != nil {
		return err
	}

	if ip == "" {
		return errors.New("Could not find Droplet address")
	}

	runner := c.Doit.SSH(user, ip, keyPath, port, opts)
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

func privateIPElsePub(droplet *do.Droplet, choice bool) (string, error) {
	if choice {
		return droplet.PrivateIPv4()
	}
	return droplet.PublicIPv4()
}
