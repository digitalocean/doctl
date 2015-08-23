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
	cmd.AddCommand(cmdDomainCreate)
	addStringFlag(cmdDomainCreate, doit.ArgDomainName, "", "Domain name")
	addStringFlag(cmdDomainCreate, doit.ArgIPAddress, "", "IP address")

	cmdDomainList := NewCmdDomainList(os.Stdout)
	cmd.AddCommand(cmdDomainList)

	cmdDomainGet := NewCmdDomainGet(os.Stdout)
	cmd.AddCommand(cmdDomainGet)
	addStringFlag(cmdDomainGet, doit.ArgDomainName, "", "Domain name")

	cmdDomainDelete := NewCmdDomainDelete(os.Stdout)
	cmd.AddCommand(cmdDomainDelete)
	addStringFlag(cmdDomainDelete, doit.ArgDomainName, "", "Domain name")

	cmdRecord := &cobra.Command{
		Use:   "records",
		Short: "domain record commands",
		Long:  "commands for interacting with an individual domain",
	}
	cmd.AddCommand(cmdRecord)

	cmdRecordList := NewCmdRecordList(os.Stdout)
	cmdRecord.AddCommand(cmdRecordList)
	addStringFlag(cmdRecordList, doit.ArgDomainName, "", "Domain name")

	cmdRecordCreate := NewCmdRecordCreate(os.Stdout)
	cmdRecord.AddCommand(cmdRecordCreate)
	addStringFlag(cmdRecordCreate, doit.ArgDomainName, "", "Domain name")
	addStringFlag(cmdRecordCreate, doit.ArgRecordType, "", "Record type")
	addStringFlag(cmdRecordCreate, doit.ArgRecordName, "", "Record name")
	addStringFlag(cmdRecordCreate, doit.ArgRecordData, "", "Record data")
	addIntFlag(cmdRecordCreate, doit.ArgRecordPriority, 0, "Record priority")
	addIntFlag(cmdRecordCreate, doit.ArgRecordPort, 0, "Record port")
	addIntFlag(cmdRecordCreate, doit.ArgRecordWeight, 0, "Record weight")

	cmdRecordDelete := NewCmdRecordDelete(os.Stdout)
	cmdRecord.AddCommand(cmdRecordDelete)
	addStringFlag(cmdRecordDelete, doit.ArgDomainName, "", "Domain name")
	addIntFlag(cmdRecordDelete, doit.ArgRecordID, 0, "Record ID")

	cmdRecordUpdate := NewCmdRecordUpdate(os.Stdout)
	cmdRecord.AddCommand(cmdRecordUpdate)
	addStringFlag(cmdRecordUpdate, doit.ArgDomainName, "", "Domain name")
	addIntFlag(cmdRecordUpdate, doit.ArgRecordID, 0, "Record ID")
	addStringFlag(cmdRecordUpdate, doit.ArgRecordType, "", "Record type")
	addStringFlag(cmdRecordUpdate, doit.ArgRecordName, "", "Record name")
	addStringFlag(cmdRecordUpdate, doit.ArgRecordData, "", "Record data")
	addIntFlag(cmdRecordUpdate, doit.ArgRecordPriority, 0, "Record priority")
	addIntFlag(cmdRecordUpdate, doit.ArgRecordPort, 0, "Record port")
	addIntFlag(cmdRecordUpdate, doit.ArgRecordWeight, 0, "Record weight")

	return cmd
}

// NewCmdDomainCreate creates a domain create command.
func NewCmdDomainCreate(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "create domain",
		Long:  "create a domain",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDomainCreate(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDomainCreate runs domain create.
func RunDomainCreate(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	req := &godo.DomainCreateRequest{
		Name:      doit.VConfig.GetString(ns, "domain-name"),
		IPAddress: doit.VConfig.GetString(ns, "ip-address"),
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
			checkErr(RunDomainList(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDomainList runs domain create.
func RunDomainList(cmdName string, out io.Writer) error {
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
			checkErr(RunDomainGet(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDomainGet retrieves a domain by name.
func RunDomainGet(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetString(ns, doit.ArgDomainName)

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
			checkErr(RunDomainDelete(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDomainDelete deletes a domain by name.
func RunDomainDelete(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	name := doit.VConfig.GetString(ns, doit.ArgDomainName)

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
			checkErr(RunRecordList(cmdNS(cmd), out), cmd)
		},
	}
}

// RunRecordList list records for a domain.
func RunRecordList(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	name := doit.VConfig.GetString(ns, doit.ArgDomainName)

	if len(name) < 1 {
		return errors.New("domain name is missing")
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

// NewCmdRecordCreate creates a record create command.
func NewCmdRecordCreate(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "create record",
		Long:  "create record for a domain",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunRecordCreate(cmdNS(cmd), out), cmd)
		},
	}
}

// RunRecordCreate creates a domain record.
func RunRecordCreate(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	name := doit.VConfig.GetString(ns, doit.ArgDomainName)

	drcr := &godo.DomainRecordEditRequest{
		Type:     doit.VConfig.GetString(ns, doit.ArgRecordType),
		Name:     doit.VConfig.GetString(ns, doit.ArgRecordName),
		Data:     doit.VConfig.GetString(ns, doit.ArgRecordData),
		Priority: doit.VConfig.GetInt(ns, doit.ArgRecordPriority),
		Port:     doit.VConfig.GetInt(ns, doit.ArgRecordPort),
		Weight:   doit.VConfig.GetInt(ns, doit.ArgRecordWeight),
	}

	if len(drcr.Type) == 0 {
		return errors.New("record request is missing type")
	}

	r, _, err := client.Domains.CreateRecord(name, drcr)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(r, out)
}

// NewCmdRecordDelete creates a record create command.
func NewCmdRecordDelete(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "delete record",
		Long:  "delete record for a domain by record id",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunRecordDelete(cmdNS(cmd), out), cmd)
		},
	}
}

// RunRecordDelete deletes a domain record.
func RunRecordDelete(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	domainName := doit.VConfig.GetString(ns, doit.ArgDomainName)
	recordID := doit.VConfig.GetInt(ns, doit.ArgRecordID)

	_, err := client.Domains.DeleteRecord(domainName, recordID)
	return err
}

// NewCmdRecordUpdate creates a command which updates a domain record.
func NewCmdRecordUpdate(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "update record",
		Long:  "update record for a domain by record id",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunRecordUpdate(cmdNS(cmd), out), cmd)
		},
	}
}

// RunRecordUpdate updates a domain record.
func RunRecordUpdate(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	domainName := doit.VConfig.GetString(ns, doit.ArgDomainName)
	recordID := doit.VConfig.GetInt(ns, doit.ArgRecordID)

	drcr := &godo.DomainRecordEditRequest{
		Type:     doit.VConfig.GetString(ns, doit.ArgRecordType),
		Name:     doit.VConfig.GetString(ns, doit.ArgRecordName),
		Data:     doit.VConfig.GetString(ns, doit.ArgRecordData),
		Priority: doit.VConfig.GetInt(ns, doit.ArgRecordPriority),
		Port:     doit.VConfig.GetInt(ns, doit.ArgRecordPort),
		Weight:   doit.VConfig.GetInt(ns, doit.ArgRecordWeight),
	}

	r, _, err := client.Domains.EditRecord(domainName, recordID, drcr)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(r, out)
}
