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

A tag is a label that can be applied to a resource (currently Droplets, images,
volumes, volume snapshots, and database clusters) in order to better organize or
facilitate the lookups and actions on it.

Tags have two attributes: a user defined name attribute and an embedded
resources attribute with information about resources that have been tagged.`,
		},
	}

	CmdBuilder(cmd, RunCmdTagCreate, "create <tag-name>", "Create a tag", `Use this command to create a new tag.`, Writer)

	CmdBuilder(cmd, RunCmdTagGet, "get <tag-name>", "Retrieve information about a tag", `Use this command to retrieve a tag, display a count of how many resources are tagged with it, and show the most recently tagged resource.`, Writer,
		displayerType(&displayers.Tag{}))

	CmdBuilder(cmd, RunCmdTagList, "list", "List all tags", `Use this command to retrieve a list of all the tags in your account.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.Tag{}))

	cmdRunTagDelete := CmdBuilder(cmd, RunCmdTagDelete, "delete <tag-name>...", "Delete a tag", `Use this command to delete a tag.

Deleting a tag also removes the tag from all the resources that had been tagged with it.`, Writer, aliasOpt("rm"))
	AddBoolFlag(cmdRunTagDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete tag without confirmation prompt")

	cmdApplyTag := CmdBuilder(cmd, RunCmdApplyTag, "apply <tag-name> --resource=<urn> [--resource=<urn> ...]", "Apply a tag to resources", `Use this command to tag one or more resources. Resources must be specified as URNs ("do:<resource_type>:<identifier>").`, Writer)
	AddStringSliceFlag(cmdApplyTag, doctl.ArgResourceType, "", []string{}, "The resource to tag in URN format (do:<resource_type>:<identifier>)", requiredOpt())

	cmdRemoveTag := CmdBuilder(cmd, RunCmdRemoveTag, "remove <tag-name> --resource=<urn> [--resource=<urn> ...]", "Remove a tag from resources", `Use this command to remove a tag from one or more resources. Resources must be specified as URNs ("do:<resource_type>:<identifier>").`, Writer)
	AddStringSliceFlag(cmdRemoveTag, doctl.ArgResourceType, "", []string{}, "The resource to tag in URN format (do:<resource_type>:<identifier>)", requiredOpt())

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
