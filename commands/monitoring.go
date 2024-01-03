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
	"errors"
	"fmt"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Monitoring creates the monitoring commands hierarchy.
func Monitoring() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "monitoring",
			Short: "[Beta] Display commands to manage monitoring",
			Long: `The sub-commands of ` + "`" + `doctl monitoring` + "`" + ` manage the monitoring on your account.

You can create alert policies to monitor the resource consumption of your Droplets, and uptime checks to monitor the availability of your websites and services`,
			GroupID: manageResourcesGroup,
		},
	}

	cmd.AddCommand(alertPolicies())
	cmd.AddCommand(UptimeCheck())
	return cmd
}

func alertPolicies() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "alert",
			Aliases: []string{"alerts", "a"},
			Short:   "Display commands for managing alert policies",
			Long: `The commands under ` + "`" + `doctl monitoring alert` + "`" + ` are for managing alert policies.

You can apply alert policies to resources in order to receive alerts on resource consumption. 
			
If you'd like to alert on the uptime of a specific URL or IP address, use ` + "`" + `doctl monitoring uptime alert` + "` instead",
		},
	}

	cmdAlertPolicyCreate := CmdBuilder(cmd, RunCmdAlertPolicyCreate, "create", "Create an alert policy", `Creates a new alert policy. You can create policies that monitor various metrics of your Droplets and send you alerts when a metric exceeds a specified threshold.
	
	For example, you can create a policy that monitors a Droplet's CPU usage and triggers an alert when the Droplet's CPU usage exceeds more than 80% for more than 10 minutes.
	
	For a full list of policy types you can set up, see our API documentation: https://docs.digitalocean.com/reference/api/api-reference/#operation/monitoring_create_alertPolicy`, Writer)
	AddStringFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyDescription, "", "", "A description of the alert policy")
	AddStringFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyType, "", "", "The type of alert policy. For example,`v1/insights/droplet/memory_utilization_percent` alerts on the percent of memory utilization. For a full list of alert types, see https://docs.digitalocean.com/reference/api/api-reference/#operation/monitoring_create_alertPolicy")
	AddStringFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyCompare, "", "", "The comparator of the alert policy. Possible values: `GreaterThan` or `LessThan`")
	AddStringFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyWindow, "", "5m", "The amount of time the resource must exceed the threshold value before an alert is triggered. Possible values: `5m`, `10m`, `30m`, or `1h`")
	AddIntFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyValue, "", 0, "The threshold value of the alert policy to compare against. For example, if the alert policy is of type `DropletCPUUtilizationPercent` and the value is set to `80`, an alert is triggered if the Droplet's CPU usage exceeds 80% for the specified window.")
	AddBoolFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyEnabled, "", true, "Enables the alert policy")
	AddStringSliceFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyEmails, "", nil, "Email address to send alerts to")
	AddStringSliceFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyTags, "", nil, "Tags to apply the alert policy to. If set to a tag, all Droplet with that tag are monitored by the alert policy.")
	AddStringSliceFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyEntities, "", nil, "Resources to apply the alert against, such as a Droplet ID.")
	AddStringSliceFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicySlackChannels, "", nil, "A Slack channel to send alerts to. For example, `production-alerts`")
	AddStringSliceFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicySlackURLs, "", nil, "A Slack webhook URL to send alerts to, for example, `https://hooks.slack.com/services/T1234567/AAAAAAAA/ZZZZZZ`.")
	cmdAlertPolicyCreate.Example = `The following example creates an alert policy that sends an email to ` + "`" + `admin@example.com` + "`" + ` whenever the memory usage on the listed Droplets (entities) exceeds 80% for more than five minutes: doctl monitoring alert create --type "v1/insights/droplet/memory_utilization_percent" --compare GreaterThan --value 80 --window 5m --entities 386734086,191669331 --emails admin@example.com`

	cmdAlertPolicyUpdate := CmdBuilder(cmd, RunCmdAlertPolicyUpdate, "update <alert-policy-uuid>...", "Update an alert policy", `Updates an existing alert policy.`, Writer)
	AddStringFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyDescription, "", "", "A description of the alert policy.")
	AddStringFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyType, "", "", "The type of alert policy. For example,`v1/insights/droplet/memory_utilization_percent` alerts on the percent of memory utilization. For a full list of alert types, see https://docs.digitalocean.com/reference/api/api-reference/#operation/monitoring_create_alertPolicy")
	AddStringFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyCompare, "", "", "The comparator of the alert policy. Either `GreaterThan` or `LessThan`")
	AddStringFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyWindow, "", "5m", "The window to apply the alert policy conditions against.")
	AddIntFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyValue, "", 0, "The value of the alert policy to compare against.")
	AddBoolFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyEnabled, "", true, "Whether the alert policy is enabled.")
	AddStringSliceFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyEmails, "", nil, "Email addresses to send alerts to")
	AddStringSliceFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyTags, "", nil, "Tags to apply the alert against")
	AddStringSliceFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyEntities, "", nil, "Resources to apply the policy to")
	AddStringSliceFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicySlackChannels, "", nil, "A Slack channel to send alerts to, for example, `production-alerts`. Must be used with `--slack-url`.")
	AddStringSliceFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicySlackURLs, "", nil, "A Slack webhook URL to send alerts to, for example, `https://hooks.slack.com/services/T1234567/AAAAAAAA/ZZZZZZ`.")
	cmdAlertPolicyUpdate.Example = `The following example updates an alert policy's details: doctl monitoring alert update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --type "v1/insights/droplet/memory_utilization_percent" --compare GreaterThan --value 80 --window 10m --entities 386734086,191669331 --emails admin@example.com`

	AlertPolicyGet := CmdBuilder(cmd, RunCmdAlertPolicyGet, "get <alert-policy-uuid>", "Retrieve information about an alert policy", `Retrieves an alert policy and its configuration.`, Writer,
		displayerType(&displayers.AlertPolicy{}))
	AlertPolicyGet.Example = `The following example retrieves information about a policy with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl monitoring alert get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdAlertPolicyList := CmdBuilder(cmd, RunCmdAlertPolicyList, "list", "List all alert policies", `Retrieves a list of all the alert policies in your account.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.AlertPolicy{}))
	cmdAlertPolicyList.Example = `The following example lists all alert policies in your account: doctl monitoring alert list`

	cmdRunAlertPolicyDelete := CmdBuilder(cmd, RunCmdAlertPolicyDelete, "delete <alert-policy-uuid>...", "Delete an alert policy", `Deletes an alert policy.`, Writer, aliasOpt("rm"))
	AddBoolFlag(cmdRunAlertPolicyDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete an alert policy without a confirmation prompt")
	cmdRunAlertPolicyDelete.Example = `The following example deletes an alert policy with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl monitoring alert delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	return cmd
}

// RunCmdAlertPolicyCreate runs alert policy create.
func RunCmdAlertPolicyCreate(c *CmdConfig) error {
	ms := c.Monitoring()

	desc, err := c.Doit.GetString(c.NS, doctl.ArgAlertPolicyDescription)
	if err != nil {
		return err
	}

	alertType, err := c.Doit.GetString(c.NS, doctl.ArgAlertPolicyType)
	if err != nil {
		return err
	}

	err = validateAlertPolicyType(alertType)
	if err != nil {
		return err
	}

	value, err := c.Doit.GetInt(c.NS, doctl.ArgAlertPolicyValue)
	if err != nil {
		return err
	}

	window, err := c.Doit.GetString(c.NS, doctl.ArgAlertPolicyWindow)
	if err != nil {
		return err
	}
	err = validateAlertPolicyWindow(window)
	if err != nil {
		return err
	}

	entities, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAlertPolicyEntities)
	if err != nil {
		return err
	}

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAlertPolicyTags)
	if err != nil {
		return err
	}

	enabled, err := c.Doit.GetBool(c.NS, doctl.ArgAlertPolicyEnabled)
	if err != nil {
		return err
	}

	compareStr, err := c.Doit.GetString(c.NS, doctl.ArgAlertPolicyCompare)
	if err != nil {
		return err
	}

	compare, err := getComparator(compareStr)
	if err != nil {
		return err
	}

	emails, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAlertPolicyEmails)
	if err != nil {
		return err
	}

	slackChannels, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAlertPolicySlackChannels)
	if err != nil {
		return err
	}

	slackURLs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAlertPolicySlackURLs)
	if err != nil {
		return err
	}

	if len(slackURLs) != len(slackChannels) {
		return errors.New("must provide the same number of slack channels as slack URLs")
	}

	if len(emails) == 0 && len(slackURLs) == 0 {
		return errors.New("must provide either emails or slack details to send the alert to")
	}

	slacks := make([]godo.SlackDetails, len(slackChannels))
	for i, channel := range slackChannels {
		slacks[i] = godo.SlackDetails{Channel: channel, URL: slackURLs[i]}
	}

	apcr := &godo.AlertPolicyCreateRequest{
		Type:        alertType,
		Description: desc,
		Compare:     compare,
		Value:       float32(value),
		Window:      window,
		Entities:    entities,
		Tags:        tags,
		Alerts: godo.Alerts{
			Slack: slacks,
			Email: emails,
		},
		Enabled: &enabled,
	}
	p, err := ms.CreateAlertPolicy(apcr)
	if err != nil {
		return err
	}

	return c.Display(&displayers.AlertPolicy{AlertPolicies: do.AlertPolicies{*p}})
}

// RunCmdAlertPolicyUpdate runs alert policy update.
func RunCmdAlertPolicyUpdate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	uuid := c.Args[0]

	ms := c.Monitoring()

	desc, err := c.Doit.GetString(c.NS, doctl.ArgAlertPolicyDescription)
	if err != nil {
		return err
	}

	alertType, err := c.Doit.GetString(c.NS, doctl.ArgAlertPolicyType)
	if err != nil {
		return err
	}
	err = validateAlertPolicyType(alertType)
	if err != nil {
		return err
	}

	value, err := c.Doit.GetInt(c.NS, doctl.ArgAlertPolicyValue)
	if err != nil {
		return err
	}

	window, err := c.Doit.GetString(c.NS, doctl.ArgAlertPolicyWindow)
	if err != nil {
		return err
	}
	err = validateAlertPolicyWindow(window)
	if err != nil {
		return err
	}

	entities, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAlertPolicyEntities)
	if err != nil {
		return err
	}

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAlertPolicyTags)
	if err != nil {
		return err
	}

	enabled, err := c.Doit.GetBool(c.NS, doctl.ArgAlertPolicyEnabled)
	if err != nil {
		return err
	}

	compareStr, err := c.Doit.GetString(c.NS, doctl.ArgAlertPolicyCompare)
	if err != nil {
		return err
	}

	compare, err := getComparator(compareStr)
	if err != nil {
		return err
	}

	emails, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAlertPolicyEmails)
	if err != nil {
		return err
	}

	slackChannels, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAlertPolicySlackChannels)
	if err != nil {
		return err
	}

	slackURLs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAlertPolicySlackURLs)
	if err != nil {
		return err
	}

	if len(slackURLs) != len(slackChannels) {
		return errors.New("must provide the same number of slack channels as slack URLs")
	}

	if len(emails) == 0 && len(slackURLs) == 0 {
		return errors.New("must provide either emails or slack details to send the alert to")
	}

	slacks := make([]godo.SlackDetails, len(slackChannels))
	for i, channel := range slackChannels {
		slacks[i] = godo.SlackDetails{Channel: channel, URL: slackURLs[i]}
	}

	apcr := &godo.AlertPolicyUpdateRequest{
		Type:        alertType,
		Description: desc,
		Compare:     compare,
		Value:       float32(value),
		Window:      window,
		Entities:    entities,
		Tags:        tags,
		Alerts: godo.Alerts{
			Slack: slacks,
			Email: emails,
		},
		Enabled: &enabled,
	}
	p, err := ms.UpdateAlertPolicy(uuid, apcr)
	if err != nil {
		return err
	}

	return c.Display(&displayers.AlertPolicy{AlertPolicies: do.AlertPolicies{*p}})
}

func getComparator(compareStr string) (godo.AlertPolicyComp, error) {
	var compare godo.AlertPolicyComp
	if strings.EqualFold("LessThan", compareStr) {
		compare = godo.LessThan
	} else if strings.EqualFold("GreaterThan", compareStr) {
		compare = godo.GreaterThan
	} else {
		return "", errors.New("comparator must be GreaterThan or LessThan")
	}
	return compare, nil
}

func validateAlertPolicyType(t string) error {
	validAlertPolicyTypes := map[string]struct{}{
		godo.DropletCPUUtilizationPercent:                     {},
		godo.DropletMemoryUtilizationPercent:                  {},
		godo.DropletDiskUtilizationPercent:                    {},
		godo.DropletDiskReadRate:                              {},
		godo.DropletDiskWriteRate:                             {},
		godo.DropletOneMinuteLoadAverage:                      {},
		godo.DropletFiveMinuteLoadAverage:                     {},
		godo.DropletFifteenMinuteLoadAverage:                  {},
		godo.DropletPublicOutboundBandwidthRate:               {},
		godo.DbaasFifteenMinuteLoadAverage:                    {},
		godo.DbaasMemoryUtilizationPercent:                    {},
		godo.DbaasDiskUtilizationPercent:                      {},
		godo.DbaasCPUUtilizationPercent:                       {},
		godo.LoadBalancerCPUUtilizationPercent:                {},
		godo.LoadBalancerDropletHealth:                        {},
		godo.LoadBalancerTLSUtilizationPercent:                {},
		godo.LoadBalancerConnectionUtilizationPercent:         {},
		godo.LoadBalancerIncreaseInHTTPErrorRateCount4xx:      {},
		godo.LoadBalancerIncreaseInHTTPErrorRateCount5xx:      {},
		godo.LoadBalancerIncreaseInHTTPErrorRatePercentage4xx: {},
		godo.LoadBalancerIncreaseInHTTPErrorRatePercentage5xx: {},
		godo.LoadBalancerHighHttpResponseTime:                 {},
		godo.LoadBalancerHighHttpResponseTime50P:              {},
		godo.LoadBalancerHighHttpResponseTime95P:              {},
		godo.LoadBalancerHighHttpResponseTime99P:              {},
	}

	_, ok := validAlertPolicyTypes[t]

	if !ok {
		return errors.New(fmt.Sprintf("'%s' is not a valid alert policy type", t))
	}

	return nil
}

func validateAlertPolicyWindow(w string) error {
	switch w {
	case "5m":
		fallthrough
	case "10m":
		fallthrough
	case "30m":
		fallthrough
	case "1h":
		return nil
	default:
		return errors.New(fmt.Sprintf("'%s' is not a valid alert policy window. Must be one of '5m', '10m', '30m', or '1h'", w))
	}
}

// RunCmdAlertPolicyGet runs alert policy get.
func RunCmdAlertPolicyGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	uuid := c.Args[0]
	ms := c.Monitoring()
	p, err := ms.GetAlertPolicy(uuid)
	if err != nil {
		return err
	}

	return c.Display(&displayers.AlertPolicy{AlertPolicies: do.AlertPolicies{*p}})
}

// RunCmdAlertPolicyList runs alert policy list.
func RunCmdAlertPolicyList(c *CmdConfig) error {
	ms := c.Monitoring()
	policies, err := ms.ListAlertPolicies()
	if err != nil {
		return err
	}

	return c.Display(&displayers.AlertPolicy{AlertPolicies: policies})
}

// RunCmdAlertPolicyDelete runs alert policy delete.
func RunCmdAlertPolicyDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("alert policy", len(c.Args)) == nil {
		for id := range c.Args {
			uuid := c.Args[id]
			ms := c.Monitoring()
			if err := ms.DeleteAlertPolicy(uuid); err != nil {
				return err
			}
		}
	} else {
		return errOperationAborted
	}

	return nil
}
