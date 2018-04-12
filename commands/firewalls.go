/*
Copyright 2017 The Doctl Authors All rights reserved.
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
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"

	"github.com/spf13/cobra"
)

// Firewall creates the firewall command.
func Firewall() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "firewall",
			Short: "firewall commands",
			Long:  "firewall is used to access firewall commands",
		},
	}

	CmdBuilder(cmd, RunFirewallGet, "get <id>", "get firewall", Writer, aliasOpt("g"))

	cmdRecordCreate := CmdBuilder(cmd, RunFirewallCreate, "create", "create firewall", Writer, aliasOpt("c"))
	AddStringFlag(cmdRecordCreate, doctl.ArgFirewallName, "", "", "firewall name", requiredOpt())
	AddStringFlag(cmdRecordCreate, doctl.ArgInboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,droplet_id:1,droplet_id:2,tag:frontend, use quoted string of space-separated values for multiple rules")
	AddStringFlag(cmdRecordCreate, doctl.ArgOutboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,address:0.0.0.0/0, use quoted string of space-separated values for multiple rules")
	AddStringSliceFlag(cmdRecordCreate, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example value: 123,456")
	AddStringSliceFlag(cmdRecordCreate, doctl.ArgTagNames, "", []string{}, "comma-separated list of tag names, example value: frontend,backend")

	cmdRecordUpdate := CmdBuilder(cmd, RunFirewallUpdate, "update <id>", "update firewall", Writer, aliasOpt("u"))
	AddStringFlag(cmdRecordUpdate, doctl.ArgFirewallName, "", "", "firewall name", requiredOpt())
	AddStringFlag(cmdRecordUpdate, doctl.ArgInboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,droplet_id:123, use quoted string of space-separated values for multiple rules")
	AddStringFlag(cmdRecordUpdate, doctl.ArgOutboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,address:0.0.0.0/0, use quoted string of space-separated values for multiple rules")
	AddStringSliceFlag(cmdRecordUpdate, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example value: 123,456")
	AddStringSliceFlag(cmdRecordUpdate, doctl.ArgTagNames, "", []string{}, "comma-separated list of tag names, example value: frontend,backend")

	CmdBuilder(cmd, RunFirewallList, "list", "list firewalls", Writer, aliasOpt("ls"), displayerType(&firewall{}))

	CmdBuilder(cmd, RunFirewallListByDroplet, "list-by-droplet <droplet_id>", "list firewalls by droplet ID", Writer)

	cmdRunRecordDelete := CmdBuilder(cmd, RunFirewallDelete, "delete <id>", "delete firewall", Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunRecordDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force firewall delete")

	cmdAddDroplets := CmdBuilder(cmd, RunFirewallAddDroplets, "add-droplets <id>", "add droplets to the firewall", Writer)
	AddStringSliceFlag(cmdAddDroplets, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example valus: 123,456")

	cmdRemoveDroplets := CmdBuilder(cmd, RunFirewallRemoveDroplets, "remove-droplets <id>", "remove droplets from the firewall", Writer)
	AddStringSliceFlag(cmdRemoveDroplets, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example value: 123,456")

	cmdAddTags := CmdBuilder(cmd, RunFirewallAddTags, "add-tags <id>", "add tags to the firewall", Writer)
	AddStringSliceFlag(cmdAddTags, doctl.ArgTagNames, "", []string{}, "comma-separated list of tag names, example valus: frontend,backend")

	cmdRemoveTags := CmdBuilder(cmd, RunFirewallRemoveTags, "remove-tags <id>", "remove tags from the firewall", Writer)
	AddStringSliceFlag(cmdRemoveTags, doctl.ArgTagNames, "", []string{}, "comma-separated list of tag names, example value: frontend,backend")

	cmdAddRules := CmdBuilder(cmd, RunFirewallAddRules, "add-rules <id>", "add inbound/outbound rules to the firewall", Writer)
	AddStringFlag(cmdAddRules, doctl.ArgInboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,address:0.0.0.0/0, use quoted string of space-separated values for multiple rules")
	AddStringFlag(cmdAddRules, doctl.ArgOutboundRules, "", "", "comma-separated key:value list, example value: protocol:tcp,ports:22,address:0.0.0.0/0, use quoted string of space-separated values for multiple rules")

	cmdRemoveRules := CmdBuilder(cmd, RunFirewallRemoveRules, "remove-rules <id>", "remove inbound/outbound rules from the firewall", Writer)
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

	item := &firewall{firewalls: do.Firewalls{*f}}
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

	item := &firewall{firewalls: do.Firewalls{*f}}
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

	item := &firewall{firewalls: do.Firewalls{*f}}
	return c.Display(item)
}

// RunFirewallList lists Firewalls.
func RunFirewallList(c *CmdConfig) error {
	fs := c.Firewalls()
	list, err := fs.List()
	if err != nil {
		return err
	}

	items := &firewall{firewalls: list}
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

	items := &firewall{firewalls: list}
	return c.Display(items)
}

// RunFirewallDelete deletes a Firewall by its identifier.
func RunFirewallDelete(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	fID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirm("delete this Firewall") == nil {
		fs := c.Firewalls()
		if err := fs.Delete(fID); err != nil {
			return err
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
