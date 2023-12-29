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
	_ "embed"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

var (
	//go:embed forwarding_detail.txt
	forwardingDetail string

	//go:embed lb_detail.txt
	lbDetail string
)

// LoadBalancer creates the load balancer command.
func LoadBalancer() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "load-balancer",
			Short:   "Display commands to manage load balancers",
			Aliases: []string{"lb"},
			Long: `The sub-commands of ` + "`" + `doctl compute load-balancer` + "`" + ` manage your load balancers.

With the load-balancer command, you can list, create, or delete load balancers, and manage their configuration details.`,
		},
	}

	forwardingRulesTxt := "A comma-separated list of key-value pairs representing forwarding rules, which define how traffic is routed, such as `entry_protocol:tcp,entry_port:3306,target_protocol:tcp,target_port:3306`."
	cmdLoadBalancerGet := CmdBuilder(cmd, RunLoadBalancerGet, "get <load-balancer-id>", "Retrieve a load balancer", "Retrieves information about a load balancer instance, including:\n\n"+lbDetail, Writer,
		aliasOpt("g"), displayerType(&displayers.LoadBalancer{}))
	cmdLoadBalancerGet.Example = `The following example retrieves information about a load balancer with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute load-balancer get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdLoadBalancerCreate := CmdBuilder(cmd, RunLoadBalancerCreate, "create",
		"Create a new load balancer", "Creates a new load balancer on your account. Valid forwarding rules are:\n"+forwardingDetail, Writer, aliasOpt("c"))
	AddStringFlag(cmdLoadBalancerCreate, doctl.ArgLoadBalancerName, "", "",
		"The load balancer's name", requiredOpt())
	AddStringFlag(cmdLoadBalancerCreate, doctl.ArgRegionSlug, "", "",
		"The load balancer's region, for example: `nyc1`", requiredOpt())
	AddStringFlag(cmdLoadBalancerCreate, doctl.ArgSizeSlug, "", "",
		fmt.Sprintf("DEPRECATED. A slug indicating the load balancer's, size, for example: `lb-small`. This flag is not compatible with the `--size-unit` flag. You can only use one or the other.", doctl.ArgSizeSlug, doctl.ArgSizeUnit))
	AddIntFlag(cmdLoadBalancerCreate, doctl.ArgSizeUnit, "", 0,
		fmt.Sprintf("The number of nodes to add to the load balancer, for example: 3. This flag is not compatible with the `--size-unit` flag. You can only use one or the other.", doctl.ArgSizeUnit, doctl.ArgSizeSlug))
	AddStringFlag(cmdLoadBalancerCreate, doctl.ArgVPCUUID, "", "", "The UUID of the VPC to create the load balancer in. If not specified, the load balancer is placed in the default VPC for the region.")
	AddStringFlag(cmdLoadBalancerCreate, doctl.ArgLoadBalancerAlgorithm, "",
		"round_robin", "DEPRECATED. You can no longer specify an algorithm for load balancers.")
	AddBoolFlag(cmdLoadBalancerCreate, doctl.ArgRedirectHTTPToHTTPS, "", false,
		"Redirects HTTP requests to the load balancer on port 80 to HTTPS on port 443")
	AddBoolFlag(cmdLoadBalancerCreate, doctl.ArgEnableProxyProtocol, "", false,
		"Enables proxy protocol")
	AddBoolFlag(cmdLoadBalancerCreate, doctl.ArgEnableBackendKeepalive, "", false,
		"Enable keepalive connections to backend target Droplets")
	AddBoolFlag(cmdLoadBalancerCreate, doctl.ArgDisableLetsEncryptDNSRecords, "", false,
		"Disables automatic DNS record creation for Let's Encrypt certificates that are added to the load balancer")
	AddStringFlag(cmdLoadBalancerCreate, doctl.ArgTagName, "", "", "Assigns Droplets with the specified tag to the load balancer")
	AddStringSliceFlag(cmdLoadBalancerCreate, doctl.ArgDropletIDs, "", []string{},
		"A comma-separated list of Droplet IDs to add to the load balancer, for example: `386734086,191669331`")
	AddStringFlag(cmdLoadBalancerCreate, doctl.ArgStickySessions, "", "",
		"A comma-separated list of key-value pairs representing a list of active sessions, for example: `type:cookies, cookie_name:DO-LB, cookie_ttl_seconds:5`")
	AddStringFlag(cmdLoadBalancerCreate, doctl.ArgHealthCheck, "", "",
		"A comma-separated list of key-value pairs representing recent health check results, for example: `protocol:http,port:80,path:/index.html,check_interval_seconds:10,response_timeout_seconds:5,healthy_threshold:5,unhealthy_threshold:3`")
	AddStringFlag(cmdLoadBalancerCreate, doctl.ArgForwardingRules, "", "",
		forwardingRulesTxt, requiredOpt())
	AddBoolFlag(cmdLoadBalancerCreate, doctl.ArgCommandWait, "", false, "Instructs the terminal to wait for the command to complete before returning access to the user")
	AddStringFlag(cmdLoadBalancerCreate, doctl.ArgProjectID, "", "", "Specifies which project to associate the load balancer with. If you do not specify a project, the load balancer is placed in your default project.")
	AddIntFlag(cmdLoadBalancerCreate, doctl.ArgHTTPIdleTimeoutSeconds, "", 0, "HTTP idle timeout that configures the idle timeout for HTTP connections on the load balancer")
	AddStringSliceFlag(cmdLoadBalancerCreate, doctl.ArgAllowList, "", []string{},
		"A comma-separated list of ALLOW rules for the load balancer,for example: `ip:203.0.113.10,cidr:192.0.2.0/24`")
	AddStringSliceFlag(cmdLoadBalancerCreate, doctl.ArgDenyList, "", []string{},
		"A comma-separated list of DENY rules for the load balancer, for example: `ip:203.0.113.10,cidr:192.0.2.0/24`")
	cmdLoadBalancerCreate.Flags().MarkHidden(doctl.ArgLoadBalancerType)
	cmdLoadBalancerCreate.Example = `The following example creates a load balancer named ` + "`" + `example-lb` + "`" + ` in the ` + "`" + `nyc1` + "`" + ` region with a forwarding rule that routes traffic from port 80 to port 8080 on the Droplets behind the load balancer. The command also adds two Droplets to the load balancer's backend pool: doctl compute load-balancer create --name example-lb --region nyc1 --forwarding-rules entry_protocol:TCP,entry_port:80,target_protocol:TCP,target_port:8080 --droplet-ids 386734086,191669331`

	cmdRecordUpdate := CmdBuilder(cmd, RunLoadBalancerUpdate, "update <load-balancer-id>",
		"Update a load balancer's configuration", `Updates the configuration of a specified load balancer. Using all applicable flags, the command should contain a full representation of the load balancer including existing attributes, such as the load balancer's name, region, forwarding rules, and Droplet IDs. Any attribute that is not provided is reset to its default value.`, Writer, aliasOpt("u"))
	AddStringFlag(cmdRecordUpdate, doctl.ArgLoadBalancerName, "", "",
		"The load balancer's name")
	AddStringFlag(cmdRecordUpdate, doctl.ArgRegionSlug, "", "",
		"The load balancer's region, for example: `nyc1`")
	AddStringFlag(cmdRecordUpdate, doctl.ArgSizeSlug, "", "",
		fmt.Sprintf("DEPRECATED. A slug indicating the load balancer's, size, for example: `lb-small`. This flag is not compatible with the `--size-unit` flag. You can only use one or the other.", doctl.ArgSizeSlug, doctl.ArgSizeUnit))
	AddIntFlag(cmdRecordUpdate, doctl.ArgSizeUnit, "", 0,
		fmt.Sprintf("The number of nodes to add to the load balancer, for example: 3. This flag is not compatible with the `--size-unit` flag. You can only use one or the other", doctl.ArgSizeUnit, doctl.ArgSizeSlug))
	AddStringFlag(cmdRecordUpdate, doctl.ArgVPCUUID, "", "", "The UUID of the VPC to create the load balancer in")
	AddStringFlag(cmdRecordUpdate, doctl.ArgLoadBalancerAlgorithm, "",
		"round_robin", "DEPRECATED. You can no longer specify an algorithm for load balancers.")
	AddBoolFlag(cmdRecordUpdate, doctl.ArgRedirectHTTPToHTTPS, "", false,
		"Redirects HTTP requests to the load balancer on port 80 to HTTPS on port 443")
	AddBoolFlag(cmdRecordUpdate, doctl.ArgEnableProxyProtocol, "", false,
		"Enables proxy protocol")
	AddBoolFlag(cmdRecordUpdate, doctl.ArgEnableBackendKeepalive, "", false,
		"Enables keepalive connections to backend target Droplets")
	AddStringFlag(cmdRecordUpdate, doctl.ArgTagName, "", "", "Assigns Droplets with the specified tag to the load balancer")
	AddStringSliceFlag(cmdRecordUpdate, doctl.ArgDropletIDs, "", []string{},
		"A comma-separated list of Droplet IDs to add to the load balancer's pool, for example: `386734086,191669331`")
	AddStringFlag(cmdRecordUpdate, doctl.ArgStickySessions, "", "",
		"A comma-separated list of key-value pairs representing a list of active sessions, for example: `type:cookies, cookie_name:DO-LB, cookie_ttl_seconds:5`")
	AddStringFlag(cmdRecordUpdate, doctl.ArgHealthCheck, "", "",
		"A comma-separated list of key-value pairs representing recent health check results, for example: `protocol:http, port:80, path:/index.html, check_interval_seconds:10, response_timeout_seconds:5, healthy_threshold:5, unhealthy_threshold:3`")
	AddStringFlag(cmdRecordUpdate, doctl.ArgForwardingRules, "", "", forwardingRulesTxt)
	AddBoolFlag(cmdRecordUpdate, doctl.ArgDisableLetsEncryptDNSRecords, "", false,
		"Disable automatic DNS record creation for Let's Encrypt certificates that are added to the load balancer")
	AddStringFlag(cmdRecordUpdate, doctl.ArgProjectID, "", "",
		"Specifies which project to associate the load balancer with. If you do not specify a project, the load balancer is placed in your default project.")
	AddStringSliceFlag(cmdRecordUpdate, doctl.ArgAllowList, "", []string{},
		"A comma-separated list of ALLOW rules for the load balancer, for example: `ip:1.2.3.4,cidr:1.2.0.0/16`")
	AddStringSliceFlag(cmdRecordUpdate, doctl.ArgDenyList, "", []string{},
		"A comma-separated list of DENY rules for the load balancer, for example: `ip:203.0.113.10,cidr:192.0.2.0/24`")
	cmdRecordUpdate.Example = `The following example updates the load balancer with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to have the name ` + "`" + `example-lb` + "`" + ` and to add the Droplet with the ID ` + "`" + `386734086` + "`" + ` to the load balancer's pool: doctl compute load-balancer update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name example-lb --droplet-ids 386734086`

	cmdLoadBalancerList := CmdBuilder(cmd, RunLoadBalancerList, "list", "List load balancers", "Retrieves a list of the load balancers on your account, including the following information for each:\n\n"+lbDetail, Writer,
		aliasOpt("ls"), displayerType(&displayers.LoadBalancer{}))
	cmdLoadBalancerList.Example = `The following example lists all of the load balancers on your account and used the --format flag to return only each load balancer's ID, IP address, and status: doctl compute load-balancer list --format "ID,IP,Status"`

	cmdRunRecordDelete := CmdBuilder(cmd, RunLoadBalancerDelete, "delete <load-balancer-id>",
		"Permanently delete a load balancer", `Permanently deletes the specified load balancer and disassociates any Droplets assigned to it. This is irreversible.`, Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunRecordDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Deletes the load balancer without a confirmation prompt")
	cmdRunRecordDelete.Example = `The following example deletes the load balancer with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute load-balancer delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdAddDroplets := CmdBuilder(cmd, RunLoadBalancerAddDroplets, "add-droplets <load-balancer-id>",
		"Add Droplets to a load balancer", `Adds Droplets to a load balancer's backend pool.`, Writer)
	AddStringSliceFlag(cmdAddDroplets, doctl.ArgDropletIDs, "", []string{},
		"A comma-separated list of Droplet IDs to add to the load balancer, for example: `386734086,191669331`")
	cmdAddDroplets.Example = `The following example adds the Droplets with the IDs ` + "`" + `386734086` + "`" + ` and ` + "`" + `191669331` + "`" + ` to a load balancer with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute load-balancer add-droplets f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --droplet-ids 386734086,191669331`

	cmdRemoveDroplets := CmdBuilder(cmd, RunLoadBalancerRemoveDroplets,
		"remove-droplets <id>", "Remove Droplets from a load balancer", `Removes Droplets from a load balancer. This command does not destroy any Droplets.`, Writer)
	AddStringSliceFlag(cmdRemoveDroplets, doctl.ArgDropletIDs, "", []string{},
		"A comma-separated list of IDs of Droplets to remove from the load balancer, for example: `386734086,191669331`")
	cmdRemoveDroplets.Example = `The following example removes the Droplets with the IDs ` + "`" + `386734086` + "`" + ` and ` + "`" + `191669331` + "`" + ` from a load balancer with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute load-balancer remove-droplets f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --droplet-ids 386734086,191669331`

	cmdAddForwardingRules := CmdBuilder(cmd, RunLoadBalancerAddForwardingRules,
		"add-forwarding-rules <id>", "Add forwarding rules to a load balancer", "Adds forwarding rules to a load balancer, specified with the `--forwarding-rules` flag. Valid rules include:\n"+forwardingDetail, Writer)
	AddStringFlag(cmdAddForwardingRules, doctl.ArgForwardingRules, "", "", forwardingRulesTxt)
	cmdAddForwardingRules.Example = `The following example adds a forwarding rule that routes traffic from port 80 to port 8080 on the Droplets behind the load balancer: doctl compute load-balancer add-forwarding-rules f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --forwarding-rules entry_protocol:TCP,entry_port:80,target_protocol:TCP,target_port:8080`

	cmdRemoveForwardingRules := CmdBuilder(cmd, RunLoadBalancerRemoveForwardingRules,
		"remove-forwarding-rules <id>", "Remove forwarding rules from a load balancer", "Removes forwarding rules from a load balancer, specified with the `--forwarding-rules` flag. Valid rules include:\n"+forwardingDetail, Writer)
	AddStringFlag(cmdRemoveForwardingRules, doctl.ArgForwardingRules, "", "", forwardingRulesTxt)

	cmdRemoveForwardingRules.Example = `The following example removes a forwarding rule that routes traffic from port 80 to port 8080 on the Droplets behind the load balancer: doctl compute load-balancer remove-forwarding-rules f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --forwarding-rules entry_protocol:TCP,entry_port:80,target_protocol:TCP,target_port:8080`

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

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		lbs := c.LoadBalancers()
		notice("Load balancer creation is in progress, waiting for load balancer to become active")

		err := waitForActiveLoadBalancer(lbs, lb.ID)
		if err != nil {
			return fmt.Errorf(
				"load balancer couldn't enter `active` state: %v",
				err,
			)
		}

		lb, _ = lbs.Get(lb.ID)
	}

	notice("Load balancer created")

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

func fillStructFromStringSliceArgs(obj any, s string) error {
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

	lbType, err := c.Doit.GetString(c.NS, doctl.ArgLoadBalancerType)
	if err != nil {
		return err
	}
	r.Type = strings.ToUpper(lbType)

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

	projectID, err := c.Doit.GetString(c.NS, doctl.ArgProjectID)
	if err != nil {
		return err
	}

	r.ProjectID = projectID

	httpIdleTimeout, err := c.Doit.GetInt(c.NS, doctl.ArgHTTPIdleTimeoutSeconds)
	if err != nil {
		return err
	}

	if httpIdleTimeout != 0 {
		t := uint64(httpIdleTimeout)
		r.HTTPIdleTimeoutSeconds = &t
	}

	allowRules, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAllowList)
	if err != nil {
		return err
	}

	denyRules, err := c.Doit.GetStringSlice(c.NS, doctl.ArgDenyList)
	if err != nil {
		return err
	}

	if len(allowRules) > 0 || len(denyRules) > 0 {
		firewall := new(godo.LBFirewall)
		firewall.Allow = allowRules
		firewall.Deny = denyRules
		r.Firewall = firewall
	}

	return nil
}

func waitForActiveLoadBalancer(lbs do.LoadBalancersService, lbID string) error {
	const maxAttempts = 180
	const wantStatus = "active"
	const errStatus = "errored"
	attempts := 0
	printNewLineSet := false

	for i := 0; i < maxAttempts; i++ {
		if attempts != 0 {
			fmt.Fprint(os.Stderr, ".")
			if !printNewLineSet {
				printNewLineSet = true
				defer fmt.Fprintln(os.Stderr)
			}
		}

		lb, err := lbs.Get(lbID)
		if err != nil {
			return err
		}

		if lb.Status == errStatus {
			return fmt.Errorf(
				"load balancer (%s) entered status `errored`",
				lbID,
			)
		}

		if lb.Status == wantStatus {
			return nil
		}

		attempts++
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf(
		"timeout waiting for load balancer (%s) to become active",
		lbID,
	)
}
