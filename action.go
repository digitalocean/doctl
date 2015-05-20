package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"

	"golang.org/x/oauth2"
)

var ActionCommand = cli.Command{
	Name:    "action",
	Aliases: []string{"a"},
	Usage:   "Action commands.",
	Action:  actionList,
	Subcommands: []cli.Command{
		{
			Name:    "show",
			Aliases: []string{"s"},
			Usage:   "Show an action.",
			Action:  actionShow,
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List most recent actions.",
			Flags: []cli.Flag{
				cli.IntFlag{Name: "page", Value: 0, Usage: "What number page of actions to fetch backward in history. Most recent first."},
				cli.IntFlag{Name: "page-size", Value: 20, Usage: "Number of actions to fetch per page."},
			},
			Action: actionList,
		},
	},
}

func actionShow(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		fmt.Printf("Error: Must provide exactly one id of an Action to show.\n")
		os.Exit(64)
	}

	id, _ := strconv.ParseInt(ctx.Args().First(), 10, 0)

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	action, _, err := client.Actions.Get(int(id))

	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	WriteOutput(action)
}

func actionList(ctx *cli.Context) {
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
	actionList, _, err := client.Actions.List(opt)

	if err != nil {
		fmt.Printf("Unable to list Actions: %s\n", err)
		os.Exit(1)
	}

	cliOut := NewCLIOutput()
	defer cliOut.Flush()
	cliOut.Header("ID", "Region", "ResourceType", "ResourceID", "Type", "StartedAt", "CompletedAt", "Status")
	for _, action := range actionList {
		cliOut.Writeln("%d\t%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
			action.ID, action.RegionSlug, action.ResourceType, action.ResourceID, action.Type, action.StartedAt, action.CompletedAt, action.Status)
	}
}
