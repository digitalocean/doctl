/*
Copyright 2016 The Doctl Authors All rights reserved.
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
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/gobwas/glob"
	"github.com/pborman/uuid"
	"github.com/spf13/cobra"
)

// Droplet creates the droplet command.
func Droplet() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "droplet",
			Aliases: []string{"d"},
			Short:   "droplet commands",
			Long:    "droplet is used to access droplet commands",
		},
		DocCategories: []string{"droplet"},
		IsIndex:       true,
	}

	CmdBuilder(cmd, RunDropletActions, "actions <droplet id>", "droplet actions", Writer,
		aliasOpt("a"), displayerType(&action{}), docCategories("droplet"))

	CmdBuilder(cmd, RunDropletBackups, "backups <droplet id>", "droplet backups", Writer,
		aliasOpt("b"), displayerType(&image{}), docCategories("droplet"))

	cmdDropletCreate := CmdBuilder(cmd, RunDropletCreate, "create NAME [NAME ...]", "create droplet", Writer,
		aliasOpt("c"), displayerType(&droplet{}), docCategories("droplet"))
	AddStringSliceFlag(cmdDropletCreate, doctl.ArgSSHKeys, []string{}, "SSH Keys or fingerprints")
	AddStringFlag(cmdDropletCreate, doctl.ArgUserData, "", "User data")
	AddStringFlag(cmdDropletCreate, doctl.ArgUserDataFile, "", "User data file")
	AddBoolFlag(cmdDropletCreate, doctl.ArgCommandWait, false, "Wait for droplet to be created")
	AddStringFlag(cmdDropletCreate, doctl.ArgRegionSlug, "", "Droplet region",
		requiredOpt())
	AddStringFlag(cmdDropletCreate, doctl.ArgSizeSlug, "", "Droplet size",
		requiredOpt())
	AddBoolFlag(cmdDropletCreate, doctl.ArgBackups, false, "Backup droplet")
	AddBoolFlag(cmdDropletCreate, doctl.ArgIPv6, false, "IPv6 support")
	AddBoolFlag(cmdDropletCreate, doctl.ArgPrivateNetworking, false, "Private networking")
	AddStringFlag(cmdDropletCreate, doctl.ArgImage, "", "Droplet image",
		requiredOpt())
	AddStringFlag(cmdDropletCreate, doctl.ArgTagName, "", "Tag name")

	AddStringSliceFlag(cmdDropletCreate, doctl.ArgVolumeList, []string{}, "Volumes to attach")

	CmdBuilder(cmd, RunDropletDelete, "delete ID [ID|Name ...]", "Delete droplet by id or name", Writer,
		aliasOpt("d", "del", "rm"), docCategories("droplet"))

	CmdBuilder(cmd, RunDropletGet, "get", "get droplet", Writer,
		aliasOpt("g"), displayerType(&droplet{}), docCategories("droplet"))

	CmdBuilder(cmd, RunDropletKernels, "kernels <droplet id>", "droplet kernels", Writer,
		aliasOpt("k"), displayerType(&kernel{}), docCategories("droplet"))

	cmdRunDropletList := CmdBuilder(cmd, RunDropletList, "list [GLOB]", "list droplets", Writer,
		aliasOpt("ls"), displayerType(&droplet{}), docCategories("droplet"))
	AddStringFlag(cmdRunDropletList, doctl.ArgRegionSlug, "", "Droplet region")
	AddStringFlag(cmdRunDropletList, doctl.ArgTagName, "", "Tag name")

	CmdBuilder(cmd, RunDropletNeighbors, "neighbors <droplet id>", "droplet neighbors", Writer,
		aliasOpt("n"), displayerType(&droplet{}), docCategories("droplet"))

	CmdBuilder(cmd, RunDropletSnapshots, "snapshots <droplet id>", "snapshots", Writer,
		aliasOpt("s"), displayerType(&image{}), docCategories("droplet"))

	cmdRunDropletTag := CmdBuilder(cmd, RunDropletTag, "tag <droplet id or name>", "tag", Writer,
		docCategories("droplet"))
	AddStringFlag(cmdRunDropletTag, doctl.ArgTagName, "", "Tag name",
		requiredOpt())

	cmdRunDropletUntag := CmdBuilder(cmd, RunDropletUntag, "untag <droplet id or name>", "untag", Writer,
		docCategories("droplet"))
	AddStringSliceFlag(cmdRunDropletUntag, doctl.ArgTagName, []string{}, "tag names")

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
	item := &action{actions: list}
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

	item := &image{images: list}
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

	keys, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSSHKeys)
	if err != nil {
		return err
	}

	tagName, err := c.Doit.GetString(c.NS, doctl.ArgTagName)
	if err != nil {
		return err
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
	ts := c.Tags()

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
			SSHKeys:           sshKeys,
			UserData:          userData,
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			d, err := ds.Create(dcr, wait)
			if err != nil {
				errs <- err
				return
			}

			if tagName != "" {
				trr := &godo.TagResourcesRequest{
					Resources: []godo.Resource{
						{ID: strconv.Itoa(d.ID), Type: godo.DropletResourceType},
					},
				}

				err := ts.TagResources(tagName, trr)
				if err != nil {
					errs <- err
				}

			}

			createdList = append(createdList, *d)
		}()
	}

	wg.Wait()
	close(errs)
	
	item := &droplet{droplets: createdList}
	c.Display(item)

	for err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
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
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return "", err
		}
		userData = string(data)
	}

	return userData, nil
}

func extractVolumes(volumeList []string) []godo.DropletCreateVolume {
	var volumes []godo.DropletCreateVolume

	notice(fmt.Sprintf("extracting volumes from %#v\n", volumeList))

	for _, v := range volumeList {
		var req godo.DropletCreateVolume
		if uuid.Parse(v) != nil {
			req.ID = v
		} else {
			req.Name = v
		}
		volumes = append(volumes, req)
	}

	return volumes
}

func allInt(in []string) ([]int, error) {
	out := []int{}
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

	tagName, err := c.Doit.GetString(c.NS, doctl.ArgTagName)
	if err != nil {
		return err
	}

	if len(c.Args) < 1 && tagName == "" {
		return doctl.NewMissingArgsErr(c.NS)
	} else if len(c.Args) > 0 && tagName != "" {
		return fmt.Errorf("please specify droplets identifiers or a tag name")
	} else if tagName != "" {
		return ds.DeleteByTag(tagName)
	}

	fn := func(ids []int) error {
		for _, id := range ids {
			if err := ds.Delete(id); err != nil {
				return fmt.Errorf("unable to delete droplet %d: %v", id, err)
			}
		}

		return nil
	}

	return matchDroplets(c.Args, ds, fn)
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
			return fmt.Errorf("there are %d Droplets with the name %q, please delete by id. [%s]",
				count, idStr, strings.Join(sum.byName[idStr], ", "))
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			id, ok := sum.byID[idStr]
			if !ok {
				return fmt.Errorf("droplet with name %q could not be found", idStr)
			}

			matchedMap[id] = true
			continue
		}

		matchedMap[id] = true
	}

	var extractedIDs []int
	for id := range matchedMap {
		extractedIDs = append(extractedIDs, id)
	}

	sort.Ints(extractedIDs)
	return fn(extractedIDs)
}

// RunDropletGet returns a droplet.
func RunDropletGet(c *CmdConfig) error {
	id, err := getDropletIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	ds := c.Droplets()

	d, err := ds.Get(id)
	if err != nil {
		return err
	}

	item := &droplet{droplets: do.Droplets{*d}}
	return c.Display(item)
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

	item := &kernel{kernels: list}
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

	matches := []glob.Glob{}
	for _, globStr := range c.Args {
		g, err := glob.Compile(globStr)
		if err != nil {
			return fmt.Errorf("unknown glob %q", globStr)
		}

		matches = append(matches, g)
	}

	var matchedList do.Droplets

	var list do.Droplets
	if tagName == "" {
		list, err = ds.List()
		if err != nil {
			return err
		}
	} else {
		list, err = ds.ListByTag(tagName)
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

	item := &droplet{droplets: matchedList}
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

	item := &droplet{droplets: list}
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

	item := &image{images: list}
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
