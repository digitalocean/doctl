package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/digitalocean/godo/util"

	"golang.org/x/oauth2"
)

var DropletCommand = cli.Command{
	Name:    "droplet",
	Aliases: []string{"d"},
	Usage:   "Droplet commands. Lists by default.",
	Action:  dropletList,
	Subcommands: []cli.Command{
		DropletActionCommand,
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "Create droplet.",
			Action:  dropletCreate,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "domain, d", Value: "", Usage: "Domain name to append to the hostname. (e.g. server01.example.com)"},
				cli.BoolFlag{Name: "add-region", Usage: "Append region to hostname. (e.g. server01.sfo1)"},
				cli.StringFlag{Name: "user-data, u", Value: "", Usage: "User data for creating server."},
				cli.StringFlag{Name: "user-data-file, uf", Value: "", Usage: "A path to a file for user data."},
				cli.StringFlag{Name: "ssh-keys, k", Value: "", Usage: "Comma seperated list of SSH Key names. (e.g. --ssh-keys Work,Home)"},
				cli.StringFlag{Name: "size, s", Value: "512mb", Usage: "Size of Droplet."},
				cli.StringFlag{Name: "region, r", Value: "nyc3", Usage: "Region of Droplet."},
				cli.StringFlag{Name: "image, i", Value: "ubuntu-14-04-x64", Usage: "Image slug of Droplet."}, // TODO handle image id
				cli.BoolFlag{Name: "backups, b", Usage: "Turn on backups."},
				cli.BoolFlag{Name: "ipv6, 6", Usage: "Turn on IPv6 networking."},
				cli.BoolFlag{Name: "private-networking, p", Usage: "Turn on private networking."},
				cli.BoolFlag{Name: "wait-for-active", Usage: "Don't return until the create has succeeded or failed."},
			},
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List droplets.",
			Action:  dropletList,
			Flags:   []cli.Flag{},
		},
		{
			Name:    "find",
			Aliases: []string{"f"},
			Usage:   "<Droplet name> Find the first Droplet whose name matches the first argument.",
			Action:  dropletFind,
		},
		{
			Name:    "destroy",
			Aliases: []string{"d"},
			Usage:   "[--id | <name>] Destroy droplet.",
			Action:  dropletDestroy,
			Flags: []cli.Flag{
				cli.IntFlag{Name: "id", Usage: "ID for Droplet. (e.g. 1234567)"},
			},
		},
	},
}

func dropletCreate(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	// Add domain to end if available.
	dropletName := ctx.Args().First()
	if ctx.String("add-region") != "" {
		dropletName = fmt.Sprintf("%s.%s", dropletName, ctx.String("region"))
	}
	if ctx.String("domain") != "" {
		dropletName = fmt.Sprintf("%s.%s", dropletName, ctx.String("domain"))
	}

	// Loop through the SSH Keys and add by name. DO API should have handled
	// this case as well.
	var sshKeys []godo.DropletCreateSSHKey
	keyNames := ctx.String("ssh-keys")
	if keyNames != "" {
		for _, keyName := range strings.Split(keyNames, ",") {
			sshKey, err := FindKeyByName(client, keyName)
			if sshKey != nil && err == nil {
				sshKeys = append(sshKeys, godo.DropletCreateSSHKey{ID: sshKey.ID})
			} else {
				fmt.Printf("Warning: Could not find key: %s.\n", keyName)
			}
		}
	}

	userData := ""
	userDataPath := ctx.String("user-data-file")
	if userDataPath != "" {
		file, err := os.Open(userDataPath)
		if err != nil {
			fmt.Printf("Error opening user data file: %s\n", err)
			os.Exit(1)
		}

		userDataFile, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Printf("Error reading user data file: %s\n", err)
			os.Exit(1)
		}
		userData = string(userDataFile)
	} else {
		userData = ctx.String("user-data")
	}

	createRequest := &godo.DropletCreateRequest{
		Name:   dropletName,
		Region: ctx.String("region"),
		Size:   ctx.String("size"),
		Image: godo.DropletCreateImage{
			Slug: ctx.String("image"),
		},
		SSHKeys:           sshKeys,
		Backups:           ctx.Bool("backups"),
		IPv6:              ctx.Bool("ipv6"),
		PrivateNetworking: ctx.Bool("private-networking"),
		UserData:          userData,
	}

	dropletRoot, resp, err := client.Droplets.Create(createRequest)
	if err != nil {
		fmt.Printf("Unable to create Droplet: %s\n", err)
		os.Exit(1)
	}

	if ctx.Bool("wait-for-active") {
		util.WaitForActive(client, resp.Links.Actions[0].HREF)
	}
	WriteOutput(dropletRoot.Droplet)
}

func dropletList(ctx *cli.Context) {
	if ctx.BoolT("help") == true {
		cli.ShowAppHelp(ctx)
		os.Exit(1)
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	opt := &godo.ListOptions{}
	dropletList := []godo.Droplet{}

	for {
		dropletPage, resp, err := client.Droplets.List(opt)
		if err != nil {
			fmt.Printf("Unable to list Droplets: %s\n", err)
			os.Exit(1)
		}

		// append the current page's droplets to our list
		for _, d := range dropletPage {
			dropletList = append(dropletList, d)
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			fmt.Printf("Unable to get pagination: %s\n", err)
			os.Exit(1)
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	cliOut := NewCLIOutput()
	defer cliOut.Flush()
	cliOut.Header("ID", "Name", "IP Address", "Status", "Memory", "Disk", "Region")
	for _, droplet := range dropletList {
		publicIP := PublicIPForDroplet(&droplet)

		cliOut.Writeln("%d\t%s\t%s\t%s\t%dMB\t%dGB\t%s\n",
			droplet.ID, droplet.Name, publicIP, droplet.Status, droplet.Memory, droplet.Disk, droplet.Region.Slug)
	}
}

func dropletFind(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		fmt.Printf("Error: Must provide one name for a Droplet search.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	droplet, err := FindDropletByName(client, name)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(64)
	}

	WriteOutput(droplet)
}

func dropletDestroy(ctx *cli.Context) {
	if ctx.Int("id") == 0 && len(ctx.Args()) != 1 {
		fmt.Printf("Error: Must provide ID or name for Droplet to destroy.\n")
		os.Exit(1)
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	id := ctx.Int("id")
	if id == 0 {
		droplet, err := FindDropletByName(client, ctx.Args()[0])
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(64)
		} else {
			id = droplet.ID
		}
	}

	dropletRoot, _, err := client.Droplets.Get(id)
	if err != nil {
		fmt.Printf("Unable to find Droplet: %s\n", err)
		os.Exit(1)
	}

	_, err = client.Droplets.Delete(id)
	if err != nil {
		fmt.Printf("Unable to destroy Droplet: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Droplet %s destroyed.\n", dropletRoot.Droplet.Name)
}
