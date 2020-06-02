/*
Copyright 2020 The Doctl Authors All rights reserved.
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
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/spf13/cobra"
)

// OneClicks creates the 1-click command.
func OneClicks() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "1-click",
			Short: "Display commands that pertain to 1-click applications",
			Long:  "The commands under `doctl 1-click` are for interacting with DigitalOcean 1-Click applications.",
		},
	}

	cmdOneClickList := CmdBuilder(cmd, RunOneClickList, "list", "Retrieve a list of 1-Click applications", "Use this command to retrieve a list of 1-Click applications. You can narrow it by type, current types: kubernetes, droplet", Writer,
		aliasOpt("ls"), displayerType(&displayers.OneClick{}))

	AddStringFlag(cmdOneClickList, doctl.ArgOneClickType, "", "", "The 1-Click type. Valid types are one of the following: kubernetes, droplet")

	return cmd
}

// RunOneClickList retrieves a list of 1-clicks by type. If no type is specified then all types are returned.
func RunOneClickList(c *CmdConfig) error {
	oneClickType, err := c.Doit.GetString(c.NS, doctl.ArgOneClickType)
	if err != nil {
		return err
	}

	oneClicks := c.OneClicks()
	oneClickList, err := oneClicks.List(oneClickType)
	if err != nil {
		return err
	}

	items := &displayers.OneClick{OneClicks: oneClickList}

	return c.Display(items)
}
