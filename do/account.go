package do

import "github.com/digitalocean/godo"

// Account is a wrapper for godo.Account
type Account struct {
	*godo.Account
}

// AccountService is an interface for interacting with DigitalOcean's account api.
type AccountService interface {
	Get() (*Account, error)
}

type accountService struct {
	client *godo.Client
}

var _ AccountService = &accountService{}

// NewAccountService builds an AccountService instance.
func NewAccountService(godoClient *godo.Client) AccountService {
	return &accountService{
		client: godoClient,
	}
}

func (as *accountService) Get() (*Account, error) {
	godoAccount, _, err := as.client.Account.Get()
	if err != nil {
		return nil, err
	}

	account := &Account{Account: godoAccount}
	return account, nil
}
