package do

import "github.com/digitalocean/godo"

// Account is a wrapper for godo.Account.
type Account struct {
	*godo.Account
}

// RateLimit is a wrapper for godo.Rate.
type RateLimit struct {
	*godo.Rate
}

// AccountService is an interface for interacting with DigitalOcean's account api.
type AccountService interface {
	Get() (*Account, error)
	RateLimit() (*RateLimit, error)
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

func (as *accountService) RateLimit() (*RateLimit, error) {
	_, resp, err := as.client.Account.Get()
	if err != nil {
		return nil, err
	}

	rateLimit := &RateLimit{Rate: &resp.Rate}
	return rateLimit, nil
}
