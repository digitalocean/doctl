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
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const insightsResourcesMaxPerPage = 200

func insightsResources() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "resource",
			Aliases: []string{"resources"},
			Short:   "Display commands for managing Insights resource signal enablement",
			Long:    "The commands under `doctl insights resource` view and update metrics, logs, and traces enablement for resources.",
		},
	}

	list := CmdBuilder(cmd, RunInsightsResourceList, "list", "List resource signal enablement",
		"Lists resources with recorded Insights signal enablement.",
		Writer, aliasOpt("ls"), displayerType(&displayers.InsightsResources{}))
	list.Args = cobra.NoArgs
	AddStringArrayFlag(list, doctl.ArgInsightsResourceProductType, "", nil, "Filter by product type; repeat to include multiple product types")
	AddStringArrayFlag(list, doctl.ArgInsightsResourceRegion, "", nil, "Filter by region; repeat to include multiple regions")
	AddIntFlag(list, doctl.ArgInsightsPage, "", 0, "Page number to retrieve. The server defaults to 1")
	AddIntFlag(list, doctl.ArgInsightsPerPage, "", 0, fmt.Sprintf("Number of resources per page, from 1 to %d. The server defaults to 20", insightsResourcesMaxPerPage))
	AddStringFlag(list, doctl.ArgInsightsResourceSort, "", "", "Sort field. Possible values: resource_urn or effective_at")
	AddStringFlag(list, doctl.ArgInsightsResourceSortDirection, "", "", "Sort direction. Possible values: asc or desc")
	list.Example = "The following example lists Droplet resources in nyc3: doctl insights resource list --product-type droplet --region nyc3"

	get := CmdBuilder(cmd, RunInsightsResourceGet, "get <resource-urn>", "Retrieve resource signal enablement",
		"Retrieves the current Insights signal enablement for a resource.",
		Writer, aliasOpt("g"), displayerType(&displayers.InsightsResource{}))
	get.Args = cobra.ExactArgs(1)
	get.Example = "The following example retrieves signal enablement for a Droplet: doctl insights resource get do:droplet:12345"

	update := CmdBuilder(cmd, RunInsightsResourceUpdate, "update <resource-urn>", "Update resource signal enablement",
		"Enables or disables metrics, logs, or traces for a resource. Omitted signals remain unchanged. Set at least one signal.",
		Writer, aliasOpt("u"), displayerType(&displayers.InsightsResource{}))
	update.Args = cobra.ExactArgs(1)
	AddStringFlag(update, doctl.ArgInsightsResourceRegion, "", "", "Region used when signal enablement is first created; it cannot be changed later")
	AddBoolFlag(update, doctl.ArgInsightsResourceMetrics, "", false, "Enable or disable metrics; use --metrics=true or --metrics=false")
	AddBoolFlag(update, doctl.ArgInsightsResourceLogs, "", false, "Enable or disable logs; use --logs=true or --logs=false")
	AddBoolFlag(update, doctl.ArgInsightsResourceTraces, "", false, "Enable or disable traces; use --traces=true or --traces=false")
	update.Example = "The following example enables metrics and logs while disabling traces: doctl insights resource update do:droplet:12345 --region nyc3 --metrics=true --logs=true --traces=false"

	return cmd
}

// RunInsightsResourceList lists resource signal enablement.
func RunInsightsResourceList(c *CmdConfig) error {
	productTypes, err := getInsightsStringArray(c, doctl.ArgInsightsResourceProductType)
	if err != nil {
		return err
	}
	productTypes, err = normalizeInsightsResourceFilters(productTypes, doctl.ArgInsightsResourceProductType)
	if err != nil {
		return err
	}
	regions, err := getInsightsStringArray(c, doctl.ArgInsightsResourceRegion)
	if err != nil {
		return err
	}
	regions, err = normalizeInsightsResourceFilters(regions, doctl.ArgInsightsResourceRegion)
	if err != nil {
		return err
	}

	page, err := c.Doit.GetInt(c.NS, doctl.ArgInsightsPage)
	if err != nil {
		return err
	}
	if page < 0 || (page == 0 && c.Doit.IsSet(doctl.ArgInsightsPage)) {
		return fmt.Errorf("--%s must be at least 1", doctl.ArgInsightsPage)
	}
	perPage, err := c.Doit.GetInt(c.NS, doctl.ArgInsightsPerPage)
	if err != nil {
		return err
	}
	if perPage < 0 || perPage > insightsResourcesMaxPerPage ||
		(perPage == 0 && c.Doit.IsSet(doctl.ArgInsightsPerPage)) {
		return fmt.Errorf("--%s must be between 1 and %d", doctl.ArgInsightsPerPage, insightsResourcesMaxPerPage)
	}

	sortField, err := insightsTrimmedString(c, doctl.ArgInsightsResourceSort)
	if err != nil {
		return err
	}
	sortField = strings.ToLower(sortField)
	if sortField != "" &&
		sortField != godo.InsightsResourceSortResourceURN &&
		sortField != godo.InsightsResourceSortEffectiveAt {
		return fmt.Errorf("invalid --%s %q: must be %q or %q", doctl.ArgInsightsResourceSort, sortField,
			godo.InsightsResourceSortResourceURN, godo.InsightsResourceSortEffectiveAt)
	}
	sortDirection, err := insightsTrimmedString(c, doctl.ArgInsightsResourceSortDirection)
	if err != nil {
		return err
	}
	sortDirection = strings.ToLower(sortDirection)
	if sortDirection != "" &&
		sortDirection != godo.InsightsResourceSortAscending &&
		sortDirection != godo.InsightsResourceSortDescending {
		return fmt.Errorf("invalid --%s %q: must be %q or %q", doctl.ArgInsightsResourceSortDirection, sortDirection,
			godo.InsightsResourceSortAscending, godo.InsightsResourceSortDescending)
	}

	response, err := c.Insights().ListResources(&godo.InsightsResourceListOptions{
		ProductTypes:  productTypes,
		Regions:       regions,
		Page:          page,
		PerPage:       perPage,
		Sort:          sortField,
		SortDirection: sortDirection,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.InsightsResources{Response: response})
}

// RunInsightsResourceGet retrieves signal enablement for a resource.
func RunInsightsResourceGet(c *CmdConfig) error {
	resourceURN, err := insightsResourceURN(c)
	if err != nil {
		return err
	}
	resource, err := c.Insights().GetResource(resourceURN)
	if err != nil {
		return err
	}
	return c.Display(&displayers.InsightsResource{Resource: resource})
}

// RunInsightsResourceUpdate changes signal enablement for a resource.
func RunInsightsResourceUpdate(c *CmdConfig) error {
	resourceURN, err := insightsResourceURN(c)
	if err != nil {
		return err
	}
	region, err := insightsTrimmedString(c, doctl.ArgInsightsResourceRegion)
	if err != nil {
		return err
	}

	request := &godo.InsightsResourceSignalsRequest{Region: region}
	signalSet := false
	for _, signal := range []struct {
		flag   string
		target **bool
	}{
		{flag: doctl.ArgInsightsResourceMetrics, target: &request.Metrics},
		{flag: doctl.ArgInsightsResourceLogs, target: &request.Logs},
		{flag: doctl.ArgInsightsResourceTraces, target: &request.Traces},
	} {
		if !c.Doit.IsSet(signal.flag) {
			continue
		}
		value, err := c.Doit.GetBool(c.NS, signal.flag)
		if err != nil {
			return err
		}
		*signal.target = godo.Bool(value)
		signalSet = true
	}
	if !signalSet {
		return fmt.Errorf("at least one of --%s, --%s, or --%s must be set",
			doctl.ArgInsightsResourceMetrics, doctl.ArgInsightsResourceLogs, doctl.ArgInsightsResourceTraces)
	}

	resource, err := c.Insights().UpdateResourceSignals(resourceURN, request)
	if err != nil {
		return err
	}
	return c.Display(&displayers.InsightsResource{Resource: resource})
}

func insightsResourceURN(c *CmdConfig) (string, error) {
	if err := ensureOneArg(c); err != nil {
		return "", err
	}
	resourceURN := strings.TrimSpace(c.Args[0])
	if resourceURN == "" {
		return "", fmt.Errorf("a resource URN is required")
	}
	return resourceURN, nil
}

func normalizeInsightsResourceFilters(values []string, flag string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	normalized := make([]string, len(values))
	for i, value := range values {
		normalized[i] = strings.TrimSpace(value)
		if normalized[i] == "" {
			return nil, fmt.Errorf("--%s values must not be empty", flag)
		}
	}
	return normalized, nil
}
