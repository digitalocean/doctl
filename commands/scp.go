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
	"os/user"
	"path/filepath"
)

func SCP(parent *Command) *Command {
	usr, err := user.Current()
	checkErr(err)

	path := filepath.Join(usr.HomeDir, ".ssh", "id_rsa")

	cmdSCP := CmdBuilder(parent, RunSCP, "scp", "scp to droplet", Writer,
		docCategories("droplet"))
	AddStringFlag(cmdSCP, doctl.ArgsSSHKeyPath, "", path, "path to private ssh key")
	AddIntFlag(cmdSCP, doctl.ArgsSSHPort, "", 22, "port sshd is running on")

	return cmdSCP
}

func RunSCP(c *CmdConfig) error {
	return nil
}
