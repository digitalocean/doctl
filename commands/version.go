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
	"strconv"

	"github.com/digitalocean/doctl"
	"github.com/spf13/cobra"
)

// Version creates a version command.
func Version() *Command {
	return &Command{
		Command: &cobra.Command{
			Use:   "version",
			Short: "Show the current version",
			Long:  "The `doctl version` command displays the version of the doctl software.",
			Run: func(cmd *cobra.Command, args []string) {
				if doctl.Build != "" {
					doctl.DoitVersion.Build = doctl.Build
				}
				if doctl.Major != "" {
					i, _ := strconv.Atoi(doctl.Major)
					doctl.DoitVersion.Major = i
				}
				if doctl.Minor != "" {
					i, _ := strconv.Atoi(doctl.Minor)
					doctl.DoitVersion.Minor = i
				}
				if doctl.Patch != "" {
					i, _ := strconv.Atoi(doctl.Patch)
					doctl.DoitVersion.Patch = i
				}
				if doctl.Label != "" {
					doctl.DoitVersion.Label = doctl.Label
				}

				fmt.Println(doctl.DoitVersion.Complete(&doctl.GithubLatestVersioner{}))
			},
		},
	}
}
