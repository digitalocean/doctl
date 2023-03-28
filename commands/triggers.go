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
	"context"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

// Triggers generates the serverless 'triggers' subtree for addition to the doctl command
func Triggers() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "triggers",
			Short: "Manage triggers associated with your functions",
			Long: `When Functions are deployed by ` + "`" + `doctl serverless deploy` + "`" + `, they may have associated triggers.
The subcommands of ` + "`" + `doctl serverless triggers` + "`" + ` are used to list and inspect
triggers.  Each trigger has an event source type, and invokes its associated function
when events from that source type occur.  Currently, only the ` + "`" + `scheduler` + "`" + ` event source type is supported.`,
			Aliases: []string{"trigger", "trig"},
			Hidden:  true, // trigger support uses APIs that are not yet universally available
		},
	}
	list := CmdBuilder(cmd, RunTriggersList, "list", "Lists your triggers",
		`Use `+"`"+`doctl serverless triggers list`+"`"+` to list your triggers.`,
		Writer, aliasOpt("ls"), displayerType(&displayers.Triggers{}))
	AddStringFlag(list, "function", "f", "", "list only triggers for the chosen function")

	CmdBuilder(cmd, RunTriggerToggle(true), "enable <triggerName>",
		"Enable a trigger", "Use `doctl serverless triggers enable <triggerName>` to enable a trigger",
		Writer, displayerType(&displayers.Triggers{}))
	CmdBuilder(cmd, RunTriggerToggle(false), "disable <triggerName>",
		"Disable a trigger", "Use `doctl serverless triggers disable <triggerName>` to disable a trigger",
		Writer, displayerType(&displayers.Triggers{}))

	CmdBuilder(cmd, RunTriggersGet, "get <triggerName>", "Get the details for a trigger",
		`Use `+"`"+`doctl serverless triggers get <triggerName>`+"`"+` for details about <triggerName>.`,
		Writer, displayerType(&displayers.Triggers{}))

	return cmd
}

// RunTriggersList provides the logic for 'doctl sls trig list'
func RunTriggersList(c *CmdConfig) error {
	if len(c.Args) > 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	fcn, _ := c.Doit.GetString(c.NS, "function")
	list, err := c.Serverless().ListTriggers(context.TODO(), fcn)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Triggers{List: list})
}

// RunTriggersGet provides the logic for 'doctl sls trig get'
func RunTriggersGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	trigger, err := c.Serverless().GetTrigger(context.TODO(), c.Args[0])
	if err != nil {
		return err
	}

	return c.Display(&displayers.Triggers{List: []do.ServerlessTrigger{trigger}})
}

// RunTriggerToggle provides the logic for 'doctl sls trig enabled/disabled'
func RunTriggerToggle(isEnabled bool) func(*CmdConfig) error {
	return func(c *CmdConfig) error {
		err := ensureOneArg(c)

		if err != nil {
			return err
		}

		trigger, err := c.Serverless().UpdateTrigger(context.TODO(), c.Args[0], &do.UpdateTriggerRequest{IsEnabled: isEnabled})

		if err != nil {
			return err
		}

		return c.Display(&displayers.Triggers{List: []do.ServerlessTrigger{trigger}})
	}
}

// cleanTriggers is the subroutine of undeploy that removes all the triggers of a namespace
func cleanTriggers(c *CmdConfig) error {
	sls := c.Serverless()
	ctx := context.TODO()
	list, err := sls.ListTriggers(ctx, "")
	if err != nil {
		return err
	}
	for _, trig := range list {
		err = sls.DeleteTrigger(ctx, trig.Name)
		if err != nil {
			return err
		}
	}
	return nil
}
