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
	"fmt"
	"sort"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

// validRegions provides the list of regions and datacenters where namespaces may be created.
// Note that AP and Functions share the same list of regions but Functions are available in only
// one datacenter per region, so we're using our own static list for the moment (AP has a dynamic
// list that can be interrogated once we have a reliable dynamic way of distinguishing which datacenters
// actually host Functions).
var validRegions = map[string]string{
	"ams": "ams3", "blr": "blr1", "fra": "fra1", "lon": "lon1",
	"nyc": "nyc1", "sfo": "sfo3", "sgp": "sgp1", "syd": "syd1", "tor": "tor1",
}

// Namespaces generates the serverless 'namespaces' subtree for addition to the doctl command
func Namespaces() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "namespaces",
			Short: "Manage your functions namespaces",
			Long: `Functions namespaces (in the cloud) contain your deployed packages and functions with.

The subcommands of ` + "`" + `doctl serverless namespaces` + "`" + ` are used to manage multiple functions namespaces within your account.
Use ` + "`" + `doctl serverless connect` + "`" + ` to connect to a specific namespace.  You can only connect to one namespace at a time.`,
			Aliases: []string{"namespace", "ns"},
		},
	}
	create := CmdBuilder(cmd, RunNamespacesCreate, "create", "Creates a namespace",
		`Creates a new functions namespace. A namespace is a collection of functions and their associated packages, triggers, and project specifications. 

Both a region and a label must be specified.`,
		Writer)
	AddStringFlag(create, "region", "r", "", "A region for the namespace to reside in", requiredOpt())
	AddStringFlag(create, "label", "l", "", "The namespace's unique name", requiredOpt())
	AddBoolFlag(create, "no-connect", "n", false, "Instructs the doctl client to not immediately connect to the newly created namespace")
	create.Example = `The following example creates a namespace named ` + "`" + `example-namespace` + "`" + ` in the 'nyc1' region: doctl serverless namespaces create --label example-namespace --region nyc1`

	delete := CmdBuilder(cmd, RunNamespacesDelete, "delete <namespace-id|label>", "Deletes a namespace",
		`Deletes a functions namespace.`,
		Writer, aliasOpt("rm"))
	AddBoolFlag(delete, "force", "f", false, "Deletes the namespace without a confirmation prompt")
	delete.Example = `The following example deletes the namespace with the label ` + "`" + `example-namespace` + "`" + `: doctl serverless namespaces delete example-namespace`

	cmdNamespacesList := CmdBuilder(cmd, RunNamespacesList, "list", "Lists your namespaces",
		`Retrieves a list of your functions namespaces.`,
		Writer, aliasOpt("ls"), displayerType(&displayers.Namespaces{}))
	cmdNamespacesList.Example = `The following example lists your functions namespaces and uses the --format flag to return only the ID and region for each namespace: doctl serverless namespaces list --format ID,Region`

	NamespacesListRegions := CmdBuilder(cmd, RunNamespacesListRegions, "list-regions", "Lists the accepted 'region' values",
		`Retrieves a list of region slugs that you can create functions namespaces in.`,
		Writer)
	NamespacesListRegions.Example = `The following example lists of region slugs for functions namespaces: doctl serverless namespaces list-regions`
	return cmd
}

// RunNamespacesCreate supports the 'serverless namespaces create' command
func RunNamespacesCreate(c *CmdConfig) error {
	label, _ := c.Doit.GetString(c.NS, "label")
	region, _ := c.Doit.GetString(c.NS, "region")
	skipConnect, _ := c.Doit.GetBool(c.NS, "no-connect")
	if label == "" || region == "" {
		return fmt.Errorf("the '--label' and '--region' flags are both required")
	}
	validRegion := getValidRegion(region)
	if validRegion == "" {
		fmt.Fprintf(c.Out, "Valid region values are %+v\n", getValidRegions())
		return fmt.Errorf("'%s' is not a valid region value", region)
	}
	ss := c.Serverless()
	ctx := context.TODO()
	uniq, err := isLabelUnique(ctx, ss, label)
	if err != nil {
		return err
	}
	if !uniq {
		return fmt.Errorf("you are using  label '%s' for another namespace; labels should be unique", label)
	}
	if !skipConnect && ss.CheckServerlessStatus() == do.ErrServerlessNotInstalled {
		skipConnect = true
		fmt.Fprintln(c.Out, "Warning: namespace will be created but not connected (serverless software is not installed)")
	}
	creds, err := ss.CreateNamespace(ctx, label, validRegion)
	if err != nil {
		return err
	}
	if skipConnect {
		fmt.Fprintf(c.Out, "New namespace %s created, but not connected.\n", creds.Namespace)
		return nil
	}
	err = ss.WriteCredentials(creds)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.Out, "Connected to functions namespace '%s' on API host '%s'\n", creds.Namespace, creds.APIHost)
	return nil
}

// RunNamespacesDelete supports the 'serverless namespaces delete' command
func RunNamespacesDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	arg := c.Args[0]
	ss := c.Serverless()
	ctx := context.TODO()
	// Since arg may be either a label or an id, match against existing namespaces
	var (
		id    string
		label string
	)
	matches, err := getMatchingNamespaces(ctx, ss, arg)
	if err != nil {
		return err
	}
	if len(matches) > 0 {
		id = matches[0].Namespace
		label = matches[0].Label
	}
	// Must be an exact match though (avoids errors).
	if len(matches) != 1 || (arg != label && arg != id) {
		return fmt.Errorf("'%s' does not exactly match the label or id of any of your namespaces", arg)
	}
	force, _ := c.Doit.GetBool(c.NS, "force")
	if !force {
		fmt.Fprintf(c.Out, "Deleting namespace '%s' with label '%s'.\n", id, label)
		if AskForConfirmDelete("namespace", 1) != nil {
			return fmt.Errorf("deletion of '%s' not confirmed, doing nothing", id)
		}
	}
	err = ss.DeleteNamespace(ctx, id)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.Out, "Namespace successfully deleted")
	return nil
}

// RunNamespacesList supports the 'serverless namespaces list' command
func RunNamespacesList(c *CmdConfig) error {
	if len(c.Args) > 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	list, err := c.Serverless().ListNamespaces(context.TODO())
	if err != nil {
		return err
	}
	return c.Display(&displayers.Namespaces{Info: list.Namespaces})
}

// RunNamespacesListRegions supports the 'serverless namespaces list-regions' command
func RunNamespacesListRegions(c *CmdConfig) error {
	if len(c.Args) > 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	fmt.Fprintf(c.Out, "%+v\n", getValidRegions())
	return nil
}

// getValidRegions returns all the region values that are accepted (region slugs and datacenter slugs)
func getValidRegions() []string {
	vrs := make([]string, len(validRegions)*2)
	i := 0
	for k, v := range validRegions {
		vrs[i] = k
		vrs[i+1] = v
		i += 2
	}
	sort.Strings(vrs)
	return vrs
}

// isLabelUnique tests that a label value is unique (not used for any other namespace in the same
// account).
func isLabelUnique(ctx context.Context, ss do.ServerlessService, label string) (bool, error) {
	resp, err := ss.ListNamespaces(ctx)
	if err != nil {
		return false, err
	}
	for _, ns := range resp.Namespaces {
		if label == ns.Label {
			return false, nil
		}
	}
	return true, nil
}

// getValidRegion returns a valid region value for the API (a four-letter datacenter slug) given either
// a datacenter slug or a three-letter region slug.  Functions are offered in one data center per region.
// The empty string is returned if the value is invalid.
func getValidRegion(value string) string {
	if len(value) == 3 {
		return validRegions[value]
	}
	if len(value) != 4 {
		return ""
	}
	for _, dc := range validRegions {
		if value == dc {
			return value
		}
	}
	return ""
}

// get the Namespaces that match a pattern, where the "pattern" has no wildcards but can be a
// prefix, infix, or suffix match to a namespace ID or label.
func getMatchingNamespaces(ctx context.Context, ss do.ServerlessService, pattern string) ([]do.OutputNamespace, error) {
	ans := []do.OutputNamespace{}
	list, err := ss.ListNamespaces(ctx)
	if err != nil {
		return ans, err
	}
	if pattern == "" {
		return list.Namespaces, nil
	}
	for _, ns := range list.Namespaces {
		if strings.Contains(ns.Namespace, pattern) || strings.Contains(ns.Label, pattern) {
			ans = append(ans, ns)
		}
	}
	return ans, nil
}
