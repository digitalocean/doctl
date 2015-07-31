package util

import (
	"golang.org/x/oauth2"

	"github.com/digitalocean/godo"
)

func ExampleWaitForActive() {
	// build client
	pat := "mytoken"
	t := &oauth2.Transport{
		Token: &oauth2.Token{AccessToken: pat},
	}

	client := godo.NewClient(t.Client())

	// create your droplet and retrieve the create action uri
	uri := "https://api.digitalocean.com/v2/actions/xxxxxxxx"

	// block until until the action is complete
	err := WaitForActive(client, uri)
	if err != nil {
		panic(err)
	}
}
