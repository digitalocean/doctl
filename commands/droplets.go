package commands

import (
	"errors"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/digitalocean/godo/util"
	"github.com/spf13/cobra"
)

// Droplet creates the droplet command.
func Droplet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "droplet",
		Short: "droplet commands",
		Long:  "droplet is used to access droplet commands",
	}

	cmdDropletActions := cmdBuilder(RunDropletActions,
		"actions", "droplet actions", writer)
	cmd.AddCommand(cmdDropletActions)
	addIntFlag(cmdDropletActions, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletBackups := cmdBuilder(RunDropletBackups,
		"backups", "droplet backups", writer)
	cmd.AddCommand(cmdDropletBackups)
	addIntFlag(cmdDropletBackups, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletCreate := cmdBuilder(RunDropletCreate,
		"create", "create droplet", writer)
	cmd.AddCommand(cmdDropletCreate)
	addStringSliceFlag(cmdDropletCreate, doit.ArgSSHKeys, []string{}, "SSH Keys or fingerprints")
	addStringFlag(cmdDropletCreate, doit.ArgUserData, "", "User data")
	addStringFlag(cmdDropletCreate, doit.ArgUserDataFile, "", "User data file")
	addBoolFlag(cmdDropletCreate, doit.ArgDropletWait, false, "Wait for droplet to be created")
	addStringFlag(cmdDropletCreate, doit.ArgDropletName, "", "Droplet name")
	addStringFlag(cmdDropletCreate, doit.ArgRegionSlug, "", "Droplet region")
	addBoolFlag(cmdDropletCreate, doit.ArgBackups, false, "Backup droplet")
	addBoolFlag(cmdDropletCreate, doit.ArgIPv6, false, "IPv6 support")
	addBoolFlag(cmdDropletCreate, doit.ArgPrivateNetworking, false, "Private networking")
	addStringFlag(cmdDropletCreate, doit.ArgImage, "", "Droplet image")

	cmdDropletDelete := cmdBuilder(RunDropletDelete,
		"delete", "delete droplet", writer)
	cmd.AddCommand(cmdDropletDelete)
	addIntFlag(cmdDropletDelete, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletGet := cmdBuilder(RunDropletGet,
		"get", "get droplet", writer)
	cmd.AddCommand(cmdDropletGet)
	addIntFlag(cmdDropletGet, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletKernels := cmdBuilder(RunDropletKernels,
		"kernels", "droplet kernels", writer)
	cmd.AddCommand(cmdDropletKernels)
	addIntFlag(cmdDropletKernels, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletList := cmdBuilder(RunDropletList,
		"list", "list droplets", writer)
	cmd.AddCommand(cmdDropletList)

	cmdDropletNeighbors := cmdBuilder(RunDropletNeighbors,
		"neighbors", "droplet neighbors", writer)
	cmd.AddCommand(cmdDropletNeighbors)
	addIntFlag(cmdDropletNeighbors, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletSnapshots := cmdBuilder(RunDropletSnapshots,
		"snapshots", "snapshots", writer)
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
			checkErr(RunDropletActions(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActions returns a list of actions for a droplet.
func RunDropletActions(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

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
func RunDropletBackups(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

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
func RunDropletCreate(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()

	sshKeys := []godo.DropletCreateSSHKey{}
	for _, rawKey := range doit.DoitConfig.GetStringSlice(ns, doit.ArgSSHKeys) {
		if i, err := strconv.Atoi(rawKey); err == nil {
			sshKeys = append(sshKeys, godo.DropletCreateSSHKey{ID: i})
			continue
		}

		sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: rawKey})
	}

	userData := doit.DoitConfig.GetString(ns, doit.ArgUserData)
	if userData == "" && doit.DoitConfig.GetString(ns, doit.ArgUserDataFile) != "" {
		data, err := ioutil.ReadFile(doit.DoitConfig.GetString(ns, doit.ArgUserDataFile))
		if err != nil {
			return err
		}
		userData = string(data)
	}

	wait := doit.DoitConfig.GetBool(ns, doit.ArgDropletWait)

	dcr := &godo.DropletCreateRequest{
		Name:              doit.DoitConfig.GetString(ns, doit.ArgDropletName),
		Region:            doit.DoitConfig.GetString(ns, doit.ArgRegionSlug),
		Size:              doit.DoitConfig.GetString(ns, doit.ArgSizeSlug),
		Backups:           doit.DoitConfig.GetBool(ns, doit.ArgBackups),
		IPv6:              doit.DoitConfig.GetBool(ns, doit.ArgIPv6),
		PrivateNetworking: doit.DoitConfig.GetBool(ns, doit.ArgPrivateNetworking),
		SSHKeys:           sshKeys,
		UserData:          userData,
	}

	imageStr := doit.DoitConfig.GetString(ns, doit.ArgImage)
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
func RunDropletDelete(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

	_, err := client.Droplets.Delete(id)
	return err
}

// RunDropletGet returns a droplet.
func RunDropletGet(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

	droplet, err := getDropletByID(client, id)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(droplet, out)
}

// RunDropletKernels returns a list of available kernels for a droplet.
func RunDropletKernels(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

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
func RunDropletList(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()

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
func RunDropletNeighbors(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

	list, _, err := client.Droplets.Neighbors(id)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(list, out)
}

// RunDropletSnapshots returns a list of available kernels for a droplet.
func RunDropletSnapshots(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

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
