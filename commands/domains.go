package commands

import (
	"io"
	"os"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Domain creates the domain commands heirarchy.
func Domain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "domain commands",
		Long:  "domain is used to access domain commands",
	}

	cmdDomainCreate := NewCmdDomainCreate(os.Stdout)
	addStringFlag(cmdDomainCreate, doit.ArgDomainName, "", "Domain name")
	addStringFlag(cmdDomainCreate, doit.ArgIPAddress, "", "IP Address")
	cmd.AddCommand(cmdDomainCreate)

	cmdDomainList := NewCmdDomainList(os.Stdout)
	cmd.AddCommand(cmdDomainList)

	return cmd
}

// NewCmdDomainCreate creates a domain create command.
func NewCmdDomainCreate(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "create domain",
		Long:  "create a domain",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDomainCreate(out))
		},
	}
}

// RunDomainCreate runs domain create.
func RunDomainCreate(out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	req := &godo.DomainCreateRequest{
		Name:      doit.VConfig.GetString("domain-name"),
		IPAddress: doit.VConfig.GetString("ip-address"),
	}

	d, _, err := client.Domains.Create(req)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(d, out)
}

// NewCmdDomainList creates a a domain list command.
func NewCmdDomainList(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list domains",
		Long:  "list all domains",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDomainList(out))
		},
	}
}

// RunDomainList runs domain create.
func RunDomainList(out io.Writer) error {
	client := doit.VConfig.GetGodoClient()

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Domains.List(opt)
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

	list := make([]godo.Domain, len(si))
	for i := range si {
		list[i] = si[i].(godo.Domain)
	}

	return doit.DisplayOutput(list, out)
}
