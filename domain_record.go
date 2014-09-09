package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
)

var DomainRecordCommand = cli.Command{
	Name:  "record",
	Usage: "Domain record commands.",
	Subcommands: []cli.Command{
		{
			Name:   "list",
			Usage:  "List domain records.",
			Action: domainRecordList,
		},
		{
			Name:   "show",
			Usage:  "Show domain record.",
			Action: domainRecordShow,
		},
		{
			Name:  "create",
			Usage: "Create domain record.",
			Flags: []cli.Flag{
				cli.IntFlag{Name: "priority", Value: 0, Usage: "Priority for domain record. (Type: MX, SRV)"},
				cli.IntFlag{Name: "port", Value: 0, Usage: "Port for domain record. (Type: SRV)"},
				cli.IntFlag{Name: "weight", Value: 0, Usage: "Weight for domain record. (Type: SRV)"},
			},
			Action: domainRecordCreate,
		},
		{
			Name:   "destroy",
			Usage:  "Destroy domain record.",
			Action: domainRecordDestroy,
		},
	},
}

func domainRecordList(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Domain.\n")
		os.Exit(64)
	}

	client := apiv2.NewClient(APIKey)

	domain := ctx.Args().First()

	records, err := client.ListAllDomainRecords(domain)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	domainMap := make(map[string]*apiv2.DomainRecordList, 1)

	domainMap[domain] = records

	WriteOutput(domainMap)

}

func domainRecordShow(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide domain record name.\n")
		os.Exit(64)
	}

	client := apiv2.NewClient(APIKey)

	recordName := ctx.Args().First()

	record, err := client.LoadDomainRecord(recordName)
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	WriteOutput(record)
}

func domainRecordCreate(ctx *cli.Context) {
	if len(ctx.Args()) < 3 {
		cli.ShowAppHelp(ctx)
		fmt.Printf("Invalid arguments.\n")
		os.Exit(1)
	}

	client := apiv2.NewClient(APIKey)

	domainRecord := client.NewDomainRecord()
	domainRecord.Name = ctx.Args().First()
	domainRecord.Type = ctx.Args()[1]
	domainRecord.Data = ctx.Args()[2]
	if domainRecord.Type == "MX" || domainRecord.Type == "SRV" {
		domainRecord.Priority = ctx.Int("priority")
	}
	if domainRecord.Type == "SRV" {
		domainRecord.Port = ctx.Int("port")
		domainRecord.Weight = ctx.Int("weight")
	}

	domain, err := client.FindDomainFromName(domainRecord.Name)
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	domainRecord, createErr := client.CreateDomainRecord(domainRecord, domain)
	if createErr != nil {
		fmt.Printf("%s\n", createErr)
		os.Exit(1)
	}

	WriteOutput(domainRecord)
}

func domainRecordDestroy(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide FQDN for domain record.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()

	client := apiv2.NewClient(APIKey)

	err := client.DestroyDomainRecord(name)
	if err != nil {
		fmt.Printf("Unable to destroy domain record: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Domain record %s destroyed.\n", name)
}
