package commands

import (
	"testing"

	"github.com/bryanl/doit/do"
	domocks "github.com/bryanl/doit/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var testAccount = &do.Account{
	Account: &godo.Account{
		DropletLimit:  10,
		Email:         "user@example.com",
		UUID:          "1234",
		EmailVerified: true,
	},
}

func TestAccountCommand(t *testing.T) {
	acctCmd := Account()
	assert.NotNil(t, acctCmd)
	assertCommandNames(t, acctCmd, "get", "ratelimit")
}

func TestAccountGet(t *testing.T) {
	withTestClient(func(config *cmdConfig) {
		as := &domocks.AccountService{}
		as.On("Get").Return(testAccount, nil)

		config.as = as

		err := RunAccountGet(config)
		assert.NoError(t, err)
	})
}
