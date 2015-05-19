package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
)

var RegionCommand = cli.Command{
	Name:  "region",
	Usage: "Region commands.",
	Subcommands: []cli.Command{
		{
			Name:   "show",
			Usage:  "Show a Region.",
			Action: regionShow,
		},
		{
			Name:   "list",
			Usage:  "List All Regions.",
			Action: regionList,
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
	client := apiv2.NewClient(APIKey)

	regionList, err := client.ListAllRegions()
	if err != nil {
		fmt.Printf("Unable to list Regions: %s\n", err)
		os.Exit(1)
	}

	cliOut := NewCLIOutput()
	defer cliOut.Flush()
	cliOut.Header("Name", "Slug", "Available")
	for _, region := range regionList.Regions {
		cliOut.Writeln("%s\t%s\t%t\n", region.Name, region.Slug, region.Available)
	}
}
