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

An alert policy can be applied to resource(s) (currently Droplets)
in order to alert on resource consumption.`,
		},
	}

	cmd.AddCommand(alertPolicies())
	return cmd
}

func alertPolicies() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "alert",
			Aliases: []string{"alerts", "a"},
			Short:   "Display commands for managing alert policies",
			Long:    "The commands under `doctl monitoring alert` are for the management of alert policies.",
		},
	}

	cmdAlertPolicyCreate := CmdBuilder(cmd, RunCmdAlertPolicyCreate, "create", "Create an alert policy", `Use this command to create a new alert policy.`, Writer)
	AddStringFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyDescription, "", "", "A description of the alert policy.")
	AddStringFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyType, "", "", "The type of alert policy.")
	AddStringFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyCompare, "", "", "The comparator of the alert policy. Either `GreaterThan` or `LessThan`")
	AddStringFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyWindow, "", "5m", "The window to apply the alert policy conditions against.")
	AddIntFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyValue, "", 0, "The value of the alert policy to compare against.")
	AddBoolFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyEnabled, "", true, "Whether the alert policy is enabled.")
	AddStringSliceFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyEmails, "", nil, "Emails to send alerts to.")
	AddStringSliceFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyTags, "", nil, "Tags to apply the alert against.")
	AddStringSliceFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicyEntities, "", nil, "Entities to apply the alert against. (e.g. a droplet ID for a droplet alert policy)")
	AddStringSliceFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicySlackChannels, "", nil, "Slack channels to send alerts to.")
	AddStringSliceFlag(cmdAlertPolicyCreate, doctl.ArgAlertPolicySlackURLs, "", nil, "Slack URLs to send alerts to.")

	cmdAlertPolicyUpdate := CmdBuilder(cmd, RunCmdAlertPolicyUpdate, "update <alert-policy-uuid>...", "Update an alert policy", `Use this command to update an existing alert policy.`, Writer)
	AddStringFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyDescription, "", "", "A description of the alert policy.")
	AddStringFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyType, "", "", "The type of alert policy.")
	AddStringFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyCompare, "", "", "The comparator of the alert policy. Either `GreaterThan` or `LessThan`")
	AddStringFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyWindow, "", "5m", "The window to apply the alert policy conditions against.")
	AddIntFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyValue, "", 0, "The value of the alert policy to compare against.")
	AddBoolFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyEnabled, "", true, "Whether the alert policy is enabled.")
	AddStringSliceFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyEmails, "", nil, "Emails to send alerts to.")
	AddStringSliceFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyTags, "", nil, "Tags to apply the alert against.")
	AddStringSliceFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicyEntities, "", nil, "Entities to apply the alert against. (e.g. a droplet ID for a droplet alert policy)")
	AddStringSliceFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicySlackChannels, "", nil, "Slack channels to send alerts to.")
	AddStringSliceFlag(cmdAlertPolicyUpdate, doctl.ArgAlertPolicySlackURLs, "", nil, "Slack URLs to send alerts to.")

	CmdBuilder(cmd, RunCmdAlertPolicyGet, "get <alert-policy-uuid>", "Retrieve information about an alert policy", `Use this command to retrieve an alert policy and see its configuration.`, Writer,
		displayerType(&displayers.AlertPolicy{}))

	CmdBuilder(cmd, RunCmdAlertPolicyList, "list", "List all alert policies", `Use this command to retrieve a list of all the alert policies in your account.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.AlertPolicy{}))

	cmdRunAlertPolicyDelete := CmdBuilder(cmd, RunCmdAlertPolicyDelete, "delete <alert-policy-uuid>...", "Delete an alert policy", `Use this command to delete an alert policy.`, Writer)
	AddBoolFlag(cmdRunAlertPolicyDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete an alert policy without confirmation prompt")

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
	switch t {
	case godo.DropletCPUUtilizationPercent:
		fallthrough
	case godo.DropletMemoryUtilizationPercent:
		fallthrough
	case godo.DropletDiskUtilizationPercent:
		fallthrough
	case godo.DropletDiskReadRate:
		fallthrough
	case godo.DropletDiskWriteRate:
		fallthrough
	case godo.DropletOneMinuteLoadAverage:
		fallthrough
	case godo.DropletFiveMinuteLoadAverage:
		fallthrough
	case godo.DropletFifteenMinuteLoadAverage:
		fallthrough
	case godo.DropletPublicOutboundBandwidthRate:
		return nil
	default:
		return errors.New(fmt.Sprintf("'%s' is not a valid alert policy type", t))
	}
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
