/*
Copyright 2024 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// BYOIPPrefix creates the command hierarchy for byoip prefixes.
func BYOIPPrefix() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "byoip-prefix",
			Short:   "Display commands to manage byoip prefixes",
			Long:    `The sub-commands of ` + "`" + `doctl network byoip-prefix` + "`" + ` manage byoip prefixes. Bring Your Own IP(BYOIP) Prefixes can be created and the IP addresses under that prefix can be used to assign to resources. BYOIP Prefixes are bound to the regions they are created in.`,
			Aliases: []string{"byoip-prefixes"},
		},
	}

	cmdBYOIPPrefixCreate := CmdBuilder(cmd, RunBYOIPPrefixCreate, "create", "Create a new BYOIP Prefix", `Creates a new BYOIP Prefix.
BYOIP Prefixes can be held in the region they were created in on your account.`, Writer,
		aliasOpt("c"), displayerType(&displayers.BYOIPPrefix{}))
	AddStringFlag(cmdBYOIPPrefixCreate, doctl.ArgRegionSlug, "", "", "The region where to create the byoip prefix")
	AddStringFlag(cmdBYOIPPrefixCreate, doctl.ArgPrefix, "", "", "The prefix to create")
	AddStringFlag(cmdBYOIPPrefixCreate, doctl.ArgSignature, "", "", "The signature for the prefix")
	cmdBYOIPPrefixCreate.Example = `The following example creates a byoip prefix in the ` + "`" + `nyc1` + "`" + ` region: doctl network byoip-prefix create --region nyc1 --prefix "10.1.1.1/24" --signature "signature"`

	cmdBYOIPPrefixGet := CmdBuilder(cmd, RunBYOIPPrefixGet, "get <prefix-uuid>", "Retrieve information about a byoip prefix", "Retrieves detailed information about a BYOIP Prefix", Writer,
		aliasOpt("g"), displayerType(&displayers.ReservedIPv6{}))
	cmdBYOIPPrefixGet.Example = `The following example retrieves information about the byoip prefix ` + "`" + `5ae545c4-0ac4-42bb-9de5-8eca3d17f1c0` + "`" + `: doctl network byoip-prefix get 5ae545c4-0ac4-42bb-9de5-8eca3d17f1c0`

	cmdRunBYOIPPrefixDelete := CmdBuilder(cmd, RunBYOIPPrefixDelete, "delete <prefix-uuid>", "Permanently delete a BYOIP Prefix", "Permanently deletes a BYOIP Prefix. This is irreversible and it needs all IPs of the prefix to be unassigned", Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunBYOIPPrefixDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the BYOIP Prefix without confirmation")
	cmdRunBYOIPPrefixDelete.Example = `The following example deletes the byoip prefix ` + "`" + `5ae545c4-0ac4-42bb-9de5-8eca3d17f1c0` + "`" + `: doctl network byoip-prefix delete 5ae545c4-0ac4-42bb-9de5-8eca3d17f1c0`

	cmdRunBYOIPPrefixList := CmdBuilder(cmd, RunBYOIPPrefixList, "list", "List all BYOIP Prefixes on your account", "Retrieves a list of all the BYOIP Prefixes in your account.", Writer,
		aliasOpt("ls"), displayerType(&displayers.BYOIPPrefix{}))
	cmdRunBYOIPPrefixList.Example = `The following example lists all byoip prefixes: doctl network byoip-prefix list`

	cmdRunBYOIPPrefixResourcesList := CmdBuilder(cmd, RunBYOIPPrefixResourcesGet, "resource", "List all the Resource for a BYOIP Prefix", "Retrieves a list of all the Resources in your prefix.", Writer,
		aliasOpt("resources"), displayerType(&displayers.BYOIPPrefixResource{}))
	cmdRunBYOIPPrefixResourcesList.Example = `The following example lists all resources in a byoip prefix: doctl network byoip-prefix resources 5ae545c4-0ac4-42bb-9de5-8eca3d17f1c0`

	return cmd
}

// RunBYOIPPrefixCreate runs byoip prefix create.
func RunBYOIPPrefixCreate(c *CmdConfig) error {
	bps := c.BYOIPPrefixes()

	// ignore errors since we don't know which one is valid
	region, _ := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)

	if region == "" {
		return doctl.NewMissingArgsErr("Region cannot be empty")
	}

	prefix, _ := c.Doit.GetString(c.NS, doctl.ArgPrefix)

	if prefix == "" {
		return doctl.NewMissingArgsErr("Prefix cannot be empty")
	}

	signature, _ := c.Doit.GetString(c.NS, doctl.ArgSignature)

	if signature == "" {
		return doctl.NewMissingArgsErr("Signature cannot be empty")
	}

	req := &godo.BYOIPPrefixCreateReq{
		Region:    region,
		Prefix:    prefix,
		Signature: signature,
	}

	bpCreateResp, err := bps.Create(req)
	if err != nil {
		return err
	}

	item := &displayers.BYOIPPrefixCreate{
		BYOIPPrefixCreate: do.BYOIPPrefixCreate{
			BYOIPPrefixCreateResp: bpCreateResp,
		},
	}

	return c.Display(item)
}

// RunBYOIPPrefixGet retrieves a byoip prefix details.
func RunBYOIPPrefixGet(c *CmdConfig) error {
	bp := c.BYOIPPrefixes()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	prefixUUID := c.Args[0]

	if len(prefixUUID) < 1 {
		return errors.New("invalid BYOIP Prefix UUID")
	}

	_, err = uuid.Parse(prefixUUID) // Validate UUID format
	if err != nil {
		return fmt.Errorf("invalid BYOIP Prefix UUID: %s", prefixUUID)
	}

	byoipPrefix, err := bp.Get(prefixUUID)
	if err != nil {
		return err
	}

	item := &displayers.BYOIPPrefix{BYOIPPrefixes: do.BYOIPPrefixes{
		*byoipPrefix,
	}}
	return c.Display(item)
}

// RunBYOIPPrefixDelete runs byoip prefix delete.
func RunBYOIPPrefixDelete(c *CmdConfig) error {
	bp := c.BYOIPPrefixes()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("byoip prefix", 1) == nil {
		prefixUUID := c.Args[0]
		return bp.Delete(prefixUUID)
	}

	return errOperationAborted
}

// RunBYOIPPrefixList runs byoip prefix list.
func RunBYOIPPrefixList(c *CmdConfig) error {
	bp := c.BYOIPPrefixes()

	list, err := bp.List()
	if err != nil {
		return err
	}

	items := &displayers.BYOIPPrefix{BYOIPPrefixes: list}

	return c.Display(items)
}

// RunBYOIPPrefixResourcesGet runs byoip prefix resources.
func RunBYOIPPrefixResourcesGet(c *CmdConfig) error {
	bp := c.BYOIPPrefixes()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	prefixUUID := c.Args[0]

	if len(prefixUUID) < 1 {
		return errors.New("Invalid BYOIP Prefix UUID")
	}

	_, err = uuid.Parse(prefixUUID) // Validate UUID format
	if err != nil {
		return fmt.Errorf("Invalid BYOIP Prefix UUID: %s", prefixUUID)
	}

	list, err := bp.GetPrefixResources(prefixUUID)
	if err != nil {
		return err
	}

	items := &displayers.BYOIPPrefixResource{BYOIPPrefixResource: list}

	return c.Display(items)
}
