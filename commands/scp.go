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

	"fmt"
	"github.com/digitalocean/doctl/do"
	"os/user"
	"path/filepath"
	"regexp"
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
	/*if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}*/

	arg1 := c.Args[0]
	file1, err := parseArg(arg1)
	if err != nil {
		return err
	}
	arg2 := c.Args[1]
	file2, err := parseArg(arg2)
	if err != nil {
		return err
	}

	fmt.Println(file1)
	fmt.Println(file2)

	return nil
}

type hostInfo struct {
	username string
	ip       string
	file     string
}

/*type scpInfo struct {
	file1 *hostInfo
	file2 *hostInfo
}*/

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
		ip:       host,
		file:     file,
	}
	return h, nil
}
