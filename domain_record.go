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
			Name:   "create",
			Usage:  "Create domain record.",
			Action: domainRecordCreate,
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

}

func domainRecordDestroy(ctx *cli.Context) {

}
