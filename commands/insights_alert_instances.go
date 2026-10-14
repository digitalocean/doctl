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
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func insightsAlertInstances() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "alert-instance",
			Aliases: []string{"alert-instances", "ai"},
			Short:   "Display commands for viewing Insights alert instances",
			Long: `The commands under ` + "`" + `doctl insights alert-instance` + "`" + ` retrieve alert instances.

An alert instance is a read-only record of an alert rule firing against a resource. Instances are created and resolved by Insights; you cannot create or modify them.`,
		},
	}

	cmdList := CmdBuilder(cmd, RunInsightsAlertInstanceList, "list", "List alert instances",
		`Lists alert instances in your account, newest trigger first. By default, all pages are retrieved. Use `+"`"+`--page`+"`"+` to retrieve a single page.`,
		Writer, aliasOpt("ls"), displayerType(&displayers.InsightsAlertInstances{}))
	AddStringFlag(cmdList, doctl.ArgInsightsAlertInstanceStatus, "", "", "Filter by status. Possible values: active or resolved")
	AddStringFlag(cmdList, doctl.ArgInsightsAlertRuleID, "", "", "Filter by the ID of the alert rule that fired")
	AddStringFlag(cmdList, doctl.ArgInsightsAlertInstanceResourceURN, "", "", "Filter by the `urn` of the resource the alert fired against, for example do:droplet:12345")
	AddIntFlag(cmdList, doctl.ArgInsightsAlertInstancePage, "", 0, "Retrieve only this page of results (1-based)")
	AddIntFlag(cmdList, doctl.ArgInsightsAlertInstancePerPage, "", 0, "Number of results per page when --page is set")
	cmdList.Example = `The following example lists active alert instances for an alert rule: doctl insights alert-instance list --status active --rule-id 4b6f2a7e-1c2d-4e5f-8a9b-0c1d2e3f4a5b`

	cmdGet := CmdBuilder(cmd, RunInsightsAlertInstanceGet, "get <alert-instance-id>", "Retrieve an alert instance",
		`Retrieves a single alert instance by ID.`,
		Writer, aliasOpt("g"), displayerType(&displayers.InsightsAlertInstances{}))
	cmdGet.Example = `The following example retrieves an alert instance: doctl insights alert-instance get 9c1f8e2a-3b4d-4c5e-9f6a-7b8c9d0e1f2a`

	return cmd
}

// RunInsightsAlertInstanceList lists alert instances.
func RunInsightsAlertInstanceList(c *CmdConfig) error {
	status, err := c.Doit.GetString(c.NS, doctl.ArgInsightsAlertInstanceStatus)
	if err != nil {
		return err
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "" && status != godo.InsightsAlertInstanceStatusActive && status != godo.InsightsAlertInstanceStatusResolved {
		return fmt.Errorf("invalid --%s %q: must be %q or %q", doctl.ArgInsightsAlertInstanceStatus, status,
			godo.InsightsAlertInstanceStatusActive, godo.InsightsAlertInstanceStatusResolved)
	}

	ruleID, err := c.Doit.GetString(c.NS, doctl.ArgInsightsAlertRuleID)
	if err != nil {
		return err
	}
	resourceURN, err := c.Doit.GetString(c.NS, doctl.ArgInsightsAlertInstanceResourceURN)
	if err != nil {
		return err
	}
	page, err := c.Doit.GetInt(c.NS, doctl.ArgInsightsAlertInstancePage)
	if err != nil {
		return err
	}
	perPage, err := c.Doit.GetInt(c.NS, doctl.ArgInsightsAlertInstancePerPage)
	if err != nil {
		return err
	}
	if page < 0 || perPage < 0 {
		return fmt.Errorf("--%s and --%s must not be negative", doctl.ArgInsightsAlertInstancePage, doctl.ArgInsightsAlertInstancePerPage)
	}

	opt := godo.AlertInstanceListOptions{
		Status:      status,
		RuleID:      strings.TrimSpace(ruleID),
		ResourceURN: strings.TrimSpace(resourceURN),
	}
	is := c.Insights()

	if page > 0 {
		opt.Page = page
		opt.PerPage = perPage
		instances, _, err := is.ListAlertInstances(&opt)
		if err != nil {
			return err
		}
		return c.Display(&displayers.InsightsAlertInstances{AlertInstances: instances})
	}

	f := func(lo *godo.ListOptions) ([]any, *godo.Response, error) {
		pageOpt := opt
		pageOpt.Page = lo.Page
		pageOpt.PerPage = lo.PerPage
		list, resp, err := is.ListAlertInstances(&pageOpt)
		if err != nil {
			return nil, nil, err
		}
		items := make([]any, len(list))
		for i := range list {
			items[i] = list[i]
		}
		return items, resp, nil
	}

	si, err := do.PaginateResp(f)
	if err != nil {
		return err
	}
	instances := make([]godo.AlertInstance, len(si))
	for i := range si {
		instances[i] = si[i].(godo.AlertInstance)
	}

	return c.Display(&displayers.InsightsAlertInstances{AlertInstances: instances})
}

// RunInsightsAlertInstanceGet retrieves an alert instance.
func RunInsightsAlertInstanceGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	id := strings.TrimSpace(c.Args[0])
	if id == "" {
		return fmt.Errorf("an alert instance ID is required")
	}

	instance, err := c.Insights().GetAlertInstance(id)
	if err != nil {
		return err
	}

	return c.Display(&displayers.InsightsAlertInstances{AlertInstances: []godo.AlertInstance{*instance}})
}
