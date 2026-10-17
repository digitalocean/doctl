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
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

// ModelEvaluationPresetCmd handles operations on model evaluation presets.
func ModelEvaluationPresetCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "preset",
			Aliases: []string{"presets"},
			Short:   "Display commands that manage GenAI model evaluation presets.",
			Long:    "The subcommands of `doctl evaluation preset` inspect and delete saved model evaluation presets.",
		},
	}

	presetDetails := `
		- The preset UUID
		- The preset name
		- The candidate model name
		- The dataset name
		- The judge model name
		- The preset creation timestamp
	`

	cmdPresetList := CmdBuilder(
		cmd,
		RunModelEvaluationPresetList,
		"list",
		"List all model evaluation presets",
		"Retrieves a list of model evaluation presets, where each preset contains:"+presetDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.ModelEvaluationPreset{}),
	)
	cmdPresetList.Example = "The following example lists model evaluation presets: " +
		"`doctl evaluation preset list`"

	cmdPresetGet := CmdBuilder(
		cmd,
		RunModelEvaluationPresetGet,
		"get <preset-uuid>",
		"Retrieve a model evaluation preset",
		"Retrieves information about a model evaluation preset, including:"+presetDetails,
		Writer, aliasOpt("g"),
		displayerType(&displayers.ModelEvaluationPreset{}),
	)
	cmdPresetGet.Example = "The following example retrieves a model evaluation preset: " +
		"`doctl evaluation preset get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdPresetDelete := CmdBuilder(
		cmd,
		RunModelEvaluationPresetDelete,
		"delete <preset-uuid>",
		"Delete a model evaluation preset",
		"Deletes a model evaluation preset by its UUID.",
		Writer, aliasOpt("del", "rm"),
	)
	AddBoolFlag(cmdPresetDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the evaluation preset without a confirmation prompt")
	cmdPresetDelete.Example = "The following example deletes a model evaluation preset: " +
		"`doctl evaluation preset delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	return cmd
}

// RunModelEvaluationPresetList lists model evaluation presets.
func RunModelEvaluationPresetList(c *CmdConfig) error {
	presets, err := c.AgentPlatform().ListModelEvaluationPresets()
	if err != nil {
		return err
	}

	return c.Display(&displayers.ModelEvaluationPreset{Presets: presets})
}

// RunModelEvaluationPresetGet retrieves a model evaluation preset by its UUID.
func RunModelEvaluationPresetGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	preset, err := c.AgentPlatform().GetModelEvaluationPreset(c.Args[0])
	if err != nil {
		return err
	}

	return c.Display(&displayers.ModelEvaluationPreset{Presets: do.ModelEvaluationPresets{*preset}})
}

// RunModelEvaluationPresetDelete deletes a model evaluation preset by its UUID.
func RunModelEvaluationPresetDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirmDelete("model evaluation preset", 1) != nil {
		return errOperationAborted
	}

	if err := c.AgentPlatform().DeleteModelEvaluationPreset(c.Args[0]); err != nil {
		return err
	}

	notice("Model evaluation preset deleted successfully")
	return nil
}
