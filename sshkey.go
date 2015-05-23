package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"

	"golang.org/x/oauth2"
)

var SSHCommand = cli.Command{
	Name:    "sshkey",
	Usage:   "SSH Key commands.",
	Aliases: []string{"ssh", "keys"},
	Action:  sshList,
	Subcommands: []cli.Command{
		{
			Name:    "create",
			Usage:   "<name> <path to ssh key(~/.ssh/id_rsa)>Create SSH key.",
			Aliases: []string{"c"},
			Action:  sshCreate,
		},
		{
			Name:    "list",
			Usage:   "List all SSH keys.",
			Aliases: []string{"l"},
			Action:  sshList,
		},
		{
			Name:    "show",
			Usage:   "<name> Show SSH key.",
			Aliases: []string{"s"},
			Action:  sshFind,
		},
		{
			Name:    "destroy",
			Usage:   "[--id | --fingerprint | <name>] Destroy SSH key.",
			Aliases: []string{"d"},
			Action:  sshDestroy,
			Flags: []cli.Flag{
				cli.IntFlag{Name: "id", Usage: "ID for SSH Key. (e.g. 1234567)"},
				cli.StringFlag{Name: "id", Usage: "Fingerprint for SSH Key. (e.g. aa:bb:cc)"},
			},
		},
	},
}

func sshCreate(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		fmt.Printf("Must provide name and public key file.\n")
		os.Exit(1)
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	file, err := os.Open(ctx.Args()[1])
	if err != nil {
		fmt.Printf("Error opening key file: %s\n", err)
		os.Exit(1)
	}

	keyData, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("Error reading key file: %s\n", err)
		os.Exit(1)
	}

	createRequest := &godo.KeyCreateRequest{
		Name:      ctx.Args().First(),
		PublicKey: string(keyData),
	}
	key, _, err := client.Keys.Create(createRequest)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	WriteOutput(key)
}

func sshList(ctx *cli.Context) {
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
	keyList := []godo.Key{}

	for {
		keyPage, resp, err := client.Keys.List(opt)
		if err != nil {
			fmt.Printf("Unable to list SSH Keys: %s\n", err)
			os.Exit(1)
		}

		// append the current page's droplets to our list
		for _, d := range keyPage {
			keyList = append(keyList, d)
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
	cliOut.Header("ID", "Name", "Fingerprint")
	for _, key := range keyList {
		cliOut.Writeln("%d\t%s\t%s\n", key.ID, key.Name, key.Fingerprint)
	}
}

func sshFind(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		fmt.Printf("Error: Must provide name for Key.\n")
		os.Exit(1)
	}

	name := ctx.Args().First()

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	key, err := FindKeyByName(client, name)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(64)
	}

	WriteOutput(key)
}

func sshDestroy(ctx *cli.Context) {
	if ctx.Int("id") == 0 && ctx.String("fingerprint") == "" && len(ctx.Args()) < 1 {
		fmt.Printf("Error: Must provide ID, fingerprint or name for SSH Key to destroy.\n")
		os.Exit(1)
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	id := ctx.Int("id")
	fingerprint := ctx.String("fingerprint")
	var key godo.Key
	if id == 0 && fingerprint == "" {
		key, err := FindKeyByName(client, ctx.Args().First())
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(64)
		} else {
			id = key.ID
		}
	} else if id != 0 {
		key, _, err := client.Keys.GetByID(id)
		if err != nil {
			fmt.Printf("Unable to find SSH Key: %s\n", err)
			os.Exit(1)
		} else {
			id = key.ID
		}
	} else {
		key, _, err := client.Keys.GetByFingerprint(fingerprint)
		if err != nil {
			fmt.Printf("Unable to find SSH Key: %s\n", err)
			os.Exit(1)
		} else {
			id = key.ID
		}
	}

	_, err := client.Keys.DeleteByID(id)
	if err != nil {
		fmt.Printf("Unable to destroy SSH Key: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Key %s destroyed.\n", key.Name)
}
