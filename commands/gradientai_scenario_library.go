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
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// ScenarioLibraryCmd handles operations on the platform scenario library.
func ScenarioLibraryCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "scenario-library",
			Aliases: []string{"sl"},
			Short:   "Display commands that browse the Gradient AI scenario library.",
			Long:    "The subcommands of `doctl gradient scenario-library` browse the curated scenarios that DigitalOcean publishes and copy them into your own scenario sets.",
		},
	}

	libraryDetails := `
		- The library scenario UUID
		- The library entry name
		- The library entry category
		- The library entry status
		- The number of scenarios in the entry
		- The library entry creation timestamp
	`

	cmdScenarioLibraryList := CmdBuilder(
		cmd,
		RunScenarioLibraryList,
		"list",
		"List the scenario library",
		"Retrieves the curated scenario library, where each entry contains:"+libraryDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.ScenarioLibraryEntry{}),
	)
	AddStringFlag(cmdScenarioLibraryList, doctl.ArgScenarioLibraryCategory, "", "", "Filters the results by category.")
	addGenAIListFlags(cmdScenarioLibraryList, "`name`, `created_at`")
	cmdScenarioLibraryList.Example = "The following example lists the scenario library: " +
		"`doctl gradient scenario-library list`"

	cmdScenarioLibraryListScenarios := CmdBuilder(
		cmd,
		RunScenarioLibraryListScenarios,
		"list-scenarios <library-scenario-uuid>",
		"List the scenarios in a library entry",
		"Retrieves the individual scenarios that belong to a scenario library entry.",
		Writer, aliasOpt("ls-s"),
		displayerType(&displayers.Scenario{}),
	)
	addGenAIListFlags(cmdScenarioLibraryListScenarios, "`file_order`, `name`, `description`")
	cmdScenarioLibraryListScenarios.Example = "The following example lists the scenarios in a library entry: " +
		"`doctl gradient scenario-library list-scenarios f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdScenarioLibraryCreateScenarioSet := CmdBuilder(
		cmd,
		RunScenarioLibraryCreateScenarioSet,
		"create-scenario-set <library-scenario-uuid>",
		"Create a scenario set from a library entry",
		"Copies a scenario library entry into a scenario set that your team owns, which you can then run simulations against.",
		Writer, aliasOpt("c-ss"),
		displayerType(&displayers.ScenarioSet{}),
	)
	AddStringFlag(cmdScenarioLibraryCreateScenarioSet, doctl.ArgGenAIName, "", "", "The name of the new scenario set.")
	cmdScenarioLibraryCreateScenarioSet.Example = "The following example copies a library entry into a scenario set: " +
		"`doctl gradient scenario-library create-scenario-set f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name support-flows`"

	return cmd
}

// RunScenarioLibraryList lists the curated scenario library.
func RunScenarioLibraryList(c *CmdConfig) error {
	category, err := c.Doit.GetString(c.NS, doctl.ArgScenarioLibraryCategory)
	if err != nil {
		return err
	}

	search, sortBy, sortDirection, err := genAIListFilters(c, scenarioLibrarySortFieldPrefix)
	if err != nil {
		return err
	}

	entries, err := c.GradientAI().ListScenarioLibrary(&godo.ScenarioLibraryListOptions{
		Category:      category,
		Search:        search,
		SortBy:        godo.ScenarioLibrarySortField(sortBy),
		SortDirection: godo.GenAISortDirection(sortDirection),
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.ScenarioLibraryEntry{Entries: entries})
}

// RunScenarioLibraryListScenarios lists the scenarios in a library entry.
func RunScenarioLibraryListScenarios(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	opt, err := scenarioListOptions(c)
	if err != nil {
		return err
	}

	scenarios, err := c.GradientAI().ListScenarioLibraryScenarios(c.Args[0], opt)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Scenario{Scenarios: scenarios})
}

// RunScenarioLibraryCreateScenarioSet copies a library entry into a scenario set.
func RunScenarioLibraryCreateScenarioSet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	name, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	libraryScenarioUUID := c.Args[0]
	scenarioSet, err := c.GradientAI().CreateScenarioSetFromLibrary(libraryScenarioUUID, &godo.CreateScenarioSetFromLibraryRequest{
		LibraryScenarioUUID: libraryScenarioUUID,
		Name:                name,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.ScenarioSet{ScenarioSets: do.ScenarioSets{*scenarioSet}})
}
