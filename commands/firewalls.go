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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"

	"github.com/spf13/cobra"
)

// Firewall creates the firewall command.
func Firewall() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "firewall",
			Short: "frovides commands to manage Cloud Firewalls",
			Long:  `The sub-commands of 'doctl compute firewall' manage DigitalOcean Cloud Firewalls.

Cloud Firewalls provide the ability to restrict network access to and from a Droplet allowing you to define which ports will accept inbound or outbound connections. With these commands you can list, create, or delete Cloud Firewalls as well as modify access rules.

A Firewall's 'inbound_rules' and 'outbound_rules' attributes will have arrays of objects as their values. These objects will contain the standard attributes of their associated types which can be found below.

Inbound access rules specify the protocol (TCP, UDP, or ICMP), ports, and sources for inbound traffic that will be allowed through the Firewall to the target Droplets. The 'ports' attribute may contain a single port, a range of ports (e.g. "8000-9000"), or "all" to allow traffic on all ports for the specified protocol. The 'sources' attribute will contain an object specifying a whitelist of sources from which traffic will be accepted.`,
		},
	}
	fwDetail := `

	- The firewall's ID
	- The firewal's name
	- The status of the firewall. This can be "waiting", "succeeded", or "failed".
	- The firewall's creation date, in ISO8601 combined date and time format.
	- Any pending changes to the firewall. These can be "droplet_id", "removing", and "status". 
	  When empty, all changes have been successfully applied.
	- The inbound rules for the firewall. 
	- The outbound rules for the firewall. 
	- The IDs of Droplets assigned to the firewall.
	- The tags assigned to the firewall.
`
	CmdBuilderWithDocs(cmd, RunFirewallGet, "get <id>", "Retrive information about a Cloud Firewall",`Use this command to get information about an existing Cloud Firewall, including:`+fwDetail, Writer, aliasOpt("g"), displayerType(&displayers.Firewall{}))

	cmdFirewallCreate := CmdBuilderWithDocs(cmd, RunFirewallCreate, "create", "Create a new Cloud Firewall", `Use this command to create a Cloud Firewall. This command must contain at least one inbound or outbound access rule.`, Writer, aliasOpt("c"), displayerType(&displayers.Firewall{}))
	AddStringFlag(cmdFirewallCreate, doctl.ArgFirewallName, "", "", "firewall name", requiredOpt())
	AddStringFlag(cmdFirewallCreate, doctl.ArgInboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,droplet_id:1,droplet_id:2,tag:frontend, use quoted string of space-separated values for multiple rules")
	AddStringFlag(cmdFirewallCreate, doctl.ArgOutboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,address:0.0.0.0/0, use quoted string of space-separated values for multiple rules")
	AddStringSliceFlag(cmdFirewallCreate, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example value: 123,456")
	AddStringSliceFlag(cmdFirewallCreate, doctl.ArgTagNames, "", []string{}, "comma-separated list of tag names, example value: frontend,backend")

	cmdFirewallUpdate := CmdBuilderWithDocs(cmd, RunFirewallUpdate, "update <id>", "Update a Cloud Firewall's configuration",`Use this command to update the configuration of an existing Cloud Firewall. The request should contain a full representation of the Firewall including existing attributes. Note that any attributes that are not provided will be reset to their default values.`, Writer, aliasOpt("u"), displayerType(&displayers.Firewall{}))
	AddStringFlag(cmdFirewallUpdate, doctl.ArgFirewallName, "", "", "firewall name", requiredOpt())
	AddStringFlag(cmdFirewallUpdate, doctl.ArgInboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,droplet_id:123, use quoted string of space-separated values for multiple rules")
	AddStringFlag(cmdFirewallUpdate, doctl.ArgOutboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,address:0.0.0.0/0, use quoted string of space-separated values for multiple rules")
	AddStringSliceFlag(cmdFirewallUpdate, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example value: 123,456")
	AddStringSliceFlag(cmdFirewallUpdate, doctl.ArgTagNames, "", []string{}, "comma-separated list of tag names, example value: frontend,backend")

	CmdBuilderWithDocs(cmd, RunFirewallList, "list", "List the Cloud Firewalls on your account", `Use this command to retrieve a list of Cloud Firewalls.`,Writer, aliasOpt("ls"), displayerType(&displayers.Firewall{}))

	CmdBuilderWithDocs(cmd, RunFirewallListByDroplet, "list-by-droplet <droplet_id>","",`Use this command to list Cloud Firewalls by the ID of a Droplet assigned to the firewall.`, Writer, displayerType(&displayers.Firewall{}))

	cmdRunRecordDelete := CmdBuilderWithDocs(cmd, RunFirewallDelete, "delete <id>...", "Permanently delete a Cloud Firewall", `Use this command to permanently delete a Cloud Firewall. This is irreversable, but does not delete any Droplets assigned to the Cloud Firewall.`,Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunRecordDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force firewall delete")

	cmdAddDroplets := CmdBuilderWithDocs(cmd, RunFirewallAddDroplets, "add-droplets <id>","Add Droplets to a Cloud Firewall",`Use this command to add Droplets to the Cloud Firewall.`, Writer)
	AddStringSliceFlag(cmdAddDroplets, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example valus: 123,456")

	cmdRemoveDroplets := CmdBuilderWithDocs(cmd, RunFirewallRemoveDroplets, "remove-droplets <id>", "Remove Droplets from a Cloud Firewall",`Use this command to remove Droplets from a Cloud Firewall.`, Writer)
	AddStringSliceFlag(cmdRemoveDroplets, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example value: 123,456")

	cmdAddTags := CmdBuilderWithDocs(cmd, RunFirewallAddTags, "add-tags <id>", "Add tags to a Cloud Firewall", `Use this command to add tags to a Cloud Firewall. This adds all assets using that tag to the firewall`,Writer)
	AddStringSliceFlag(cmdAddTags, doctl.ArgTagNames, "", []string{}, "comma-separated list of tag names, example valus: frontend,backend")

	cmdRemoveTags := CmdBuilderWithDocs(cmd, RunFirewallRemoveTags, "remove-tags <id>", "Remove tags from a Cloud Firewall",`Use this command to remove tags from a Cloud Firewall. This removes all assets using that tag from the firewall.`, Writer)
	AddStringSliceFlag(cmdRemoveTags, doctl.ArgTagNames, "", []string{}, "comma-separated list of tag names, example value: frontend,backend")

	cmdAddRules := CmdBuilderWithDocs(cmd, RunFirewallAddRules, "add-rules <id>", "Add inbound or outbound rules to a Cloud Firewall",`Use this command to add inbound or outbound rules to a Cloud Firewall.`, Writer)
	AddStringFlag(cmdAddRules, doctl.ArgInboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,address:0.0.0.0/0, use quoted string of space-separated values for multiple rules")
	AddStringFlag(cmdAddRules, doctl.ArgOutboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,address:0.0.0.0/0, use quoted string of space-separated values for multiple rules")

	cmdRemoveRules := CmdBuilderWithDocs(cmd, RunFirewallRemoveRules, "remove-rules <id>", "Remove inbound or outbound rules from a Cloud Firewall",`Use this command to remove inbound or outbound rules from a Cloud Firewall.`, Writer)
	AddStringFlag(cmdRemoveRules, doctl.ArgInboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,load_balancer_uid:lb-uuid, use quoted string of space-separated values for multiple rules")
	AddStringFlag(cmdRemoveRules, doctl.ArgOutboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,address:0.0.0.0/0, use quoted string of space-separated values for multiple rules")

	return cmd
}

// RunFirewallGet retrieves an existing Firewall by its identifier.
func RunFirewallGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]

	fs := c.Firewalls()
	f, err := fs.Get(id)
	if err != nil {
		return err
	}

	item := &displayers.Firewall{Firewalls: do.Firewalls{*f}}
	return c.Display(item)
}

// RunFirewallCreate creates a new Firewall with a given configuration.
func RunFirewallCreate(c *CmdConfig) error {
	r := new(godo.FirewallRequest)
	if err := buildFirewallRequestFromArgs(c, r); err != nil {
		return err
	}

	fs := c.Firewalls()
	f, err := fs.Create(r)
	if err != nil {
		return err
	}

	item := &displayers.Firewall{Firewalls: do.Firewalls{*f}}
	return c.Display(item)
}

// RunFirewallUpdate updates an existing Firewall with new configuration.
func RunFirewallUpdate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	fID := c.Args[0]

	r := new(godo.FirewallRequest)
	if err := buildFirewallRequestFromArgs(c, r); err != nil {
		return err
	}

	fs := c.Firewalls()
	f, err := fs.Update(fID, r)
	if err != nil {
		return err
	}

	item := &displayers.Firewall{Firewalls: do.Firewalls{*f}}
	return c.Display(item)
}

// RunFirewallList lists Firewalls.
func RunFirewallList(c *CmdConfig) error {
	fs := c.Firewalls()
	list, err := fs.List()
	if err != nil {
		return err
	}

	items := &displayers.Firewall{Firewalls: list}
	return c.Display(items)
}

// RunFirewallListByDroplet lists Firewalls for a given Droplet.
func RunFirewallListByDroplet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	dID, err := strconv.Atoi(c.Args[0])
	if err != nil {
		return fmt.Errorf("invalid droplet id: [%v]", c.Args[0])
	}

	fs := c.Firewalls()
	list, err := fs.ListByDroplet(dID)
	if err != nil {
		return err
	}

	items := &displayers.Firewall{Firewalls: list}
	return c.Display(items)
}

// RunFirewallDelete deletes a Firewall by its identifier.
func RunFirewallDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	fs := c.Firewalls()
	if force || AskForConfirm(fmt.Sprintf("delete %d firewall(s)", len(c.Args))) == nil {
		for _, id := range c.Args {
			if err := fs.Delete(id); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("operation aborted")
	}

	return nil
}

// RunFirewallAddDroplets adds droplets to a Firewall.
func RunFirewallAddDroplets(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	fID := c.Args[0]

	dropletIDsList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgDropletIDs)
	if err != nil {
		return err
	}

	dropletIDs, err := extractDropletIDs(dropletIDsList)
	if err != nil {
		return err
	}

	return c.Firewalls().AddDroplets(fID, dropletIDs...)
}

// RunFirewallRemoveDroplets removes droplets from a Firewall.
func RunFirewallRemoveDroplets(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	fID := c.Args[0]

	dropletIDsList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgDropletIDs)
	if err != nil {
		return err
	}

	dropletIDs, err := extractDropletIDs(dropletIDsList)
	if err != nil {
		return err
	}

	return c.Firewalls().RemoveDroplets(fID, dropletIDs...)
}

// RunFirewallAddTags adds tags to a Firewall.
func RunFirewallAddTags(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	fID := c.Args[0]

	tagList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTagNames)
	if err != nil {
		return err
	}

	return c.Firewalls().AddTags(fID, tagList...)
}

// RunFirewallRemoveTags removes tags from a Firewall.
func RunFirewallRemoveTags(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	fID := c.Args[0]

	tagList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTagNames)
	if err != nil {
		return err
	}

	return c.Firewalls().RemoveTags(fID, tagList...)
}

// RunFirewallAddRules adds rules to a Firewall.
func RunFirewallAddRules(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	fID := c.Args[0]

	rr := new(godo.FirewallRulesRequest)
	if err := buildFirewallRulesRequestFromArgs(c, rr); err != nil {
		return err
	}

	return c.Firewalls().AddRules(fID, rr)
}

// RunFirewallRemoveRules removes rules from a Firewall.
func RunFirewallRemoveRules(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	fID := c.Args[0]

	rr := new(godo.FirewallRulesRequest)
	if err := buildFirewallRulesRequestFromArgs(c, rr); err != nil {
		return err
	}

	return c.Firewalls().RemoveRules(fID, rr)
}

func buildFirewallRequestFromArgs(c *CmdConfig, r *godo.FirewallRequest) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgFirewallName)
	if err != nil {
		return err
	}
	r.Name = name

	ira, err := c.Doit.GetString(c.NS, doctl.ArgInboundRules)
	if err != nil {
		return err
	}

	inboundRules, err := extractInboundRules(ira)
	if err != nil {
		return err
	}
	r.InboundRules = inboundRules

	ora, err := c.Doit.GetString(c.NS, doctl.ArgOutboundRules)
	if err != nil {
		return err
	}

	outboundRules, err := extractOutboundRules(ora)
	if err != nil {
		return err
	}
	r.OutboundRules = outboundRules

	dropletIDsList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgDropletIDs)
	if err != nil {
		return err
	}

	dropletIDs, err := extractDropletIDs(dropletIDsList)
	if err != nil {
		return err
	}
	r.DropletIDs = dropletIDs

	tagsList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTagNames)
	if err != nil {
		return err
	}
	r.Tags = tagsList

	return nil
}

func buildFirewallRulesRequestFromArgs(c *CmdConfig, rr *godo.FirewallRulesRequest) error {
	ira, err := c.Doit.GetString(c.NS, doctl.ArgInboundRules)
	if err != nil {
		return err
	}

	inboundRules, err := extractInboundRules(ira)
	if err != nil {
		return err
	}
	rr.InboundRules = inboundRules

	ora, err := c.Doit.GetString(c.NS, doctl.ArgOutboundRules)
	if err != nil {
		return err
	}

	outboundRules, err := extractOutboundRules(ora)
	if err != nil {
		return err
	}
	rr.OutboundRules = outboundRules

	return nil
}

func extractInboundRules(s string) (rules []godo.InboundRule, err error) {
	if len(s) == 0 {
		return nil, nil
	}

	list := strings.Split(s, " ")
	for _, v := range list {
		rule, err := extractRule(v, "sources")
		if err != nil {
			return nil, err
		}
		mr, _ := json.Marshal(rule)
		ir := &godo.InboundRule{}
		json.Unmarshal(mr, ir)
		rules = append(rules, *ir)
	}

	return rules, nil
}

func extractOutboundRules(s string) (rules []godo.OutboundRule, err error) {
	if len(s) == 0 {
		return nil, nil
	}

	list := strings.Split(s, " ")
	for _, v := range list {
		rule, err := extractRule(v, "destinations")
		if err != nil {
			return nil, err
		}
		mr, _ := json.Marshal(rule)
		or := &godo.OutboundRule{}
		json.Unmarshal(mr, or)
		rules = append(rules, *or)
	}

	return rules, nil
}

func extractRule(ruleStr string, sd string) (map[string]interface{}, error) {
	rule := map[string]interface{}{}
	var dropletIDs []int
	var addresses, lbUIDs, tags []string

	kvs := strings.Split(ruleStr, ",")
	for _, v := range kvs {
		pair := strings.SplitN(v, ":", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("Unexpected input value [%v], must be a key:value pair", pair)
		}

		switch pair[0] {
		case "address":
			addresses = append(addresses, pair[1])
		case "droplet_id":
			i, err := strconv.Atoi(pair[1])
			if err != nil {
				return nil, fmt.Errorf("Provided value [%v] for droplet id is not of type int", pair[0])
			}
			dropletIDs = append(dropletIDs, i)
		case "load_balancer_uid":
			lbUIDs = append(lbUIDs, pair[1])
		case "tag":
			tags = append(tags, pair[1])
		default:
			rule[pair[0]] = pair[1]
		}
	}

	rule[sd] = map[string]interface{}{
		"addresses":          addresses,
		"droplet_ids":        dropletIDs,
		"load_balancer_uids": lbUIDs,
		"tags":               tags,
	}

	return rule, nil
}
