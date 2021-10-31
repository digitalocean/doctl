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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// LoadBalancer creates the load balancer command.
func LoadBalancer() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "load-balancer",
			Short: "Display commands to manage load balancers",
			Long: `The sub-commands of ` + "`" + `doctl compute load-balancer` + "`" + ` manage your load balancers.

With the load-balancer command, you can list, create, or delete load balancers, and manage their configuration details.`,
		},
	}
	lbDetail := `

- The load balancer's ID
- The load balancer's name
- The load balancer's IP address
- The load balancer's traffic algorithm. Must
  be either ` + "`" + `round_robin` + "`" + ` or ` + "`" + `least_connections` + "`" + `
- The current state of the load balancer. This can be ` + "`" + `new` + "`" + `, ` + "`" + `active` + "`" + `, or ` + "`" + `errored` + "`" + `.
- The load balancer's creation date, in ISO8601 combined date and time format.
- The load balancer's forwarding rules. See ` + "`" + `doctl compute load-balancer add-forwarding-rules --help` + "`" + ` for a list.
- The ` + "`" + `health_check` + "`" + ` settings for the load balancer.
- The ` + "`" + `sticky_sessions` + "`" + ` settings for the load balancer.
- The datacenter region the load balancer is located in.
- The Droplet tag corresponding to the Droplets assigned to the load balancer.
- The IDs of the Droplets assigned to the load balancer.
- Whether HTTP request to the load balancer on port 80 will be redirected to HTTPS on port 443.
- Whether the PROXY protocol is in use on the load balancer.
`
	forwardingDetail := `

- ` + "`" + `entry_protocol` + "`" + `: The entry protocol used for traffic to the load balancer. Possible values are: ` + "`" + `http` + "`" + `, ` + "`" + `https` + "`" + `, ` + "`" + `http2` + "`" + `, or ` + "`" + `tcp` + "`" + `.
- ` + "`" + `entry_port` + "`" + `: The entry port used for incoming traffic to the load balancer.
- ` + "`" + `target_protocol` + "`" + `: The target protocol used for traffic from the load balancer to the backend Droplets. Possible values are: ` + "`" + `http` + "`" + `, ` + "`" + `https` + "`" + `, ` + "`" + `http2` + "`" + `, or ` + "`" + `tcp` + "`" + `.
- ` + "`" + `target_port` + "`" + `: The target port used to send traffic from the load balancer to the backend Droplets.
- ` + "`" + `certificate_id` + "`" + `: The ID of the TLS certificate used for SSL termination, if enabled. Can be obtained with ` + "`" + `doctl certificate list` + "`" + `
- ` + "`" + `tls_passthrough` + "`" + `: Whether SSL passthrough is enabled on the load balancer.
`
	forwardingRulesTxt := "A comma-separated list of key-value pairs representing forwarding rules, which define how traffic is routed, e.g.: `entry_protocol:tcp,entry_port:3306,target_protocol:tcp,target_port:3306`."
	CmdBuilder(cmd, RunLoadBalancerGet, "get <id>", "Retrieve a load balancer", "Use this command to retrieve information about a load balancer instance, including:"+lbDetail, Writer,
		aliasOpt("g"), displayerType(&displayers.LoadBalancer{}))

	cmdRecordCreate := CmdBuilder(cmd, RunLoadBalancerCreate, "create",
		"Create a new load balancer", "Use this command to create a new load balancer on your account. Valid forwarding rules are:"+forwardingDetail, Writer, aliasOpt("c"))
	AddStringFlag(cmdRecordCreate, doctl.ArgLoadBalancerName, "", "",
		"The load balancer's name", requiredOpt())
	AddStringFlag(cmdRecordCreate, doctl.ArgRegionSlug, "", "",
		"The load balancer's region, e.g.: `nyc1`", requiredOpt())
	AddStringFlag(cmdRecordCreate, doctl.ArgSizeSlug, "", "",
		fmt.Sprintf("The load balancer's size, e.g.: `lb-small`. Only one of %s and %s should be used", doctl.ArgSizeSlug, doctl.ArgSizeUnit))
	AddIntFlag(cmdRecordCreate, doctl.ArgSizeUnit, "", 0,
		fmt.Sprintf("The load balancer's size, e.g.: 1. Only one of %s and %s should be used", doctl.ArgSizeUnit, doctl.ArgSizeSlug))
	AddStringFlag(cmdRecordCreate, doctl.ArgVPCUUID, "", "", "The UUID of the VPC to create the load balancer in")
	AddStringFlag(cmdRecordCreate, doctl.ArgLoadBalancerAlgorithm, "",
		"round_robin", "The algorithm to use when traffic is distributed across your Droplets; possible values: `round_robin` or `least_connections`")
	AddBoolFlag(cmdRecordCreate, doctl.ArgRedirectHTTPToHTTPS, "", false,
		"Redirects HTTP requests to the load balancer on port 80 to HTTPS on port 443")
	AddBoolFlag(cmdRecordCreate, doctl.ArgEnableProxyProtocol, "", false,
		"enable proxy protocol")
	AddBoolFlag(cmdRecordCreate, doctl.ArgEnableBackendKeepalive, "", false,
		"enable keepalive connections to backend target droplets")
	AddBoolFlag(cmdRecordCreate, doctl.ArgDisableLetsEncryptDNSRecords, "", false,
		"disable automatic DNS record creation for Let's Encrypt certificates that are added to the load balancer")
	AddStringFlag(cmdRecordCreate, doctl.ArgTagName, "", "", "droplet tag name")
	AddStringSliceFlag(cmdRecordCreate, doctl.ArgDropletIDs, "", []string{},
		"A comma-separated list of Droplet IDs to add to the load balancer, e.g.: `12,33`")
	AddStringFlag(cmdRecordCreate, doctl.ArgStickySessions, "", "",
		"A comma-separated list of key-value pairs representing a list of active sessions, e.g.: `type:cookies, cookie_name:DO-LB, cookie_ttl_seconds:5`")
	AddStringFlag(cmdRecordCreate, doctl.ArgHealthCheck, "", "",
		"A comma-separated list of key-value pairs representing recent health check results, e.g.: `protocol:http,port:80,path:/index.html,check_interval_seconds:10,response_timeout_seconds:5,healthy_threshold:5,unhealthy_threshold:3`")
	AddStringFlag(cmdRecordCreate, doctl.ArgForwardingRules, "", "",
		forwardingRulesTxt)

	cmdRecordUpdate := CmdBuilder(cmd, RunLoadBalancerUpdate, "update <id>",
		"Update a load balancer's configuration", `Use this command to update the configuration of a specified load balancer.`, Writer, aliasOpt("u"))
	AddStringFlag(cmdRecordUpdate, doctl.ArgLoadBalancerName, "", "",
		"The load balancer's name", requiredOpt())
	AddStringFlag(cmdRecordUpdate, doctl.ArgRegionSlug, "", "",
		"The load balancer's region, e.g.: `nyc1`", requiredOpt())
	AddStringFlag(cmdRecordUpdate, doctl.ArgSizeSlug, "", "",
		fmt.Sprintf("The load balancer's size, e.g.: `lb-small`. Only one of %s and %s should be used", doctl.ArgSizeSlug, doctl.ArgSizeUnit))
	AddIntFlag(cmdRecordUpdate, doctl.ArgSizeUnit, "", 0,
		fmt.Sprintf("The load balancer's size, e.g.: 1. Only one of %s and %s should be used", doctl.ArgSizeUnit, doctl.ArgSizeSlug))
	AddStringFlag(cmdRecordUpdate, doctl.ArgVPCUUID, "", "", "The UUID of the VPC to create the load balancer in")
	AddStringFlag(cmdRecordUpdate, doctl.ArgLoadBalancerAlgorithm, "",
		"round_robin", "The algorithm to use when traffic is distributed across your Droplets; possible values: `round_robin` or `least_connections`")
	AddBoolFlag(cmdRecordUpdate, doctl.ArgRedirectHTTPToHTTPS, "", false,
		"Flag to redirect HTTP requests to the load balancer on port 80 to HTTPS on port 443")
	AddBoolFlag(cmdRecordUpdate, doctl.ArgEnableProxyProtocol, "", false,
		"enable proxy protocol")
	AddBoolFlag(cmdRecordUpdate, doctl.ArgEnableBackendKeepalive, "", false,
		"enable keepalive connections to backend target droplets")
	AddStringFlag(cmdRecordUpdate, doctl.ArgTagName, "", "", "Assigns Droplets with the specified tag to the load balancer")
	AddStringSliceFlag(cmdRecordUpdate, doctl.ArgDropletIDs, "", []string{},
		"A comma-separated list of Droplet IDs, e.g.: `215,378`")
	AddStringFlag(cmdRecordUpdate, doctl.ArgStickySessions, "", "",
		"A comma-separated list of key-value pairs representing a list of active sessions, e.g.: `type:cookies, cookie_name:DO-LB, cookie_ttl_seconds:5`")
	AddStringFlag(cmdRecordUpdate, doctl.ArgHealthCheck, "", "",
		"A comma-separated list of key-value pairs representing recent health check results, e.g.: `protocol:http, port:80, path:/index.html, check_interval_seconds:10, response_timeout_seconds:5, healthy_threshold:5, unhealthy_threshold:3`")
	AddStringFlag(cmdRecordUpdate, doctl.ArgForwardingRules, "", "", forwardingRulesTxt)
	AddBoolFlag(cmdRecordUpdate, doctl.ArgDisableLetsEncryptDNSRecords, "", false,
		"disable automatic DNS record creation for Let's Encrypt certificates that are added to the load balancer")

	CmdBuilder(cmd, RunLoadBalancerList, "list", "List load balancers", "Use this command to get a list of the load balancers on your account, including the following information for each:"+lbDetail, Writer,
		aliasOpt("ls"), displayerType(&displayers.LoadBalancer{}))

	cmdRunRecordDelete := CmdBuilder(cmd, RunLoadBalancerDelete, "delete <id>",
		"Permanently delete a load balancer", `Use this command to permanently delete the specified load balancer. This is irreversible.`, Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunRecordDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the load balancer without a confirmation prompt")

	cmdAddDroplets := CmdBuilder(cmd, RunLoadBalancerAddDroplets, "add-droplets <id>",
		"Add Droplets to a load balancer", `Use this command to add Droplets to a load balancer.`, Writer)
	AddStringSliceFlag(cmdAddDroplets, doctl.ArgDropletIDs, "", []string{},
		"A comma-separated list of IDs of Droplet to add to the load balancer, example value: `12,33`")

	cmdRemoveDroplets := CmdBuilder(cmd, RunLoadBalancerRemoveDroplets,
		"remove-droplets <id>", "Remove Droplets from a load balancer", `Use this command to remove Droplets from a load balancer. This command does not destroy any Droplets.`, Writer)
	AddStringSliceFlag(cmdRemoveDroplets, doctl.ArgDropletIDs, "", []string{},
		"A comma-separated list of IDs of Droplets to remove from the load balancer, example value: `12,33`")

	cmdAddForwardingRules := CmdBuilder(cmd, RunLoadBalancerAddForwardingRules,
		"add-forwarding-rules <id>", "Add forwarding rules to a load balancer", "Use this command to add forwarding rules to a load balancer, specified with the `--forwarding-rules` flag. Valid rules include:"+forwardingDetail, Writer)
	AddStringFlag(cmdAddForwardingRules, doctl.ArgForwardingRules, "", "", forwardingRulesTxt)

	cmdRemoveForwardingRules := CmdBuilder(cmd, RunLoadBalancerRemoveForwardingRules,
		"remove-forwarding-rules <id>", "Remove forwarding rules from a load balancer", "Use this command to remove forwarding rules from a load balancer, specified with the `--forwarding-rules` flag. Valid rules include:"+forwardingDetail, Writer)
	AddStringFlag(cmdRemoveForwardingRules, doctl.ArgForwardingRules, "", "", forwardingRulesTxt)

	return cmd
}

// RunLoadBalancerGet retrieves an existing load balancer by its identifier.
func RunLoadBalancerGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]

	lbs := c.LoadBalancers()
	lb, err := lbs.Get(id)
	if err != nil {
		return err
	}

	item := &displayers.LoadBalancer{LoadBalancers: do.LoadBalancers{*lb}}
	return c.Display(item)
}

// RunLoadBalancerList lists load balancers.
func RunLoadBalancerList(c *CmdConfig) error {
	lbs := c.LoadBalancers()
	list, err := lbs.List()
	if err != nil {
		return err
	}

	item := &displayers.LoadBalancer{LoadBalancers: list}
	return c.Display(item)
}

// RunLoadBalancerCreate creates a new load balancer with a given configuration.
func RunLoadBalancerCreate(c *CmdConfig) error {
	r := new(godo.LoadBalancerRequest)
	if err := buildRequestFromArgs(c, r); err != nil {
		return err
	}

	lbs := c.LoadBalancers()
	lb, err := lbs.Create(r)
	if err != nil {
		return err
	}

	item := &displayers.LoadBalancer{LoadBalancers: do.LoadBalancers{*lb}}
	return c.Display(item)
}

// RunLoadBalancerUpdate updates an existing load balancer with new configuration.
func RunLoadBalancerUpdate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	lbID := c.Args[0]

	r := new(godo.LoadBalancerRequest)
	if err := buildRequestFromArgs(c, r); err != nil {
		return err
	}

	lbs := c.LoadBalancers()
	lb, err := lbs.Update(lbID, r)
	if err != nil {
		return err
	}

	item := &displayers.LoadBalancer{LoadBalancers: do.LoadBalancers{*lb}}
	return c.Display(item)
}

// RunLoadBalancerDelete deletes a load balancer by its identifier.
func RunLoadBalancerDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	lbID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("load balancer", 1) == nil {
		lbs := c.LoadBalancers()
		if err := lbs.Delete(lbID); err != nil {
			return err
		}
	} else {
		return errOperationAborted
	}

	return nil
}

// RunLoadBalancerAddDroplets adds droplets to a load balancer.
func RunLoadBalancerAddDroplets(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	lbID := c.Args[0]

	dropletIDsList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgDropletIDs)
	if err != nil {
		return err
	}

	dropletIDs, err := extractDropletIDs(dropletIDsList)
	if err != nil {
		return err
	}

	return c.LoadBalancers().AddDroplets(lbID, dropletIDs...)
}

// RunLoadBalancerRemoveDroplets removes droplets from a load balancer.
func RunLoadBalancerRemoveDroplets(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	lbID := c.Args[0]

	dropletIDsList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgDropletIDs)
	if err != nil {
		return err
	}

	dropletIDs, err := extractDropletIDs(dropletIDsList)
	if err != nil {
		return err
	}

	return c.LoadBalancers().RemoveDroplets(lbID, dropletIDs...)
}

// RunLoadBalancerAddForwardingRules adds forwarding rules to a load balancer.
func RunLoadBalancerAddForwardingRules(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	lbID := c.Args[0]

	fra, err := c.Doit.GetString(c.NS, doctl.ArgForwardingRules)
	if err != nil {
		return err
	}

	forwardingRules, err := extractForwardingRules(fra)
	if err != nil {
		return err
	}

	return c.LoadBalancers().AddForwardingRules(lbID, forwardingRules...)
}

// RunLoadBalancerRemoveForwardingRules removes forwarding rules from a load balancer.
func RunLoadBalancerRemoveForwardingRules(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	lbID := c.Args[0]

	fra, err := c.Doit.GetString(c.NS, doctl.ArgForwardingRules)
	if err != nil {
		return err
	}

	forwardingRules, err := extractForwardingRules(fra)
	if err != nil {
		return err
	}

	return c.LoadBalancers().RemoveForwardingRules(lbID, forwardingRules...)
}

func extractForwardingRules(s string) (forwardingRules []godo.ForwardingRule, err error) {
	if len(s) == 0 {
		return forwardingRules, err
	}

	list := strings.Split(s, " ")

	for _, v := range list {
		forwardingRule := new(godo.ForwardingRule)
		if err := fillStructFromStringSliceArgs(forwardingRule, v); err != nil {
			return nil, err
		}

		forwardingRules = append(forwardingRules, *forwardingRule)
	}

	return forwardingRules, err
}

func fillStructFromStringSliceArgs(obj interface{}, s string) error {
	if len(s) == 0 {
		return nil
	}

	kvs := strings.Split(s, ",")
	m := map[string]string{}

	for _, v := range kvs {
		p := strings.Split(v, ":")
		if len(p) == 2 {
			m[p[0]] = p[1]
		} else {
			return fmt.Errorf("Unexpected input value %v: must be a key:value pair", p)
		}
	}

	structValue := reflect.Indirect(reflect.ValueOf(obj))
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		f := structValue.Field(i)
		jv := strings.Split(structType.Field(i).Tag.Get("json"), ",")[0]

		if val, exists := m[jv]; exists {
			switch f.Kind() {
			case reflect.Bool:
				if v, err := strconv.ParseBool(val); err == nil {
					f.Set(reflect.ValueOf(v))
				}
			case reflect.Int:
				if v, err := strconv.Atoi(val); err == nil {
					f.Set(reflect.ValueOf(v))
				}
			case reflect.String:
				f.Set(reflect.ValueOf(val))
			default:
				return fmt.Errorf("Unexpected type for struct field %v", val)
			}
		}
	}

	return nil
}

func buildRequestFromArgs(c *CmdConfig, r *godo.LoadBalancerRequest) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgLoadBalancerName)
	if err != nil {
		return err
	}
	r.Name = name

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	r.Region = region

	size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return err
	}
	r.SizeSlug = size

	sizeUnit, err := c.Doit.GetInt(c.NS, doctl.ArgSizeUnit)
	if err != nil {
		return err
	}
	r.SizeUnit = uint32(sizeUnit)

	algorithm, err := c.Doit.GetString(c.NS, doctl.ArgLoadBalancerAlgorithm)
	if err != nil {
		return err
	}
	r.Algorithm = algorithm

	tag, err := c.Doit.GetString(c.NS, doctl.ArgTagName)
	if err != nil {
		return err
	}
	r.Tag = tag

	vpcUUID, err := c.Doit.GetString(c.NS, doctl.ArgVPCUUID)
	if err != nil {
		return err
	}
	r.VPCUUID = vpcUUID

	redirectHTTPToHTTPS, err := c.Doit.GetBool(c.NS, doctl.ArgRedirectHTTPToHTTPS)
	if err != nil {
		return err
	}
	r.RedirectHttpToHttps = redirectHTTPToHTTPS

	enableProxyProtocol, err := c.Doit.GetBool(c.NS, doctl.ArgEnableProxyProtocol)
	if err != nil {
		return err
	}
	r.EnableProxyProtocol = enableProxyProtocol

	enableBackendKeepalive, err := c.Doit.GetBool(c.NS, doctl.ArgEnableBackendKeepalive)
	if err != nil {
		return err
	}
	r.EnableBackendKeepalive = enableBackendKeepalive

	disableLetsEncryptDNSRecords, err := c.Doit.GetBool(c.NS, doctl.ArgDisableLetsEncryptDNSRecords)
	if err != nil {
		return err
	}
	r.DisableLetsEncryptDNSRecords = &disableLetsEncryptDNSRecords

	dropletIDsList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgDropletIDs)
	if err != nil {
		return err
	}

	dropletIDs, err := extractDropletIDs(dropletIDsList)
	if err != nil {
		return err
	}
	r.DropletIDs = dropletIDs

	ssa, err := c.Doit.GetString(c.NS, doctl.ArgStickySessions)
	if err != nil {
		return err
	}

	stickySession := new(godo.StickySessions)
	if err := fillStructFromStringSliceArgs(stickySession, ssa); err != nil {
		return err
	}
	r.StickySessions = stickySession

	hca, err := c.Doit.GetString(c.NS, doctl.ArgHealthCheck)
	if err != nil {
		return err
	}

	healthCheck := new(godo.HealthCheck)
	if err := fillStructFromStringSliceArgs(healthCheck, hca); err != nil {
		return err
	}
	r.HealthCheck = healthCheck

	fra, err := c.Doit.GetString(c.NS, doctl.ArgForwardingRules)
	if err != nil {
		return err
	}

	forwardingRules, err := extractForwardingRules(fra)
	if err != nil {
		return err
	}
	r.ForwardingRules = forwardingRules

	return nil
}
