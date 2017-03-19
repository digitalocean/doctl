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
	"bytes"
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/spf13/cobra"
)

var bashCompletionSuccMsg = "Bash Completion file generated successfull.\n\n" +
	"To make completion available, you need to manually copy it to:\n" +
	"\t/etc/bash_completion.d/\n" +
	"or source it using\n" +
	"\tsource <(doctl completion bash -l \"\")\n"

// Completion creates the completion command
func Completion() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "completion",
			Short: "completion commands",
			Long:  "completion is used to create completion file for bash/zsh/fish",
		},
		DocCategories: []string{"snapshot"},
		IsIndex:       true,
	}

	cmdRunCompletionBash := CmdBuilder(cmd, RunCompletionBash, "bash", "generate bash completion file",
		Writer, aliasOpt("b"))
	AddStringFlag(cmdRunCompletionBash, doctl.ArgCompletionLocation, doctl.ArgShortCompletionLocation,
		"./doctl.sh", "location to store completion file; if location is empty, print to screen")

	return cmd
}

func RunCompletionBash(c *CmdConfig) error {
	var buf bytes.Buffer

	loc, err := c.Doit.GetString(c.NS, doctl.ArgCompletionLocation)
	if err != nil {
		return err
	}

	err = DoitCmd.GenBashCompletion(&buf)
	if err != nil {
		return fmt.Errorf("error while generating bash completion: %v", err)
	}

	if loc != "" {
		err = writeCompletion(loc, &buf)
		if err != nil {
			return fmt.Errorf("error while writing bash completion file: %v", err)
		}

		fmt.Printf("%s", bashCompletionSuccMsg)
	} else {
		fmt.Printf("%s", buf.String())
	}

	return nil
}

func RunCompletionZsh(c *CmdConfig) error {
	return fmt.Errorf("ZSH completion generation is not available now")
}

func RunCompletionFish(c *CmdConfig) error {
	return fmt.Errorf("Fish completion generation is not available now")
}
