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
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"

	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// SpacesKeys creates a new command that groups the subcommands for managing DigitalOcean Spaces Keys.
func SpacesKeys() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "keys",
			Aliases: []string{"k"},
			Short:   "Display commands that manage DigitalOcean Spaces Keys.",
			Long:    "The subcommands of `doctl spaces keys` allow you to access and manage Spaces Keys.",
		},
	}

	createSpacesKeyDesc := "Create a key for a Space with the provided name."
	cmdSpacesKeysCreate := CmdBuilder(cmd, spacesKeysCreate, "create <name>", "Create a key for a Space.", createSpacesKeyDesc, Writer)
	AddStringSliceFlag(cmdSpacesKeysCreate, "grants", "g", []string{},
		`A comma-separated list of grants to add to the key. The permission should be either 'read', 'readwrite', or 'fullaccess'.
Format: `+"`"+`"bucket=your-bucket;permission=your-permission"`+"`", requiredOpt())
	cmdSpacesKeysCreate.Example =
		`doctl spaces keys create my-key --grants 'bucket=my-bucket;permission=readwrite,bucket=my-other-bucket;permission=read'
doctl spaces keys create my-key --grants 'bucket=;permission=fullaccess'`

	listSpacesKeysDesc := "List all keys for a Space."
	cmdSpacesKeysList := CmdBuilder(
		cmd,
		spacesKeysList,
		"list",
		"List all keys for a Space.",
		listSpacesKeysDesc,
		Writer, aliasOpt("ls"), displayerType(&displayers.SpacesKey{}),
	)
	cmdSpacesKeysList.Example = "The following command lists all Spaces Keys and uses the `--format` flag to return only the Name and Grants of each key. `doctl spaces keys list --format Name,Grants`"

	deleteSpacesKeyDesc := "Delete a key for a Space."
	cmdSpacesKeysDelete := CmdBuilder(
		cmd,
		spacesKeysDelete,
		"delete <access key>",
		"Delete a key for a Space.",
		deleteSpacesKeyDesc,
		Writer, aliasOpt("rm"),
	)
	cmdSpacesKeysDelete.Example = "doctl spaces keys delete DOACCESSKEY"

	updateSpacesKeyDesc := "Update a key for a Space."
	cmdSpacesKeysUpdate := CmdBuilder(cmd, spacesKeysUpdate, "update <access key>", "Update a key for a Space.", updateSpacesKeyDesc, Writer)
	AddStringFlag(cmdSpacesKeysUpdate, "name", "n", "", "The new name for the key.", requiredOpt())
	AddStringSliceFlag(cmdSpacesKeysUpdate, "grants", "g", []string{},
		`A comma-separated list of grants to set to the key. The permission should be either 'read', 'readwrite', or 'fullaccess'.
Format: `+"`"+`"bucket=your-bucket;permission=your-permission"`+"`", requiredOpt())
	cmdSpacesKeysUpdate.Example = "doctl spaces keys update DOACCESSKEY --name new-key --grants 'bucket=my-bucket;permission=readwrite,bucket=my-other-bucket;permission=read'"

	return cmd
}

func spacesKeysCreate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	grants, err := c.Doit.GetStringSlice(c.NS, "grants")
	if err != nil {
		return err
	}

	parsedGrants, err := parseGrantsFromArg(grants)
	if err != nil {
		return err
	}

	r := &godo.SpacesKeyCreateRequest{
		Name:   c.Args[0],
		Grants: parsedGrants,
	}

	key, err := c.SpacesKeys().Create(r)
	if err != nil {
		return err
	}

	return displaySpacesKeys(c, *key)
}

func spacesKeysList(c *CmdConfig) error {
	keys, err := c.SpacesKeys().List()
	if err != nil {
		return err
	}

	return displaySpacesKeys(c, keys...)
}

func spacesKeysDelete(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	err := c.SpacesKeys().Delete(c.Args[0])
	if err != nil {
		return err
	}

	return nil
}

func spacesKeysUpdate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	grants, err := c.Doit.GetStringSlice(c.NS, "grants")
	if err != nil {
		return err
	}

	parsedGrants, err := parseGrantsFromArg(grants)
	if err != nil {
		return err
	}

	name, err := c.Doit.GetString(c.NS, "name")
	if err != nil {
		return err
	}

	r := &godo.SpacesKeyUpdateRequest{
		Name:   name,
		Grants: parsedGrants,
	}

	key, err := c.SpacesKeys().Update(c.Args[0], r)
	if err != nil {
		return err
	}

	return displaySpacesKeys(c, *key)
}

func displaySpacesKeys(c *CmdConfig, keys ...do.SpacesKey) error {
	item := &displayers.SpacesKey{SpacesKeys: keys}
	return c.Display(item)
}

func parseGrantsFromArg(grants []string) ([]*godo.Grant, error) {
	parsedGrants := []*godo.Grant{}
	for _, grant := range grants {
		parsedGrant, err := parseGrant(grant)
		if err != nil {
			return nil, err
		}
		parsedGrants = append(parsedGrants, parsedGrant)
	}
	return parsedGrants, nil
}

func parseGrant(grant string) (*godo.Grant, error) {
	const (
		argSeparator = ";"
		kvSeparator  = "="
	)
	trimmedGrant := strings.TrimSuffix(grant, argSeparator)
	parsedGrant := &godo.Grant{
		Bucket:     "",
		Permission: "",
	}
	for _, arg := range strings.Split(trimmedGrant, argSeparator) {
		kv := strings.Split(arg, kvSeparator)
		if len(kv) != 2 {
			return nil, fmt.Errorf("A Grant must be in the format 'key=value'. Provided: %v", kv)
		}

		key := kv[0]
		value := kv[1]

		switch key {
		case "bucket":
			parsedGrant.Bucket = value
		case "permission":
			// Validate permission
			switch value {
			case "read":
				parsedGrant.Permission = godo.SpacesKeyRead
			case "readwrite":
				parsedGrant.Permission = godo.SpacesKeyReadWrite
			case "fullaccess":
				parsedGrant.Permission = godo.SpacesKeyFullAccess
			default:
				return nil, fmt.Errorf("Unsupported permission %q", value)
			}
		default:
			return nil, fmt.Errorf("Unsupported grant argument %q", key)
		}
	}
	return parsedGrant, nil
}
