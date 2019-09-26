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
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// GenDocs creates the gen-docs command
func GenDocs(parent *Command) *Command {
	cmdGenDocs := cmdBuilderWithInit(parent, RunGenDocs, "gen-docs", "generates documentation", "this is a poorly documented command to generate docs", Writer, false, hiddenCmd())
	AddStringFlag(cmdGenDocs, "dir", "", "", "path to a directory for yaml output", requiredOpt())

	return cmdGenDocs
}

// RunGenDocs outputs docs.
func RunGenDocs(c *CmdConfig) error {
	dir, err := c.Doit.GetString(c.NS, "dir")
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); err != nil {
		return err
	}

	for _, cmd := range DoitCmd.Commands() {
		err := writeDocs(cmd, dir)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func writeDocs(cmd *cobra.Command, dir string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := writeDocs(c, dir); err != nil {
			return err
		}
	}

	basename := strings.Replace(cmd.CommandPath(), " ", "_", -1) + ".yaml"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	doc.GenYaml(cmd, f)
	use := cmd.UseLine()
	usage := fmt.Sprintf("usage: %s\n", use)
	if _, err := f.WriteString(usage); err != nil {
		return err
	}

	return nil
}
