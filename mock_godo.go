package doit

import "github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"

// AccountServiceMock mocks github.com/digitalocean/AccountService.
type AccountServiceMock struct {
	GetFn func() (*godo.Account, *godo.Response, error)
}

var _ godo.AccountService = &AccountServiceMock{}

// Get mocks github.com/digitalocean/AccountService.Get.
func (s *AccountServiceMock) Get() (*godo.Account, *godo.Response, error) {
	return s.GetFn()
}

// ActionsServiceMock mocks github.com/digitalocean/godo/ActionsService.
type ActionsServiceMock struct {
	GetFn  func(id int) (*godo.Action, *godo.Response, error)
	ListFn func(opts *godo.ListOptions) ([]godo.Action, *godo.Response, error)
}

var _ godo.ActionsService = &ActionsServiceMock{}

// List is a mocked method.
func (s *ActionsServiceMock) List(opts *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
	return s.ListFn(opts)
}

// Get is a mocked method.
func (s *ActionsServiceMock) Get(id int) (*godo.Action, *godo.Response, error) {
	return s.GetFn(id)
}

// DomainsServiceMock mocks github.com/digitalocean/godo/DomainsService.
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

// List is a mocked method.
func (s *DomainsServiceMock) List(opts *godo.ListOptions) ([]godo.Domain, *godo.Response, error) {
	return s.ListFn(opts)
}

// Get is a mocked method.
func (s *DomainsServiceMock) Get(name string) (*godo.Domain, *godo.Response, error) {
	return s.GetFn(name)
}

// Create is a mocked method.
func (s *DomainsServiceMock) Create(req *godo.DomainCreateRequest) (*godo.Domain, *godo.Response, error) {
	return s.CreateFn(req)
}

// Delete is a mocked method.
func (s *DomainsServiceMock) Delete(name string) (*godo.Response, error) {
	return s.DeleteFn(name)
}

// Records is a mocked method.
func (s *DomainsServiceMock) Records(name string, opts *godo.ListOptions) ([]godo.DomainRecord, *godo.Response, error) {
	return s.RecordsFn(name, opts)
}

// Record is a mocked method.
func (s *DomainsServiceMock) Record(name string, id int) (*godo.DomainRecord, *godo.Response, error) {
	return s.RecordFn(name, id)
}

// DeleteRecord is a mocked method.
func (s *DomainsServiceMock) DeleteRecord(name string, id int) (*godo.Response, error) {
	return s.DeleteRecordFn(name, id)
}

// EditRecord is a mocked method.
func (s *DomainsServiceMock) EditRecord(name string, id int, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
	return s.EditRecordFn(name, id, req)
}

// CreateRecord is a mocked method.
func (s *DomainsServiceMock) CreateRecord(name string, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
	return s.CreateRecordFn(name, req)
}

// DropletActionsServiceMock mocks github.com/digitalocean/godo/DropletActionsServiceMock.
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

// ChangeKernel is a mocked method.
func (s *DropletActionsServiceMock) ChangeKernel(id, kernelID int) (*godo.Action, *godo.Response, error) {
	return s.ChangeKernelFn(id, kernelID)
}

// DisableBackups is a mocked method.
func (s *DropletActionsServiceMock) DisableBackups(id int) (*godo.Action, *godo.Response, error) {
	return s.DisableBackupsFn(id)
}

// EnableIPv6 is a mocked method.
func (s *DropletActionsServiceMock) EnableIPv6(id int) (*godo.Action, *godo.Response, error) {
	return s.EnableIPv6Fn(id)
}

// EnablePrivateNetworking is a mocked method.
func (s *DropletActionsServiceMock) EnablePrivateNetworking(id int) (*godo.Action, *godo.Response, error) {
	return s.EnablePrivateNetworkingFn(id)
}

// Get is a mocked method.
func (s *DropletActionsServiceMock) Get(dropletID, actionID int) (*godo.Action, *godo.Response, error) {
	return s.GetFn(dropletID, actionID)
}

// GetByURI is a mocked method.
func (s *DropletActionsServiceMock) GetByURI(rawurl string) (*godo.Action, *godo.Response, error) {
	return s.GetByURIFn(rawurl)
}

// PasswordReset is a mocked method.
func (s *DropletActionsServiceMock) PasswordReset(id int) (*godo.Action, *godo.Response, error) {
	return s.PasswordResetFn(id)
}

// PowerCycle is a mocked method.
func (s *DropletActionsServiceMock) PowerCycle(id int) (*godo.Action, *godo.Response, error) {
	return s.PowerCycleFn(id)
}

// PowerOff is a mocked method.
func (s *DropletActionsServiceMock) PowerOff(id int) (*godo.Action, *godo.Response, error) {
	return s.PowerOffFn(id)
}

// PowerOn is a mocked method.
func (s *DropletActionsServiceMock) PowerOn(id int) (*godo.Action, *godo.Response, error) {
	return s.PowerOnFn(id)
}

// Reboot is a mocked method.
func (s *DropletActionsServiceMock) Reboot(id int) (*godo.Action, *godo.Response, error) {
	return s.RebootFn(id)
}

// RebuildByImageID is a mocked method.
func (s *DropletActionsServiceMock) RebuildByImageID(id, imageID int) (*godo.Action, *godo.Response, error) {
	return s.RebuildByImageIDFn(id, imageID)
}

// RebuildByImageSlug is a mocked method.
func (s *DropletActionsServiceMock) RebuildByImageSlug(id int, slug string) (*godo.Action, *godo.Response, error) {
	return s.RebuildByImageSlugFn(id, slug)
}

// Rename is a mocked method.
func (s *DropletActionsServiceMock) Rename(id int, name string) (*godo.Action, *godo.Response, error) {
	return s.RenameFn(id, name)
}

// Resize is a mocked method.
func (s *DropletActionsServiceMock) Resize(id int, sizeSlug string, resizeDisk bool) (*godo.Action, *godo.Response, error) {
	return s.ResizeFn(id, sizeSlug, resizeDisk)
}

// Restore is a mocked method.
func (s *DropletActionsServiceMock) Restore(id, imageID int) (*godo.Action, *godo.Response, error) {
	return s.RestoreFn(id, imageID)
}

// Shutdown is a mocked method.
func (s *DropletActionsServiceMock) Shutdown(id int) (*godo.Action, *godo.Response, error) {
	return s.ShutdownFn(id)
}

// Snapshot is a mocked method.
func (s *DropletActionsServiceMock) Snapshot(id int, name string) (*godo.Action, *godo.Response, error) {
	return s.SnapshotFn(id, name)
}

// Upgrade is a mocked method.
func (s *DropletActionsServiceMock) Upgrade(id int) (*godo.Action, *godo.Response, error) {
	return s.UpgradeFn(id)
}

// DropletsServiceMock mocks github.com/digitalocean/godo/DropletsService.
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

// Actions is a mocked method.
func (s *DropletsServiceMock) Actions(dropletID int, opt *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
	return s.ActionsFn(dropletID, opt)
}

// Backups is a mocked method.
func (s *DropletsServiceMock) Backups(dropletID int, opt *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.BackupsFn(dropletID, opt)
}

// Create is a mocked method.
func (s *DropletsServiceMock) Create(createRequest *godo.DropletCreateRequest) (*godo.Droplet, *godo.Response, error) {
	return s.CreateFn(createRequest)
}

// Delete is a mocked method.
func (s *DropletsServiceMock) Delete(dropletID int) (*godo.Response, error) {
	return s.DeleteFn(dropletID)
}

// Get is a mocked method.
func (s *DropletsServiceMock) Get(dropletID int) (*godo.Droplet, *godo.Response, error) {
	return s.GetFn(dropletID)
}

// Kernels is a mocked method.
func (s *DropletsServiceMock) Kernels(dropletID int, opt *godo.ListOptions) ([]godo.Kernel, *godo.Response, error) {
	return s.KernelsFn(dropletID, opt)
}

// List is a mocked method.
func (s *DropletsServiceMock) List(opt *godo.ListOptions) ([]godo.Droplet, *godo.Response, error) {
	return s.ListFn(opt)
}

// Neighbors is a mocked method.
func (s *DropletsServiceMock) Neighbors(dropletID int) ([]godo.Droplet, *godo.Response, error) {
	return s.NeighborsFn(dropletID)
}

// Snapshots is a mocked method.
func (s *DropletsServiceMock) Snapshots(dropletID int, opt *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.SnapshotsFn(dropletID, opt)
}

// FloatingIPsServiceMock mocks github.com/digitalocean/godo/FloatingIPsService.
type FloatingIPsServiceMock struct {
	ListFn   func(*godo.ListOptions) ([]godo.FloatingIP, *godo.Response, error)
	GetFn    func(string) (*godo.FloatingIP, *godo.Response, error)
	CreateFn func(*godo.FloatingIPCreateRequest) (*godo.FloatingIP, *godo.Response, error)
	DeleteFn func(string) (*godo.Response, error)
}

var _ godo.FloatingIPsService = &FloatingIPsServiceMock{}

// List is a mocked method.
func (f *FloatingIPsServiceMock) List(opt *godo.ListOptions) ([]godo.FloatingIP, *godo.Response, error) {
	return f.ListFn(opt)
}

// Get is a mocked method.
func (f *FloatingIPsServiceMock) Get(ip string) (*godo.FloatingIP, *godo.Response, error) {
	return f.GetFn(ip)
}

// Create is a mocked method.
func (f *FloatingIPsServiceMock) Create(createRequest *godo.FloatingIPCreateRequest) (*godo.FloatingIP, *godo.Response, error) {
	return f.CreateFn(createRequest)
}

// Delete is a mocked method.
func (f *FloatingIPsServiceMock) Delete(ip string) (*godo.Response, error) {
	return f.DeleteFn(ip)
}

// FloatingIPActionsServiceMock mocks github.com/digitalocean/godo/FloatingIPActionsService.
type FloatingIPActionsServiceMock struct {
	AssignFn   func(ip string, dropletID int) (*godo.Action, *godo.Response, error)
	UnassignFn func(ip string) (*godo.Action, *godo.Response, error)
	GetFn      func(ip string, actionID int) (*godo.Action, *godo.Response, error)
}

var _ godo.FloatingIPActionsService = &FloatingIPActionsServiceMock{}

// Assign is a mocked method.
func (s *FloatingIPActionsServiceMock) Assign(ip string, dropletID int) (*godo.Action, *godo.Response, error) {
	return s.AssignFn(ip, dropletID)
}

// Unassign is a mocked method.
func (s *FloatingIPActionsServiceMock) Unassign(ip string) (*godo.Action, *godo.Response, error) {
	return s.UnassignFn(ip)
}

// Get is a mocked method.
func (s *FloatingIPActionsServiceMock) Get(ip string, actionID int) (*godo.Action, *godo.Response, error) {
	return s.GetFn(ip, actionID)
}

// ImagesServiceMock mocks github.com/digitalocean/godo/ImagesService.
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

// List is a mocked method.
func (s *ImagesServiceMock) List(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.ListFn(opts)
}

// ListDistribution is a mocked method.
func (s *ImagesServiceMock) ListDistribution(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.ListDistributionFn(opts)
}

// ListApplication is a mocked method.
func (s *ImagesServiceMock) ListApplication(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.ListApplicationFn(opts)
}

// ListUser is a mocked method.
func (s *ImagesServiceMock) ListUser(opts *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	return s.ListUserFn(opts)
}

// GetByID is a mocked method.
func (s *ImagesServiceMock) GetByID(id int) (*godo.Image, *godo.Response, error) {
	return s.GetByIDFn(id)
}

// GetBySlug is a mocked method.
func (s *ImagesServiceMock) GetBySlug(slug string) (*godo.Image, *godo.Response, error) {
	return s.GetBySlugFn(slug)
}

// Update is a mocked method.
func (s *ImagesServiceMock) Update(id int, req *godo.ImageUpdateRequest) (*godo.Image, *godo.Response, error) {
	return s.UpdateFn(id, req)
}

// Delete is a mocked method.
func (s *ImagesServiceMock) Delete(id int) (*godo.Response, error) {
	return s.DeleteFn(id)
}

// ImageActionsServiceMock mocks github.com/digitalocean/godo/ImagesActionsService.
type ImageActionsServiceMock struct {
	GetFn      func(imageID, actionID int) (*godo.Action, *godo.Response, error)
	TransferFn func(imageID int, transferRequest *godo.ActionRequest) (*godo.Action, *godo.Response, error)
}

var _ godo.ImageActionsService = &ImageActionsServiceMock{}

// Get is a mocked method.
func (s *ImageActionsServiceMock) Get(imageID, actionID int) (*godo.Action, *godo.Response, error) {
	return s.GetFn(imageID, actionID)
}

// Transfer is a mocked method.
func (s *ImageActionsServiceMock) Transfer(imageID int, transferRequest *godo.ActionRequest) (*godo.Action, *godo.Response, error) {
	return s.TransferFn(imageID, transferRequest)
}

// RegionsServiceMock mocks github.com/digitalocean/godo/RegionsService.
type RegionsServiceMock struct {
	ListFn func(opts *godo.ListOptions) ([]godo.Region, *godo.Response, error)
}

var _ godo.RegionsService = &RegionsServiceMock{}

// List is a mocked method.
func (s *RegionsServiceMock) List(opts *godo.ListOptions) ([]godo.Region, *godo.Response, error) {
	return s.ListFn(opts)
}

// SizesServiceMock mocks github.com/digitalocean/godo/SizesService.
type SizesServiceMock struct {
	ListFn func(opts *godo.ListOptions) ([]godo.Size, *godo.Response, error)
}

var _ godo.SizesService = &SizesServiceMock{}

// List is a mocked method.
func (s *SizesServiceMock) List(opts *godo.ListOptions) ([]godo.Size, *godo.Response, error) {
	return s.ListFn(opts)
}

// KeysServiceMock mocks github.com/digitalocean/godo/KeysService.
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

// List is a mocked method.
func (s *KeysServiceMock) List(opts *godo.ListOptions) ([]godo.Key, *godo.Response, error) {
	return s.ListFn(opts)
}

// GetByID is a mocked method.
func (s *KeysServiceMock) GetByID(id int) (*godo.Key, *godo.Response, error) {
	return s.GetByIDFn(id)
}

// GetByFingerprint is a mocked method.
func (s *KeysServiceMock) GetByFingerprint(fingerprint string) (*godo.Key, *godo.Response, error) {
	return s.GetByFingerprintFn(fingerprint)
}

// Create is a mocked method.
func (s *KeysServiceMock) Create(req *godo.KeyCreateRequest) (*godo.Key, *godo.Response, error) {
	return s.CreateFn(req)
}

// UpdateByID is a mocked method.
func (s *KeysServiceMock) UpdateByID(id int, req *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error) {
	return s.UpdateByIDFn(id, req)
}

// UpdateByFingerprint is a mocked method.
func (s *KeysServiceMock) UpdateByFingerprint(fingerprint string, req *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error) {
	return s.UpdateByFingerprintFn(fingerprint, req)
}

// DeleteByID is a mocked method.
func (s *KeysServiceMock) DeleteByID(id int) (*godo.Response, error) {
	return s.DeleteByIDFn(id)
}

// DeleteByFingerprint is a mocked method.
func (s *KeysServiceMock) DeleteByFingerprint(fingerprint string) (*godo.Response, error) {
	return s.DeleteByFingerprintFn(fingerprint)
}
