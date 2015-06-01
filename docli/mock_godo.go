package docli

import "github.com/digitalocean/godo"

type AccountServiceMock struct {
	GetFn func() (*godo.Account, *godo.Response, error)
}

var _ godo.AccountService = &AccountServiceMock{}

func (s *AccountServiceMock) Get() (*godo.Account, *godo.Response, error) {
	return s.GetFn()
}
