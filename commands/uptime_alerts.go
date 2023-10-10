package commands

import (
	"fmt"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func uptimeAlertCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "alerts",
			Short: "Display commands to manage uptime check alerts",
			Long: `The sub-commands of ` + "`" + `doctl monitoring` + "`" + ` manage your uptime check alerts.

DigitalOcean Uptime Check Alerts sends you message when Uptime Checks are showing anomalies.`,
		},
	}

	cmdUptimeAlertsCreate := CmdBuilder(
		cmd, RunUptimeAlertsCreate, "create <uptime-check-id>", "Create an alert for the uptime check",
		`Use this command to create an uptime alert on your account.

You can use flags to specify the alert name, type, threshold, comparison operand, alert channels and period to poll the uptime check.`,
		Writer, aliasOpt("c"), displayerType(&displayers.UptimeAlert{}),
	)
	AddStringFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertName, "", "", "Uptime alert name", requiredOpt())
	AddStringFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertType, "", "", "Uptime alert type, must be one of latency, down, down_global, or ssl_expiry", requiredOpt())
	AddIntFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertThreshold, "", 0, "Threshold at which the alert will enter a trigger state. The specific threshold is dependent on the alert type")
	AddStringFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertComparison, "", "", "The comparison operator used against the alert's threshold: greater_than or less_than")
	AddStringFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertPeriod, "", "", "Period of time the threshold must be exceeded to trigger the alert: 2m 3m 5m 10m 15m 30m 1h", requiredOpt())
	AddStringSliceFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertEmails, "", nil, "Emails to send alerts to")
	AddStringSliceFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertSlackChannels, "", nil, "Slack channels to send alerts to")
	AddStringSliceFlag(cmdUptimeAlertsCreate, doctl.ArgUptimeAlertSlackURLs, "", nil, "Slack URLs to send alerts to")

	CmdBuilder(cmd, RunUptimeAlertsGet, "get <uptime-check-id> <uptime-alert-id>", "Get an uptime alert", `Use this command to get an uptime check alert by check and alert ID.`, Writer,
		aliasOpt("g"), displayerType(&displayers.UptimeAlert{}))

	CmdBuilder(cmd, RunUptimeAlertsList, "list <uptime-check-id>", "List uptime alerts", `Use this command to list all of the alerts for a given uptime check.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.UptimeCheck{}))

	cmdUptimeAlertsUpdate := CmdBuilder(
		cmd, RunUptimeAlertsUpdate, "update <uptime-check-id> <uptime-alert-id>", "Update an alert for the uptime check",
		`Use this command to update an uptime alert on your account.

You can use flags to specify the alert name, type, threshold, comparison operand, alert channels and period to poll the uptime check.`,
		Writer, aliasOpt("u"), displayerType(&displayers.UptimeAlert{}),
	)
	AddStringFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertName, "", "", "Uptime alert name", requiredOpt())
	AddStringFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertType, "", "", "Uptime alert type, must be one of latency, down, down_global, or ssl_expiry", requiredOpt())
	AddIntFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertThreshold, "", 0, "Threshold at which the alert will enter a trigger state. The specific threshold is dependent on the alert type")
	AddStringFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertComparison, "", "", "The comparison operator used against the alert's threshold: greater_than or less_than")
	AddStringFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertPeriod, "", "", "Period of time the threshold must be exceeded to trigger the alert: 2m 3m 5m 10m 15m 30m 1h", requiredOpt())
	AddStringSliceFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertEmails, "", nil, "Emails to send alerts to")
	AddStringSliceFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertSlackChannels, "", nil, "Slack channels to send alerts to")
	AddStringSliceFlag(cmdUptimeAlertsUpdate, doctl.ArgUptimeAlertSlackURLs, "", nil, "Slack URLs to send alerts to")

	CmdBuilder(cmd, RunUptimeAlertsDelete, "delete <uptime-check-id>  <uptime-alert-id>", "Delete an uptime check", `Use this command to delete an uptime check on your account by ID.`, Writer,
		aliasOpt("d", "del", "rm"))

	return cmd
}

func RunUptimeAlertsCreate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	alertName, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertName)
	if err != nil {
		return err
	}

	alertType, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertType)
	if err != nil {
		return err
	}
	if alertType != "latency" && alertType != "down" && alertType != "down_global" && alertType != "ssl_expiry" {
		return fmt.Errorf("the uptime alert type must be one of latency, down, down_global, or ssl_expiry, got %s", alertType)
	}

	alertThreshold, err := c.Doit.GetInt(c.NS, doctl.ArgUptimeAlertThreshold)
	if err != nil {
		return err
	}

	alertComparison, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertComparison)
	if err != nil {
		return err
	}
	if alertComparison != "greater_than" && alertComparison != "less_than" {
		return fmt.Errorf("the uptime alert comparison operator must be one of greater_than or less_than, got %s", alertComparison)
	}

	alertPeriod, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertPeriod)
	if err != nil {
		return err
	}
	if alertPeriod != "2m" && alertPeriod != "3m" && alertPeriod != "5m" && alertPeriod != "10m" && alertPeriod != "15m" && alertPeriod != "30m" && alertPeriod != "1h" {
		return fmt.Errorf("the uptime alert period must be one of 2m, 3m, 5m, 10m, 15m, 30m, or 1h, got %s", alertComparison)
	}

	alertEmails, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertEmails)
	if err != nil {
		return err
	}

	alertSlackChannels, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertSlackChannels)
	if err != nil {
		return err
	}

	alertSlackURLs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertSlackURLs)
	if err != nil {
		return err
	}

	if len(alertSlackURLs) != len(alertSlackChannels) {
		return fmt.Errorf("must provide the same number of slack channels as slack URLs")
	}

	if len(alertEmails) == 0 && len(alertSlackURLs) == 0 {
		return fmt.Errorf("must provide either emails or slack details to send the alert to")
	}

	alertSlacks := make([]godo.SlackDetails, len(alertSlackChannels))
	for i, channel := range alertSlackChannels {
		alertSlacks[i] = godo.SlackDetails{Channel: channel, URL: alertSlackURLs[i]}
	}

	uptimeAlert, err := c.UptimeAlerts().Create(c.Args[0], &godo.CreateUptimeAlertRequest{
		Name:       alertName,
		Type:       alertType,
		Threshold:  alertThreshold,
		Comparison: alertComparison,
		Period:     alertPeriod,
		Notifications: &godo.Notifications{
			Email: alertEmails,
			Slack: alertSlacks,
		},
	})
	if err != nil {
		return err
	}

	item := &displayers.UptimeAlert{UptimeAlerts: []do.UptimeAlert{*uptimeAlert}}
	return c.Display(item)
}

func RunUptimeAlertsGet(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	uptimeAlert, err := c.UptimeAlerts().Get(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}

	item := &displayers.UptimeAlert{UptimeAlerts: []do.UptimeAlert{*uptimeAlert}}
	return c.Display(item)
}

func RunUptimeAlertsList(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	uptimeAlerts, err := c.UptimeAlerts().List(c.Args[0])
	if err != nil {
		return err
	}

	items := &displayers.UptimeAlert{UptimeAlerts: uptimeAlerts}
	return c.Display(items)
}

func RunUptimeAlertsUpdate(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	alertName, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertName)
	if err != nil {
		return err
	}

	alertType, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertType)
	if err != nil {
		return err
	}
	if alertType != "latency" && alertType != "down" && alertType != "down_global" && alertType != "ssl_expiry" {
		return fmt.Errorf("the uptime alert type must be one of latency, down, down_global, or ssl_expiry, got %s", alertType)
	}

	alertThreshold, err := c.Doit.GetInt(c.NS, doctl.ArgUptimeAlertThreshold)
	if err != nil {
		return err
	}

	alertComparison, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertComparison)
	if err != nil {
		return err
	}
	if alertComparison != "greater_than" && alertComparison != "less_than" {
		return fmt.Errorf("the uptime alert comparison operator must be one of greater_than or less_than, got %s", alertComparison)
	}

	alertPeriod, err := c.Doit.GetString(c.NS, doctl.ArgUptimeAlertPeriod)
	if err != nil {
		return err
	}
	if alertPeriod != "2m" && alertPeriod != "3m" && alertPeriod != "5m" && alertPeriod != "10m" && alertPeriod != "15m" && alertPeriod != "30m" && alertPeriod != "1h" {
		return fmt.Errorf("the uptime alert comparison operator must be one of 2m, 3m, 5m, 10m, 15m, 30m, or 1h, got %s", alertComparison)
	}

	alertEmails, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertEmails)
	if err != nil {
		return err
	}

	alertSlackChannels, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertSlackChannels)
	if err != nil {
		return err
	}

	alertSlackURLs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeAlertSlackURLs)
	if err != nil {
		return err
	}

	if len(alertSlackURLs) != len(alertSlackChannels) {
		return fmt.Errorf("must provide the same number of slack channels as slack URLs")
	}

	if len(alertEmails) == 0 && len(alertSlackURLs) == 0 {
		return fmt.Errorf("must provide either emails or slack details to send the alert to")
	}

	alertSlacks := make([]godo.SlackDetails, len(alertSlackChannels))
	for i, channel := range alertSlackChannels {
		alertSlacks[i] = godo.SlackDetails{Channel: channel, URL: alertSlackURLs[i]}
	}

	uptimeAlert, err := c.UptimeAlerts().Update(c.Args[0], c.Args[1], &godo.UpdateUptimeAlertRequest{
		Name:       alertName,
		Type:       alertType,
		Threshold:  alertThreshold,
		Comparison: alertComparison,
		Period:     alertPeriod,
		Notifications: &godo.Notifications{
			Email: alertEmails,
			Slack: alertSlacks,
		},
	})
	if err != nil {
		return err
	}

	item := &displayers.UptimeAlert{UptimeAlerts: []do.UptimeAlert{*uptimeAlert}}
	return c.Display(item)
}

func RunUptimeAlertsDelete(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	return c.UptimeAlerts().Delete(c.Args[0], c.Args[1])
}
