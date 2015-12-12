package commands

import (
	"errors"
	"io"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// FloatingIP creates the command heirarchy for floating ips.
func FloatingIP() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "floating-ip",
		Short:   "floating IP commands",
		Long:    "floating-ip is used to access commands on floating IPs",
		Aliases: []string{"fip"},
	}

	cmdFloatingIPCreate := cmdBuilder(RunFloatingIPCreate, "create", "create a floating IP", writer, aliasOpt("c"))
	cmd.AddCommand(cmdFloatingIPCreate)
	addStringFlag(cmdFloatingIPCreate, doit.ArgRegionSlug, "", "Region where to create the floating IP.", requiredOpt())
	addIntFlag(cmdFloatingIPCreate, doit.ArgDropletID, 0, "ID of the droplet to assign the IP to. (Optional)")

	cmdFloatingIPGet := cmdBuilder(RunFloatingIPGet, "get <floating-ip>", "get the details of a floating IP", writer, aliasOpt("g"))
	cmd.AddCommand(cmdFloatingIPGet)

	cmdFloatingIPDelete := cmdBuilder(RunFloatingIPDelete, "delete <floating-ip>", "delete a floating IP address", writer, aliasOpt("d"))
	cmd.AddCommand(cmdFloatingIPDelete)

	cmdFloatingIPList := cmdBuilder(RunFloatingIPList, "list", "list all floating IP addresses", writer, aliasOpt("ls"))
	cmd.AddCommand(cmdFloatingIPList)

	return cmd
}

// RunFloatingIPCreate runs floating IP create.
func RunFloatingIPCreate(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	region, err := config.GetString(ns, doit.ArgRegionSlug)
	if err != nil {
		return err
	}

	dropletID, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	req := &godo.FloatingIPCreateRequest{
		Region:    region,
		DropletID: dropletID,
	}
	ip, _, err := client.FloatingIPs.Create(req)
	if err != nil {
		return err
	}
	return displayOutput(&floatingIP{floatingIPs: floatingIPs{*ip}}, out)
}

// RunFloatingIPGet retrieves a floating IP's details.
func RunFloatingIPGet(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	ip := args[0]

	if len(ip) < 1 {
		return errors.New("invalid ip address")
	}

	d, _, err := client.FloatingIPs.Get(ip)
	if err != nil {
		return err
	}

	return displayOutput(&floatingIP{floatingIPs: floatingIPs{*d}}, out)
}

// RunFloatingIPDelete runs floating IP delete.
func RunFloatingIPDelete(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	ip := args[0]

	_, err := client.FloatingIPs.Delete(ip)
	return err
}

// RunFloatingIPList runs floating IP create.
func RunFloatingIPList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.FloatingIPs.List(opt)
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

	list := make([]godo.FloatingIP, len(si))
	for i := range si {
		list[i] = si[i].(godo.FloatingIP)
	}

	return displayOutput(&floatingIP{floatingIPs: list}, out)
}
