package do

import (
	"testing"

	"github.com/bryanl/godomock"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestAccountServiceGet(t *testing.T) {

	gAccountSvc := &godomock.MockAccountService{}

	gAccount := &godo.Account{UUID: "uuid"}
	gAccountSvc.On("Get").Return(gAccount, nil, nil)

	client := &godo.Client{
		Account: gAccountSvc,
	}
	as := NewAccountService(client)

	account, err := as.Get()
	assert.NoError(t, err)
	assert.Equal(t, "uuid", account.UUID)
}
