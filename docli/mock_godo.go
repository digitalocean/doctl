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
	return s.EditRecordFn(name, id, req)
}

func (s *DomainsServiceMock) CreateRecord(name string, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
	return s.CreateRecordFn(name, req)
}

type DropletActionsServiceMock struct {
	ChangeKernelFn            func(id, kernelID int) (*godo.Action, *godo.Response, error)
	DisableBackupsFn          func(id int) (*godo.Action, *godo.Response, error)
	EnableIPv6Fn              func(id int) (*godo.Action, *godo.Response, error)
	EnablePrivateNetworkingFn func(id int) (*godo.Action, *godo.Response, error)
	GetFn                     func(dropletID, actionID int) (*godo.Action, *godo.Response, error)
	GetByURIFn                func(rawurl string) (*godo.Action, *godo.Response, error)
	PasswordResetFn           func(id int) (*godo.Action, *godo.Response, error)
	PowerCycleFn              func(id int) (*godo.Action, *godo.Response, error)
	PowerOffFn                func(id int) (*godo.Action, *godo.Response, error)
	PowerOnFn                 func(id int) (*godo.Action, *godo.Response, error)
	RebootFn                  func(id int) (*godo.Action, *godo.Response, error)
	RebuildByImageIDFn        func(id, imageID int) (*godo.Action, *godo.Response, error)
	RebuildByImageSlugFn      func(id int, slug string) (*godo.Action, *godo.Response, error)
	RenameFn                  func(id int, name string) (*godo.Action, *godo.Response, error)
	ResizeFn                  func(id int, sizeSlug string, resizeDisk bool) (*godo.Action, *godo.Response, error)
	RestoreFn                 func(id, imageID int) (*godo.Action, *godo.Response, error)
	ShutdownFn                func(id int) (*godo.Action, *godo.Response, error)
	SnapshotFn                func(id int, name string) (*godo.Action, *godo.Response, error)
	UpgradeFn                 func(id int) (*godo.Action, *godo.Response, error)
}

var _ godo.DropletActionsService = &DropletActionsServiceMock{}

func (s *DropletActionsServiceMock) ChangeKernel(id, kernelID int) (*godo.Action, *godo.Response, error) {
	return s.ChangeKernelFn(id, kernelID)
}

func (s *DropletActionsServiceMock) DisableBackups(id int) (*godo.Action, *godo.Response, error) {
	return s.DisableBackupsFn(id)

}

func (s *DropletActionsServiceMock) EnableIPv6(id int) (*godo.Action, *godo.Response, error) {
	return s.EnableIPv6Fn(id)
}

func (s *DropletActionsServiceMock) EnablePrivateNetworking(id int) (*godo.Action, *godo.Response, error) {
	return s.EnablePrivateNetworkingFn(id)
}

func (s *DropletActionsServiceMock) Get(dropletID, actionID int) (*godo.Action, *godo.Response, error) {
	return s.GetFn(dropletID, actionID)
}

func (s *DropletActionsServiceMock) GetByURI(rawurl string) (*godo.Action, *godo.Response, error) {
	return s.GetByURIFn(rawurl)
}

func (s *DropletActionsServiceMock) PasswordReset(id int) (*godo.Action, *godo.Response, error) {
	return s.PasswordResetFn(id)
}

func (s *DropletActionsServiceMock) PowerCycle(id int) (*godo.Action, *godo.Response, error) {
	return s.PowerCycleFn(id)
}

func (s *DropletActionsServiceMock) PowerOff(id int) (*godo.Action, *godo.Response, error) {
	return s.PowerOffFn(id)
}

func (s *DropletActionsServiceMock) PowerOn(id int) (*godo.Action, *godo.Response, error) {
	return s.PowerOnFn(id)
}

func (s *DropletActionsServiceMock) Reboot(id int) (*godo.Action, *godo.Response, error) {
	return s.RebootFn(id)
}

func (s *DropletActionsServiceMock) RebuildByImageID(id, imageID int) (*godo.Action, *godo.Response, error) {
	return s.RebuildByImageIDFn(id, imageID)
}

func (s *DropletActionsServiceMock) RebuildByImageSlug(id int, slug string) (*godo.Action, *godo.Response, error) {
	return s.RebuildByImageSlugFn(id, slug)
}

func (s *DropletActionsServiceMock) Rename(id int, name string) (*godo.Action, *godo.Response, error) {
	return s.RenameFn(id, name)
}

func (s *DropletActionsServiceMock) Resize(id int, sizeSlug string, resizeDisk bool) (*godo.Action, *godo.Response, error) {
	return s.ResizeFn(id, sizeSlug, resizeDisk)
}

func (s *DropletActionsServiceMock) Restore(id, imageID int) (*godo.Action, *godo.Response, error) {
	return s.RestoreFn(id, imageID)
}

func (s *DropletActionsServiceMock) Shutdown(id int) (*godo.Action, *godo.Response, error) {
	return s.ShutdownFn(id)
}

func (s *DropletActionsServiceMock) Snapshot(id int, name string) (*godo.Action, *godo.Response, error) {
	return s.SnapshotFn(id, name)
}

func (s *DropletActionsServiceMock) Upgrade(id int) (*godo.Action, *godo.Response, error) {
	return s.UpgradeFn(id)
}
