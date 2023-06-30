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
	"fmt"
	"net/url"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// UptimeCheck creates the UptimeCheck command
func UptimeCheck() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "uptime",
			Short: "Display commands to manage uptime checks",
			Long: `The sub-commands of ` + "`" + `doctl uptime` + "`" + ` manage your uptime checks.

DigitalOcean Uptime Checks provide the ability to monitor your endpoints from around the world,
and alert you when they're slow, unavailable, or SSL certificates are expiring.`,
		},
	}

	cmdUptimeChecksCreate := CmdBuilder(cmd, RunUptimeChecksCreate, "create <uptime-check-name>", "Create an uptime check", `Use this command to create an uptime check on your account.

You can use flags to specify the uptime check, target, type, regions, and whether or not the check is enabled.`, Writer,
		aliasOpt("c"), displayerType(&displayers.UptimeCheck{}))
	AddStringFlag(cmdUptimeChecksCreate, doctl.ArgUptimeCheckTarget, "", "", "Uptime check target, must be a valid URL", requiredOpt())
	AddStringFlag(cmdUptimeChecksCreate, doctl.ArgUptimeCheckType, "", "", "Uptime check type, must be one of ping, http, or https. Defaults to either http or https, depending on the URL target provided")
	AddStringSliceFlag(cmdUptimeChecksCreate, doctl.ArgUptimeCheckRegions, "", []string{"us_east"}, "Uptime check regions, must be a comma-separated list from any of us_east, us_west, eu_west, or se_asia. Defaults to [us_east]")
	AddBoolFlag(cmdUptimeChecksCreate, doctl.ArgUptimeCheckEnabled, "", true, "Whether or not the uptime check is enabled, defaults to true")

	CmdBuilder(cmd, RunUptimeChecksGet, "get <uptime-check-id>", "Get an uptime check", `Use this command to get an uptime check on your account by ID.`, Writer,
		aliasOpt("g"), displayerType(&displayers.UptimeCheck{}))

	CmdBuilder(cmd, RunUptimeChecksList, "list", "List uptime checks", `Use this command to list all of the uptime checks on your account.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.UptimeCheck{}))

	cmdUptimeCheckUpdate := CmdBuilder(cmd, RunUptimeChecksUpdate, "update <uptime-check-id>", "Update an uptime check", `Use this command to update an uptime check on your account.

You can use flags to specify an updated uptime check name, target, type, and regions. All of these flags are required. An uptime check cannot be disabled via doctl, you must do so through the Cloud control panel or through the public API directly`, Writer,
		aliasOpt("u"), displayerType(&displayers.UptimeCheck{}))
	AddStringFlag(cmdUptimeCheckUpdate, doctl.ArgUptimeCheckName, "", "", "Uptime check name", requiredOpt())
	AddStringFlag(cmdUptimeCheckUpdate, doctl.ArgUptimeCheckTarget, "", "", "Uptime check target, must be a valid URL", requiredOpt())
	AddStringFlag(cmdUptimeCheckUpdate, doctl.ArgUptimeCheckType, "", "", "Uptime check type, must be one of ping, http, or https", requiredOpt())
	AddStringSliceFlag(cmdUptimeCheckUpdate, doctl.ArgUptimeCheckRegions, "", []string{"us_east"}, "Uptime check regions, must be a comma-separated list from any of us_east, us_west, eu_west, or se_asia", requiredOpt())

	CmdBuilder(cmd, RunUptimeChecksDelete, "delete <uptime-check-id>", "Delete an uptime check", `Use this command to delete an uptime check on your account by ID.`, Writer,
		aliasOpt("d", "del", "rm"))

	return cmd
}

// RunUptimeChecksCreate creates an uptime check.
func RunUptimeChecksCreate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	checkName := c.Args[0]

	checkTarget, err := c.Doit.GetString(c.NS, doctl.ArgUptimeCheckTarget)
	if err != nil {
		return err
	}
	checkURL, err := url.Parse(checkTarget)
	if err != nil {
		return fmt.Errorf("the uptime check target %s is not a valid URL: %w", checkTarget, err)
	}

	checkType := checkURL.Scheme
	checkTypeArg, err := c.Doit.GetString(c.NS, doctl.ArgUptimeCheckType)
	if err != nil {
		return err
	}
	if checkTypeArg != "" {
		checkType = checkTypeArg
	}
	if checkType != "ping" && checkType != "http" && checkType != "https" {
		return fmt.Errorf("the uptime check type must be one of ping, http, or https, got %s", checkType)
	}

	checkRegions := []string{"us_east"}
	checkRegionsArg, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeCheckRegions)
	if err != nil {
		return err
	}
	if len(checkRegionsArg) > 0 {
		checkRegions = checkRegionsArg
	}

	checkEnabled, err := c.Doit.GetBool(c.NS, doctl.ArgUptimeCheckEnabled)
	if err != nil {
		return err
	}

	uptimeCheck, err := c.UptimeChecks().Create(&godo.CreateUptimeCheckRequest{
		Name:    checkName,
		Type:    checkType,
		Target:  checkTarget,
		Regions: checkRegions,
		Enabled: checkEnabled,
	})
	if err != nil {
		return err
	}

	item := &displayers.UptimeCheck{UptimeChecks: []do.UptimeCheck{*uptimeCheck}}
	return c.Display(item)
}

// RunUptimeChecksGet gets an uptime check by ID.
func RunUptimeChecksGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	uptimeCheck, err := c.UptimeChecks().Get(c.Args[0])
	if err != nil {
		return nil
	}

	item := &displayers.UptimeCheck{UptimeChecks: []do.UptimeCheck{*uptimeCheck}}
	return c.Display(item)
}

// RunUptimeChecksList returns a list of uptime checks.
func RunUptimeChecksList(c *CmdConfig) error {
	uptimeChecks, err := c.UptimeChecks().List()
	if err != nil {
		return err
	}
	item := &displayers.UptimeCheck{UptimeChecks: uptimeChecks}
	return c.Display(item)
}

// RunUptimeChecksUpdate updates an uptime check by ID.
func RunUptimeChecksUpdate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	checkID := c.Args[0]

	checkName, err := c.Doit.GetString(c.NS, doctl.ArgUptimeCheckName)
	if err != nil {
		return err
	}

	checkTarget, err := c.Doit.GetString(c.NS, doctl.ArgUptimeCheckTarget)
	if err != nil {
		return err
	}

	checkType, err := c.Doit.GetString(c.NS, doctl.ArgUptimeCheckType)
	if err != nil {
		return err
	}
	if checkType != "ping" && checkType != "http" && checkType != "https" {
		return fmt.Errorf("the uptime check type must be one of ping, http, or https, got %s", checkType)
	}

	checkRegions, err := c.Doit.GetStringSlice(c.NS, doctl.ArgUptimeCheckRegions)
	if err != nil {
		return err
	}

	uptimeCheck, err := c.UptimeChecks().Update(checkID, &godo.UpdateUptimeCheckRequest{
		Name:    checkName,
		Type:    checkType,
		Target:  checkTarget,
		Regions: checkRegions,
	})
	if err != nil {
		return err
	}

	item := &displayers.UptimeCheck{UptimeChecks: []do.UptimeCheck{*uptimeCheck}}
	return c.Display(item)
}

// RunUptimeChecksDelete deletes an uptime check by ID.
func RunUptimeChecksDelete(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	return c.UptimeChecks().Delete(c.Args[0])
}
