package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
)

var DomainCommand = cli.Command{
	Name:  "domain",
	Usage: "Domain commands.",
	Subcommands: []cli.Command{
		DomainRecordCommand,
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

	WriteOutput(domain)
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
	if len(ctx.Args()) < 2 {
		fmt.Printf("Must provide domain name and droplet name.\n")
		os.Exit(1)
	}

	client := apiv2.NewClient(APIKey)

	droplet, err := client.FindDropletByName(ctx.Args()[1])
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(64)
	}

	domainRequest := client.NewDomainRequest(ctx.Args().First())
	domainRequest.IPAddress = droplet.PublicIPAddress()

	domain, createErr := client.CreateDomain(domainRequest)
	if createErr != nil {
		fmt.Printf("%s\n", createErr)
		os.Exit(1)
	}

	WriteOutput(domain)
}

func domainDestroy(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for domain.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()

	client := apiv2.NewClient(APIKey)

	err := client.DestroyDomain(name)
	if err != nil {
		fmt.Printf("Unable to destroy domain: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Domain %s destroyed.\n", name)
}
