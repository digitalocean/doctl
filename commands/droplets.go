package commands

import (
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo/util"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/cobra"
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

	cmdDropletCreate := cmdBuilder(RunDropletCreate, "create", "create droplet", writer, aliasOpt("c"))
	cmd.AddCommand(cmdDropletCreate)
	addStringSliceFlag(cmdDropletCreate, doit.ArgSSHKeys, []string{}, "SSH Keys or fingerprints")
	addStringFlag(cmdDropletCreate, doit.ArgUserData, "", "User data")
	addStringFlag(cmdDropletCreate, doit.ArgUserDataFile, "", "User data file")
	addBoolFlag(cmdDropletCreate, doit.ArgDropletWait, false, "Wait for droplet to be created")
	addStringFlag(cmdDropletCreate, doit.ArgDropletName, "", "Droplet name (required)")
	addStringFlag(cmdDropletCreate, doit.ArgRegionSlug, "", "Droplet region (required)")
	addStringFlag(cmdDropletCreate, doit.ArgSizeSlug, "", "Droplet size (required)")
	addBoolFlag(cmdDropletCreate, doit.ArgBackups, false, "Backup droplet")
	addBoolFlag(cmdDropletCreate, doit.ArgIPv6, false, "IPv6 support")
	addBoolFlag(cmdDropletCreate, doit.ArgPrivateNetworking, false, "Private networking")
	addStringFlag(cmdDropletCreate, doit.ArgImage, "", "Droplet image (required)")

	cmdDropletDelete := cmdBuilder(RunDropletDelete, "delete", "delete droplet", writer, aliasOpt("d"))
	cmd.AddCommand(cmdDropletDelete)
	addIntFlag(cmdDropletDelete, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletGet := cmdBuilder(RunDropletGet, "get", "get droplet", writer, aliasOpt("g"))
	cmd.AddCommand(cmdDropletGet)
	addIntFlag(cmdDropletGet, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletKernels := cmdBuilder(RunDropletKernels, "kernels", "droplet kernels", writer, aliasOpt("k"))
	cmd.AddCommand(cmdDropletKernels)
	addIntFlag(cmdDropletKernels, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletList := cmdBuilder(RunDropletList, "list", "list droplets", writer, aliasOpt("ls"))
	cmd.AddCommand(cmdDropletList)

	cmdDropletNeighbors := cmdBuilder(RunDropletNeighbors, "neighbors", "droplet neighbors", writer, aliasOpt("n"))
	cmd.AddCommand(cmdDropletNeighbors)
	addIntFlag(cmdDropletNeighbors, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletSnapshots := cmdBuilder(RunDropletSnapshots, "snapshots", "snapshots", writer, aliasOpt("s"))
	cmd.AddCommand(cmdDropletSnapshots)
	addIntFlag(cmdDropletSnapshots, doit.ArgDropletID, 0, "Droplet ID")

	return cmd
}

// NewCmdDropletActions creates a droplet action get command.
func NewCmdDropletActions(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "actions",
		Short: "get droplet actions",
		Long:  "get droplet actions",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActions(cmdNS(cmd), doit.DoitConfig, out), cmd)
		},
	}
}

// RunDropletActions returns a list of actions for a droplet.
func RunDropletActions(ns string, config doit.Config, out io.Writer) error {
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

	return doit.DisplayOutput(list, out)
}

// RunDropletBackups returns a list of backup images for a droplet.
func RunDropletBackups(ns string, config doit.Config, out io.Writer) error {
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

	return doit.DisplayOutput(list, out)
}

// RunDropletCreate creates a droplet.
func RunDropletCreate(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()

	name, err := config.GetString(ns, doit.ArgDropletName)
	if name == "" {
		return NewMissingArgsErr(ns)
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

	sshKeys := []godo.DropletCreateSSHKey{}
	keys, err := config.GetStringSlice(ns, doit.ArgSSHKeys)
	if err != nil {
		return err
	}

	for _, rawKey := range keys {
		rawKey = strings.TrimPrefix(rawKey, "[")
		rawKey = strings.TrimSuffix(rawKey, "]")
		if i, err := strconv.Atoi(rawKey); err == nil {
			sshKeys = append(sshKeys, godo.DropletCreateSSHKey{ID: i})
			continue
		}

		sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: rawKey})
	}

	userData, err := config.GetString(ns, doit.ArgUserData)
	if err != nil {
		return err
	}

	filename, err := config.GetString(ns, doit.ArgUserDataFile)
	if err != nil {
		return err
	}

	if userData == "" && filename != "" {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		userData = string(data)
	}

	wait, err := config.GetBool(ns, doit.ArgDropletWait)
	if err != nil {
		return err
	}

	dcr := &godo.DropletCreateRequest{
		Name:              name,
		Region:            region,
		Size:              size,
		Backups:           backups,
		IPv6:              ipv6,
		PrivateNetworking: privateNetworking,
		SSHKeys:           sshKeys,
		UserData:          userData,
	}

	imageStr, err := config.GetString(ns, doit.ArgImage)
	if i, err := strconv.Atoi(imageStr); err == nil {
		dcr.Image = godo.DropletCreateImage{ID: i}
	} else {
		dcr.Image = godo.DropletCreateImage{Slug: imageStr}
	}

	r, resp, err := client.Droplets.Create(dcr)
	if err != nil {
		return err
	}

	var action *godo.LinkAction

	if wait {
		for _, a := range resp.Links.Actions {
			if a.Rel == "create" {
				action = &a
			}
		}
	}

	if action != nil {
		err = util.WaitForActive(client, action.HREF)
		if err != nil {
			return err
		}

		r, err = getDropletByID(client, r.ID)
		if err != nil {
			return err
		}
	}

	return doit.DisplayOutput(r, out)
}

// RunDropletDelete destroy a droplet by id.
func RunDropletDelete(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()
	id, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	_, err = client.Droplets.Delete(id)
	return err
}

// RunDropletGet returns a droplet.
func RunDropletGet(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()
	id, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	if id < 1 {
		return NewMissingArgsErr(ns)
	}

	droplet, err := getDropletByID(client, id)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(droplet, out)
}

// RunDropletKernels returns a list of available kernels for a droplet.
func RunDropletKernels(ns string, config doit.Config, out io.Writer) error {
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

	return doit.DisplayOutput(list, out)
}

// RunDropletList returns a list of droplets.
func RunDropletList(ns string, config doit.Config, out io.Writer) error {
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

	si, err := doit.PaginateResp(f)
	if err != nil {
		return err
	}

	list := make([]godo.Droplet, len(si))
	for i := range si {
		list[i] = si[i].(godo.Droplet)
	}

	return doit.DisplayOutput(list, out)
}

// RunDropletNeighbors returns a list of droplet neighbors.
func RunDropletNeighbors(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()
	id, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	list, _, err := client.Droplets.Neighbors(id)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(list, out)
}

// RunDropletSnapshots returns a list of available kernels for a droplet.
func RunDropletSnapshots(ns string, config doit.Config, out io.Writer) error {
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

	return doit.DisplayOutput(list, out)
}

func getDropletByID(client *godo.Client, id int) (*godo.Droplet, error) {
	if id < 1 {
		return nil, errors.New("missing droplet id")
	}

	droplet, _, err := client.Droplets.Get(id)
	return droplet, err
}
