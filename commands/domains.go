package commands

import (
	"errors"
	"fmt"
	"io"
	"strconv"

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

	cmdDomainCreate := cmdBuilder(cmd, RunDomainCreate, "create <domain>", "create domain", writer,
		aliasOpt("c"), displayerType(&domain{}))
	addStringFlag(cmdDomainCreate, doit.ArgIPAddress, "", "IP address", requiredOpt())

	cmdBuilder(cmd, RunDomainList, "list", "list comains", writer,
		aliasOpt("ls"), displayerType(&domain{}))

	cmdBuilder(cmd, RunDomainGet, "get <domain>", "get domain", writer,
		aliasOpt("g"), displayerType(&domain{}))

	cmdBuilder(cmd, RunDomainDelete, "delete <domain>", "delete droplet", writer, aliasOpt("g"))

	cmdRecord := &cobra.Command{
		Use:   "records",
		Short: "domain record commands",
		Long:  "commands for interacting with an individual domain",
	}
	cmd.AddCommand(cmdRecord)

	cmdRecordList := cmdBuilder(cmdRecord, RunRecordList, "list <domain>", "list records", writer,
		aliasOpt("ls"), displayerType(&domainRecord{}))
	addStringFlag(cmdRecordList, doit.ArgDomainName, "", "Domain name")

	cmdRecordCreate := cmdBuilder(cmdRecord, RunRecordCreate, "create <domain>", "create record", writer,
		aliasOpt("c"), displayerType(&domainRecord{}))
	addStringFlag(cmdRecordCreate, doit.ArgRecordType, "", "Record type")
	addStringFlag(cmdRecordCreate, doit.ArgRecordName, "", "Record name")
	addStringFlag(cmdRecordCreate, doit.ArgRecordData, "", "Record data")
	addIntFlag(cmdRecordCreate, doit.ArgRecordPriority, 0, "Record priority")
	addIntFlag(cmdRecordCreate, doit.ArgRecordPort, 0, "Record port")
	addIntFlag(cmdRecordCreate, doit.ArgRecordWeight, 0, "Record weight")

	cmdBuilder(cmdRecord, RunRecordDelete, "delete <domain> <record id...>", "delete record", writer, aliasOpt("d"))

	cmdRecordUpdate := cmdBuilder(cmdRecord, RunRecordUpdate, "update <domain>", "update record", writer,
		aliasOpt("u"), displayerType(&domainRecord{}))
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
func RunDomainCreate(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}
	domainName := args[0]

	client := config.GetGodoClient()

	ipAddress, err := config.GetString(ns, "ip-address")
	if err != nil {
		return err
	}

	req := &godo.DomainCreateRequest{
		Name:      domainName,
		IPAddress: ipAddress,
	}

	d, _, err := client.Domains.Create(req)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &domain{domains: domains{*d}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDomainList runs domain create.
func RunDomainList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

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

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &domain{domains: list},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDomainGet retrieves a domain by name.
func RunDomainGet(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}
	id := args[0]

	client := config.GetGodoClient()

	if len(id) < 1 {
		return errors.New("invalid domain name")
	}

	d, _, err := client.Domains.Get(id)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &domain{domains: domains{*d}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunDomainDelete deletes a domain by name.
func RunDomainDelete(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}
	name := args[0]

	client := config.GetGodoClient()

	if len(name) < 1 {
		return errors.New("invalid domain name")
	}

	_, err := client.Domains.Delete(name)
	return err
}

// RunRecordList list records for a domain.
func RunRecordList(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}
	name := args[0]

	client := config.GetGodoClient()

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

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &domainRecord{domainRecords: list},
		out:    out,
	}

	return displayOutput(dc)
}

// RunRecordCreate creates a domain record.
func RunRecordCreate(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}
	name := args[0]

	client := config.GetGodoClient()

	rType, err := config.GetString(ns, doit.ArgRecordType)
	if err != nil {
		return err
	}

	rName, err := config.GetString(ns, doit.ArgRecordName)
	if err != nil {
		return err
	}

	rData, err := config.GetString(ns, doit.ArgRecordData)
	if err != nil {
		return err
	}

	rPriority, err := config.GetInt(ns, doit.ArgRecordPriority)
	if err != nil {
		return err
	}

	rPort, err := config.GetInt(ns, doit.ArgRecordPort)
	if err != nil {
		return err
	}

	rWeight, err := config.GetInt(ns, doit.ArgRecordWeight)
	if err != nil {
		return err
	}

	drcr := &godo.DomainRecordEditRequest{
		Type:     rType,
		Name:     rName,
		Data:     rData,
		Priority: rPriority,
		Port:     rPort,
		Weight:   rWeight,
	}

	if len(drcr.Type) == 0 {
		return errors.New("record request is missing type")
	}

	r, _, err := client.Domains.CreateRecord(name, drcr)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &domainRecord{domainRecords{*r}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunRecordDelete deletes a domain record.
func RunRecordDelete(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) < 2 {
		return doit.NewMissingArgsErr(ns)
	}

	domainName, ids := args[0], args[1:]
	if len(ids) < 1 {
		return doit.NewMissingArgsErr(ns)
	}

	client := config.GetGodoClient()

	for _, i := range ids {
		id, err := strconv.Atoi(i)
		if err != nil {
			return fmt.Errorf("invalid record id %q", i)
		}

		_, err = client.Domains.DeleteRecord(domainName, id)
		if err != nil {
			return err
		}
	}

	return nil
}

// RunRecordUpdate updates a domain record.
func RunRecordUpdate(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}
	domainName := args[0]

	client := config.GetGodoClient()

	recordID, err := config.GetInt(ns, doit.ArgRecordID)
	if err != nil {
		return err
	}

	rType, err := config.GetString(ns, doit.ArgRecordType)
	if err != nil {
		return err
	}

	rName, err := config.GetString(ns, doit.ArgRecordName)
	if err != nil {
		return err
	}

	rData, err := config.GetString(ns, doit.ArgRecordData)
	if err != nil {
		return err
	}

	rPriority, err := config.GetInt(ns, doit.ArgRecordPriority)
	if err != nil {
		return err
	}

	rPort, err := config.GetInt(ns, doit.ArgRecordPort)
	if err != nil {
		return err
	}

	rWeight, err := config.GetInt(ns, doit.ArgRecordWeight)
	if err != nil {
		return err
	}

	drcr := &godo.DomainRecordEditRequest{
		Type:     rType,
		Name:     rName,
		Data:     rData,
		Priority: rPriority,
		Port:     rPort,
		Weight:   rWeight,
	}

	r, _, err := client.Domains.EditRecord(domainName, recordID, drcr)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &domainRecord{domainRecords{*r}},
		out:    out,
	}

	return displayOutput(dc)
}
