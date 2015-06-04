package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/digitalocean/godo"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/golang.org/x/oauth2"
)

var DomainCommand = cli.Command{
	Name:    "domain",
	Aliases: []string{"dns"},
	Usage:   "Domain commands.",
	Action:  domainList,
	Subcommands: []cli.Command{
		{
			Name:    "show",
			Aliases: []string{"s"},
			Usage:   "<name> Show an domain.",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "verbose, v", Usage: "Include domain records in listing"},
			},
			Action: domainShow,
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List all domains.",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "verbose, v", Usage: "Include domain records in listing"}, // TODO
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
			Usage:   "<name> Destroy a domain.",
			Action:  domainDestroy,
		},
		{
			Name:    "list-records",
			Aliases: []string{"records", "r"},
			Usage:   "<domain> List domain records for a domain.",
			Flags: []cli.Flag{
				cli.IntFlag{Name: "page", Value: 0, Usage: "What number page of actions to fetch backward in history. Most recent first."},
				cli.IntFlag{Name: "page-size", Value: 20, Usage: "Number of actions to fetch per page."},
			},
			Action: domainRecordList,
		},
		{
			Name:    "show-record",
			Aliases: []string{"record"},
			Usage:   "<domain> <id> Show a domain record.",
			Action:  domainRecordShow,
		},
		{
			Name:    "add",
			Aliases: []string{"create-record"},
			Usage:   "<domain> Create domain record.",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "type", Value: "A", Usage: "Type for domain record."},
				cli.StringFlag{Name: "name", Value: "", Usage: "Name for domain record. The host name, alias, or service being defined by the record."},
				cli.StringFlag{Name: "data", Value: "", Usage: "Data for domain record."},
				cli.IntFlag{Name: "priority", Value: 0, Usage: "Priority for domain record. (Type: MX, SRV)"},
				cli.IntFlag{Name: "port", Value: 0, Usage: "Port for domain record. (Type: SRV)"},
				cli.IntFlag{Name: "weight", Value: 0, Usage: "Weight for domain record. (Type: SRV)"},
			},
			Action: domainRecordCreate,
		},
		{
			Name:   "destroy-record",
			Usage:  "<domain> <id> Destroy domain record.",
			Action: domainRecordDestroy,
		},
	},
}

func domainShow(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		log.Fatal("Error: Must provide name for Domain.")
	}

	name := ctx.Args().First()

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	domain, _, err := client.Domains.Get(name)
	if err != nil {
		log.Fatal(err)
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
		log.Fatalf("Unable to list Domains: %s.", err)
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
		log.Fatal("Must provide domain name and Droplet name.")
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	droplet, err := FindDropletByName(client, ctx.Args()[1])
	if err != nil {
		log.Fatal(err)
	}

	createRequest := &godo.DomainCreateRequest{
		Name:      ctx.Args().First(),
		IPAddress: PublicIPForDroplet(droplet),
	}
	domain, _, err := client.Domains.Create(createRequest)
	if err != nil {
		log.Fatal(err)
	}

	WriteOutput(domain)
}

func domainDestroy(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		log.Fatal("Error: Must provide a name for the domain to destroy.")
	}

	name := ctx.Args().First()

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	_, err := client.Domains.Delete(name)
	if err != nil {
		log.Fatalf("Unable to destroy domain: %s.", err)
	}

	log.Printf("Domain %s destroyed", name)
}

//
// Domain Records
//

func domainRecordList(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		log.Fatal("Error: Must provide a domain name for which to list records.")
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	domainName := ctx.Args().First()

	opt := &godo.ListOptions{
		Page:    ctx.Int("page"),
		PerPage: ctx.Int("page-size"),
	}
	domainDecords, _, err := client.Domains.Records(domainName, opt)
	if err != nil {
		log.Fatal(err)
	}

	WriteOutput(domainDecords)
}

func domainRecordShow(ctx *cli.Context) {
	if len(ctx.Args()) == 2 {
		log.Fatal("Error: Must provide domain name and domain record id.")
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	domainName := ctx.Args().First()
	recordID, err := strconv.Atoi(ctx.Args()[1])
	if err != nil {
		log.Fatal(err)
	}

	domainDecord, _, err := client.Domains.Record(domainName, recordID)
	if err != nil {
		log.Fatal(err)
	}

	WriteOutput(domainDecord)
}

func domainRecordCreate(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowAppHelp(ctx)
		log.Print("Must specify a domain name to add a record to.")
		os.Exit(1)
	}

	domainName := ctx.Args().First()

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	createRequest := &godo.DomainRecordEditRequest{
		Type: strings.ToUpper(ctx.String("type")),
		Name: ctx.String("name"),
		Data: ctx.String("data"),
	}

	if createRequest.Type == "MX" || createRequest.Type == "SRV" {
		createRequest.Priority = ctx.Int("priority")
	}
	if createRequest.Type == "SRV" {
		createRequest.Port = ctx.Int("port")
		createRequest.Weight = ctx.Int("weight")
	}

	domainRecord, _, err := client.Domains.CreateRecord(domainName, createRequest)
	if err != nil {
		log.Fatal(err)
	}

	WriteOutput(domainRecord)
}

func domainRecordDestroy(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		log.Fatal("Error: Must provide domain name and domain record id.")
	}

	domainName := ctx.Args().First()
	recordID, err := strconv.Atoi(ctx.Args()[1])
	if err != nil {
		log.Fatal(err)
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	_, err = client.Domains.DeleteRecord(domainName, recordID)
	if err != nil {
		log.Fatalf("Unable to destroy domain record: %s.", err)
	}

	log.Printf("Domain record %d destroyed", recordID)
}
