package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
)

var SizeCommand = cli.Command{
	Name:  "size",
	Usage: "Size commands.",
	Subcommands: []cli.Command{
		{
			Name:   "show",
			Usage:  "Show a size.",
			Action: sizeShow,
		},
		{
			Name:   "list",
			Usage:  "List all sizes.",
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

	client := apiv2.NewClient(APIKey)

	size, err := client.LoadSize(name)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	WriteOutput(size)
}

func sizeList(ctx *cli.Context) {
	client := apiv2.NewClient(APIKey)

	sizeList, err := client.ListAllSizes()
	if err != nil {
		fmt.Printf("Unable to list Sizes: %s\n", err)
		os.Exit(1)
	}

	cliOut := NewCLIOutput()
	defer cliOut.Flush()
	cliOut.Header("Slug", "Memory", "VCPUs", "Disk", "Transfer", "Price Monthly", "Price Hourly")
	for _, size := range sizeList.Sizes {
		cliOut.Writeln("%s\t%dMB\t%d\t%dGB\t%d\t$%.0f\t$%.5f\n",
			size.Slug, size.Memory, size.VCPUS, size.Disk, size.Transfer, size.PriceMonthly, size.PriceHourly)
	}
}
