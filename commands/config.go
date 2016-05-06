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

import "github.com/spf13/cobra"

// Config creates config commands for doctl.
func Config() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "config",
			Short: "config commands",
			Long:  "config is used to access config commands",
		},
	}

	CmdBuilder(cmd, RunConfigGet, "get", "get configuration item", Writer, docCategories("config"))
	CmdBuilder(cmd, RunConfigSet, "set", "set configuration item", Writer, docCategories("config"))
	CmdBuilder(cmd, RunConfigDelete, "delete", "delete configuration item", Writer, docCategories("config"))
	CmdBuilder(cmd, RunConfigList, "list", "list configuration items", Writer,
		docCategories("config"), aliasOpt("ls"))

	return cmd
}

// RunConfigGet retrieves configuration items.
func RunConfigGet(c *CmdConfig) error {
	return nil
}

// RunConfigSet sets configuration items.
func RunConfigSet(c *CmdConfig) error {
	return nil
}

// RunConfigDelete deletes configuration items.
func RunConfigDelete(c *CmdConfig) error {
	return nil
}

// RunConfigList lists configuration items.
func RunConfigList(c *CmdConfig) error {
	return nil
}
