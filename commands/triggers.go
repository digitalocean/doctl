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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
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
			Aliases: []string{"trig"},
			Hidden:  true, // trigger support is experimental and currently using a temporary prototype API
		},
	}
	list := CmdBuilder(cmd, RunTriggersList, "list", "Lists your triggers",
		`Use `+"`"+`doctl serverless triggers list`+"`"+` to list your triggers.`,
		Writer, displayerType(&displayers.Triggers{}))
	AddStringFlag(list, "function", "f", "", "list only triggers for the chosen function")

	CmdBuilder(cmd, RunTriggersGet, "get <triggerName>", "Get the details for a trigger",
		`Use `+"`"+`doctl serverless triggers get <triggerName>`+"`"+` for details about <triggerName>.`,
		Writer)

	CmdBuilder(cmd, RunTriggersEnable, "enable <triggerName>", "Enable a trigger",
		`Use `+"`"+`doctl serverless triggers enable <triggerName>`+"`"+` to enable the trigger <triggerName>.`,
		Writer, hiddenCmd())

	CmdBuilder(cmd, RunTriggersDisable, "disable <triggerName>", "Disable a trigger",
		`Use `+"`"+`doctl serverless triggers disable <triggerName>`+"`"+` to disable the trigger <triggerName>.
When a trigger is disable it does not invoke its target function`,
		Writer, hiddenCmd())

	CmdBuilder(cmd, RunTriggersFire, "fire <triggerName>", "Test-fire a trigger",
		`Use `+"`"+`doctl serverless triggers fire <triggerName>`+"`"+` to invoke the function associated with a trigger using
the same method and parameters that an event occurence would use`,
		Writer, hiddenCmd())

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
	json, err := json.MarshalIndent(&trigger, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(json))
	return nil
}

// RunTriggersEnable provides the logic for 'doctl sls trig enable'
// This command is hidden.  It will work (if you know about it) when using the prototype API
// but will not be supported in the real API at first.
func RunTriggersEnable(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	newContent, err := c.Serverless().SetTriggerEnablement(context.TODO(), c.Args[0], true)
	if err != nil {
		return err
	}
	if !newContent.Enabled {
		return errors.New("failed to enable trigger (cause unknown)")
	}
	return nil
}

// RunTriggersDisable provides the logic for 'doctl sls trig disable'
// This command is hidden.  It will work (if you know about it) when using the prototype API
// but will not be supported in the real API at first.
func RunTriggersDisable(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	newContent, err := c.Serverless().SetTriggerEnablement(context.TODO(), c.Args[0], false)
	if err != nil {
		return err
	}
	if newContent.Enabled {
		return errors.New("failed to disable trigger (cause unknown)")
	}
	return nil
}

// RunTriggersFire provides the logic for 'doctl sls trig fire'
// This command is hidden.  It will work (if you know about it) when using the prototype API
// but will not be supported in the real API at first.
func RunTriggersFire(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	return c.Serverless().FireTrigger(context.TODO(), c.Args[0])
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