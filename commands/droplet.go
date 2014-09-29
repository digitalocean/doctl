package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
)

var DropletCommand = cli.Command{
	Name:  "droplet",
	Usage: "Droplet commands.",
	Subcommands: []cli.Command{
		DropletActionCommand,
		{
			Name:   "create",
			Usage:  "Create droplet.",
			Action: dropletCreate,
			Flags: []cli.Flag{
				cli.StringFlag{Name: "domain", Value: "", Usage: "Domain name to append to server name. (e.g. server01.example.com)"},
				cli.StringFlag{Name: "user-data", Value: "", Usage: "User data for creating server."},
				cli.StringFlag{Name: "ssh-keys", Value: "", Usage: "Comma seperated list of SSH Keys for server access. (e.g. --ssh-keys Work,Home)"},
				cli.StringFlag{Name: "size", Value: "512mb", Usage: "Size of Droplet."},
				cli.StringFlag{Name: "region", Value: "nyc3", Usage: "Region of Droplet."},
				cli.StringFlag{Name: "image", Value: "ubuntu-14-04-x64", Usage: "Image slug of Droplet."},
				cli.BoolFlag{Name: "backups", Usage: "Turn on backups."},
				cli.BoolFlag{Name: "ipv6", Usage: "Turn on IPv6 networking."},
				cli.BoolFlag{Name: "private-networking", Usage: "Turn on private networking."},
				cli.BoolFlag{Name: "add-region", Usage: "Append region to server name. (e.g. server01.sfo1)"},
			},
		},
		{
			Name:   "list",
			Usage:  "List droplets.",
			Action: dropletList,
		},
		{
			Name:   "show",
			Usage:  "Show droplet.",
			Action: dropletShow,
		},
		{
			Name:   "destroy",
			Usage:  "Destroy droplet.",
			Action: dropletDestroy,
		},
	},
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
	client := apiv2.NewClient(APIKey)

	dropletList, err := client.ListDroplets()
	if err != nil {
		fmt.Printf("Unable to list Droplets: %s\n", err)
		os.Exit(1)
	}

	cliOut := NewCLIOutput()
	defer cliOut.Flush()
	cliOut.Header("ID", "Name", "IP Address", "Status", "Memory", "Disk", "Region")
	for _, droplet := range dropletList.Droplets {
		cliOut.Writeln("%d\t%s\t%s\t%s\t%dMB\t%dGB\t%s\n",
			droplet.ID, droplet.Name, droplet.PublicIPAddress(), droplet.Status, droplet.Memory, droplet.Disk, droplet.Region.Slug)
	}
}

func dropletShow(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()

	client := apiv2.NewClient(APIKey)

	dropletList, err := client.ListDroplets()
	if err != nil {
		fmt.Printf("Unable to list Droplets: %s\n", err)
		os.Exit(1)
	}

	for _, droplet := range dropletList.Droplets {
		if droplet.Name == name {
			WriteOutput(droplet)
		}
	}
}

func dropletDestroy(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()

	client := apiv2.NewClient(APIKey)

	err := client.DestroyDroplet(name)
	if err != nil {
		fmt.Printf("Unable to destroy Droplet: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Droplet %s destroyed.\n", name)
}
