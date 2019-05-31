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
			Short: "load-balancer commands",
			Long:  "load-balancer is used to access load-balancer commands",
		},
	}

	CmdBuilder(cmd, RunLoadBalancerGet, "get <id>", "get load balancer", Writer, aliasOpt("g"), displayerType(&displayers.LoadBalancer{}))

	cmdRecordCreate := CmdBuilder(cmd, RunLoadBalancerCreate, "create", "create load balancer", Writer, aliasOpt("c"))
	AddStringFlag(cmdRecordCreate, doctl.ArgLoadBalancerName, "", "", "load balancer name", requiredOpt())
	AddStringFlag(cmdRecordCreate, doctl.ArgRegionSlug, "", "", "load balancer region location, example value: nyc1", requiredOpt())
	AddStringFlag(cmdRecordCreate, doctl.ArgLoadBalancerAlgorithm, "", "round_robin", "load balancing algorithm, possible values: round_robin or least_connections")
	AddBoolFlag(cmdRecordCreate, doctl.ArgRedirectHttpToHttps, "", false, "flag to redirect HTTP requests to the load balancer on port 80 to HTTPS on port 443")
	AddStringFlag(cmdRecordCreate, doctl.ArgTagName, "", "", "droplet tag name")
	AddStringSliceFlag(cmdRecordCreate, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example value: 12,33")
	AddStringFlag(cmdRecordCreate, doctl.ArgStickySessions, "", "", "comma-separated key:value list, example value: type:cookies,cookie_name:DO-LB,cookie_ttl_seconds:5")
	AddStringFlag(cmdRecordCreate, doctl.ArgHealthCheck, "", "", "comma-separated key:value list, example value: protocol:http,port:80,path:/index.html,check_interval_seconds:10,response_timeout_seconds:5,healthy_threshold:5,unhealthy_threshold:3")
	AddStringFlag(cmdRecordCreate, doctl.ArgForwardingRules, "", "", "comma-separated key:value list, example value: entry_protocol:tcp,entry_port:3306,target_protocol:tcp,target_port:3306, use quoted string of space-separated values for multiple rules")

	cmdRecordUpdate := CmdBuilder(cmd, RunLoadBalancerUpdate, "update <id>", "update load balancer", Writer, aliasOpt("u"))
	AddStringFlag(cmdRecordUpdate, doctl.ArgLoadBalancerName, "", "", "load balancer name", requiredOpt())
	AddStringFlag(cmdRecordUpdate, doctl.ArgRegionSlug, "", "", "load balancer region location, example value: nyc1", requiredOpt())
	AddStringFlag(cmdRecordUpdate, doctl.ArgLoadBalancerAlgorithm, "", "round_robin", "load balancing algorithm, possible values: round_robin or least_connections")
	AddBoolFlag(cmdRecordUpdate, doctl.ArgRedirectHttpToHttps, "", false, "flag to redirect HTTP requests to the load balancer on port 80 to HTTPS on port 443")
	AddStringFlag(cmdRecordUpdate, doctl.ArgTagName, "", "", "droplet tag name")
	AddStringSliceFlag(cmdRecordUpdate, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example value: 12,33")
	AddStringFlag(cmdRecordUpdate, doctl.ArgStickySessions, "", "", "comma-separated key:value list, example value, example value: type:cookies,cookie_name:DO-LB,cookie_ttl_seconds:5")
	AddStringFlag(cmdRecordUpdate, doctl.ArgHealthCheck, "", "", "comma-separated key:value list, example value: protocol:http,port:80,path:/index.html,check_interval_seconds:10,response_timeout_seconds:5,healthy_threshold:5,unhealthy_threshold:3")
	AddStringFlag(cmdRecordUpdate, doctl.ArgForwardingRules, "", "", "comma-separated key:value list, example value: entry_protocol:tcp,entry_port:3306,target_protocol:tcp,target_port:3306, use quoted string of space-separated values for multiple rules")

	CmdBuilder(cmd, RunLoadBalancerList, "list", "list load balancers", Writer, aliasOpt("ls"), displayerType(&displayers.LoadBalancer{}))

	cmdRunRecordDelete := CmdBuilder(cmd, RunLoadBalancerDelete, "delete <id>", "delete load balancer", Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunRecordDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force load balancer delete")

	cmdAddDroplets := CmdBuilder(cmd, RunLoadBalancerAddDroplets, "add-droplets <id>", "add droplets to the load balancer", Writer)
	AddStringSliceFlag(cmdAddDroplets, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example valus: 12,33")

	cmdRemoveDroplets := CmdBuilder(cmd, RunLoadBalancerRemoveDroplets, "remove-droplets <id>", "remove droplets from the load balancer", Writer)
	AddStringSliceFlag(cmdRemoveDroplets, doctl.ArgDropletIDs, "", []string{}, "comma-separated list of droplet IDs, example value: 12,33")

	cmdAddForwardingRules := CmdBuilder(cmd, RunLoadBalancerAddForwardingRules, "add-forwarding-rules <id>", "add forwarding rules to the load balancer", Writer)
	AddStringFlag(cmdAddForwardingRules, doctl.ArgForwardingRules, "", "", "comma-separated key:value list, example value: entry_protocol:tcp,entry_port:3306,target_protocol:tcp,target_port:3306, use quoted string of space-separated values for multiple rules")

	cmdRemoveForwardingRules := CmdBuilder(cmd, RunLoadBalancerRemoveForwardingRules, "remove-forwarding-rules <id>", "remove forwarding rules from the load balancer", Writer)
	AddStringFlag(cmdRemoveForwardingRules, doctl.ArgForwardingRules, "", "", "comma-separated key:value list, example value: entry_protocol:tcp,entry_port:3306,target_protocol:tcp,target_port:3306, use quoted string of space-separated values for multiple rules")

	return cmd
}

// RunLoadBalancerGet retrieves an existing load balancer by its identifier.
func RunLoadBalancerGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
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
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	lbID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirm("delete this load balancer") == nil {
		lbs := c.LoadBalancers()
		if err := lbs.Delete(lbID); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("operation aborted")
	}

	return nil
}

// RunLoadBalancerAddDroplets adds droplets to a load balancer.
func RunLoadBalancerAddDroplets(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
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
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
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
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
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
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
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

	redirectHTTPToHTTPS, err := c.Doit.GetBool(c.NS, doctl.ArgRedirectHttpToHttps)
	if err != nil {
		return err
	}
	r.RedirectHttpToHttps = redirectHTTPToHTTPS

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
