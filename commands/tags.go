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
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/pkg/urn"
	"github.com/digitalocean/godo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Tags creates the tag commands hierarchy.
func Tags() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "tag",
			Short: "Display commands to manage tags",
			Long: `The sub-commands of ` + "`" + `doctl compute tag` + "`" + ` manage the tags on your account.

Tags are labels that you can apply to resources to better organize them and more efficiently take actions on them. For example, if you have a group of Droplets that you want to place behind the same set of cloud firewall rules, you can tag those Droplets with a common tag and then apply the firewall rules to all Droplets with that tag.

You can tag Droplets, images, volumes, volume snapshots, and database clusters.

Tags have two attributes: a user defined name attribute and an embedded
resources attribute with information about resources that have been tagged.`,
		},
	}

	cmdTagCreate := CmdBuilder(cmd, RunCmdTagCreate, "create <tag-name>", "Create a tag", `Creates a new tag that you can apply to resources.`, Writer)
	cmdTagCreate.Example = `The following example creates a tag name ` + "`" + `web` + "`" + `: doctl compute tag create web`

	cmdTagGet := CmdBuilder(cmd, RunCmdTagGet, "get <tag-name>", "Retrieve information about a tag", `Retrieves the number of resources using the tag.`, Writer,
		displayerType(&displayers.Tag{}))
	cmdTagGet.Example = `The following example retrieves information about the tag named ` + "`" + `web` + "`" + `: doctl compute tag get web`

	CmdBuilder(cmd, RunCmdTagList, "list", "List all tags", `Retrieves a list of all the tags in your account and how many resources are using each tag.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.Tag{}))

	cmdRunTagDelete := CmdBuilder(cmd, RunCmdTagDelete, "delete <tag-name>...", "Delete a tag", `Deletes a tag from your account.

Deleting a tag also removes the tag from all the resources that had been tagged with it.`, Writer, aliasOpt("rm"))
	AddBoolFlag(cmdRunTagDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete tag without confirmation prompt")
	cmdRunTagDelete.Example = `The following example deletes the tag named ` + "`" + `web` + "`" + `: doctl compute tag delete web`

	cmdApplyTag := CmdBuilder(cmd, RunCmdApplyTag, "apply <tag-name> --resource=<urn> [--resource=<urn> ...]", "Apply a tag to resources", `Tag one or more resources. You can tag Droplets, images, volumes, volume snapshots, and database clusters.
	
Resources must be specified as Uniform Resource Names (URNs) and has the following syntax: `+"`"+`do:<resource_type>:<identifier>`+"`"+`.`, Writer)
	AddStringSliceFlag(cmdApplyTag, doctl.ArgResourceType, "", []string{}, "The resource to tag in URN format", requiredOpt())
	cmdApplyTag.Example = `The following example tags two Droplet with the tag named ` + "`" + `web` + "`" + `: doctl compute tag apply web --resource=do:droplet:386734086,do:droplet:191669331`

	cmdRemoveTag := CmdBuilder(cmd, RunCmdRemoveTag, "remove <tag-name> --resource=<urn> [--resource=<urn> ...]", "Remove a tag from resources", `Removes a tag from one or more resources. Resources must be specified as Uniform Resource Names (URNs) and has the following syntax: `+"`"+`do:<resource_type>:<identifier>`+"`"+`.`, Writer)
	AddStringSliceFlag(cmdRemoveTag, doctl.ArgResourceType, "", []string{}, "The resource to untag in URN format", requiredOpt())
	cmdRemoveTag.Example = `The following example removes the tag named ` + "`" + `web` + "`" + ` from two Droplets: doctl compute tag remove web --resource=do:droplet:386734086,do:droplet:191669331`

	return cmd
}

// RunCmdTagCreate runs tag create.
func RunCmdTagCreate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	name := c.Args[0]
	ts := c.Tags()

	tcr := &godo.TagCreateRequest{Name: name}
	t, err := ts.Create(tcr)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Tag{Tags: do.Tags{*t}})
}

// RunCmdTagGet runs tag get.
func RunCmdTagGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	name := c.Args[0]
	ts := c.Tags()
	t, err := ts.Get(name)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Tag{Tags: do.Tags{*t}})
}

// RunCmdTagList runs tag list.
func RunCmdTagList(c *CmdConfig) error {
	ts := c.Tags()
	tags, err := ts.List()
	if err != nil {
		return err
	}

	return c.Display(&displayers.Tag{Tags: tags})
}

// RunCmdTagDelete runs tag delete.
func RunCmdTagDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("tag", len(c.Args)) == nil {
		for id := range c.Args {
			name := c.Args[id]
			ts := c.Tags()
			if err := ts.Delete(name); err != nil {
				return err
			}
		}
	} else {
		return errOperationAborted
	}

	return nil
}

// RunCmdApplyTag applies a tag to one or more resources.
func RunCmdApplyTag(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	tagName := c.Args[0]

	urns, err := c.Doit.GetStringSlice(c.NS, doctl.ArgResourceType)
	if err != nil {
		return err
	}

	resourceReq, err := buildTagResources(urns)
	if err != nil {
		return err
	}

	tagReq := &godo.TagResourcesRequest{Resources: resourceReq}

	return c.Tags().TagResources(tagName, tagReq)
}

// RunCmdRemoveTag removes a tag from one or more resources.
func RunCmdRemoveTag(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	tagName := c.Args[0]

	urns, err := c.Doit.GetStringSlice(c.NS, doctl.ArgResourceType)
	if err != nil {
		return err
	}

	resourceReq, err := buildTagResources(urns)
	if err != nil {
		return err
	}

	tagReq := &godo.UntagResourcesRequest{Resources: resourceReq}

	return c.Tags().UntagResources(tagName, tagReq)
}

func buildTagResources(urns []string) ([]godo.Resource, error) {
	resources := []godo.Resource{}
	for _, u := range urns {
		parsedURN, err := urn.ParseURN(u)
		if err != nil {
			return nil, errors.Wrap(err, `URN must be in the format "do:<resource_type>:<resource_id>"`)
		}

		resource := godo.Resource{
			ID:   parsedURN.Identifier(),
			Type: godo.ResourceType(parsedURN.Collection()),
		}
		// The URN collection for databases is actually "dbaas" but the resource
		// type for tags is "database" Support the use of a real URN.
		if resource.Type == "dbaas" {
			resource.Type = "database"
		}
		resources = append(resources, resource)
	}

	return resources, nil
}
