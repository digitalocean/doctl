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

// ScenarioSetCmd handles operations on simulation scenario sets.
func ScenarioSetCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "scenario-set",
			Aliases: []string{"ss", "scenario-sets"},
			Short:   "Display commands that manage Gradient AI scenario sets.",
			Long:    "The subcommands of `doctl gradient scenario-set` manage the scenario sets that Gradient AI simulations run against.",
		},
	}

	scenarioSetDetails := `
		- The scenario set UUID
		- The scenario set name
		- The scenario set status
		- The scenario set source kind
		- The number of scenarios in the set
		- The scenario set creation and last-updated timestamps
	`

	cmdScenarioSetCreate := CmdBuilder(
		cmd,
		RunScenarioSetCreate,
		"create",
		"Create a scenario set",
		"Creates a scenario set from a local file or from inline scenarios and returns:"+scenarioSetDetails,
		Writer, aliasOpt("c"),
		displayerType(&displayers.ScenarioSet{}),
	)
	AddStringFlag(cmdScenarioSetCreate, doctl.ArgGenAIName, "", "", "The name of the scenario set.", requiredOpt())
	AddStringFlag(cmdScenarioSetCreate, doctl.ArgScenarioSetFile, "", "", "The path to a local scenario file to upload. Mutually exclusive with `--scenarios`.")
	AddStringFlag(cmdScenarioSetCreate, doctl.ArgScenarioSetScenarios, "", "", "A JSON array of scenario objects. Mutually exclusive with `--file`.")
	cmdScenarioSetCreate.Example = "The following example creates a scenario set from a local file: " +
		"`doctl gradient scenario-set create --name support-flows --file ./scenarios.jsonl`"

	cmdScenarioSetGenerate := CmdBuilder(
		cmd,
		RunScenarioSetGenerate,
		"generate",
		"Generate a scenario set from a goal",
		"Generates scenarios from a goal description. Generation runs asynchronously, so the returned scenario set starts in the `GENERATING` status and returns:"+scenarioSetDetails,
		Writer, aliasOpt("gen"),
		displayerType(&displayers.ScenarioSet{}),
	)
	AddStringFlag(cmdScenarioSetGenerate, doctl.ArgGenAIName, "", "", "The name of the scenario set.", requiredOpt())
	AddStringFlag(cmdScenarioSetGenerate, doctl.ArgScenarioSetGoalDescription, "", "", "A description of what the generated scenarios should cover.", requiredOpt())
	AddIntFlag(cmdScenarioSetGenerate, doctl.ArgScenarioSetNumScenarios, "", 0, "The number of scenarios to generate.")
	AddStringFlag(cmdScenarioSetGenerate, doctl.ArgScenarioSetGeneratorModelUUID, "", "", "The UUID of the model used to generate the scenarios.")
	cmdScenarioSetGenerate.Example = "The following example generates ten scenarios from a goal: " +
		"`doctl gradient scenario-set generate --name refund-flows --goal-description \"Customers asking for refunds\" --num-scenarios 10`"

	cmdScenarioSetList := CmdBuilder(
		cmd,
		RunScenarioSetList,
		"list",
		"List all scenario sets",
		"Retrieves a list of scenario sets, where each scenario set contains:"+scenarioSetDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.ScenarioSet{}),
	)
	AddStringSliceFlag(cmdScenarioSetList, doctl.ArgGenAIStatuses, "", []string{}, "Filters the results by status. One of: `generating`, `ready`, `failed`, `cancelled`")
	AddStringSliceFlag(cmdScenarioSetList, doctl.ArgScenarioSetSourceKinds, "", []string{}, "Filters the results by source kind. One of: `user_upload`, `goal_generated`, `library`, `signal_generated`")
	addGenAIListFlags(cmdScenarioSetList, "`created_at`, `name`, `status`, `scenario_count`, `updated_at`")
	cmdScenarioSetList.Example = "The following example lists every scenario set that is ready to run: " +
		"`doctl gradient scenario-set list --statuses ready`"

	cmdScenarioSetGet := CmdBuilder(
		cmd,
		RunScenarioSetGet,
		"get <scenario-set-uuid>",
		"Retrieve a scenario set",
		"Retrieves information about a scenario set, including:"+scenarioSetDetails,
		Writer, aliasOpt("g"),
		displayerType(&displayers.ScenarioSet{}),
	)
	cmdScenarioSetGet.Example = "The following example retrieves a scenario set: " +
		"`doctl gradient scenario-set get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdScenarioSetListScenarios := CmdBuilder(
		cmd,
		RunScenarioSetListScenarios,
		"list-scenarios <scenario-set-uuid>",
		"List the scenarios in a scenario set",
		"Retrieves the individual scenarios that belong to a scenario set.",
		Writer, aliasOpt("ls-s"),
		displayerType(&displayers.Scenario{}),
	)
	addGenAIListFlags(cmdScenarioSetListScenarios, "`file_order`, `name`, `description`")
	cmdScenarioSetListScenarios.Example = "The following example lists the scenarios in a scenario set: " +
		"`doctl gradient scenario-set list-scenarios f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdScenarioSetUpdate := CmdBuilder(
		cmd,
		RunScenarioSetUpdate,
		"update <scenario-set-uuid>",
		"Update a scenario set",
		"Renames a scenario set and, when `--scenarios` is provided, replaces its scenarios.",
		Writer, aliasOpt("u"),
		displayerType(&displayers.ScenarioSet{}),
	)
	AddStringFlag(cmdScenarioSetUpdate, doctl.ArgGenAIName, "", "", "The new name of the scenario set.")
	AddStringFlag(cmdScenarioSetUpdate, doctl.ArgScenarioSetScenarios, "", "", "A JSON array of scenario objects that replaces the existing scenarios.")
	cmdScenarioSetUpdate.Example = "The following example renames a scenario set: " +
		"`doctl gradient scenario-set update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name support-flows-v2`"

	cmdScenarioSetDelete := CmdBuilder(
		cmd,
		RunScenarioSetDelete,
		"delete <scenario-set-uuid>",
		"Delete a scenario set",
		"Deletes a scenario set by its UUID.",
		Writer, aliasOpt("del", "rm"),
	)
	AddBoolFlag(cmdScenarioSetDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the scenario set without a confirmation prompt")
	cmdScenarioSetDelete.Example = "The following example deletes a scenario set: " +
		"`doctl gradient scenario-set delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdScenarioSetDownloadURL := CmdBuilder(
		cmd,
		RunScenarioSetDownloadURL,
		"download-url <scenario-set-uuid>",
		"Retrieve a download URL for a scenario set",
		"Retrieves a temporary presigned URL for downloading the file that backs a scenario set.",
		Writer, aliasOpt("dl-url"),
		displayerType(&displayers.GenAIDownloadURL{}),
	)
	cmdScenarioSetDownloadURL.Example = "The following example retrieves a download URL for a scenario set: " +
		"`doctl gradient scenario-set download-url f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	return cmd
}

// RunScenarioSetCreate creates a scenario set.
func RunScenarioSetCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	path, err := c.Doit.GetString(c.NS, doctl.ArgScenarioSetFile)
	if err != nil {
		return err
	}

	rawScenarios, err := c.Doit.GetString(c.NS, doctl.ArgScenarioSetScenarios)
	if err != nil {
		return err
	}

	if path == "" && rawScenarios == "" {
		return fmt.Errorf("either --%s or --%s is required", doctl.ArgScenarioSetFile, doctl.ArgScenarioSetScenarios)
	}
	if path != "" && rawScenarios != "" {
		return fmt.Errorf("--%s and --%s cannot be used together", doctl.ArgScenarioSetFile, doctl.ArgScenarioSetScenarios)
	}

	req := &godo.CreateScenarioSetRequest{Name: name}

	if path != "" {
		dataSource, err := uploadScenarioSetFile(c, path)
		if err != nil {
			return err
		}
		req.FileUploadScenarioSet = dataSource
	} else {
		scenarios, err := parseScenarios(rawScenarios)
		if err != nil {
			return err
		}
		req.Scenarios = scenarios
	}

	scenarioSet, err := c.GradientAI().CreateScenarioSet(req)
	if err != nil {
		return err
	}

	return c.Display(&displayers.ScenarioSet{ScenarioSets: do.ScenarioSets{*scenarioSet}})
}

// RunScenarioSetGenerate generates a scenario set from a goal description.
func RunScenarioSetGenerate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	goalDescription, err := c.Doit.GetString(c.NS, doctl.ArgScenarioSetGoalDescription)
	if err != nil {
		return err
	}

	numScenarios, err := c.Doit.GetInt(c.NS, doctl.ArgScenarioSetNumScenarios)
	if err != nil {
		return err
	}

	generatorModelUUID, err := c.Doit.GetString(c.NS, doctl.ArgScenarioSetGeneratorModelUUID)
	if err != nil {
		return err
	}

	scenarioSet, err := c.GradientAI().GenerateScenarioSet(&godo.GenerateScenarioSetRequest{
		Name:               name,
		GoalDescription:    goalDescription,
		NumScenarios:       uint32(numScenarios),
		GeneratorModelUUID: generatorModelUUID,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.ScenarioSet{ScenarioSets: do.ScenarioSets{*scenarioSet}})
}

// RunScenarioSetList lists scenario sets.
func RunScenarioSetList(c *CmdConfig) error {
	rawStatuses, err := c.Doit.GetStringSlice(c.NS, doctl.ArgGenAIStatuses)
	if err != nil {
		return err
	}

	rawSourceKinds, err := c.Doit.GetStringSlice(c.NS, doctl.ArgScenarioSetSourceKinds)
	if err != nil {
		return err
	}

	search, sortBy, sortDirection, err := genAIListFilters(c, scenarioSetSortFieldPrefix)
	if err != nil {
		return err
	}

	scenarioSets, err := c.GradientAI().ListScenarioSets(&godo.ScenarioSetListOptions{
		Statuses:      genAIEnums[godo.ScenarioSetStatus](scenarioSetStatusPrefix, rawStatuses),
		SourceKinds:   genAIEnums[godo.ScenarioSetSourceKind](scenarioSetSourceKindPrefix, rawSourceKinds),
		Search:        search,
		SortBy:        godo.ScenarioSetSortField(sortBy),
		SortDirection: godo.GenAISortDirection(sortDirection),
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.ScenarioSet{ScenarioSets: scenarioSets})
}

// RunScenarioSetGet retrieves a scenario set by its UUID.
func RunScenarioSetGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	scenarioSet, err := c.GradientAI().GetScenarioSet(c.Args[0])
	if err != nil {
		return err
	}

	return c.Display(&displayers.ScenarioSet{ScenarioSets: do.ScenarioSets{*scenarioSet}})
}

// RunScenarioSetListScenarios lists the scenarios in a scenario set.
func RunScenarioSetListScenarios(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	opt, err := scenarioListOptions(c)
	if err != nil {
		return err
	}

	scenarios, err := c.GradientAI().ListScenarios(c.Args[0], opt)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Scenario{Scenarios: scenarios})
}

// RunScenarioSetUpdate updates a scenario set.
func RunScenarioSetUpdate(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	name, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	rawScenarios, err := c.Doit.GetString(c.NS, doctl.ArgScenarioSetScenarios)
	if err != nil {
		return err
	}

	if name == "" && rawScenarios == "" {
		return fmt.Errorf("either --%s or --%s is required", doctl.ArgGenAIName, doctl.ArgScenarioSetScenarios)
	}

	scenarios, err := parseScenarios(rawScenarios)
	if err != nil {
		return err
	}

	scenarioSetUUID := c.Args[0]
	scenarioSet, err := c.GradientAI().UpdateScenarioSet(scenarioSetUUID, &godo.UpdateScenarioSetRequest{
		ScenarioSetUUID: scenarioSetUUID,
		Name:            name,
		Scenarios:       scenarios,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.ScenarioSet{ScenarioSets: do.ScenarioSets{*scenarioSet}})
}

// RunScenarioSetDelete deletes a scenario set by its UUID.
func RunScenarioSetDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirmDelete("scenario set", 1) != nil {
		return errOperationAborted
	}

	if err := c.GradientAI().DeleteScenarioSet(c.Args[0]); err != nil {
		return err
	}

	notice("Scenario set deleted successfully")
	return nil
}

// RunScenarioSetDownloadURL retrieves a presigned download URL for a scenario set.
func RunScenarioSetDownloadURL(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	downloadURL, err := c.GradientAI().GetScenarioSetDownloadURL(c.Args[0])
	if err != nil {
		return err
	}

	return c.Display(&displayers.GenAIDownloadURL{DownloadURL: downloadURL})
}
