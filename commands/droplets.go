package commands

import (
	"fmt"
	"io"
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
	addStringSliceFlag(cmdDropletCreate, doit.ArgSSHKeys, []string{}, "SSH Keys or fingerprints")
	addStringFlag(cmdDropletCreate, doit.ArgUserData, "", "User data")
	addStringFlag(cmdDropletCreate, doit.ArgUserDataFile, "", "User data file")
	addBoolFlag(cmdDropletCreate, doit.ArgDropletWait, false, "Wait for droplet to be created")
	addStringFlag(cmdDropletCreate, doit.ArgRegionSlug, "", "Droplet region", requiredOpt())
	addStringFlag(cmdDropletCreate, doit.ArgSizeSlug, "", "Droplet size", requiredOpt())
	addBoolFlag(cmdDropletCreate, doit.ArgBackups, false, "Backup droplet")
	addBoolFlag(cmdDropletCreate, doit.ArgIPv6, false, "IPv6 support")
	addBoolFlag(cmdDropletCreate, doit.ArgPrivateNetworking, false, "Private networking")
	addStringFlag(cmdDropletCreate, doit.ArgImage, "", "Droplet image", requiredOpt())

	cmdBuilder(cmd, RunDropletDelete, "delete ID [ID ...]", "delete droplet", writer, aliasOpt("d", "del"))

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

// NewCmdDropletActions creates a droplet action get command.
func NewCmdDropletActions(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "actions",
		Short: "get droplet actions",
		Long:  "get droplet actions",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActions(cmdNS(cmd), doit.DoitConfig, out, args), cmd)
		},
	}
}

// RunDropletActions returns a list of actions for a droplet.
func RunDropletActions(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ds := do.NewDropletsService(client)

	id, err := getDropletIDArg(ns, args)
	if err != nil {
		return err
	}

	si, err := ds.Actions(id)

	list := make([]godo.Action, len(si))
	for i := range si {
		list[i] = *si[i].Action
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &action{actions: list},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDropletBackups returns a list of backup images for a droplet.
func RunDropletBackups(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ds := do.NewDropletsService(client)

	id, err := getDropletIDArg(ns, args)
	if err != nil {
		return err
	}

	si, err := ds.Backups(id)
	if err != nil {
		return err
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = *si[i].Image
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &image{images: list},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDropletCreate creates a droplet.
func RunDropletCreate(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	if len(args) < 1 {
		return doit.NewMissingArgsErr(ns)
	}

	region, err := config.GetString(ns, doit.ArgRegionSlug)
	if err != nil {
		return err
	}

	size, err := config.GetString(ns, doit.ArgSizeSlug)
	if err != nil {
		return err
	}

	backups, err := config.GetBool(ns, doit.ArgBackups)
	if err != nil {
		return err
	}

	ipv6, err := config.GetBool(ns, doit.ArgIPv6)
	if err != nil {
		return err
	}

	privateNetworking, err := config.GetBool(ns, doit.ArgPrivateNetworking)
	if err != nil {
		return err
	}

	keys, err := config.GetStringSlice(ns, doit.ArgSSHKeys)
	if err != nil {
		return err
	}

	sshKeys := extractSSHKeys(keys)

	userData, err := config.GetString(ns, doit.ArgUserData)
	if err != nil {
		return err
	}

	filename, err := config.GetString(ns, doit.ArgUserDataFile)
	if err != nil {
		return err
	}

	userData, err = extractUserData(userData, filename)
	if err != nil {
		return err
	}

	var createImage godo.DropletCreateImage

	imageStr, err := config.GetString(ns, doit.ArgImage)
	if i, err := strconv.Atoi(imageStr); err == nil {
		createImage = godo.DropletCreateImage{ID: i}
	} else {
		createImage = godo.DropletCreateImage{Slug: imageStr}
	}

	wait, err := config.GetBool(ns, doit.ArgDropletWait)
	if err != nil {
		return err
	}

	ds := do.NewDropletsService(client)

	var wg sync.WaitGroup
	errs := make(chan error)
	for _, name := range args {
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

			dc := &outputConfig{
				ns:     ns,
				config: config,
				item:   &droplet{droplets{*d.Droplet}},
				out:    out,
			}

			displayOutput(dc)
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
func RunDropletDelete(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ds := do.NewDropletsService(client)

	if len(args) < 1 {
		return doit.NewMissingArgsErr(ns)
	}

	for _, idStr := range args {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return err
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
func RunDropletGet(ns string, config doit.Config, out io.Writer, args []string) error {
	id, err := getDropletIDArg(ns, args)
	if err != nil {
		return err
	}

	client := config.GetGodoClient()
	ds := do.NewDropletsService(client)

	d, err := ds.Get(id)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &droplet{droplets{*d.Droplet}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDropletKernels returns a list of available kernels for a droplet.
func RunDropletKernels(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ds := do.NewDropletsService(client)
	id, err := getDropletIDArg(ns, args)
	if err != nil {
		return err
	}

	list, err := ds.Kernels(id)
	if err != nil {
		return err
	}

	godoKernels := &kernel{kernels: kernels{}}
	for _, k := range list {
		godoKernels.kernels = append(godoKernels.kernels, *k.Kernel)
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   godoKernels,
		out:    out,
	}

	return displayOutput(dc)
}

// RunDropletList returns a list of droplets.
func RunDropletList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ds := do.NewDropletsService(client)

	region, err := config.GetString(ns, doit.ArgRegionSlug)
	if err != nil {
		return err
	}

	matches := []glob.Glob{}
	for _, globStr := range args {
		g, err := glob.Compile(globStr)
		if err != nil {
			return fmt.Errorf("unknown glob %q", globStr)
		}

		matches = append(matches, g)
	}

	list, err := ds.List()
	var godoDroplets []godo.Droplet
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
			godoDroplets = append(godoDroplets, *droplet.Droplet)
		}
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &droplet{droplets: godoDroplets},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDropletNeighbors returns a list of droplet neighbors.
func RunDropletNeighbors(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	id, err := getDropletIDArg(ns, args)
	if err != nil {
		return err
	}

	list, _, err := client.Droplets.Neighbors(id)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &droplet{droplets: list},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDropletSnapshots returns a list of available kernels for a droplet.
func RunDropletSnapshots(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ds := do.NewDropletsService(client)
	id, err := getDropletIDArg(ns, args)
	if err != nil {
		return err
	}

	si, err := ds.Snapshots(id)
	if err != nil {
		return err
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = *si[i].Image
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &image{images: list},
		out:    out,
	}

	return displayOutput(dc)
}

func getDropletIDArg(ns string, args []string) (int, error) {
	if len(args) != 1 {
		return 0, doit.NewMissingArgsErr(ns)
	}

	return strconv.Atoi(args[0])
}
