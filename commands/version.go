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
	"strconv"

	"github.com/bryanl/doit"
	"github.com/spf13/cobra"
)

type version struct {
	Major, Minor, Patch int
	Name, Build, Label  string
}

// Version creates a version command.
func Version() *Command {
	return &Command{
		Command: &cobra.Command{
			Use:   "version",
			Short: "show the current version",
			Run: func(cmd *cobra.Command, args []string) {
				if doit.Build != "" {
					doit.DoitVersion.Build = doit.Build
				}
				if doit.Major != "" {
					i, _ := strconv.Atoi(doit.Major)
					doit.DoitVersion.Major = i
				}
				if doit.Minor != "" {
					i, _ := strconv.Atoi(doit.Minor)
					doit.DoitVersion.Minor = i
				}
				if doit.Patch != "" {
					i, _ := strconv.Atoi(doit.Patch)
					doit.DoitVersion.Patch = i
				}
				if doit.Label != "" {
					doit.DoitVersion.Label = doit.Label
				}

				fmt.Println(doit.DoitVersion.Complete(&doit.GithubLatestVersioner{}))
			},
		},
	}
}
