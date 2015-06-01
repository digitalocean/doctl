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

type DomainsServiceMock struct {
	ListFn   func(opts *godo.ListOptions) ([]godo.Domain, *godo.Response, error)
	GetFn    func(string) (*godo.Domain, *godo.Response, error)
	CreateFn func(*godo.DomainCreateRequest) (*godo.Domain, *godo.Response, error)
	DeleteFn func(string) (*godo.Response, error)

	RecordsFn      func(string, *godo.ListOptions) ([]godo.DomainRecord, *godo.Response, error)
	RecordFn       func(string, int) (*godo.DomainRecord, *godo.Response, error)
	DeleteRecordFn func(string, int) (*godo.Response, error)
	EditRecordFn   func(string, int, *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error)
	CreateRecordFn func(string, *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error)
}

var _ godo.DomainsService = &DomainsServiceMock{}

func (s *DomainsServiceMock) List(opts *godo.ListOptions) ([]godo.Domain, *godo.Response, error) {
	return s.ListFn(opts)
}

func (s *DomainsServiceMock) Get(name string) (*godo.Domain, *godo.Response, error) {
	return s.GetFn(name)
}

func (s *DomainsServiceMock) Create(req *godo.DomainCreateRequest) (*godo.Domain, *godo.Response, error) {
	return s.CreateFn(req)
}

func (s *DomainsServiceMock) Delete(name string) (*godo.Response, error) {
	return s.DeleteFn(name)
}

func (s *DomainsServiceMock) Records(name string, opts *godo.ListOptions) ([]godo.DomainRecord, *godo.Response, error) {
	return s.RecordsFn(name, opts)
}

func (s *DomainsServiceMock) Record(name string, id int) (*godo.DomainRecord, *godo.Response, error) {
	return s.RecordFn(name, id)
}

func (s *DomainsServiceMock) DeleteRecord(name string, id int) (*godo.Response, error) {
	return s.DeleteRecordFn(name, id)
}

func (s *DomainsServiceMock) EditRecord(name string, id int, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
	return s.EditRecord(name, id, req)
}

func (s *DomainsServiceMock) CreateRecord(name string, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
	return s.CreateRecordFn(name, req)
}
