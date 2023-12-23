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
			Short: "Display commands to manage cloud firewalls",
			Long: `The sub-commands of ` + "`" + `doctl compute firewall` + "`" + ` manage DigitalOcean cloud firewalls.

Cloud firewalls allow you to restrict network access to and from a Droplet by defining which ports accept inbound or outbound connections. With these commands, you can list, create, or delete Cloud firewalls, as well as modify access rules. 

Note: Cloud firewalls are not internal Droplet firewalls on Droplets, such as UFW or FirewallD.

A firewall's ` + "`" + `inbound_rules` + "`" + ` and ` + "`" + `outbound_rules` + "`" + ` attributes contain arrays of objects as their values. These objects contain the standard attributes of their associated types, which can be found below.

Inbound access rules specify the protocol (TCP, UDP, or ICMP), ports, and sources for inbound traffic that will be allowed through the Firewall to the target Droplets. The ` + "`" + `ports` + "`" + ` attribute may contain a single port, a range of ports (e.g. ` + "`" + `8000-9000` + "`" + `), or ` + "`" + `all` + "`" + ` to allow traffic on all ports for the specified protocol. The ` + "`" + `sources` + "`" + ` attribute will contain an object specifying a whitelist of sources from which traffic will be accepted.`,
		},
	}
	fwDetail := `

- The firewall's UUID
- The firewall's name
- The status of the firewall. Possible values: ` + "`" + `waiting` + "`" + `, ` + "`" + `succeeded` + "`" + `, ` + "`" + `failed` + "`" + `.
- The firewall's creation date, in ISO8601 combined date and time format.
- Any pending changes to the firewall. Possible values:` + "`" + `droplet_id` + "`" + `, ` + "`" + `removing` + "`" + `, ` + "`" + `status` + "`" + `.
	  When empty, all changes have been successfully applied.
- The inbound rules for the firewall
- The outbound rules for the firewall
- The IDs of Droplets assigned to the firewall
- The tags assigned to the firewall
`
	inboundRulesTxt := `A comma-separated key-value list that defines an inbound rule. The rule must define a communication protocol, a port number, and a traffic source location, such as a Droplet ID, IP address, or a tag. For example, the following rule defines that resources can only receive TCP traffic on port 22 from addresses in the specified CIDR: ` + "`" + `protocol:tcp,ports:22,address:192.0.2.0/24` + "`" + `. 
	
Available source keys are: ` + "`" + `address` + "`" + `, ` + "`" + `droplet_id` + "`" + `, ` + "`" + `load_balancer_uid` + "`" + `, ` + "`" + `kubernetes_id` + "`" + `, and ` + "`" + `tag` + "`" + `. 

Use a quoted string of space-separated values for multiple rules.`
	outboundRulesTxt := `A comma-separate key-value list that defines an outbound rule. The rule must define a communication protocol, a port number, and a destination location, such as a Droplet ID, IP address, or a tag. For example, the following rule defines that the firewall only allows traffic to be sent to port 22 of any IPv4 address on the internet: ` + "`" + `protocol:tcp,ports:22,address:0.0.0.0/0` + "`" + `.

Available destination keys are: ` + "`" + `address` + "`" + `, ` + "`" + `droplet_id` + "`" + `, ` + "`" + `load_balancer_uid` + "`" + `, ` + "`" + `kubernetes_id` + "`" + `, and ` + "`" + `tag` + "`" + `. 

Use a quoted string of space-separated values for multiple rules.`
	dropletIDRulesTxt := "A comma-separated list of Droplet IDs to place behind the cloud firewall, for example: `386734086,391669331`"
	tagNameRulesTxt := "A comma-separated list of tag names to apply to the cloud firewall, for example: `frontend,backend`"

	cmdFirewallGet := CmdBuilder(cmd, RunFirewallGet, "get <id>", "Retrieve information about a cloud firewall", `Retrieves information about an existing cloud firewall, including:`+fwDetail, Writer, aliasOpt("g"), displayerType(&displayers.Firewall{}))
	cmdFirewallGet.Example = `The following example retrieves information about the cloud firewall with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute firewall get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdFirewallCreate := CmdBuilder(cmd, RunFirewallCreate, "create", "Create a new cloud firewall", `Creates a cloud firewall. This command must contain at least one inbound or outbound access rule.`, Writer, aliasOpt("c"), displayerType(&displayers.Firewall{}))
	AddStringFlag(cmdFirewallCreate, doctl.ArgFirewallName, "", "", "The firewall's name", requiredOpt())
	AddStringFlag(cmdFirewallCreate, doctl.ArgInboundRules, "", "", inboundRulesTxt)
	AddStringFlag(cmdFirewallCreate, doctl.ArgOutboundRules, "", "", outboundRulesTxt)
	AddStringSliceFlag(cmdFirewallCreate, doctl.ArgDropletIDs, "", []string{}, dropletIDRulesTxt)
	AddStringSliceFlag(cmdFirewallCreate, doctl.ArgTagNames, "", []string{}, tagNameRulesTxt)
	cmdFirewallCreate.Example = `The following example creates a cloud firewall named ` + "`" + `example-firewall` + "`" + ` that contains an inbound rule and an outbound rule and applies them to the specified Droplet: doctl compute firewall create --name "example-firewall" --inbound-rules "protocol:tcp,ports:22,droplet_id:386734086" --outbound-rules "protocol:tcp,ports:22,address:0.0.0.0/0" --droplet-ids "386734086,391669331"`

	cmdFirewallUpdate := CmdBuilder(cmd, RunFirewallUpdate, "update <id>", "Update a cloud firewall's configuration", `Updates the configuration of an existing cloud firewall. The request should contain a full representation of the firewall, including existing attributes. Any attributes that are not provided are reset to their default values.`, Writer, aliasOpt("u"), displayerType(&displayers.Firewall{}))
	AddStringFlag(cmdFirewallUpdate, doctl.ArgFirewallName, "", "", "The firewall's name", requiredOpt())
	AddStringFlag(cmdFirewallUpdate, doctl.ArgInboundRules, "", "", inboundRulesTxt)
	AddStringFlag(cmdFirewallUpdate, doctl.ArgOutboundRules, "", "", outboundRulesTxt)
	AddStringSliceFlag(cmdFirewallUpdate, doctl.ArgDropletIDs, "", []string{}, dropletIDRulesTxt)
	AddStringSliceFlag(cmdFirewallUpdate, doctl.ArgTagNames, "", []string{}, tagNameRulesTxt)
	cmdFirewallUpdate.Example = `The following example updates a cloud firewall named ` + "`" + `example-firewall` + "`" + ` that contains an inbound rule and an outbound rule and applies them to the specified Droplet: doctl compute firewall update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name "example-firewall" --inbound-rules "protocol:tcp,ports:22,droplet_id:386734086" --outbound-rules "protocol:tcp,ports:22,address:0.0.0.0/0" --droplet-ids "386734086,391669331"`

	cmdFirewallList := CmdBuilder(cmd, RunFirewallList, "list", "List the cloud firewalls on your account", `Retrieves a list of cloud firewalls on your account.`, Writer, aliasOpt("ls"), displayerType(&displayers.Firewall{}))
	cmdFirewallList.Example = `The following example lists all cloud firewalls on your account and uses the ` + "`" + `--format` + "`" + ` flag to return only the ID, name and inbound rules for each firewall: doctl compute firewall list --format ID,Name,InboundRules`

	cmdirewallListByDroplet := CmdBuilder(cmd, RunFirewallListByDroplet, "list-by-droplet <droplet_id>", "List firewalls by Droplet", `Lists the cloud firewalls assigned to a Droplet.`, Writer, displayerType(&displayers.Firewall{}))
	cmdirewallListByDroplet.Example = `The following example lists all cloud firewalls assigned to the Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute firewall list-by-droplet 386734086`

	cmdRunRecordDelete := CmdBuilder(cmd, RunFirewallDelete, "delete <id>...", "Permanently delete a cloud firewall", `Permanently deletes a cloud firewall. This is irreversible, but does not delete any Droplets assigned to the cloud firewall.`, Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunRecordDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the firewall without a confirmation prompt")
	cmdRunRecordDelete.Example = `The following example deletes a cloud firewall with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute firewall delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdAddDroplets := CmdBuilder(cmd, RunFirewallAddDroplets, "add-droplets <id>", "Add Droplets to a cloud firewall", `Assigns Droplets to a cloud firewall on your account.`, Writer)
	AddStringSliceFlag(cmdAddDroplets, doctl.ArgDropletIDs, "", []string{}, dropletIDRulesTxt)
	cmdAddDroplets.Example = `The following example assigns two Droplets to the cloud firewall with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute firewall add-droplets f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --droplet-ids "386734086,391669331"`

	cmdRemoveDroplets := CmdBuilder(cmd, RunFirewallRemoveDroplets, "remove-droplets <id>", "Remove Droplets from a cloud firewall", `Removes Droplets from a cloud firewall.`, Writer)
	AddStringSliceFlag(cmdRemoveDroplets, doctl.ArgDropletIDs, "", []string{}, dropletIDRulesTxt)
	cmdRemoveDroplets.Example = `The following example removes two Droplets from a cloud firewall with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute firewall remove-droplets f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --droplet-ids "386734086,391669331"`

	cmdAddTags := CmdBuilder(cmd, RunFirewallAddTags, "add-tags <id>", "Add tags to a cloud firewall", `Add tags to a cloud firewall. This adds all assets using that tag to the firewall.`, Writer)
	AddStringSliceFlag(cmdAddTags, doctl.ArgTagNames, "", []string{}, tagNameRulesTxt)
	cmdAddTags.Example = `The following example adds two tags to a cloud firewall with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute firewall add-tags f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --tag-names "frontend,backend"`

	cmdRemoveTags := CmdBuilder(cmd, RunFirewallRemoveTags, "remove-tags <id>", "Remove tags from a cloud firewall", `Removes tags from a cloud firewall. This removes all assets using that tag from the firewall.`, Writer)
	AddStringSliceFlag(cmdRemoveTags, doctl.ArgTagNames, "", []string{}, tagNameRulesTxt)
	cmdRemoveTags.Example = `The following example removes two tags from a cloud firewall with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute firewall remove-tags f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --tag-names "frontend,backend"`

	cmdAddRules := CmdBuilder(cmd, RunFirewallAddRules, "add-rules <id>", "Add inbound or outbound rules to a cloud firewall", `Add inbound or outbound rules to a cloud firewall.`, Writer)
	AddStringFlag(cmdAddRules, doctl.ArgInboundRules, "", "", inboundRulesTxt)
	AddStringFlag(cmdAddRules, doctl.ArgOutboundRules, "", "", outboundRulesTxt)
	cmdAddRules.Example = `The following example adds an inbound rule and an outbound rule to a cloud firewall with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute firewall add-rules f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --inbound-rules "protocol:tcp,ports:22,droplet_id:386734086" --outbound-rules "protocol:tcp,ports:22,address:0.0.0.0/0"`

	cmdRemoveRules := CmdBuilder(cmd, RunFirewallRemoveRules, "remove-rules <id>", "Remove inbound or outbound rules from a cloud firewall", `Remove inbound or outbound rules from a cloud firewall.`, Writer)
	AddStringFlag(cmdRemoveRules, doctl.ArgInboundRules, "", "", inboundRulesTxt)
	AddStringFlag(cmdRemoveRules, doctl.ArgOutboundRules, "", "", outboundRulesTxt)
	cmdRemoveRules.Example = `The following example removes an inbound rule and an outbound rule from a cloud firewall with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute firewall remove-rules f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --inbound-rules "protocol:tcp,ports:22,droplet_id:386734086" --outbound-rules "protocol:tcp,ports:22,address:0.0.0.0/0"`

	return cmd
}

// RunFirewallGet retrieves an existing Firewall by its identifier.
func RunFirewallGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
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
	err := ensureOneArg(c)
	if err != nil {
		return err
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
	if force || AskForConfirmDelete("firewall", len(c.Args)) == nil {
		for _, id := range c.Args {
			if err := fs.Delete(id); err != nil {
				return err
			}
		}
	} else {
		return errOperationAborted
	}

	return nil
}

// RunFirewallAddDroplets adds droplets to a Firewall.
func RunFirewallAddDroplets(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
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
	err := ensureOneArg(c)
	if err != nil {
		return err
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
	err := ensureOneArg(c)
	if err != nil {
		return err
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
	err := ensureOneArg(c)
	if err != nil {
		return err
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
	err := ensureOneArg(c)
	if err != nil {
		return err
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
	err := ensureOneArg(c)
	if err != nil {
		return err
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
		err = json.Unmarshal(mr, ir)
		if err != nil {
			return nil, err
		}
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
		err = json.Unmarshal(mr, or)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *or)
	}

	return rules, nil
}

func extractRule(ruleStr string, sd string) (map[string]any, error) {
	rule := map[string]any{}
	var dropletIDs []int
	var addresses, lbUIDs, k8sIDs, tags []string

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
		case "kubernetes_id":
			k8sIDs = append(k8sIDs, pair[1])
		case "tag":
			tags = append(tags, pair[1])
		default:
			rule[pair[0]] = pair[1]
		}
	}

	rule[sd] = map[string]any{
		"addresses":          addresses,
		"droplet_ids":        dropletIDs,
		"load_balancer_uids": lbUIDs,
		"kubernetes_ids":     k8sIDs,
		"tags":               tags,
	}

	return rule, nil
}
