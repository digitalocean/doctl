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
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/gobwas/glob"
	"github.com/spf13/cobra"
)

// Droplet creates the droplet command.
func Droplet() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "droplet",
			Aliases: []string{"d"},
			Short:   "Manage virtual machines (Droplets)",
			Long:    "A Droplet is a DigitalOcean virtual machine. Use the subcommands of `doctl compute droplet` to create, manage, and retrieve information about Droplets.",
		},
	}
	dropletDetails := `

- The Droplet's ID
- The Droplet's name
- The Droplet's public IPv4 address
- The Droplet's private IPv4 address
- The Droplet's IPv6 address
- The memory size of the Droplet in MB
- The number of vCPUs on the Droplet
- The size of the Droplet's disk in GB
- The Droplet's data center region
- The image the Droplet was created from
- The status of the Droplet. Possible values: ` + "`" + `new` + "`" + `, ` + "`" + `active` + "`" + `, ` + "`" + `off` + "`" + `, or ` + "`" + `archive` + "`" + `
- The tags assigned to the Droplet
- A list of features enabled for the Droplet, such as ` + "`" + `backups` + "`" + `, ` + "`" + `ipv6` + "`" + `, ` + "`" + `monitoring` + "`" + `, and ` + "`" + `private_networking` + "`" + `
- The IDs of block storage volumes attached to the Droplet
	`
	cmdDropletActions := CmdBuilder(cmd, RunDropletActions, "actions <droplet-id>", "List Droplet actions", `Retrieves a list of previous actions taken on the Droplet, such as reboots, resizes, and snapshots actions.`, Writer,
		aliasOpt("a"), displayerType(&displayers.Action{}))
	cmdDropletActions.Example = `The following example retrieves a list of actions taken on a Droplet with the ID ` + "`" + `386734086` + "`" + `. Additionally, the command uses the ` + "`" + `--format` + "`" + ` flag to return only the ID, status, and type of action: doctl compute droplet actions 386734086 --format ID,Status,Type`

	cmdDropletBackups := CmdBuilder(cmd, RunDropletBackups, "backups <droplet-id>", "List Droplet backups", `Lists backup images for a Droplet, including each image's slug and ID.`, Writer,
		aliasOpt("b"), displayerType(&displayers.Image{}))
	cmdDropletBackups.Example = `The following example retrieves a list of backups for a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet backups 386734086`

	dropletCreateLongDesc := `Creates a new Droplet on your account. The command requires values for the ` + "`" + `--size` + "`" + `, and ` + "`" + `--image` + "`" + ` flags.

To retrieve a list of size slugs, use the ` + "`" + `doctl compute size list` + "`" + ` command. To retrieve a list of image slugs, use the ` + "`" + `doctl compute image list` + "`" + ` command.

If you do not specify a region, the Droplet is created in the default region for your account. If you do not specify any SSH keys, we email a temporary password to your account's email address.`

	cmdDropletCreate := CmdBuilder(cmd, RunDropletCreate, "create <droplet-name>...", "Create a new Droplet", dropletCreateLongDesc, Writer,
		aliasOpt("c"), displayerType(&displayers.Droplet{}))
	AddStringSliceFlag(cmdDropletCreate, doctl.ArgSSHKeys, "", []string{}, "A list of SSH key IDs or fingerprints to embed in the Droplet's root account upon creation")
	AddStringFlag(cmdDropletCreate, doctl.ArgUserData, "", "", "A shell script to run on the Droplet's first boot")
	AddStringFlag(cmdDropletCreate, doctl.ArgUserDataFile, "", "", "The path to a file containing a shell script or Cloud-init YAML file to run on the Droplet's first boot. Example: `path/to/file.yaml`")
	AddBoolFlag(cmdDropletCreate, doctl.ArgCommandWait, "", false, "Instructs the terminal to wait for the action to complete before returning access to the user")
	AddStringFlag(cmdDropletCreate, doctl.ArgRegionSlug, "", "", "A slug specifying the region to create the Droplet in, such as `nyc1`. Use the `doctl compute region list` command for a list of valid regions.")
	AddStringFlag(cmdDropletCreate, doctl.ArgSizeSlug, "", "", "A slug indicating the Droplet's number of vCPUs, RAM, and disk size. For example, `s-1vcpu-1gb` specifies a Droplet with one vCPU and 1 GiB of RAM. The disk size is defined by the slug's plan. Run `doctl compute size list` for a list of valid size slugs and their disk sizes.",
		requiredOpt())
	AddBoolFlag(cmdDropletCreate, doctl.ArgBackups, "", false, "Enables backups for the Droplet. Backups are created on a weekly basis.")
	AddBoolFlag(cmdDropletCreate, doctl.ArgIPv6, "", false, "Enables IPv6 support and assigns an IPv6 address to the Droplet")
	AddBoolFlag(cmdDropletCreate, doctl.ArgPrivateNetworking, "", false, "Enables private networking for the Droplet by provisioning it inside of your account's default VPC for the region")
	AddBoolFlag(cmdDropletCreate, doctl.ArgMonitoring, "", false, "Installs the DigitalOcean agent for additional monitoring")
	AddStringFlag(cmdDropletCreate, doctl.ArgImage, "", "", "An ID or slug specifying the image to use to create the Droplet, such as `ubuntu-20-04-x64`. Use the commands under `doctl compute image` to find additional images.",
		requiredOpt())
	AddStringFlag(cmdDropletCreate, doctl.ArgTagName, "", "", "Applies a tag to the Droplet")
	AddStringFlag(cmdDropletCreate, doctl.ArgVPCUUID, "", "", "The UUID of a non-default VPC to create the Droplet in")
	AddStringFlag(cmdDropletCreate, doctl.ArgProjectID, "", "", "The UUID of the project to assign the Droplet to")
	AddStringSliceFlag(cmdDropletCreate, doctl.ArgTagNames, "", []string{}, "Applies a list of tags to the Droplet")
	AddBoolFlag(cmdDropletCreate, doctl.ArgDropletAgent, "", false, "Specifies whether or not the Droplet monitoring agent should be installed. By default, the agent is installed on new Droplets but installation errors are ignored. Set `--droplet-agent=false` to prevent installation. Set to `true` to make installation errors fatal.")
	AddStringSliceFlag(cmdDropletCreate, doctl.ArgVolumeList, "", []string{}, "A list of block storage volume IDs to attach to the Droplet")
	cmdDropletCreate.Example = `The following example creates a Droplet named ` + "`" + `example-droplet` + "`" + ` with a two vCPUs, two GiB of RAM, and 20 GBs of disk space. The Droplet is created in the ` + "`" + `nyc1` + "`" + ` region and is based on the ` + "`" + `ubuntu-20-04-x64` + "`" + ` image. Additionally, the command uses the ` + "`" + `--user-data` + "`" + ` flag to run a Bash script the first time the Droplet boots up: doctl compute droplet create example-droplet --size s-2vcpu-2gb --image ubuntu-20-04-x64 --region nyc1 --user-data $'#!/bin/bash\n touch /root/example.txt; sudo apt update;sudo snap install doctl'`

	cmdRunDropletDelete := CmdBuilder(cmd, RunDropletDelete, "delete <droplet-id|droplet-name>...", "Permanently delete a Droplet", `Permanently deletes a Droplet. This is irreversible.`, Writer,
		aliasOpt("d", "del", "rm"))
	AddBoolFlag(cmdRunDropletDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the Droplet without a confirmation prompt")
	AddStringFlag(cmdRunDropletDelete, doctl.ArgTagName, "", "", "Tag name")
	cmdRunDropletDelete.Example = `The following example deletes a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet delete 386734086`

	cmdRunDropletGet := CmdBuilder(cmd, RunDropletGet, "get <droplet-id|droplet-name>", "Retrieve information about a Droplet", `Retrieves information about a Droplet, including:`+dropletDetails, Writer,
		aliasOpt("g"), displayerType(&displayers.Droplet{}))
	AddStringFlag(cmdRunDropletGet, doctl.ArgTemplate, "", "", "Go template format. Sample values: `{{.ID}}`, `{{.Name}}`, `{{.Memory}}`, `{{.Region.Name}}`, `{{.Image}}`, `{{.Tags}}`")
	cmdRunDropletGet.Example = `The following example retrieves information about a Droplet with the ID ` + "`" + `386734086` + "`" + `. The command also uses the ` + "`" + `--format` + "`" + ` flag to only return the Droplet's name, ID, and public IPv4 address: doctl compute droplet get 386734086 --format Name,ID,PublicIPv4`

	cmdDropletKernels := CmdBuilder(cmd, RunDropletKernels, "kernels <droplet-id>", "List available Droplet kernels", `Retrieves a list of all kernels available to a Droplet. This command is only available for Droplets with externally managed kernels. All Droplets created after March 2017 have internally managed kernels by default.`, Writer,
		aliasOpt("k"), displayerType(&displayers.Kernel{}))
	cmdDropletKernels.Example = `The following example retrieves a list of available kernels for a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet kernels 386734086`

	cmdRunDropletList := CmdBuilder(cmd, RunDropletList, "list [GLOB]", "List Droplets on your account", `Retrieves a list of Droplets on your account, including the following information about each:`+dropletDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.Droplet{}))
	AddStringFlag(cmdRunDropletList, doctl.ArgRegionSlug, "", "", "Retrieves a list of Droplets in a specified region")
	AddStringFlag(cmdRunDropletList, doctl.ArgTagName, "", "", "Retrieves a list of Droplets with the specified tag name")
	cmdRunDropletList.Example = `The following example retrieves a list of all Droplets in the ` + "`" + `nyc1` + "`" + ` region: doctl compute droplet list --region nyc1`

	cmdDropletNeighbors := CmdBuilder(cmd, RunDropletNeighbors, "neighbors <droplet-id>", "List a Droplet's neighbors on your account", `Lists your Droplets that are on the same physical hardware, including the following details:`+dropletDetails, Writer,
		aliasOpt("n"), displayerType(&displayers.Droplet{}))
	cmdDropletNeighbors.Example = `The following example retrieves a list of Droplets that are on the same physical hardware as the Droplet with the ID ` + "`" + `386734086` + "`" + ` and uses the ` + "`" + `--format` + "`" + ` flag to return only each Droplet's ID, name and public IPv4 address: doctl compute droplet neighbors 386734086 --format ID,Name,PublicIPv4`

	cmdDropletSnapshots := CmdBuilder(cmd, RunDropletSnapshots, "snapshots <droplet-id>", "List all snapshots for a Droplet", `Retrieves a list of snapshots created from this Droplet.`, Writer,
		aliasOpt("s"), displayerType(&displayers.Image{}))
	cmdDropletSnapshots.Example = `The following example retrieves a list of snapshots for a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet snapshots 386734086`

	cmdRunDropletTag := CmdBuilder(cmd, RunDropletTag, "tag <droplet-id|droplet-name>", "Add a tag to a Droplet", "Applies a tag to a Droplet. Specify the tag with the `--tag-name` flag.", Writer)
	AddStringFlag(cmdRunDropletTag, doctl.ArgTagName, "", "", "the tag name apply to the Droplet. You can use a new or existing tag.",
		requiredOpt())
	cmdRunDropletTag.Example = `The following example applies the tag ` + "`" + `frontend` + "`" + ` to a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet tag 386734086 --tag-name frontend`

	cmdRunDropletUntag := CmdBuilder(cmd, RunDropletUntag, "untag <droplet-id|droplet-name>", "Remove a tag from a Droplet", "Removes a tag from a Droplet. Specify the tag with the `--tag-name` flag.", Writer)
	AddStringSliceFlag(cmdRunDropletUntag, doctl.ArgTagName, "", []string{}, "The tag name to remove from Droplet")
	cmdRunDropletUntag.Example = `The following example removes the tag ` + "`" + `frontend` + "`" + ` from a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet untag 386734086 --tag-name frontend`

	cmd.AddCommand(dropletOneClicks())

	return cmd
}

// RunDropletActions returns a list of actions for a droplet.
func RunDropletActions(c *CmdConfig) error {

	ds := c.Droplets()

	id, err := getDropletIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	list, err := ds.Actions(id)
	if err != nil {
		return err
	}
	item := &displayers.Action{Actions: list}
	return c.Display(item)
}

// RunDropletBackups returns a list of backup images for a droplet.
func RunDropletBackups(c *CmdConfig) error {

	ds := c.Droplets()

	id, err := getDropletIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	list, err := ds.Backups(id)
	if err != nil {
		return err
	}

	item := &displayers.Image{Images: list}
	return c.Display(item)
}

// RunDropletCreate creates a droplet.
func RunDropletCreate(c *CmdConfig) error {

	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}

	size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return err
	}

	backups, err := c.Doit.GetBool(c.NS, doctl.ArgBackups)
	if err != nil {
		return err
	}

	ipv6, err := c.Doit.GetBool(c.NS, doctl.ArgIPv6)
	if err != nil {
		return err
	}

	privateNetworking, err := c.Doit.GetBool(c.NS, doctl.ArgPrivateNetworking)
	if err != nil {
		return err
	}

	monitoring, err := c.Doit.GetBool(c.NS, doctl.ArgMonitoring)
	if err != nil {
		return err
	}

	agent, err := c.Doit.GetBoolPtr(c.NS, doctl.ArgDropletAgent)
	if err != nil {
		return err
	}

	keys, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSSHKeys)
	if err != nil {
		return err
	}

	tagName, err := c.Doit.GetString(c.NS, doctl.ArgTagName)
	if err != nil {
		return err
	}

	vpcUUID, err := c.Doit.GetString(c.NS, doctl.ArgVPCUUID)
	if err != nil {
		return err
	}

	projectUUID, err := c.Doit.GetString(c.NS, doctl.ArgProjectID)
	if err != nil {
		return err
	}

	tagNames, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTagNames)
	if err != nil {
		return err
	}
	if len(tagName) > 0 {
		tagNames = append(tagNames, tagName)
	}

	sshKeys := extractSSHKeys(keys)

	userData, err := c.Doit.GetString(c.NS, doctl.ArgUserData)
	if err != nil {
		return err
	}

	volumeList, err := c.Doit.GetStringSlice(c.NS, doctl.ArgVolumeList)
	if err != nil {
		return err
	}
	volumes := extractVolumes(volumeList)

	filename, err := c.Doit.GetString(c.NS, doctl.ArgUserDataFile)
	if err != nil {
		return err
	}

	userData, err = extractUserData(userData, filename)
	if err != nil {
		return err
	}

	imageStr, err := c.Doit.GetString(c.NS, doctl.ArgImage)
	if err != nil {
		return err
	}

	createImage := godo.DropletCreateImage{Slug: imageStr}

	i, err := strconv.Atoi(imageStr)
	if err == nil {
		createImage = godo.DropletCreateImage{ID: i}
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	ds := c.Droplets()

	var wg sync.WaitGroup
	var createdList do.Droplets
	errs := make(chan error, len(c.Args))
	for _, name := range c.Args {
		dcr := &godo.DropletCreateRequest{
			Name:              name,
			Region:            region,
			Size:              size,
			Image:             createImage,
			Volumes:           volumes,
			Backups:           backups,
			IPv6:              ipv6,
			PrivateNetworking: privateNetworking,
			Monitoring:        monitoring,
			SSHKeys:           sshKeys,
			UserData:          userData,
			VPCUUID:           vpcUUID,
			Tags:              tagNames,
		}

		if agent != nil {
			dcr.WithDropletAgent = agent
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			d, err := ds.Create(dcr, wait)
			if err != nil {
				errs <- err
				return
			}

			createdList = append(createdList, *d)
		}()
	}

	wg.Wait()
	close(errs)

	item := &displayers.Droplet{Droplets: createdList}

	for err := range errs {
		if err != nil {
			return err
		}
	}

	for _, createdDroplet := range createdList {
		if err := c.moveToProject(projectUUID, createdDroplet); err != nil {
			return err
		}
	}

	return c.Display(item)
}

// RunDropletTag adds a tag to a droplet.
func RunDropletTag(c *CmdConfig) error {
	ds := c.Droplets()
	ts := c.Tags()

	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	tag, err := c.Doit.GetString(c.NS, doctl.ArgTagName)
	if err != nil {
		return err
	}

	fn := func(ids []int) error {
		trr := &godo.TagResourcesRequest{}
		for _, id := range ids {
			r := godo.Resource{
				ID:   strconv.Itoa(id),
				Type: godo.DropletResourceType,
			}
			trr.Resources = append(trr.Resources, r)
		}

		return ts.TagResources(tag, trr)
	}

	return matchDroplets(c.Args, ds, fn)
}

// RunDropletUntag untags a droplet.
func RunDropletUntag(c *CmdConfig) error {
	ds := c.Droplets()
	ts := c.Tags()

	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	dropletIDStrs := c.Args

	tagNames, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTagName)
	if err != nil {
		return err
	}

	fn := func(ids []int) error {
		urr := &godo.UntagResourcesRequest{}

		for _, id := range ids {
			for _, tagName := range tagNames {
				r := godo.Resource{
					ID:   strconv.Itoa(id),
					Type: godo.DropletResourceType,
				}

				urr.Resources = append(urr.Resources, r)

				err := ts.UntagResources(tagName, urr)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	return matchDroplets(dropletIDStrs, ds, fn)
}

func extractSSHKeys(keys []string) []godo.DropletCreateSSHKey {
	sshKeys := []godo.DropletCreateSSHKey{}

	for _, k := range keys {
		if i, err := strconv.Atoi(k); err == nil {
			if i > 0 {
				sshKeys = append(sshKeys, godo.DropletCreateSSHKey{ID: i})
			}
			continue
		}

		if k != "" {
			sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: k})
		}
	}

	return sshKeys
}

func extractUserData(userData, filename string) (string, error) {
	if userData == "" && filename != "" {
		data, err := os.ReadFile(filename)
		if err != nil {
			return "", err
		}
		userData = string(data)
	}

	return userData, nil
}

func extractVolumes(volumeList []string) []godo.DropletCreateVolume {
	var volumes []godo.DropletCreateVolume

	for _, v := range volumeList {
		var req godo.DropletCreateVolume
		if looksLikeUUID(v) {
			req.ID = v
		} else {
			req.Name = v
		}
		volumes = append(volumes, req)
	}

	return volumes
}

func allInt(in []string) ([]int, error) {
	out := make([]int, 0, len(in))
	seen := map[string]bool{}

	for _, i := range in {
		if seen[i] {
			continue
		}

		seen[i] = true

		id, err := strconv.Atoi(i)
		if err != nil {
			return nil, fmt.Errorf("%s is not an int", i)
		}
		out = append(out, id)
	}
	return out, nil
}

// RunDropletDelete destroy a droplet by id.
func RunDropletDelete(c *CmdConfig) error {
	ds := c.Droplets()

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	tagName, err := c.Doit.GetString(c.NS, doctl.ArgTagName)
	if err != nil {
		return err
	}

	if len(c.Args) < 1 && tagName == "" {
		return doctl.NewMissingArgsErr(c.NS)
	} else if len(c.Args) > 0 && tagName != "" {
		return fmt.Errorf("Please specify Droplet identifier or a tag name.")
	} else if tagName != "" {
		// Collect affected Droplet IDs to show in confirmation message.
		var affectedIDs string
		list, err := ds.ListByTag(tagName)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			fmt.Fprintf(c.Out, "Nothing to delete: no Droplets are using the \"%s\" tag\n", tagName)
			return nil
		}
		ids := make([]string, 0, len(list))
		for _, droplet := range list {
			ids = append(ids, strconv.Itoa(droplet.ID))
		}
		affectedIDs = strings.Join(ids, " ")
		resourceType := "Droplet"
		if len(list) > 1 {
			resourceType = "Droplets"
		}

		if force || AskForConfirm(fmt.Sprintf("delete %d %s tagged \"%s\"? [affected %s: %s]", len(list), resourceType, tagName, resourceType, affectedIDs)) == nil {
			return ds.DeleteByTag(tagName)
		}
		return errOperationAborted
	}

	if force || AskForConfirmDelete("Droplet", len(c.Args)) == nil {

		fn := func(ids []int) error {
			for _, id := range ids {
				if err := ds.Delete(id); err != nil {
					return fmt.Errorf("Unable to delete Droplet %d: %v", id, err)
				}
			}
			return nil
		}
		return matchDroplets(c.Args, ds, fn)
	}
	return errOperationAborted
}

type matchDropletsFn func(ids []int) error

func matchDroplets(ids []string, ds do.DropletsService, fn matchDropletsFn) error {
	if extractedIDs, err := allInt(ids); err == nil {
		return fn(extractedIDs)
	}

	sum, err := buildDropletSummary(ds)
	if err != nil {
		return err
	}

	matchedMap := map[int]bool{}
	for _, idStr := range ids {
		count := sum.count[idStr]
		if count > 1 {
			return fmt.Errorf("There are %d Droplets with the name %q; please provide a specific Droplet ID. [%s]",
				count, idStr, strings.Join(sum.byName[idStr], ", "))
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			id, ok := sum.byID[idStr]
			if !ok {
				return fmt.Errorf("Droplet with the name %q could not be found.", idStr)
			}

			matchedMap[id] = true
			continue
		}

		matchedMap[id] = true
	}

	extractedIDs := make([]int, 0, len(matchedMap))
	for id := range matchedMap {
		extractedIDs = append(extractedIDs, id)
	}

	sort.Ints(extractedIDs)
	return fn(extractedIDs)
}

// RunDropletGet returns a droplet.
func RunDropletGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	getTemplate, err := c.Doit.GetString(c.NS, doctl.ArgTemplate)
	if err != nil {
		return err
	}

	ds := c.Droplets()
	fn := func(ids []int) error {
		for _, id := range ids {
			d, err := ds.Get(id)
			if err != nil {
				return err
			}

			item := &displayers.Droplet{Droplets: do.Droplets{*d}}

			if getTemplate != "" {
				t := template.New("Get template")
				t, err = t.Parse(getTemplate)
				if err != nil {
					return err
				}
				return t.Execute(c.Out, d)
			}
			return c.Display(item)
		}
		return nil
	}
	return matchDroplets(c.Args, ds, fn)

}

// RunDropletKernels returns a list of available kernels for a droplet.
func RunDropletKernels(c *CmdConfig) error {

	ds := c.Droplets()
	id, err := getDropletIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	list, err := ds.Kernels(id)
	if err != nil {
		return err
	}

	item := &displayers.Kernel{Kernels: list}
	return c.Display(item)
}

// RunDropletList returns a list of droplets.
func RunDropletList(c *CmdConfig) error {

	ds := c.Droplets()

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}

	tagName, err := c.Doit.GetString(c.NS, doctl.ArgTagName)
	if err != nil {
		return err
	}

	matches := make([]glob.Glob, 0, len(c.Args))
	for _, globStr := range c.Args {
		g, err := glob.Compile(globStr)
		if err != nil {
			return fmt.Errorf("Unknown glob %q", globStr)
		}

		matches = append(matches, g)
	}

	var matchedList do.Droplets

	var list do.Droplets
	if tagName == "" {
		list, err = ds.List()
	} else {
		list, err = ds.ListByTag(tagName)
	}
	if err != nil {
		return err
	}

	for _, droplet := range list {
		var skip = true
		if len(matches) == 0 {
			skip = false
		} else {
			for _, m := range matches {
				if m.Match(droplet.Name) {
					skip = false
				}
			}
		}

		if !skip && region != "" {
			if region != droplet.Region.Slug {
				skip = true
			}
		}

		if !skip {
			matchedList = append(matchedList, droplet)
		}
	}

	item := &displayers.Droplet{Droplets: matchedList}
	return c.Display(item)
}

// RunDropletNeighbors returns a list of droplet neighbors.
func RunDropletNeighbors(c *CmdConfig) error {

	ds := c.Droplets()

	id, err := getDropletIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	list, err := ds.Neighbors(id)
	if err != nil {
		return err
	}

	item := &displayers.Droplet{Droplets: list}
	return c.Display(item)
}

// RunDropletSnapshots returns a list of available kernels for a droplet.
func RunDropletSnapshots(c *CmdConfig) error {

	ds := c.Droplets()
	id, err := getDropletIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	list, err := ds.Snapshots(id)
	if err != nil {
		return err
	}

	item := &displayers.Image{Images: list}
	return c.Display(item)
}

func getDropletIDArg(ns string, args []string) (int, error) {
	if len(args) != 1 {
		return 0, doctl.NewMissingArgsErr(ns)
	}

	return strconv.Atoi(args[0])
}

type dropletSummary struct {
	count  map[string]int
	byID   map[string]int
	byName map[string][]string
}

func buildDropletSummary(ds do.DropletsService) (*dropletSummary, error) {
	list, err := ds.List()
	if err != nil {
		return nil, err
	}

	var sum dropletSummary

	sum.count = map[string]int{}
	sum.byID = map[string]int{}
	sum.byName = map[string][]string{}
	for _, d := range list {
		sum.count[d.Name]++
		sum.byID[d.Name] = d.ID
		sum.byName[d.Name] = append(sum.byName[d.Name], strconv.Itoa(d.ID))
	}

	return &sum, nil
}

// kubernetesOneClicks creates the 1-click command.
func dropletOneClicks() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "1-click",
			Short: "Display commands that pertain to Droplet 1-click applications",
			Long:  "The commands under `doctl compute droplet 1-click` are for interacting with DigitalOcean Droplet 1-Click applications.",
		},
	}

	cmdDropletOneClickList := CmdBuilder(cmd, RunDropletOneClickList, "list", "Retrieve a list of Droplet 1-Click applications", `Retrieves a list of Droplet 1-Click application slugs.

You can use 1-click slugs to create Droplets by using them as the argument for the `+"`"+`--image`+"`"+` flag in the `+"`"+`doctl compute droplet create`+"`"+` command. For example, the following command creates a Droplet with an Openblocks installation on it: `+"`"+`doctl compute droplet create example-droplet --image openblocks --size s-2vcpu-2gb --region nyc1`+"`"+``, Writer,
		aliasOpt("ls"), displayerType(&displayers.OneClick{}))
	cmdDropletOneClickList.Example = `The following example retrieves a list of 1-clicks for Droplets: doctl compute droplet 1-click list`

	return cmd
}

// RunDropletOneClickList retrieves a list of 1-clicks for Droplets.
func RunDropletOneClickList(c *CmdConfig) error {
	oneClicks := c.OneClicks()
	oneClickList, err := oneClicks.List("droplet")
	if err != nil {
		return err
	}

	items := &displayers.OneClick{OneClicks: oneClickList}

	return c.Display(items)
}
