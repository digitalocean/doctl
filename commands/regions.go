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
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/spf13/cobra"
)

// Region creates the region commands hierarchy.
func Region() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "region",
			Short: "Display commands to list datacenter regions",
			Long:  "The subcommands of `doctl compute region` retrieve information about DigitalOcean datacenter regions.",
		},
	}

	regionDesc := `List DigitalOcean datacenter regions displaying their name, slug, and availability.

Use the slugs displayed by this command to specify regions in other commands.
`
	CmdBuilder(cmd, RunRegionList, "list", "List datacenter regions", regionDesc,
		Writer, aliasOpt("ls"), displayerType(&displayers.Region{}))

	return cmd
}

// RunRegionList all regions.
func RunRegionList(c *CmdConfig) error {
	rs := c.Regions()

	list, err := rs.List()
	if err != nil {
		return err
	}

	image := &displayers.Region{Regions: list}
	return c.Display(image)
}
