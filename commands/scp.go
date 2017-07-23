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
	"fmt"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
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
	/*if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}*/

	file1 := c.Args[0]
	parseArg(file1)
	//file2 := c.Args[1]

	return nil
}

type hostInfo struct {
	username string
	ip       string
	file     string
}

type scpInfo struct {
	file1 *hostInfo
	file2 *hostInfo
}

func parseArg(arg string) (*hostInfo, error) {
	if !strings.Contains(arg, ":") {
		return nil, nil
	}
	// zero index will contain username and host
	// one index will contain file location
	file := strings.Split(arg, ":")
	// if host or file is empty return error
	if file[0] == "" || file[1] == "" {
		return nil, nil
	}
	// check is username provided and split to array if it is
	// zero index will contain username
	// one index will contain host
	var host []string
	if strings.Contains(file[0], "@") {
		host = strings.Split(file[0], "@")
	}
	// if host is not provided return error
	if host[1] == "" {
		return nil, nil
	}
	id, err := getDropletIP(host[1])
	if err != nil {
		return nil, err
	}
	// if username is not provided assume default
	if host[0] == "" {

	}

	// get IP

	h := &hostInfo{
		username: host[0],
		ip:       host[1],
		file:     file[1],
	}

	fmt.Println(h)
	return h, nil
}

func getDropletIP(data string) (int, error) {
	return 0, nil
}
