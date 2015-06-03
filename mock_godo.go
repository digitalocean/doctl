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

type DropletsServiceMock struct {
	ActionsFn   func(dropletID int, opt *godo.ListOptions) ([]godo.Action, *godo.Response, error)
	BackupsFn   func(dropletID int, opt *godo.ListOptions) ([]godo.Image, *godo.Response, error)
	CreateFn    func(createRequest *godo.DropletCreateRequest) (*godo.Droplet, *godo.Response, error)
	DeleteFn    func(dropletID int) (*godo.Response, error)
	GetFn       func(dropletID int) (*godo.Droplet, *godo.Response, error)
	KernelsFn   func(dropletID int, opt *godo.ListOptions) ([]godo.Kernel, *godo.Response, error)
	ListFn      func(opt *godo.ListOptions) ([]godo.Droplet, *godo.Response, error)
	NeighborsFn func(dropletID int) ([]godo.Droplet, *godo.Response, error)
	SnapshotsFn func(dropletID int, opt *godo.ListOptions) ([]godo.Image, *godo.Response, error)
}

var _ godo.DropletsService = &DropletsServiceMock{}

func (s *DropletsServiceMock) Actions(dropletID int, opt *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
	return s.ActionsFn(dropletID, opt)
}

func (s *DropletsServiceMock) Backups(dropletID int, opt *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.BackupsFn(dropletID, opt)
}

func (s *DropletsServiceMock) Create(createRequest *godo.DropletCreateRequest) (*godo.Droplet, *godo.Response, error) {
	return s.CreateFn(createRequest)
}

func (s *DropletsServiceMock) Delete(dropletID int) (*godo.Response, error) {
	return s.DeleteFn(dropletID)
}

func (s *DropletsServiceMock) Get(dropletID int) (*godo.Droplet, *godo.Response, error) {
	return s.GetFn(dropletID)
}

func (s *DropletsServiceMock) Kernels(dropletID int, opt *godo.ListOptions) ([]godo.Kernel, *godo.Response, error) {
	return s.KernelsFn(dropletID, opt)
}

func (s *DropletsServiceMock) List(opt *godo.ListOptions) ([]godo.Droplet, *godo.Response, error) {
	return s.ListFn(opt)
}

func (s *DropletsServiceMock) Neighbors(dropletID int) ([]godo.Droplet, *godo.Response, error) {
	return s.NeighborsFn(dropletID)
}

func (s *DropletsServiceMock) Snapshots(dropletID int, opt *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.SnapshotsFn(dropletID, opt)
}

type ImagesServiceMock struct {
	ListFn             func(*godo.ListOptions) ([]godo.Image, *godo.Response, error)
	ListDistributionFn func(opt *godo.ListOptions) ([]godo.Image, *godo.Response, error)
	ListApplicationFn  func(opt *godo.ListOptions) ([]godo.Image, *godo.Response, error)
	ListUserFn         func(opt *godo.ListOptions) ([]godo.Image, *godo.Response, error)
	GetByIDFn          func(int) (*godo.Image, *godo.Response, error)
	GetBySlugFn        func(string) (*godo.Image, *godo.Response, error)
	UpdateFn           func(int, *godo.ImageUpdateRequest) (*godo.Image, *godo.Response, error)
	DeleteFn           func(int) (*godo.Response, error)
}

var _ godo.ImagesService = &ImagesServiceMock{}

func (s *ImagesServiceMock) List(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.ListFn(opts)
}

func (s *ImagesServiceMock) ListDistribution(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.ListDistributionFn(opts)
}

func (s *ImagesServiceMock) ListApplication(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.ListApplicationFn(opts)
}

func (s *ImagesServiceMock) ListUser(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.ListUserFn(opts)
}

func (s *ImagesServiceMock) GetByID(id int) (*godo.Image, *godo.Response, error) {
	return s.GetByIDFn(id)
}

func (s *ImagesServiceMock) GetBySlug(slug string) (*godo.Image, *godo.Response, error) {
	return s.GetBySlugFn(slug)
}

func (s *ImagesServiceMock) Update(id int, req *godo.ImageUpdateRequest) (*godo.Image, *godo.Response, error) {
	return s.UpdateFn(id, req)
}

func (s *ImagesServiceMock) Delete(id int) (*godo.Response, error) {
	return s.DeleteFn(id)
}

type ImageActionsServiceMock struct {
	GetFn      func(imageID, actionID int) (*godo.Action, *godo.Response, error)
	TransferFn func(imageID int, transferRequest *godo.ActionRequest) (*godo.Action, *godo.Response, error)
}

var _ godo.ImageActionsService = &ImageActionsServiceMock{}

func (s *ImageActionsServiceMock) Get(imageID, actionID int) (*godo.Action, *godo.Response, error) {
	return s.GetFn(imageID, actionID)
}

func (s *ImageActionsServiceMock) Transfer(imageID int, transferRequest *godo.ActionRequest) (*godo.Action, *godo.Response, error) {
	return s.TransferFn(imageID, transferRequest)
}

type RegionsServiceMock struct {
	ListFn func(opts *godo.ListOptions) ([]godo.Region, *godo.Response, error)
}

var _ godo.RegionsService = &RegionsServiceMock{}

func (s *RegionsServiceMock) List(opts *godo.ListOptions) ([]godo.Region, *godo.Response, error) {
	return s.ListFn(opts)
}

type SizesServiceMock struct {
	ListFn func(opts *godo.ListOptions) ([]godo.Size, *godo.Response, error)
}

var _ godo.SizesService = &SizesServiceMock{}

func (s *SizesServiceMock) List(opts *godo.ListOptions) ([]godo.Size, *godo.Response, error) {
	return s.ListFn(opts)
}

type KeysServiceMock struct {
	ListFn                func(*godo.ListOptions) ([]godo.Key, *godo.Response, error)
	GetByIDFn             func(int) (*godo.Key, *godo.Response, error)
	GetByFingerprintFn    func(string) (*godo.Key, *godo.Response, error)
	CreateFn              func(*godo.KeyCreateRequest) (*godo.Key, *godo.Response, error)
	UpdateByIDFn          func(int, *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error)
	UpdateByFingerprintFn func(string, *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error)
	DeleteByIDFn          func(int) (*godo.Response, error)
	DeleteByFingerprintFn func(string) (*godo.Response, error)
}

var _ godo.KeysService = &KeysServiceMock{}

func (s *KeysServiceMock) List(opts *godo.ListOptions) ([]godo.Key, *godo.Response, error) {
	return s.ListFn(opts)
}

func (s *KeysServiceMock) GetByID(id int) (*godo.Key, *godo.Response, error) {
	return s.GetByIDFn(id)
}

func (s *KeysServiceMock) GetByFingerprint(fingerprint string) (*godo.Key, *godo.Response, error) {
	return s.GetByFingerprintFn(fingerprint)
}

func (s *KeysServiceMock) Create(req *godo.KeyCreateRequest) (*godo.Key, *godo.Response, error) {
	return s.CreateFn(req)
}

func (s *KeysServiceMock) UpdateByID(id int, req *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error) {
	return s.UpdateByIDFn(id, req)
}

func (s *KeysServiceMock) UpdateByFingerprint(fingerprint string, req *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error) {
	return s.UpdateByFingerprintFn(fingerprint, req)
}

func (s *KeysServiceMock) DeleteByID(id int) (*godo.Response, error) {
	return s.DeleteByIDFn(id)
}

func (s *KeysServiceMock) DeleteByFingerprint(fingerprint string) (*godo.Response, error) {
	return s.DeleteByFingerprintFn(fingerprint)
}
