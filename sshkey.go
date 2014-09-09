package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/api/v2"
	"gopkg.in/yaml.v1"
)

var SSHCommand = cli.Command{
	Name:  "sshkey",
	Usage: "SSH Key commands.",
	Subcommands: []cli.Command{
		{
			Name:   "create",
			Usage:  "Create SSH key.",
			Action: sshCreate,
		},
		{
			Name:   "list",
			Usage:  "List all SSH keys.",
			Action: sshList,
		},
		{
			Name:   "show",
			Usage:  "Show SSH key.",
			Action: sshShow,
		},
		{
			Name:   "destroy",
			Usage:  "Destroy SSH key.",
			Action: sshDestroy,
		},
	},
}

func sshCreate(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		fmt.Printf("Must provide name and public key file.\n")
		os.Exit(1)
	}

	client := apiv2.NewClient(APIKey)

	key := client.NewSSHKey()
	key.Name = ctx.Args().First()

	file, err := os.Open(ctx.Args()[1])
	if err != nil {
		fmt.Printf("Error opening key file: %s\n", err)
		os.Exit(1)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("Error reading key file: %s\n", err)
		os.Exit(1)
	}

	key.PublicKey = string(data)

	key, err = client.CreateKey(key)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	WriteOutput(key)
}

func sshList(ctx *cli.Context) {
	client := apiv2.NewClient(APIKey)

	keyList, err := client.ListAllKeys()
	if err != nil {
		fmt.Printf("Unable to list Keys: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("ID\t%-16s\t%-48s\n", "Name", "Fingerprint")
	for _, key := range keyList.SSHKeys {
		fmt.Printf("%d\t%-16s\t%-48s\n", key.ID, key.Name, key.Fingerprint)
	}
}

func sshShow(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Key.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()

	client := apiv2.NewClient(APIKey)

	keyList, err := client.ListAllKeys()
	if err != nil {
		fmt.Printf("Unable to list Keys: %s\n", err)
		os.Exit(1)
	}

	for _, key := range keyList.SSHKeys {
		if key.Name == name {
			data, errMarshal := yaml.Marshal(key)
			if errMarshal != nil {
				fmt.Printf("YAML Error: %s", errMarshal)
				os.Exit(1)
			}
			fmt.Printf("%s", string(data))
		}
	}
}

func sshDestroy(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		fmt.Printf("Error: Must provide name for Key.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()

	client := apiv2.NewClient(APIKey)

	err := client.DestroyKey(name)
	if err != nil {
		fmt.Printf("Unable to destroy Key: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Key %s destroyed.\n", name)
}
