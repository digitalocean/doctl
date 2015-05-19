package util

import (
	"code.google.com/p/goauth2/oauth"

	"github.com/digitalocean/godo"
)

func ExampleWaitForActive() {
	// build client
	pat := "mytoken"
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: pat},
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
