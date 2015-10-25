package commands

import (
	"errors"
	"io"

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

	cmdDomainCreate := cmdBuilder(RunDomainCreate, "create", "create domain", writer, "c")
	cmd.AddCommand(cmdDomainCreate)
	addStringFlag(cmdDomainCreate, doit.ArgDomainName, "", "Domain name")
	addStringFlag(cmdDomainCreate, doit.ArgIPAddress, "", "IP address")

	cmdDomainList := cmdBuilder(RunDomainList, "list", "list comains", writer, "ls")
	cmd.AddCommand(cmdDomainList)

	cmdDomainGet := cmdBuilder(RunDomainGet, "get", "get domain", writer, "g")
	cmd.AddCommand(cmdDomainGet)
	addStringFlag(cmdDomainGet, doit.ArgDomainName, "", "Domain name")

	cmdDomainDelete := cmdBuilder(RunDomainDelete, "delete", "delete droplet", writer, "g")
	cmd.AddCommand(cmdDomainDelete)
	addStringFlag(cmdDomainDelete, doit.ArgDomainName, "", "Domain name")

	cmdRecord := &cobra.Command{
		Use:   "records",
		Short: "domain record commands",
		Long:  "commands for interacting with an individual domain",
	}
	cmd.AddCommand(cmdRecord)

	cmdRecordList := cmdBuilder(RunRecordList, "list", "list records", writer, "ls")
	cmdRecord.AddCommand(cmdRecordList)
	addStringFlag(cmdRecordList, doit.ArgDomainName, "", "Domain name")

	cmdRecordCreate := cmdBuilder(RunRecordCreate, "create", "create record", writer, "c")
	cmdRecord.AddCommand(cmdRecordCreate)
	addStringFlag(cmdRecordCreate, doit.ArgDomainName, "", "Domain name")
	addStringFlag(cmdRecordCreate, doit.ArgRecordType, "", "Record type")
	addStringFlag(cmdRecordCreate, doit.ArgRecordName, "", "Record name")
	addStringFlag(cmdRecordCreate, doit.ArgRecordData, "", "Record data")
	addIntFlag(cmdRecordCreate, doit.ArgRecordPriority, 0, "Record priority")
	addIntFlag(cmdRecordCreate, doit.ArgRecordPort, 0, "Record port")
	addIntFlag(cmdRecordCreate, doit.ArgRecordWeight, 0, "Record weight")

	cmdRecordDelete := cmdBuilder(RunRecordDelete, "delete", "delete record", writer, "d")
	cmdRecord.AddCommand(cmdRecordDelete)
	addStringFlag(cmdRecordDelete, doit.ArgDomainName, "", "Domain name")
	addIntFlag(cmdRecordDelete, doit.ArgRecordID, 0, "Record ID")

	cmdRecordUpdate := cmdBuilder(RunRecordUpdate, "update", "update record", writer, "u")
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

// RunDomainCreate runs domain create.
func RunDomainCreate(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	req := &godo.DomainCreateRequest{
		Name:      doit.DoitConfig.GetString(ns, "domain-name"),
		IPAddress: doit.DoitConfig.GetString(ns, "ip-address"),
	}

	d, _, err := client.Domains.Create(req)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(d, out)
}

// RunDomainList runs domain create.
func RunDomainList(cmdName string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()

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

// RunDomainGet retrieves a domain by name.
func RunDomainGet(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	id := doit.DoitConfig.GetString(ns, doit.ArgDomainName)

	if len(id) < 1 {
		return errors.New("invalid domain name")
	}

	d, _, err := client.Domains.Get(id)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(d, out)
}

// RunDomainDelete deletes a domain by name.
func RunDomainDelete(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	name := doit.DoitConfig.GetString(ns, doit.ArgDomainName)

	if len(name) < 1 {
		return errors.New("invalid domain name")
	}

	_, err := client.Domains.Delete(name)
	return err
}

// RunRecordList list records for a domain.
func RunRecordList(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	name := doit.DoitConfig.GetString(ns, doit.ArgDomainName)

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

// RunRecordCreate creates a domain record.
func RunRecordCreate(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	name := doit.DoitConfig.GetString(ns, doit.ArgDomainName)

	drcr := &godo.DomainRecordEditRequest{
		Type:     doit.DoitConfig.GetString(ns, doit.ArgRecordType),
		Name:     doit.DoitConfig.GetString(ns, doit.ArgRecordName),
		Data:     doit.DoitConfig.GetString(ns, doit.ArgRecordData),
		Priority: doit.DoitConfig.GetInt(ns, doit.ArgRecordPriority),
		Port:     doit.DoitConfig.GetInt(ns, doit.ArgRecordPort),
		Weight:   doit.DoitConfig.GetInt(ns, doit.ArgRecordWeight),
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

// RunRecordDelete deletes a domain record.
func RunRecordDelete(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	domainName := doit.DoitConfig.GetString(ns, doit.ArgDomainName)
	recordID := doit.DoitConfig.GetInt(ns, doit.ArgRecordID)

	_, err := client.Domains.DeleteRecord(domainName, recordID)
	return err
}

// RunRecordUpdate updates a domain record.
func RunRecordUpdate(ns string, out io.Writer) error {
	client := doit.DoitConfig.GetGodoClient()
	domainName := doit.DoitConfig.GetString(ns, doit.ArgDomainName)
	recordID := doit.DoitConfig.GetInt(ns, doit.ArgRecordID)

	drcr := &godo.DomainRecordEditRequest{
		Type:     doit.DoitConfig.GetString(ns, doit.ArgRecordType),
		Name:     doit.DoitConfig.GetString(ns, doit.ArgRecordName),
		Data:     doit.DoitConfig.GetString(ns, doit.ArgRecordData),
		Priority: doit.DoitConfig.GetInt(ns, doit.ArgRecordPriority),
		Port:     doit.DoitConfig.GetInt(ns, doit.ArgRecordPort),
		Weight:   doit.DoitConfig.GetInt(ns, doit.ArgRecordWeight),
	}

	r, _, err := client.Domains.EditRecord(domainName, recordID, drcr)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(r, out)
}
