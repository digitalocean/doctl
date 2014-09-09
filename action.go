package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
)

var ActionCommand = cli.Command{
	Name:  "action",
	Usage: "Action commands.",
	Subcommands: []cli.Command{
		{
			Name:   "show",
			Usage:  "Show an action.",
			Action: actionShow,
		},
		{
			Name:   "list",
			Usage:  "List all actions.",
			Action: actionList,
		},
	},
}

func actionShow(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Action.\n")
		os.Exit(64)
	}

	id, _ := strconv.ParseInt(ctx.Args().First(), 10, 0)

	client := apiv2.NewClient(APIKey)

	action, err := client.LoadAction(int(id))
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	WriteOutput(action)
}

func actionList(ctx *cli.Context) {
	client := apiv2.NewClient(APIKey)

	actionList, err := client.ListAllActions()
	if err != nil {
		fmt.Printf("Unable to list Actions: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-10s\t%-6s\t%-10s\t%-10s\t%-10s\t%-16s\t%-16s\t%s\n",
		"ID", "Region", "ResourceType", "ResourceID", "Type", "StartedAt", "CompletedAt", "Status")
	for _, action := range actionList.Actions {
		fmt.Printf("%-10d\t%-6s\t%-10s\t%-10d\t%-10s\t%-16s\t%-16s\t%s\n",
			action.ID, action.Region, action.ResourceType, action.ResourceID, action.Type, action.StartedAt,
			action.CompletedAt, action.Status)
	}
}
