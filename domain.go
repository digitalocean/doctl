package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
	"gopkg.in/yaml.v1"
)

var DomainCommand = cli.Command{
	Name:  "domain",
	Usage: "Domain commands.",
	Subcommands: []cli.Command{
		{
			Name:   "show",
			Usage:  "Show an domain.",
			Action: domainShow,
		},
		{
			Name:   "list",
			Usage:  "List all domains.",
			Action: domainList,
		},
		{
			Name:   "create",
			Usage:  "Create new domain.",
			Action: domainCreate,
		},
		{
			Name:   "destroy",
			Usage:  "Destroy a domain.",
			Action: domainDestroy,
		},
		{
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
			},
		},
	},
}

func domainShow(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Domain.\n")
		os.Exit(64)
	}

	name := ctx.Args().First()

	client := apiv2.NewClient(APIKey)

	domain, err := client.LoadDomain(name)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	data, errMarshal := yaml.Marshal(domain)
	if errMarshal != nil {
		fmt.Printf("YAML Error: %s", errMarshal)
		os.Exit(1)
	}
	fmt.Printf("%s", string(data))
}

func domainList(ctx *cli.Context) {
	client := apiv2.NewClient(APIKey)

	domainList, err := client.ListAllDomains()
	if err != nil {
		fmt.Printf("Unable to list Domains: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-16s\t%s\n", "Name", "TTL")
	for _, domain := range domainList.Domains {
		fmt.Printf("%-16s\t%d\n", domain.Name, domain.TTL)
	}
}

func domainCreate(ctx *cli.Context) {

}

func domainDestroy(ctx *cli.Context) {

}

func domainRecordList(ctx *cli.Context) {

}

func domainRecordShow(ctx *cli.Context) {

}
