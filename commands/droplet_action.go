package commands

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
)

var DropletActionCommand = cli.Command{
	Name:  "action",
	Usage: "Droplet Action Commands.",
	Subcommands: []cli.Command{
		{
			Name:   "reboot",
			Usage:  "Reboot droplet.",
			Action: dropletActionReboot,
		},
		{
			Name:   "power_cycle",
			Usage:  "Powercycle droplet.",
			Action: dropletActionPowercycle,
		},
		{
			Name:   "shutdown",
			Usage:  "Shutdown droplet.",
			Action: dropletActionShutdown,
		},
		{
			Name:   "poweroff",
			Usage:  "Power off droplet.",
			Action: dropletActionPoweroff,
		},
		{
			Name:   "poweron",
			Usage:  "Power on droplet.",
			Action: dropletActionPoweron,
		},
		{
			Name:   "password_reset",
			Usage:  "Reset password for droplet.",
			Action: dropletActionPasswordReset,
		},
		{
			Name:  "resize",
			Usage: "Resize droplet.",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "size", Value: "512mb", Usage: "Size slug."},
				cli.BoolFlag{Name: "disk", Usage: "Whether to increase disk size"},
			},
			Action: dropletActionResize,
		},
	},
}

func dropletActionReboot(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()
	client := apiv2.NewClient(APIKey)

	droplet, err := client.FindDropletByName(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	action, rebootErr := droplet.Reboot()
	if rebootErr != nil {
		fmt.Println(rebootErr)
		os.Exit(1)
	}

	WriteOutput(action)
}

func dropletActionPowercycle(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()
	client := apiv2.NewClient(APIKey)

	droplet, err := client.FindDropletByName(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	action, rebootErr := droplet.Powercycle()
	if rebootErr != nil {
		fmt.Println(rebootErr)
		os.Exit(1)
	}

	WriteOutput(action)
}

func dropletActionShutdown(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()
	client := apiv2.NewClient(APIKey)

	droplet, err := client.FindDropletByName(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	action, rebootErr := droplet.Shutdown()
	if rebootErr != nil {
		fmt.Println(rebootErr)
		os.Exit(1)
	}

	WriteOutput(action)
}

func dropletActionPoweroff(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()
	client := apiv2.NewClient(APIKey)

	droplet, err := client.FindDropletByName(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	action, rebootErr := droplet.Poweroff()
	if rebootErr != nil {
		fmt.Println(rebootErr)
		os.Exit(1)
	}

	WriteOutput(action)
}

func dropletActionPoweron(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()
	client := apiv2.NewClient(APIKey)

	droplet, err := client.FindDropletByName(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	action, rebootErr := droplet.Poweron()
	if rebootErr != nil {
		fmt.Println(rebootErr)
		os.Exit(1)
	}

	WriteOutput(action)
}

func dropletActionPasswordReset(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()
	client := apiv2.NewClient(APIKey)

	droplet, err := client.FindDropletByName(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	action, rebootErr := droplet.PasswordReset()
	if rebootErr != nil {
		fmt.Println(rebootErr)
		os.Exit(1)
	}

	WriteOutput(action)
}

func dropletActionResize(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Droplet.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()
	size := ctx.String("size")
	disk := ctx.Bool("disk")

	client := apiv2.NewClient(APIKey)

	droplet, err := client.FindDropletByName(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	action, rebootErr := droplet.Resize(size, disk)
	if rebootErr != nil {
		fmt.Println(rebootErr)
		os.Exit(1)
	}

	WriteOutput(action)
}
