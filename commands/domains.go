package commands

import (
	"errors"
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
	addStringFlag(cmdDomainCreate, doit.ArgIPAddress, "", "IP address")
	cmd.AddCommand(cmdDomainCreate)

	cmdDomainList := NewCmdDomainList(os.Stdout)
	cmd.AddCommand(cmdDomainList)

	cmdDomainGet := NewCmdDomainGet(os.Stdout)
	addStringFlag(cmdDomainGet, doit.ArgDomainName, "", "Domain name")
	cmd.AddCommand(cmdDomainGet)

	cmdDomainDelete := NewCmdDomainDelete(os.Stdout)
	addStringFlag(cmdDomainDelete, doit.ArgDomainName, "", "Domain name")
	cmd.AddCommand(cmdDomainDelete)

	recordCmd := &cobra.Command{
		Use:   "records",
		Short: "domain record commands",
		Long:  "commands for interacting with an individual domain",
	}
	cmd.AddCommand(recordCmd)

	cmdRecordList := NewCmdRecordList(os.Stdout)
	addStringFlag(cmdRecordList, doit.ArgDomainName, "", "Domain name")
	recordCmd.AddCommand(cmdRecordList)

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

// NewCmdDomainGet creates a command to retrieve a domain.
func NewCmdDomainGet(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "get domain",
		Long:  "retrieve an individual domain",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDomainGet(out))
		},
	}
}

// RunDomainGet retrieves a domain by name.
func RunDomainGet(out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetString(doit.ArgDomainName)

	if len(id) < 1 {
		return errors.New("invalid domain name")
	}

	d, _, err := client.Domains.Get(id)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(d, out)
}

// NewCmdDomainDelete creates a command to delete a domain.
func NewCmdDomainDelete(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "delete domain",
		Long:  "delete a domain an all associated records",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDomainDelete(out))
		},
	}
}

// RunDomainDelete deletes a domain by name.
func RunDomainDelete(out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	name := doit.VConfig.GetString(doit.ArgDomainName)

	if len(name) < 1 {
		return errors.New("invalid domain name")
	}

	_, err := client.Domains.Delete(name)
	return err
}

// NewCmdRecordList creates a domain record listing command.
func NewCmdRecordList(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list records",
		Long:  "list all records in a domain",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunRecordList(out))
		},
	}
}

// RunRecordList list records for a domain.
func RunRecordList(out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	name := doit.VConfig.GetString(doit.ArgDomainName)

	if len(name) < 1 {
		return errors.New("invalid domain name")
	}

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Domains.Records(name, opt)
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

	list := make([]godo.DomainRecord, len(si))
	for i := range si {
		list[i] = si[i].(godo.DomainRecord)
	}

	return doit.DisplayOutput(list, out)
}
