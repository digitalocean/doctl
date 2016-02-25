package commands

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
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

	cmdBuilder(cmd, RunDomainList, "list", "list domains", writer,
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
func RunDomainCreate(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}
	domainName := c.args[0]

	ds := c.domainsService()

	ipAddress, err := c.doitConfig.GetString(c.ns, "ip-address")
	if err != nil {
		return err
	}

	req := &godo.DomainCreateRequest{
		Name:      domainName,
		IPAddress: ipAddress,
	}

	d, err := ds.Create(req)
	if err != nil {
		return err
	}

	return c.display(&domain{domains: do.Domains{*d}})
}

// RunDomainList runs domain create.
func RunDomainList(c *cmdConfig) error {

	ds := c.domainsService()

	domains, err := ds.List()
	if err != nil {
		return err
	}

	item := &domain{domains: domains}
	return c.display(item)
}

// RunDomainGet retrieves a domain by name.
func RunDomainGet(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}
	id := c.args[0]

	ds := c.domainsService()

	if len(id) < 1 {
		return errors.New("invalid domain name")
	}

	d, err := ds.Get(id)
	if err != nil {
		return err
	}

	item := &domain{domains: do.Domains{*d}}
	return c.display(item)
}

// RunDomainDelete deletes a domain by name.
func RunDomainDelete(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}
	name := c.args[0]

	ds := c.domainsService()

	if len(name) < 1 {
		return errors.New("invalid domain name")
	}

	err := ds.Delete(name)
	return err
}

// RunRecordList list records for a domain.
func RunRecordList(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}
	name := c.args[0]

	ds := c.domainsService()

	if len(name) < 1 {
		return errors.New("domain name is missing")
	}

	list, err := ds.Records(name)
	if err != nil {
		return err
	}

	items := &domainRecord{domainRecords: list}
	return c.display(items)

}

// RunRecordCreate creates a domain record.
func RunRecordCreate(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}
	name := c.args[0]

	ds := c.domainsService()

	rType, err := c.doitConfig.GetString(c.ns, doit.ArgRecordType)
	if err != nil {
		return err
	}

	rName, err := c.doitConfig.GetString(c.ns, doit.ArgRecordName)
	if err != nil {
		return err
	}

	rData, err := c.doitConfig.GetString(c.ns, doit.ArgRecordData)
	if err != nil {
		return err
	}

	rPriority, err := c.doitConfig.GetInt(c.ns, doit.ArgRecordPriority)
	if err != nil {
		return err
	}

	rPort, err := c.doitConfig.GetInt(c.ns, doit.ArgRecordPort)
	if err != nil {
		return err
	}

	rWeight, err := c.doitConfig.GetInt(c.ns, doit.ArgRecordWeight)
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

	r, err := ds.CreateRecord(name, drcr)
	if err != nil {
		return err
	}

	item := &domainRecord{domainRecords: do.DomainRecords{*r}}
	return c.display(item)

}

// RunRecordDelete deletes a domain record.
func RunRecordDelete(c *cmdConfig) error {
	if len(c.args) < 2 {
		return doit.NewMissingArgsErr(c.ns)
	}

	domainName, ids := c.args[0], c.args[1:]
	if len(ids) < 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	ds := c.domainsService()

	for _, i := range ids {
		id, err := strconv.Atoi(i)
		if err != nil {
			return fmt.Errorf("invalid record id %q", i)
		}

		err = ds.DeleteRecord(domainName, id)
		if err != nil {
			return err
		}
	}

	return nil
}

// RunRecordUpdate updates a domain record.
func RunRecordUpdate(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}
	domainName := c.args[0]

	ds := c.domainsService()

	recordID, err := c.doitConfig.GetInt(c.ns, doit.ArgRecordID)
	if err != nil {
		return err
	}

	rType, err := c.doitConfig.GetString(c.ns, doit.ArgRecordType)
	if err != nil {
		return err
	}

	rName, err := c.doitConfig.GetString(c.ns, doit.ArgRecordName)
	if err != nil {
		return err
	}

	rData, err := c.doitConfig.GetString(c.ns, doit.ArgRecordData)
	if err != nil {
		return err
	}

	rPriority, err := c.doitConfig.GetInt(c.ns, doit.ArgRecordPriority)
	if err != nil {
		return err
	}

	rPort, err := c.doitConfig.GetInt(c.ns, doit.ArgRecordPort)
	if err != nil {
		return err
	}

	rWeight, err := c.doitConfig.GetInt(c.ns, doit.ArgRecordWeight)
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

	r, err := ds.EditRecord(domainName, recordID, drcr)
	if err != nil {
		return err
	}

	item := &domainRecord{domainRecords: do.DomainRecords{*r}}
	return c.display(item)
}
