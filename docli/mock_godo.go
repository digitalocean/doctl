package docli

import "github.com/digitalocean/godo"

type AccountServiceMock struct {
	GetFn func() (*godo.Account, *godo.Response, error)
}

var _ godo.AccountService = &AccountServiceMock{}

func (s *AccountServiceMock) Get() (*godo.Account, *godo.Response, error) {
	return s.GetFn()
}

type ActionsServiceMock struct {
	GetFn  func(id int) (*godo.Action, *godo.Response, error)
	ListFn func(opts *godo.ListOptions) ([]godo.Action, *godo.Response, error)
}

var _ godo.ActionsService = &ActionsServiceMock{}

func (s *ActionsServiceMock) List(opts *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
	return s.ListFn(opts)
}

func (s *ActionsServiceMock) Get(id int) (*godo.Action, *godo.Response, error) {
	return s.GetFn(id)
}
