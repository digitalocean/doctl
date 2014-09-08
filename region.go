package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
	"gopkg.in/yaml.v1"
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

	client := apiv2.NewClient(os.Getenv("DIGITAL_OCEAN_API_KEY"))

	region, err := client.LoadRegion(name)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	data, errMarshal := yaml.Marshal(region)
	if errMarshal != nil {
		fmt.Printf("YAML Error: %s", errMarshal)
		os.Exit(1)
	}
	fmt.Printf("%s", string(data))
}

func regionList(ctx *cli.Context) {
	client := apiv2.NewClient(os.Getenv("DIGITAL_OCEAN_API_KEY"))

	regionList, err := client.ListAllRegions()
	if err != nil {
		fmt.Printf("Unable to list Regions: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-16s\t%s\t%s\n", "Name", "Slug", "Available")
	for _, region := range regionList.Regions {
		fmt.Printf("%-16s\t%s\t%t\n", region.Name, region.Slug, region.Available)
	}
}
