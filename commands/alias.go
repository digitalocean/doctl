/*
Copyright 2017 The Doctl Authors All rights reserved.
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
	"github.com/gobwas/glob"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type aliasDesc struct {
	Alias        string `json:"alias"`
	AliasCommand string `json:"alias_command"`
}

// Alias creates a alias command.
func Alias() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "alias",
			Short: "alias commands",
			Long:  "alias is used to manage aliases",
		},
	}

	CmdBuilder(cmd, RunAliasList, "list", "list [glob]", Writer, aliasOpt("ls"))

	CmdBuilder(cmd, RunAliasGet, "get", "get <alias-name>", Writer, aliasOpt("g"))

	CmdBuilder(cmd, RunAliasAdd, "add", "add <alias-name> <alias-command>", Writer, aliasOpt("a"))

	cmdRunAliasDelete := CmdBuilder(cmd, RunAliasDelete, "delete", "delete <alias-name> [alias-name ...]",
		Writer, aliasOpt("d"))
	AddBoolFlag(cmdRunAliasDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force alias delete")

	return cmd
}

// RunAliasList returns the list of all aliases
func RunAliasList(c *CmdConfig) error {

	list := AliasReader.AllKeys()
	var al []aliasDesc

	matches := []glob.Glob{}
	for _, globStr := range c.Args {
		g, err := glob.Compile(globStr)
		if err != nil {
			return fmt.Errorf("unknown glob %q", globStr)
		}

		matches = append(matches, g)
	}

	for _, a := range list {
		var skip = true
		if len(matches) == 0 {
			skip = false
		} else {
			for _, m := range matches {
				if m.Match(a) {
					skip = false
				}
			}
		}

		if !skip {
			al = append(al, aliasDesc{a, AliasReader.GetString(a)})
		}
	}

	item := &alias{aliases: al}
	return c.Display(item)

}

// RunAliasGet returns alias details
func RunAliasGet(c *CmdConfig) error {

	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	ali := c.Args[0]
	var a []aliasDesc

	if value := AliasReader.GetString(ali); value != "" {
		a = append(a, aliasDesc{ali, AliasReader.GetString(ali)})
	} else {
		return fmt.Errorf("alias %s not found", ali)
	}

	item := &alias{aliases: a}
	return c.Display(item)

}

// RunAliasSet creates new alias
func RunAliasAdd(c *CmdConfig) error {

	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	alias := c.Args[0]
	command := c.Args[1]

	AliasReader.Set(alias, command)

	return writeAlias()
}

// RunAliasDelete deletes an existing alias
func RunAliasDelete(c *CmdConfig) error {

	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirm("delete alias(es)") == nil {
		newAliasReader := viper.New()
		keys := AliasReader.AllKeys()
		var match bool

		for _, key := range keys {
			for _, alias := range c.Args {
				if key == alias {
					match = true
					break
				}
			}
			if !match {
				newAliasReader.Set(key, AliasReader.GetString(key))
			} else {
				match = false
			}
		}

		AliasReader = newAliasReader

		return writeAlias()
	}

	return nil
}
