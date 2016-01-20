package commands

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/digitalocean/godo/util"
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

	cmdDropletActions := cmdBuilder(RunDropletActions, "actions", "droplet actions", writer, aliasOpt("a"))
	cmd.AddCommand(cmdDropletActions)
	addIntFlag(cmdDropletActions, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletBackups := cmdBuilder(RunDropletBackups, "backups", "droplet backups", writer, aliasOpt("b"))
	cmd.AddCommand(cmdDropletBackups)
	addIntFlag(cmdDropletBackups, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletCreate := cmdBuilder(RunDropletCreate, "create NAME [NAME ...]", "create droplet", writer, aliasOpt("c"))
	cmd.AddCommand(cmdDropletCreate)
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

	cmdDropletDelete := cmdBuilder(RunDropletDelete, "delete ID [ID ...]", "delete droplet", writer, aliasOpt("d", "del"))
	cmd.AddCommand(cmdDropletDelete)

	cmdDropletGet := cmdBuilder(RunDropletGet, "get", "get droplet", writer, aliasOpt("g"))
	cmd.AddCommand(cmdDropletGet)

	cmdDropletKernels := cmdBuilder(RunDropletKernels, "kernels", "droplet kernels", writer, aliasOpt("k"))
	cmd.AddCommand(cmdDropletKernels)
	addIntFlag(cmdDropletKernels, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletList := cmdBuilder(RunDropletList, "list [REGEX]", "list droplets", writer, aliasOpt("ls"))
	cmd.AddCommand(cmdDropletList)
	addStringFlag(cmdDropletList, doit.ArgFormat, "", "Format")
	addBoolFlag(cmdDropletList, doit.ArgNoHeader, false, "hide headers")

	cmdDropletNeighbors := cmdBuilder(RunDropletNeighbors, "neighbors", "droplet neighbors", writer, aliasOpt("n"))
	cmd.AddCommand(cmdDropletNeighbors)
	addIntFlag(cmdDropletNeighbors, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletSnapshots := cmdBuilder(RunDropletSnapshots, "snapshots", "snapshots", writer, aliasOpt("s"))
	cmd.AddCommand(cmdDropletSnapshots)
	addIntFlag(cmdDropletSnapshots, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

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
	id, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Actions(id, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(f)
	if err != nil {
		return err
	}

	list := make([]godo.Action, len(si))
	for i := range si {
		list[i] = si[i].(godo.Action)
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
	id, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Backups(id, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(f)
	if err != nil {
		return err
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
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

	var wg sync.WaitGroup
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
		r, resp, err := client.Droplets.Create(dcr)
		if err != nil {
			return err
		}

		if wait {
			wg.Add(1)

			go func() {
				defer wg.Done()
				var action *godo.LinkAction

				for _, a := range resp.Links.Actions {
					if a.Rel == "create" {
						action = &a
						break
					}
				}

				if action != nil {
					_ = util.WaitForActive(client, action.HREF)
					r, _ = getDropletByID(client, r.ID)
				}

				dc := &outputConfig{
					ns:     ns,
					config: config,
					item:   &droplet{droplets{*r}},
					out:    out,
				}

				displayOutput(dc)
			}()
		} else {
			dc := &outputConfig{
				ns:     ns,
				config: config,
				item:   &droplet{droplets{*r}},
				out:    out,
			}

			displayOutput(dc)
		}
	}

	wg.Wait()

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

	if len(args) < 1 {
		return doit.NewMissingArgsErr(ns)
	}

	for _, idStr := range args {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return err
		}

		_, err = client.Droplets.Delete(id)
		fmt.Printf("deleted droplet %d\n", id)
	}

	return nil
}

// RunDropletGet returns a droplet.
func RunDropletGet(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	client := config.GetGodoClient()

	d, err := getDropletByID(client, id)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &droplet{droplets{*d}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDropletKernels returns a list of available kernels for a droplet.
func RunDropletKernels(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	id, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Kernels(id, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(f)
	if err != nil {
		return err
	}

	list := make([]godo.Kernel, len(si))
	for i := range si {
		list[i] = si[i].(godo.Kernel)
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &kernel{kernels: list},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDropletList returns a list of droplets.
func RunDropletList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.List(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	matches := []*regexp.Regexp{}
	for _, reStr := range args {
		re, err := regexp.Compile(reStr)
		if err != nil {
			return fmt.Errorf("unknown regex %q", reStr)
		}

		matches = append(matches, re)
	}

	si, err := doit.PaginateResp(f)
	if err != nil {
		return err
	}

	list := []godo.Droplet{}
	for _, d := range si {
		droplet := d.(godo.Droplet)

		var skip = true
		if len(matches) == 0 {
			skip = false
		} else {
			for _, m := range matches {
				if m.MatchString(droplet.Name) {
					skip = false
				}
			}
		}

		if !skip {
			list = append(list, d.(godo.Droplet))
		}
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &droplet{droplets: list},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDropletNeighbors returns a list of droplet neighbors.
func RunDropletNeighbors(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	id, err := config.GetInt(ns, doit.ArgDropletID)
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
	id, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Snapshots(id, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(f)
	if err != nil {
		return err
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &image{images: list},
		out:    out,
	}

	return displayOutput(dc)
}

func getDropletByID(client *godo.Client, id int) (*godo.Droplet, error) {
	if id < 1 {
		return nil, errors.New("missing droplet id")
	}

	droplet, _, err := client.Droplets.Get(id)
	return droplet, err
}
