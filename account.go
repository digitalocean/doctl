package main

import (
	"fmt"
	"os"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/digitalocean/godo"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/golang.org/x/oauth2"
)

var AccountCommand = cli.Command{
	Name:    "account",
	Aliases: []string{"whoami"},
	Usage:   "Account commands.",
	Action:  accountShow,
	Subcommands: []cli.Command{
		{
			Name:    "show",
			Aliases: []string{"s"},
			Usage:   "Show an account.",
			Action:  accountShow,
		},
	},
}

func accountShow(ctx *cli.Context) {
	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	account, _, err := client.Account.Get()

	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	fmt.Printf(
		"  _______________________________________\n" +
			"/   Hi there! I'm Sammy.                 \\\n" +
			"\\                                        /\n" +
			" ---------------------------------------\n" +
			"                                          \\\n" +
			"                                           \\       \n" +
			"                 `.                        |      \n" +
			"                 `:::                      |      \n" +
			"         :        .:::.                    |       \n" +
			"         :,        :::::                   |       \n" +
			"         ,:        ::::::                  |       \n" +
			"         .:,       ;:::::.                 /       \n" +
			"          ::       ;:::::::::::::::::::,` /        \n" +
			"          ::: :,.,::::::::::::::::::::::::        \n" +
			"          ;::::::::::::::::::: `:`::::::::        \n" +
			"         `::::::::::::::::;::.`;'#`::::::.        \n" +
			"         ::,,:::::::::::;;;::``.;' :::::;         \n" +
			"         :   ,:::::::::::;;::. ,::`:::::          \n" +
			"               :::::::::::::::    ::::;           \n" +
			"                ;::::::::::,.:::;:.```            \n" +
			"                 ::::::::::..,.```````            \n" +
			"                 `:::::::::,..```````             \n" +
			"                  :::::::,``,....``               \n" +
			"                  `::::````` :,...`               \n" +
			"                `:::::,`````` `.,..`              \n" +
			"              :,::::::````````  ,,.`              \n" +
			"                ...`  `````````````               \n" +
			"                        `````````                 \n")
	WriteOutput(account)
}
