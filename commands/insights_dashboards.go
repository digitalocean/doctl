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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const insightsDashboardsMaxPerPage = 200

func insightsDashboards() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "dashboard",
			Aliases: []string{"dashboards"},
			Short:   "Display commands for managing Insights dashboards",
			Long: "The commands under `doctl insights dashboard` create, view, update, and delete Insights dashboards.\n\n" +
				"A dashboard is defined by a JSON document. Retrieve the JSON Schema it must satisfy with `doctl insights dashboard schema`. " +
				"The dashboard's name and description are taken from `spec.display.name` and `spec.display.description` in the document.",
		},
	}

	list := CmdBuilder(cmd, RunInsightsDashboardList, "list", "List dashboards",
		"Lists the team's Insights dashboards. Without `--sort`, dashboards are ordered newest first.",
		Writer, aliasOpt("ls"), displayerType(&displayers.InsightsDashboards{}))
	list.Args = cobra.NoArgs
	AddIntFlag(list, doctl.ArgInsightsPage, "", 0, "Page number to retrieve. The server defaults to 1")
	AddIntFlag(list, doctl.ArgInsightsPerPage, "", 0, fmt.Sprintf("Number of dashboards per page, from 1 to %d. The server defaults to 20", insightsDashboardsMaxPerPage))
	AddStringFlag(list, doctl.ArgInsightsDashboardSort, "", "", "Sort field. Possible values: name, created_at, or updated_at. When omitted, dashboards are ordered by created_at descending")
	AddStringFlag(list, doctl.ArgInsightsDashboardSortDirection, "", "", "Sort direction. Possible values: asc or desc. Requires --sort; defaults to asc when --sort is set")
	list.Example = "The following example lists dashboards by name: doctl insights dashboard list --sort name --sort-direction asc"

	get := CmdBuilder(cmd, RunInsightsDashboardGet, "get <dashboard-uuid>", "Retrieve a dashboard",
		"Retrieves an Insights dashboard, including its full spec in JSON output.",
		Writer, aliasOpt("g"), displayerType(&displayers.InsightsDashboard{}))
	get.Args = cobra.ExactArgs(1)
	get.Example = "The following example saves a dashboard's spec to a file: doctl insights dashboard get 7c9e6679-7425-40de-944b-e07fc1f90ae7 --output json | jq .dashboard.spec > dashboard.json"

	create := CmdBuilder(cmd, RunInsightsDashboardCreate, "create", "Create a dashboard",
		"Creates an Insights dashboard from a spec document. The document's `spec.display.name` must be unique within the team.",
		Writer, aliasOpt("c"), displayerType(&displayers.InsightsDashboard{}))
	create.Args = cobra.NoArgs
	AddStringFlag(create, doctl.ArgInsightsDashboardSpecFile, "", "", "Path to a dashboard spec in JSON or YAML format. Set to - to read from stdin", requiredOpt())
	create.Example = "The following example creates a dashboard from a file: doctl insights dashboard create --spec-file dashboard.json"

	update := CmdBuilder(cmd, RunInsightsDashboardUpdate, "update <dashboard-uuid>", "Update a dashboard",
		"Replaces an Insights dashboard's spec with the provided document. The name and description are recomputed from the new spec.",
		Writer, aliasOpt("u"), displayerType(&displayers.InsightsDashboard{}))
	update.Args = cobra.ExactArgs(1)
	AddStringFlag(update, doctl.ArgInsightsDashboardSpecFile, "", "", "Path to a dashboard spec in JSON or YAML format. Set to - to read from stdin", requiredOpt())
	update.Example = "The following example replaces a dashboard's spec: doctl insights dashboard update 7c9e6679-7425-40de-944b-e07fc1f90ae7 --spec-file dashboard.json"

	deleteCmd := CmdBuilder(cmd, RunInsightsDashboardDelete, "delete <dashboard-uuid>", "Delete a dashboard",
		"Deletes an Insights dashboard.", Writer, aliasOpt("rm"))
	deleteCmd.Args = cobra.ExactArgs(1)
	AddBoolFlag(deleteCmd, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the dashboard without a confirmation prompt")
	deleteCmd.Example = "The following example deletes a dashboard without prompting: doctl insights dashboard delete 7c9e6679-7425-40de-944b-e07fc1f90ae7 --force"

	schema := CmdBuilder(cmd, RunInsightsDashboardSchema, "schema", "Retrieve the dashboard spec schema",
		"Prints the JSON Schema that a dashboard spec document must satisfy.", Writer)
	schema.Args = cobra.NoArgs
	schema.Example = "The following example saves the schema to a file: doctl insights dashboard schema > dashboard.schema.json"

	return cmd
}

// RunInsightsDashboardList lists dashboards.
func RunInsightsDashboardList(c *CmdConfig) error {
	page, perPage, err := insightsPageOptions(c, insightsDashboardsMaxPerPage)
	if err != nil {
		return err
	}

	sortField, err := insightsTrimmedString(c, doctl.ArgInsightsDashboardSort)
	if err != nil {
		return err
	}
	sortField = strings.ToLower(sortField)
	if sortField != "" &&
		sortField != godo.InsightsDashboardSortName &&
		sortField != godo.InsightsDashboardSortCreatedAt &&
		sortField != godo.InsightsDashboardSortUpdatedAt {
		return fmt.Errorf("invalid --%s %q: must be %q, %q, or %q", doctl.ArgInsightsDashboardSort, sortField,
			godo.InsightsDashboardSortName, godo.InsightsDashboardSortCreatedAt, godo.InsightsDashboardSortUpdatedAt)
	}
	sortDirection, err := insightsTrimmedString(c, doctl.ArgInsightsDashboardSortDirection)
	if err != nil {
		return err
	}
	sortDirection = strings.ToLower(sortDirection)
	if sortDirection != "" &&
		sortDirection != godo.InsightsDashboardSortAscending &&
		sortDirection != godo.InsightsDashboardSortDescending {
		return fmt.Errorf("invalid --%s %q: must be %q or %q", doctl.ArgInsightsDashboardSortDirection, sortDirection,
			godo.InsightsDashboardSortAscending, godo.InsightsDashboardSortDescending)
	}
	// The server ignores sort_direction unless sort is set.
	if sortDirection != "" && sortField == "" {
		return fmt.Errorf("--%s requires --%s", doctl.ArgInsightsDashboardSortDirection, doctl.ArgInsightsDashboardSort)
	}

	response, err := c.Insights().ListDashboards(&godo.DashboardListOptions{
		Page:          page,
		PerPage:       perPage,
		Sort:          sortField,
		SortDirection: sortDirection,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.InsightsDashboards{Response: response})
}

// RunInsightsDashboardGet retrieves a dashboard.
func RunInsightsDashboardGet(c *CmdConfig) error {
	uuid, err := insightsDashboardUUID(c)
	if err != nil {
		return err
	}
	dashboard, err := c.Insights().GetDashboard(uuid)
	if err != nil {
		return err
	}
	return c.Display(&displayers.InsightsDashboard{Dashboard: dashboard})
}

// RunInsightsDashboardCreate creates a dashboard from a spec document.
func RunInsightsDashboardCreate(c *CmdConfig) error {
	spec, err := insightsDashboardSpec(c)
	if err != nil {
		return err
	}
	dashboard, err := c.Insights().CreateDashboard(&godo.DashboardRequest{Spec: spec})
	if err != nil {
		return err
	}
	return c.Display(&displayers.InsightsDashboard{Dashboard: dashboard})
}

// RunInsightsDashboardUpdate replaces a dashboard's spec.
func RunInsightsDashboardUpdate(c *CmdConfig) error {
	uuid, err := insightsDashboardUUID(c)
	if err != nil {
		return err
	}
	spec, err := insightsDashboardSpec(c)
	if err != nil {
		return err
	}
	dashboard, err := c.Insights().UpdateDashboard(uuid, &godo.DashboardRequest{Spec: spec})
	if err != nil {
		return err
	}
	return c.Display(&displayers.InsightsDashboard{Dashboard: dashboard})
}

// RunInsightsDashboardDelete deletes a dashboard.
func RunInsightsDashboardDelete(c *CmdConfig) error {
	uuid, err := insightsDashboardUUID(c)
	if err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if err := confirmDelete(force, "Insights dashboard", 1); err != nil {
		return err
	}
	return c.Insights().DeleteDashboard(uuid)
}

// RunInsightsDashboardSchema prints the JSON Schema for dashboard specs.
func RunInsightsDashboardSchema(c *CmdConfig) error {
	if len(c.Args) > 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	schema, err := c.Insights().GetDashboardSchema()
	if err != nil {
		return err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, schema, "", "  "); err != nil {
		return err
	}
	out.WriteByte('\n')
	_, err = c.Out.Write(out.Bytes())
	return err
}

func insightsDashboardUUID(c *CmdConfig) (string, error) {
	if err := ensureOneArg(c); err != nil {
		return "", err
	}
	uuid := strings.TrimSpace(c.Args[0])
	if uuid == "" {
		return "", fmt.Errorf("a dashboard UUID is required")
	}
	return uuid, nil
}

func insightsDashboardSpec(c *CmdConfig) (json.RawMessage, error) {
	path, err := insightsTrimmedString(c, doctl.ArgInsightsDashboardSpecFile)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("--%s is required", doctl.ArgInsightsDashboardSpecFile)
	}
	spec, err := readInsightsDashboardSpec(insightsStdin, path)
	if err != nil {
		return nil, fmt.Errorf("--%s: %w", doctl.ArgInsightsDashboardSpecFile, err)
	}
	return spec, nil
}
