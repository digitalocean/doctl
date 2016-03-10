package commands

import (
	"testing"

	"github.com/bryanl/doit/do"
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
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.account.On("Get").Return(testAccount, nil)

		err := RunAccountGet(config)
		assert.NoError(t, err)
	})
}
