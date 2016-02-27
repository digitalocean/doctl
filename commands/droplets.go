package commands

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/digitalocean/godo"
	"github.com/gobwas/glob"
	"github.com/spf13/cobra"
)

// Droplet creates the droplet command.
func Droplet() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "droplet",
		Aliases: []string{"d"},
		Short:   "droplet commands",
		Long:    "droplet is used to access droplet commands",
	}

	cmdBuilder(cmd, RunDropletActions, "actions <droplet id>", "droplet actions", writer,
		aliasOpt("a"), displayerType(&action{}))

	cmdBuilder(cmd, RunDropletBackups, "backups <droplet id>", "droplet backups", writer,
		aliasOpt("b"), displayerType(&image{}))

	cmdDropletCreate := cmdBuilder(cmd, RunDropletCreate, "create NAME [NAME ...]", "create droplet", writer,
		aliasOpt("c"), displayerType(&droplet{}))
	addStringSliceFlag(cmdDropletCreate, doit.ArgSSHKeys, []string{}, "SSH Keys or fingerprints",
		shortFlag("k"))
	addStringFlag(cmdDropletCreate, doit.ArgUserData, "", "User data",
		shortFlag("u"))
	addStringFlag(cmdDropletCreate, doit.ArgUserDataFile, "", "User data file",
		shortFlag("f"))
	addBoolFlag(cmdDropletCreate, doit.ArgDropletWait, false, "Wait for droplet to be created",
		shortFlag("w"))
	addStringFlag(cmdDropletCreate, doit.ArgRegionSlug, "", "Droplet region",
		requiredOpt(), shortFlag("r"))
	addStringFlag(cmdDropletCreate, doit.ArgSizeSlug, "", "Droplet size",
		requiredOpt(), shortFlag("s"))
	addBoolFlag(cmdDropletCreate, doit.ArgBackups, false, "Backup droplet",
		shortFlag("b"))
	addBoolFlag(cmdDropletCreate, doit.ArgIPv6, false, "IPv6 support",
		shortFlag("6"))
	addBoolFlag(cmdDropletCreate, doit.ArgPrivateNetworking, false, "Private networking",
		shortFlag("p"))
	addStringFlag(cmdDropletCreate, doit.ArgImage, "", "Droplet image",
		requiredOpt(), shortFlag("i"))

	cmdBuilder(cmd, RunDropletDelete, "delete ID [ID|Name ...]", "Delete droplet by id or name", writer,
		aliasOpt("d", "del", "rm"))

	cmdBuilder(cmd, RunDropletGet, "get", "get droplet", writer,
		aliasOpt("g"), displayerType(&droplet{}))

	cmdBuilder(cmd, RunDropletKernels, "kernels <droplet id>", "droplet kernels", writer,
		aliasOpt("k"), displayerType(&kernel{}))

	cmdRunDropletList := cmdBuilder(cmd, RunDropletList, "list [GLOB]", "list droplets", writer,
		aliasOpt("ls"), displayerType(&droplet{}))
	addStringFlag(cmdRunDropletList, doit.ArgRegionSlug, "", "Droplet region")

	cmdBuilder(cmd, RunDropletNeighbors, "neighbors <droplet id>", "droplet neighbors", writer,
		aliasOpt("n"), displayerType(&droplet{}))

	cmdBuilder(cmd, RunDropletSnapshots, "snapshots <droplet id>", "snapshots", writer,
		aliasOpt("s"), displayerType(&image{}))

	return cmd
}

// RunDropletActions returns a list of actions for a droplet.
func RunDropletActions(c *cmdConfig) error {

	ds := c.dropletsService()

	id, err := getDropletIDArg(c.ns, c.args)
	if err != nil {
		return err
	}

	list, err := ds.Actions(id)
	item := &action{actions: list}
	return c.display(item)
}

// RunDropletBackups returns a list of backup images for a droplet.
func RunDropletBackups(c *cmdConfig) error {

	ds := c.dropletsService()

	id, err := getDropletIDArg(c.ns, c.args)
	if err != nil {
		return err
	}

	list, err := ds.Backups(id)
	if err != nil {
		return err
	}

	item := &image{images: list}
	return c.display(item)
}

// RunDropletCreate creates a droplet.
func RunDropletCreate(c *cmdConfig) error {

	if len(c.args) < 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	region, err := c.doitConfig.GetString(c.ns, doit.ArgRegionSlug)
	if err != nil {
		return err
	}

	size, err := c.doitConfig.GetString(c.ns, doit.ArgSizeSlug)
	if err != nil {
		return err
	}

	backups, err := c.doitConfig.GetBool(c.ns, doit.ArgBackups)
	if err != nil {
		return err
	}

	ipv6, err := c.doitConfig.GetBool(c.ns, doit.ArgIPv6)
	if err != nil {
		return err
	}

	privateNetworking, err := c.doitConfig.GetBool(c.ns, doit.ArgPrivateNetworking)
	if err != nil {
		return err
	}

	keys, err := c.doitConfig.GetStringSlice(c.ns, doit.ArgSSHKeys)
	if err != nil {
		return err
	}

	sshKeys := extractSSHKeys(keys)

	userData, err := c.doitConfig.GetString(c.ns, doit.ArgUserData)
	if err != nil {
		return err
	}

	filename, err := c.doitConfig.GetString(c.ns, doit.ArgUserDataFile)
	if err != nil {
		return err
	}

	userData, err = extractUserData(userData, filename)
	if err != nil {
		return err
	}

	var createImage godo.DropletCreateImage

	imageStr, err := c.doitConfig.GetString(c.ns, doit.ArgImage)
	if i, err := strconv.Atoi(imageStr); err == nil {
		createImage = godo.DropletCreateImage{ID: i}
	} else {
		createImage = godo.DropletCreateImage{Slug: imageStr}
	}

	wait, err := c.doitConfig.GetBool(c.ns, doit.ArgDropletWait)
	if err != nil {
		return err
	}

	ds := c.dropletsService()

	var wg sync.WaitGroup
	errs := make(chan error, len(c.args))
	for _, name := range c.args {
		dcr := &godo.DropletCreateRequest{
			Name:              name,
			Region:            region,
			Size:              size,
			Image:             createImage,
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

			item := &droplet{droplets: do.Droplets{*d}}
			c.display(item)
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

func extractSSHKeys(keys []string) []godo.DropletCreateSSHKey {
	sshKeys := []godo.DropletCreateSSHKey{}

	for _, rawKey := range keys {
		rawKey = strings.TrimPrefix(rawKey, "[")
		rawKey = strings.TrimSuffix(rawKey, "]")
		if i, err := strconv.Atoi(rawKey); err == nil {
			sshKeys = append(sshKeys, godo.DropletCreateSSHKey{ID: i})
			continue
		}

		sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: rawKey})
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

// RunDropletDelete destroy a droplet by id.
func RunDropletDelete(c *cmdConfig) error {

	ds := c.dropletsService()

	if len(c.args) < 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	listedDroplets := false
	list := do.Droplets{}

	for _, idStr := range c.args {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			if !listedDroplets {
				list, err = ds.List()
				if err != nil {
					return errors.New("unable to build list of droplets")
				}
				listedDroplets = true
			}

			var matchedDroplet *do.Droplet
			for _, d := range list {
				if d.Name == idStr {
					matchedDroplet = &d
					break
				}
			}

			if matchedDroplet == nil {
				return fmt.Errorf("unable to find droplet with name %q", idStr)
			}

			id = matchedDroplet.ID
		}

		err = ds.Delete(id)
		if err != nil {
			return err
		}

		fmt.Printf("deleted droplet %d\n", id)
	}

	return nil
}

// RunDropletGet returns a droplet.
func RunDropletGet(c *cmdConfig) error {
	id, err := getDropletIDArg(c.ns, c.args)
	if err != nil {
		return err
	}

	ds := c.dropletsService()

	d, err := ds.Get(id)
	if err != nil {
		return err
	}

	item := &droplet{droplets: do.Droplets{*d}}
	return c.display(item)
}

// RunDropletKernels returns a list of available kernels for a droplet.
func RunDropletKernels(c *cmdConfig) error {

	ds := c.dropletsService()
	id, err := getDropletIDArg(c.ns, c.args)
	if err != nil {
		return err
	}

	list, err := ds.Kernels(id)
	if err != nil {
		return err
	}

	item := &kernel{kernels: list}
	return c.display(item)
}

// RunDropletList returns a list of droplets.
func RunDropletList(c *cmdConfig) error {

	ds := c.dropletsService()

	region, err := c.doitConfig.GetString(c.ns, doit.ArgRegionSlug)
	if err != nil {
		return err
	}

	matches := []glob.Glob{}
	for _, globStr := range c.args {
		g, err := glob.Compile(globStr)
		if err != nil {
			return fmt.Errorf("unknown glob %q", globStr)
		}

		matches = append(matches, g)
	}

	var matchedList do.Droplets

	list, err := ds.List()
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

	item := &droplet{droplets: matchedList}
	return c.display(item)
}

// RunDropletNeighbors returns a list of droplet neighbors.
func RunDropletNeighbors(c *cmdConfig) error {

	ds := c.dropletsService()

	id, err := getDropletIDArg(c.ns, c.args)
	if err != nil {
		return err
	}

	list, err := ds.Neighbors(id)
	if err != nil {
		return err
	}

	item := &droplet{droplets: list}
	return c.display(item)
}

// RunDropletSnapshots returns a list of available kernels for a droplet.
func RunDropletSnapshots(c *cmdConfig) error {

	ds := c.dropletsService()
	id, err := getDropletIDArg(c.ns, c.args)
	if err != nil {
		return err
	}

	list, err := ds.Snapshots(id)
	if err != nil {
		return err
	}

	item := &image{images: list}
	return c.display(item)
}

func getDropletIDArg(ns string, args []string) (int, error) {
	if len(args) != 1 {
		return 0, doit.NewMissingArgsErr(ns)
	}

	return strconv.Atoi(args[0])
}
