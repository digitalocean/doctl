package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/slantview/doctl/api/v2"

	"golang.org/x/oauth2"
)

var RegionCommand = cli.Command{
	Name:   "region",
	Usage:  "Region commands.",
	Action: regionList,
	Subcommands: []cli.Command{
		{
			Name:    "show",
			Aliases: []string{"s"},
			Usage:   "Show a Region.",
			Action:  regionShow,
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List All Regions.",
			Action:  regionList,
		},
	},
}

func regionShow(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Region.\n")
		os.Exit(64)
	}

	name := ctx.Args().First()

	client := apiv2.NewClient(APIKey)

	region, err := client.LoadRegion(name)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	WriteOutput(region)
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
		fmt.Printf("Unable to list Regions: %s\n", err)
		os.Exit(1)
	}

	cliOut := NewCLIOutput()
	defer cliOut.Flush()
	cliOut.Header("Name", "Slug", "Available")
	for _, region := range regionList {
		cliOut.Writeln("%s\t%s\t%t\n", region.Name, region.Slug, region.Available)
	}
}
