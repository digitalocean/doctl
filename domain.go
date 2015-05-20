package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"

	"golang.org/x/oauth2"
)

var DomainCommand = cli.Command{
	Name:    "domain",
	Aliases: []string{"dns"},
	Usage:   "Domain commands.",
	Action:  domainList,
	Subcommands: []cli.Command{
		DomainRecordCommand,
		{
			Name:    "show",
			Aliases: []string{"s"},
			Usage:   "Show an domain.",
			Action:  domainShow,
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List all domains.",
			Flags: []cli.Flag{
				cli.IntFlag{Name: "page", Value: 0, Usage: "What number page of actions to fetch backward in history. Most recent first."},
				cli.IntFlag{Name: "page-size", Value: 20, Usage: "Number of actions to fetch per page."},
			},
			Action: domainList,
		},
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "<domain> <Droplet name> Create new domain.",
			Action:  domainCreate,
		},
		{
			Name:    "destroy",
			Aliases: []string{"d"},
			Usage:   "Destroy a domain.",
			Action:  domainDestroy,
		},
	},
}

func domainShow(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		fmt.Printf("Error: Must provide name for Domain.\n")
		os.Exit(64)
	}

	name := ctx.Args().First()

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	domain, _, err := client.Domains.Get(name)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	WriteOutput(domain)
}

func domainList(ctx *cli.Context) {
	if ctx.BoolT("help") == true {
		cli.ShowAppHelp(ctx)
		os.Exit(1)
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	opt := &godo.ListOptions{
		Page:    ctx.Int("page"),
		PerPage: ctx.Int("page-size"),
	}
	domainList, _, err := client.Domains.List(opt)
	if err != nil {
		fmt.Printf("Unable to list Domains: %s\n", err)
		os.Exit(1)
	}

	cliOut := NewCLIOutput()
	defer cliOut.Flush()
	cliOut.Header("Name", "TTL")
	for _, domain := range domainList {
		cliOut.Writeln("%s\t%d\n", domain.Name, domain.TTL)
	}
}

func domainCreate(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		fmt.Printf("Must provide domain name and droplet name.\n")
		os.Exit(1)
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	droplet, err := FindDropletByName(client, ctx.Args()[1])
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(64)
	}

	createRequest := &godo.DomainCreateRequest{
		Name:      ctx.Args().First(),
		IPAddress: PublicIPForDroplet(droplet),
	}
	domainRoot, _, err := client.Domains.Create(createRequest)

	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	WriteOutput(domainRoot.Domain)
}

func domainDestroy(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		fmt.Printf("Error: Must provide a name for the domain to destroy.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	_, err := client.Domains.Delete(name)
	if err != nil {
		fmt.Printf("Unable to destroy domain: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Domain %s destroyed.\n", name)
}
