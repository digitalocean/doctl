package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
	"gopkg.in/yaml.v1"
)

var SizeCommand = cli.Command{
	Name:  "size",
	Usage: "Size commands.",
	Subcommands: []cli.Command{
		{
			Name:   "show",
			Usage:  "Show a Size.",
			Action: sizeShow,
		},
		{
			Name:   "list",
			Usage:  "List All Sizes.",
			Action: sizeList,
		},
	},
}

func sizeShow(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Size.\n")
		os.Exit(64)
	}

	name := ctx.Args().First()

	client := apiv2.NewClient(os.Getenv("DIGITAL_OCEAN_API_KEY"))

	size, err := client.LoadSize(name)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	data, errMarshal := yaml.Marshal(size)
	if errMarshal != nil {
		fmt.Printf("YAML Error: %s", errMarshal)
		os.Exit(1)
	}
	fmt.Printf("%s", string(data))
}

func sizeList(ctx *cli.Context) {
	client := apiv2.NewClient(os.Getenv("DIGITAL_OCEAN_API_KEY"))

	sizeList, err := client.ListAllSizes()
	if err != nil {
		fmt.Printf("Unable to list Sizes: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-8s\t%s\t%s\t%s\t%-8s\t%-11s\t%s\n", "Slug", "Memory", "VCPUs", "Disk", "Transfer", "Price Monthly", "Price Hourly")
	for _, size := range sizeList.Sizes {
		fmt.Printf("%-8s\t%dMB\t%d\t%dGB\t%-8d\t$%-11.0f\t$%0.5f\n",
			size.Slug, size.Memory, size.VCPUS, size.Disk, size.Transfer, size.PriceMonthly, size.PriceHourly)
	}
}
