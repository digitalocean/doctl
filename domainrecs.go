package doit

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func RecordCreate(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	domainName := c.String("domain-name")

	drcr := &godo.DomainRecordEditRequest{
		Type:     c.String("record-type"),
		Name:     c.String("record-name"),
		Data:     c.String("record-data"),
		Priority: c.Int("record-priority"),
		Port:     c.Int("record-port"),
		Weight:   c.Int("record-weight"),
	}

	r, _, err := client.Domains.CreateRecord(domainName, drcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create record")
	}

	WriteJSON(r, c.App.Writer)
}

func RecordDelete(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	domainName := c.String("domain-name")
	recordID := c.Int("record-id")

	_, err := client.Domains.DeleteRecord(domainName, recordID)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not delete record")
	}
}

// List records for a domain.
func RecordList(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	opts := LoadOpts(c)
	name := c.String("domain-name")

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

	si, err := PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list domain")
	}

	list := make([]godo.DomainRecord, len(si))
	for i := range si {
		list[i] = si[i].(godo.DomainRecord)
	}

	WriteJSON(list, c.App.Writer)
}

// Retrieve a domain record.
func RecordGet(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	domainName := c.String("domain-name")
	recordID := c.Int("record-id")

	r, _, err := client.Domains.Record(domainName, recordID)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not display record")
	}

	WriteJSON(r, c.App.Writer)
}

func RecordUpdate(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	domainName := c.String("domain-name")
	recordID := c.Int("record-id")

	drcr := &godo.DomainRecordEditRequest{
		Type:     c.String("record-type"),
		Name:     c.String("record-name"),
		Data:     c.String("record-data"),
		Priority: c.Int("record-priority"),
		Port:     c.Int("record-port"),
		Weight:   c.Int("record-weight"),
	}

	r, _, err := client.Domains.EditRecord(domainName, recordID, drcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not update record")
	}

	WriteJSON(r, c.App.Writer)
}
