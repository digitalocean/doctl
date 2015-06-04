package main

import (
	"log"
	"os"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/digitalocean/godo"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/golang.org/x/oauth2"
)

var RegionCommand = cli.Command{
	Name:   "region",
	Usage:  "Region commands.",
	Action: regionList,
	Subcommands: []cli.Command{
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List All Regions.",
			Action:  regionList,
		},
	},
}

func regionList(ctx *cli.Context) {
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
		Page:    1,
		PerPage: 50, // Not likely to have more than 50 regions soon
	}
	regionList, _, err := client.Regions.List(opt)
	if err != nil {
		log.Fatalf("Unable to list Regions: %s.", err)
	}

	cliOut := NewCLIOutput()
	defer cliOut.Flush()
	cliOut.Header("Name", "Slug", "Available")
	for _, region := range regionList {
		cliOut.Writeln("%s\t%s\t%t\n", region.Name, region.Slug, region.Available)
	}
}
