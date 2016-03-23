package godomock

import "github.com/digitalocean/godo"

// DropletActionsService is the godo DropletActions interface.
type DropletActionsService interface {
	Shutdown(int) (*godo.Action, *godo.Response, error)
	ShutdownByTag(string) (*godo.Action, *godo.Response, error)
	PowerOff(int) (*godo.Action, *godo.Response, error)
	PowerOffByTag(string) (*godo.Action, *godo.Response, error)
	PowerOn(int) (*godo.Action, *godo.Response, error)
	PowerOnByTag(string) (*godo.Action, *godo.Response, error)
	PowerCycle(int) (*godo.Action, *godo.Response, error)
	PowerCycleByTag(string) (*godo.Action, *godo.Response, error)
	Reboot(int) (*godo.Action, *godo.Response, error)
	Restore(int, int) (*godo.Action, *godo.Response, error)
	Resize(int, string, bool) (*godo.Action, *godo.Response, error)
	Rename(int, string) (*godo.Action, *godo.Response, error)
	Snapshot(int, string) (*godo.Action, *godo.Response, error)
	SnapshotByTag(string, string) (*godo.Action, *godo.Response, error)
	EnableBackups(int) (*godo.Action, *godo.Response, error)
	EnableBackupsByTag(string) (*godo.Action, *godo.Response, error)
	DisableBackups(int) (*godo.Action, *godo.Response, error)
	DisableBackupsByTag(string) (*godo.Action, *godo.Response, error)
	PasswordReset(int) (*godo.Action, *godo.Response, error)
	RebuildByImageID(int, int) (*godo.Action, *godo.Response, error)
	RebuildByImageSlug(int, string) (*godo.Action, *godo.Response, error)
	ChangeKernel(int, int) (*godo.Action, *godo.Response, error)
	EnableIPv6(int) (*godo.Action, *godo.Response, error)
	EnableIPv6ByTag(string) (*godo.Action, *godo.Response, error)
	EnablePrivateNetworking(int) (*godo.Action, *godo.Response, error)
	EnablePrivateNetworkingByTag(string) (*godo.Action, *godo.Response, error)
	Upgrade(int) (*godo.Action, *godo.Response, error)
	Get(int, int) (*godo.Action, *godo.Response, error)
	GetByURI(string) (*godo.Action, *godo.Response, error)
}
