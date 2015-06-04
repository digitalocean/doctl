package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/digitalocean/godo"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/golang.org/x/oauth2"
)

var SSHCommand = cli.Command{
	Name:    "sshkey",
	Usage:   "SSH Key commands.",
	Aliases: []string{"ssh", "keys"},
	Action:  sshList,
	Subcommands: []cli.Command{
		{
			Name:    "create",
			Usage:   "<name> <path to ssh key(~/.ssh/id_rsa)> Create SSH key.",
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
			Name:    "find",
			Usage:   "<name> Find SSH key.",
			Aliases: []string{"f"},
			Action:  sshFind,
		},
		{
			Name:    "destroy",
			Usage:   "[--id | --fingerprint | <name>] Destroy SSH key.",
			Aliases: []string{"d"},
			Action:  sshDestroy,
			Flags: []cli.Flag{
				cli.IntFlag{Name: "id", Usage: "ID for SSH Key. (e.g. 1234567)"},
				cli.StringFlag{Name: "fingerprint", Usage: "Fingerprint for SSH Key. (e.g. aa:bb:cc)"},
			},
		},
	},
}

func sshCreate(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		log.Fatal("Must provide name and public key file.")
	}

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	file, err := os.Open(ctx.Args()[1])
	if err != nil {
		log.Fatalf("Error opening key file: %s.", err)
	}

	keyData, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Error reading key file: %s.", err)
	}

	createRequest := &godo.KeyCreateRequest{
		Name:      ctx.Args().First(),
		PublicKey: string(keyData),
	}
	key, _, err := client.Keys.Create(createRequest)
	if err != nil {
		log.Fatal(err)
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
			log.Fatalf("Unable to list SSH Keys: %s.", err)
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
			log.Fatalf("Unable to get pagination: %s.", err)
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
		log.Fatal("Error: Must provide name for Key.")
	}

	name := ctx.Args().First()

	tokenSource := &TokenSource{
		AccessToken: APIKey,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	key, err := FindKeyByName(client, name)
	if err != nil {
		log.Fatal(err)
	}

	WriteOutput(key)
}

func sshDestroy(ctx *cli.Context) {
	if ctx.Int("id") == 0 && ctx.String("fingerprint") == "" && len(ctx.Args()) < 1 {
		log.Fatal("Error: Must provide ID, fingerprint or name for SSH Key to destroy.")
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
			log.Fatal(err)
		} else {
			id = key.ID
		}
	} else if id != 0 {
		key, _, err := client.Keys.GetByID(id)
		if err != nil {
			log.Fatalf("Unable to find SSH Key: %d.", id)
		} else {
			id = key.ID
		}
	} else {
		key, _, err := client.Keys.GetByFingerprint(fingerprint)
		if err != nil {
			log.Fatalf("Unable to find SSH Key: %q.", fingerprint)
		} else {
			id = key.ID
		}
	}

	_, err := client.Keys.DeleteByID(id)
	if err != nil {
		log.Fatalf("Unable to destroy SSH Key: %s.", err)
	}

	log.Printf("Key %d, %q destroyed.", key.ID, key.Name)
}
