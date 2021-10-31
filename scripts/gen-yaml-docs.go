/*
Copyright 2019 The Doctl Authors All rights reserved.
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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitalocean/doctl/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	dir := os.Getenv("DOCS_OUT")
	if dir == "" {
		fmt.Printf("DOCS_OUT environment variable not set.\n")
		os.Exit(1)
	}
	if _, err := os.Stat(dir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for _, cmd := range commands.DoitCmd.Commands() {
		err := writeDocs(cmd, dir)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// Iterate through commands in commands/*.go and run Cobra's GenYaml function.
func writeDocs(cmd *cobra.Command, dir string) error {
	// Exit if there's an error
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := writeDocs(c, dir); err != nil {
			return err
		}
	}
	// Set filename to doctl_namespace_command.yaml, and create file
	basename := strings.Replace(cmd.CommandPath(), " ", "_", -1) + ".yaml"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Call Cobra's GenYaml command, passing in the created file
	err = doc.GenYaml(cmd, f)
	if err != nil {
		return err
	}
	// Append alias information to the standard YAML output
	aliases := fmt.Sprintf("aliases: %s\n", strings.Join(cmd.Aliases, ", "))

	if _, err := f.WriteString(aliases); err != nil {
		return err
	}
	return nil
}
