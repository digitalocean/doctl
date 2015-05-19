package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/slantview/doctl/api/v2"
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
				cli.StringFlag{Name: "domain", Value: "", Usage: "Domain name to append to server name. (e.g. server01.example.com)"},
				cli.BoolFlag{Name: "add-region", Usage: "Append region to server name. (e.g. server01.sfo1)"},
				cli.StringFlag{Name: "user-data", Value: "", Usage: "User data for creating server."},
				cli.StringFlag{Name: "ssh-keys", Value: "", Usage: "Comma seperated list of SSH Keys for server access. (e.g. --ssh-keys Work,Home)"},
				cli.StringFlag{Name: "size", Value: "512mb", Usage: "Size of Droplet."},
				cli.StringFlag{Name: "region", Value: "nyc3", Usage: "Region of Droplet."},
				cli.StringFlag{Name: "image", Value: "ubuntu-14-04-x64", Usage: "Image slug of Droplet."},
				cli.BoolFlag{Name: "backups", Usage: "Turn on backups."},
				cli.BoolFlag{Name: "ipv6", Usage: "Turn on IPv6 networking."},
				cli.BoolFlag{Name: "private-networking", Usage: "Turn on private networking."},
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
			Usage:   "Find the first Droplet whose name matches the first argument.",
			Action:  dropletFind,
		},
		{
			Name:    "destroy",
			Aliases: []string{"d"},
			Usage:   "Destroy droplet.",
			Action:  dropletDestroy,
			Flags: []cli.Flag{
				cli.IntFlag{Name: "id", Usage: "ID for Droplet. (e.g. 1234567)"},
			},
		},
	},
}

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func dropletCreate(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	client := apiv2.NewClient(APIKey)

	// Add domain to end if available.
	name := ctx.Args().First()
	if ctx.String("add-region") != "" {
		name = fmt.Sprintf("%s.%s", name, ctx.String("region"))
	}
	if ctx.String("domain") != "" {
		name = fmt.Sprintf("%s.%s", name, ctx.String("domain"))
	}

	request := client.NewDropletRequest(name)
	request.Size = ctx.String("size")
	request.Image = ctx.String("image")
	request.PrivateNetworking = ctx.Bool("private-networking")
	request.IPv6 = ctx.Bool("ipv6")
	request.Backups = ctx.Bool("backups")
	request.UserData = ctx.String("user-data")

	// Loop through the SSH Keys and add by ID.  D.O. API should have handled this case as well.
	var sshKeys []string
	for _, key := range strings.Split(ctx.String("ssh-keys"), ",") {
		sshKey, err := client.FindKey(key)
		if sshKey != nil && err == nil {
			sshKeys = append(sshKeys, fmt.Sprintf("%d", sshKey.ID))
		}
	}
	request.SSHKeys = sshKeys

	droplet, errCreate := client.CreateDroplet(request)
	if errCreate != nil {
		fmt.Printf("Unable to create Droplet: %s\n", errCreate)
		os.Exit(1)
	}

	WriteOutput(droplet)
}

func dropletList(ctx *cli.Context) {
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
		cliOut.Writeln("%d\t%s\t%s\t%s\t%dMB\t%dGB\t%s\n",
			droplet.ID, droplet.Name, droplet.Networks.V4[0].IPAddress, droplet.Status, droplet.Memory, droplet.Disk, droplet.Region.Slug)
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

	opt := &godo.ListOptions{}
	for {
		dropletPage, resp, err := client.Droplets.List(opt)
		if err != nil {
			fmt.Printf("Unable to list Droplets: %s\n", err)
			os.Exit(1)
		}

		// append the current page's droplets to our list
		for _, d := range dropletPage {
			if d.Name == name {
				WriteOutput(d)
			}
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			fmt.Printf("Unable to find the Droplet: %s\n", name)
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
}

func dropletDestroy(ctx *cli.Context) {
	if ctx.Int("id") == 0 {
		fmt.Printf("Error: Must provide ID for Droplet for Destroy.\n")
		os.Exit(1)
	}

	id := ctx.Int("id")
	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

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
