/*
Copyright 2023 The Doctl Authors All rights reserved.
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

// UptimeAlert creates the UptimeAlert command
func UptimeAlert() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "alert",
			Aliases: []string{"alerts"},
			Short:   "Display commands to manage uptime alerts",
			Long: `The sub-commands of ` + "`" + `doctl monitoring uptime alert` + "`" + ` manage your uptime alerts.

DigitalOcean Uptime Alerts provide the ability to monitor your endpoints from around the world,
and alert you when they're slow, unavailable, or SSL certificates are expiring.`,
		},
	}

	cmdUptimeAlertsCreate := CmdBuilder(cmd, RunUptimeAlertsCreate, "create <uptime-check-id>", "Create an uptime alert", `Use this command to create an uptime alert on your account.

You can use flags to specify the uptime alert, type, threshold, comparison, notifications, and period.`, Writer,
		aliasOpt("c"), displayerType(&displayers.UptimeAlert{}), overrideCmdNS("uptime-alert"))

	AddStringFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertName, "", "", "Uptime alert name", requiredOpt())
	AddStringFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertType, "", "", "Uptime alert type, must be one of latency, down, down_global or ssl_expiry", requiredOpt())
	AddIntFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertThreshold, "", 0, "Uptime alert threshold at which the alert will enter a trigger state. The specific threshold is dependent on the alert type.")
	AddStringFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertComparison, "", "", "Uptime alert comparison operator used against the alert's threshold. Must be either `greater_than` or `less_than`")
	AddStringSliceFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertEmails, "", nil, "Emails to send uptime alerts to. Must be emails associated with the DigitalOcean account")
	AddStringSliceFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertSlackChannels, "", nil, "Slack channels to send uptime alerts to.")
	AddStringSliceFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertSlackURLs, "", nil, "Slack URLs to send uptime alerts to.")
	AddStringFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertPeriod, "", "", "Uptime alert period of time the threshold must be exceeded to trigger the alert. Must be one of 2m, 3m, 5m, 10m, 15m, 30m or 1h", requiredOpt())

	CmdBuilder(cmd, RunUptimeAlertsGet, "get <uptime-check-id> <uptime-alert-id>", "Get uptime alert", `Use this command to get an uptime alert on your account by ID.`, Writer,
		aliasOpt("g"), displayerType(&displayers.UptimeAlert{}))

	CmdBuilder(cmd, RunUptimeAlertsList, "list <uptime-check-id>", "List uptime alerts", `Use this command to list all of the uptime alerts for an uptime check.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.UptimeAlert{}))

	cmdUptimeAlertsUpdate := CmdBuilder(cmd, RunUptimeAlertsUpdate, "update <uptime-check-id> <uptime-alert-id>", "Update an uptime alert", `Use this command to update an uptime alert on your account.

You can use flags to specify the uptime alert, type, threshold, comparison, notifications, and period.`, Writer,
		aliasOpt("u"), displayerType(&displayers.UptimeAlert{}), overrideCmdNS("uptime-alert"))
	AddStringFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertName, "", "", "Uptime alert name", requiredOpt())
	AddStringFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertType, "", "", "Uptime alert type, must be one of latency, down, down_global or ssl_expiry", requiredOpt())
	AddIntFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertThreshold, "", 0, "Uptime alert threshold at which the alert will enter a trigger state. The specific threshold is dependent on the alert type.")
	AddStringFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertComparison, "", "", "Uptime alert comparison operator used against the alert's threshold. Must be either `greater_than` or `less_than`")
	AddStringSliceFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertEmails, "", nil, "Emails to send uptime alerts to. Must be emails associated with the DigitalOcean account")
	AddStringSliceFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertSlackChannels, "", nil, "Slack channels to send uptime alerts to.")
	AddStringSliceFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertSlackURLs, "", nil, "Slack URLs to send uptime alerts to.")
	AddStringFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertPeriod, "", "", "Uptime alert period of time the threshold must be exceeded to trigger the alert. Must be one of 2m, 3m, 5m, 10m, 15m, 30m or 1h", requiredOpt())

	CmdBuilder(cmd, RunUptimeAlertsDelete, "delete <uptime-check-id> <uptime-alert-id>", "Delete an uptime alert", `Use this command to delete an uptime check on your account by ID.`, Writer,
		aliasOpt("d", "del", "rm"))

	return cmd
}

func getUptimeAlertComparator(compareStr string) (godo.UptimeAlertComp, error) {
	var compare godo.UptimeAlertComp
	if strings.EqualFold("less_than", compareStr) {
		compare = godo.UptimeAlertLessThan
	} else if strings.EqualFold("greater_than", compareStr) {
		compare = godo.UptimeAlertGreaterThan
	} else {
		return "", errors.New("comparator must be greater_than or less_than")
	}
	return compare, nil
}

func validateUptimeAlertType(t string) error {
	validUptimeAlertTypes := map[string]struct{}{
		"latency":     {},
		"down":        {},
		"down_global": {},
		"ssl_expiry":  {},
	}

	_, ok := validUptimeAlertTypes[t]

	if !ok {
		return errors.New(fmt.Sprintf("'%s' is not a valid uptime alert type", t))
	}

	return nil
}

func validateUptimeAlertPeriod(t string) error {
	validUptimeAlertPeriods := map[string]struct{}{
		"2m":  {},
		"3m":  {},
		"5m":  {},
		"10m": {},
		"15m": {},
		"30m": {},
		"1h":  {},
	}

	_, ok := validUptimeAlertPeriods[t]

	if !ok {
		return errors.New(fmt.Sprintf("'%s' is not a valid uptime alert period", t))
	}

	return nil
}

// RunUptimeAlertsCreate creates an uptime alert.
func RunUptimeAlertsCreate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertName)
	if err != nil {
		return err
	}

	alertType, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertType)
	if err != nil {
		return err
	}

	err = validateUptimeAlertType(alertType)
	if err != nil {
		return err
	}

	threshold, err := c.Doit.GetInt(c.NS, doctl.ArgUptimeAlertThreshold)
	if err != nil {
		return err
	}

	compareStr, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertComparison)
	if err != nil {
		return err
	}
	comparison, err := getUptimeAlertComparator(compareStr)
	if err != nil {
		return err
	}

	emails, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertEmails)
	if err != nil {
		return err
	}

	slackChannels, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertSlackChannels)
	if err != nil {
		return err
	}

	slackURLs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertSlackURLs)
	if err != nil {
		return err
	}

	if len(slackURLs) != len(slackChannels) {
		return errors.New("must provide the same number of slack channels as slack URLs")
	}

	if len(emails) == 0 && len(slackURLs) == 0 {
		return errors.New("must provide either emails or slack details to send the uptime alert to")
	}

	slacks := make([]godo.SlackDetails, len(slackChannels))
	for i, channel := range slackChannels {
		slacks[i] = godo.SlackDetails{Channel: channel, URL: slackURLs[i]}
	}

	period, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertPeriod)
	if err != nil {
		return err
	}

	err = validateUptimeAlertPeriod(period)
	if err != nil {
		return err
	}

	cuar := &godo.CreateUptimeAlertRequest{
		Name:       name,
		Type:       alertType,
		Threshold:  threshold,
		Comparison: comparison,
		Notifications: &godo.Notifications{
			Slack: slacks,
			Email: emails,
		},
		Period: period,
	}

	uptimeAlert, err := c.UptimeChecks().CreateAlert(c.Args[0], cuar)
	if err != nil {
		return err
	}

	return c.Display(&displayers.UptimeAlert{UptimeAlerts: []do.UptimeAlert{*uptimeAlert}})
}

// RunUptimeAlertsGet gets an uptime alert by ID.
func RunUptimeAlertsGet(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	uptimeAlert, err := c.UptimeChecks().GetAlert(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}
	item := &displayers.UptimeAlert{UptimeAlerts: []do.UptimeAlert{*uptimeAlert}}
	return c.Display(item)
}

// RunUptimeAlertsList returns a list of uptime alerts.
func RunUptimeAlertsList(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	uptimeAlerts, err := c.UptimeChecks().ListAlerts(c.Args[0])
	if err != nil {
		return err
	}
	item := &displayers.UptimeAlert{UptimeAlerts: uptimeAlerts}
	return c.Display(item)
}

// RunUptimeAlertsUpdate updates an uptime alert by ID.
func RunUptimeAlertsUpdate(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertName)
	if err != nil {
		return err
	}

	alertType, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertType)
	if err != nil {
		return err
	}

	err = validateUptimeAlertType(alertType)
	if err != nil {
		return err
	}

	threshold, err := c.Doit.GetInt(c.NS, doctl.ArgUptimeAlertThreshold)
	if err != nil {
		return err
	}

	compareStr, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertComparison)
	if err != nil {
		return err
	}
	comparison, err := getUptimeAlertComparator(compareStr)
	if err != nil {
		return err
	}

	emails, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertEmails)
	if err != nil {
		return err
	}

	slackChannels, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertSlackChannels)
	if err != nil {
		return err
	}

	slackURLs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertSlackURLs)
	if err != nil {
		return err
	}

	if len(slackURLs) != len(slackChannels) {
		return errors.New("must provide the same number of slack channels as slack URLs")
	}

	if len(emails) == 0 && len(slackURLs) == 0 {
		return errors.New("must provide either emails or slack details to send the uptime alert to")
	}

	slacks := make([]godo.SlackDetails, len(slackChannels))
	for i, channel := range slackChannels {
		slacks[i] = godo.SlackDetails{Channel: channel, URL: slackURLs[i]}
	}

	period, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertPeriod)
	if err != nil {
		return err
	}

	err = validateUptimeAlertPeriod(period)
	if err != nil {
		return err
	}

	uuar := &godo.UpdateUptimeAlertRequest{
		Name:       name,
		Type:       alertType,
		Threshold:  threshold,
		Comparison: comparison,
		Notifications: &godo.Notifications{
			Slack: slacks,
			Email: emails,
		},
		Period: period,
	}

	uptimeAlert, err := c.UptimeChecks().UpdateAlert(c.Args[0], c.Args[1], uuar)
	if err != nil {
		return err
	}

	return c.Display(&displayers.UptimeAlert{UptimeAlerts: []do.UptimeAlert{*uptimeAlert}})
}

// RunUptimeAlertsDelete deletes an uptime alert by ID.
func RunUptimeAlertsDelete(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	return c.UptimeChecks().DeleteAlert(c.Args[0], c.Args[1])
}
