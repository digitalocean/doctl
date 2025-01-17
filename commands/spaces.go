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

import "github.com/spf13/cobra"

func Spaces() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "spaces",
			Aliases: []string{"sp"},
			Short:   "Display commands that manage DigitalOcean Spaces.",
			Long:    "The subcommands of `doctl spaces` allow you to access and manage Spaces.",
			GroupID: manageResourcesGroup,
		},
	}

	cmd.AddCommand(SpacesKeys())

	return cmd
}
