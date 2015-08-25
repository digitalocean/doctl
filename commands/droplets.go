package commands

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
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

	cmdDropletActions := NewCmdDropletActions(os.Stdout)
	cmd.AddCommand(cmdDropletActions)
	addIntFlag(cmdDropletActions, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletBackups := NewCmdDropletBackups(os.Stdout)
	cmd.AddCommand(cmdDropletBackups)
	addIntFlag(cmdDropletBackups, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletCreate := NewCmdDropletCreate(os.Stdout)
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

	cmdDropletDelete := NewCmdDropletDelete(os.Stdout)
	cmd.AddCommand(cmdDropletDelete)
	addIntFlag(cmdDropletDelete, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletGet := NewCmdDropletGet(os.Stdout)
	cmd.AddCommand(cmdDropletGet)
	addIntFlag(cmdDropletGet, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletKernels := NewCmdDropletKernels(os.Stdout)
	cmd.AddCommand(cmdDropletKernels)
	addIntFlag(cmdDropletKernels, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletList := NewCmdDropletList(os.Stdout)
	cmd.AddCommand(cmdDropletList)

	cmdDropletNeighbors := NewCmdDropletNeighbors(os.Stdout)
	cmd.AddCommand(cmdDropletNeighbors)
	addIntFlag(cmdDropletNeighbors, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletSnapshots := NewCmdDropletSnapshots(os.Stdout)
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
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

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

// NewCmdDropletBackups creates a droplet backups command.
func NewCmdDropletBackups(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "backups",
		Short: "get droplet backups",
		Long:  "get droplet backups",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletBackups(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletBackups returns a list of backup images for a droplet.
func RunDropletBackups(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

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

// NewCmdDropletCreate creates a droplet create command.
func NewCmdDropletCreate(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "create droplet",
		Long:  "create droplet",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletCreate(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletCreate creates a droplet.
func RunDropletCreate(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()

	sshKeys := []godo.DropletCreateSSHKey{}
	for _, rawKey := range doit.VConfig.GetStringSlice(ns, doit.ArgSSHKeys) {
		if i, err := strconv.Atoi(rawKey); err == nil {
			sshKeys = append(sshKeys, godo.DropletCreateSSHKey{ID: i})
			continue
		}

		sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: rawKey})
	}

	userData := doit.VConfig.GetString(ns, doit.ArgUserData)
	if userData == "" && doit.VConfig.GetString(ns, doit.ArgUserDataFile) != "" {
		data, err := ioutil.ReadFile(doit.VConfig.GetString(ns, doit.ArgUserDataFile))
		if err != nil {
			return err
		}
		userData = string(data)
	}

	wait := doit.VConfig.GetBool(ns, doit.ArgDropletWait)

	dcr := &godo.DropletCreateRequest{
		Name:              doit.VConfig.GetString(ns, doit.ArgDropletName),
		Region:            doit.VConfig.GetString(ns, doit.ArgRegionSlug),
		Size:              doit.VConfig.GetString(ns, doit.ArgSizeSlug),
		Backups:           doit.VConfig.GetBool(ns, doit.ArgBackups),
		IPv6:              doit.VConfig.GetBool(ns, doit.ArgIPv6),
		PrivateNetworking: doit.VConfig.GetBool(ns, doit.ArgPrivateNetworking),
		SSHKeys:           sshKeys,
		UserData:          userData,
	}

	imageStr := doit.VConfig.GetString(ns, doit.ArgImage)
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

// NewCmdDropletDelete creates a droplet delete command.
func NewCmdDropletDelete(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "delete droplet",
		Long:  "delete droplet",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletDelete(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletDelete destroy a droplet by id.
func RunDropletDelete(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

	_, err := client.Droplets.Delete(id)
	return err
}

// NewCmdDropletGet creates a droplet get command.
func NewCmdDropletGet(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "get droplet",
		Long:  "get droplet",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletGet(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletGet returns a droplet.
func RunDropletGet(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

	droplet, err := getDropletByID(client, id)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(droplet, out)
}

// NewCmdDropletKernels creates a droplet kernels command.
func NewCmdDropletKernels(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "kernels",
		Short: "droplet kernels",
		Long:  "droplet kernels",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletKernels(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletKernels returns a list of available kernels for a droplet.
func RunDropletKernels(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

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

// NewCmdDropletList creates a droplet list command.
func NewCmdDropletList(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list droplet",
		Long:  "list droplet",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletList(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletList returns a list of droplets.
func RunDropletList(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()

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

// NewCmdDropletNeighbors creates a droplet neighbors command.
func NewCmdDropletNeighbors(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "neighbors",
		Short: "droplet neighbors",
		Long:  "droplet neighbors",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletNeighbors(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletNeighbors returns a list of droplet neighbors.
func RunDropletNeighbors(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

	list, _, err := client.Droplets.Neighbors(id)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(list, out)
}

// NewCmdDropletSnapshots creates a droplet snapshots command.
func NewCmdDropletSnapshots(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "snapshots",
		Short: "droplet snapshots",
		Long:  "droplet snapshots",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletSnapshots(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletSnapshots returns a list of available kernels for a droplet.
func RunDropletSnapshots(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

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
