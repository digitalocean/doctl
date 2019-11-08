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
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Registry creates the registry command
func Registry() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "registry",
			Aliases: []string{"reg", "r"},
			Short:   "registry commands",
			Long:    "registry is used to access container registry commands",
		},
	}

	CmdBuilder(cmd, RunRegistryCreate, "create <registry-name>", "create container registry", Writer)

	CmdBuilder(cmd, RunRegistryGet, "get", "get the container registry", Writer, aliasOpt("g"), displayerType(&displayers.Registry{}))

	cmdRunRegistryDelete := CmdBuilder(cmd, RunRegistryDelete, "delete", "delete the container registry", Writer, aliasOpt("del"))
	AddBoolFlag(cmdRunRegistryDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force registry delete")

	return cmd
}

// Registry

// RunRegistryCreate creates a registry
func RunRegistryCreate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name := c.Args[0]
	rs := c.Registry()

	rcr := &godo.RegistryCreateRequest{Name: name}
	r, err := rs.Create(rcr)
	if err != nil {
		return err
	}

	return displayRegistries(c, *r)
}

// RunRegistryGet returns the registry
func RunRegistryGet(c *CmdConfig) error {
	reg, err := c.Registry().Get()
	if err != nil {
		return err
	}

	return displayRegistries(c, *reg)
}

// RunRegistryDelete delete the registry
func RunRegistryDelete(c *CmdConfig) error {
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirm("delete registry") != nil {
		return fmt.Errorf("operation aborted")
	}

	return c.Registry().Delete()
}

func displayRegistries(c *CmdConfig, registries ...do.Registry) error {
	item := &displayers.Registry{
		Registries: registries,
	}
	return c.Display(item)
}
