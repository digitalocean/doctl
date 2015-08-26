package commands

import (
	"io/ioutil"
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
)

var testAccount = &godo.Account{
	DropletLimit:  10,
	Email:         "user@example.com",
	UUID:          "1234",
	EmailVerified: true,
}

func TestAccountGet(t *testing.T) {
	accountDidGet := false

	client := &godo.Client{
		Account: &doit.AccountServiceMock{
			GetFn: func() (*godo.Account, *godo.Response, error) {
				accountDidGet = true
				return testAccount, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestViperConfig) {
		cmd := NewCmdAccountGet(ioutil.Discard)
		cmd.Run(cmd, []string{})

		if !accountDidGet {
			t.Errorf("could not retrieve account")
		}
	})
}
