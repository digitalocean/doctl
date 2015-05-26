package dropletactions

import (
	"strconv"

	"github.com/digitalocean/godo"
)

// DisableBackups disables backups for a droplet.
func DisableBackups(client *godo.Client, id int) (*godo.Action, error) {
	r, _, err := client.DropletActions.DisableBackups(id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Reboot reboots a droplet.
func Reboot(client *godo.Client, id int) (*godo.Action, error) {
	r, _, err := client.DropletActions.Reboot(id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// PowerCycle power cycles a droplet.
func PowerCycle(client *godo.Client, id int) (*godo.Action, error) {
	r, _, err := client.DropletActions.PowerCycle(id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Shutdown shuts a droplet down.
func Shutdown(client *godo.Client, id int) (*godo.Action, error) {
	r, _, err := client.DropletActions.Shutdown(id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// PowerOff turns droplet power off.
func PowerOff(client *godo.Client, id int) (*godo.Action, error) {
	r, _, err := client.DropletActions.PowerOff(id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// PowerOn turns droplet power on.
func PowerOn(client *godo.Client, id int) (*godo.Action, error) {
	r, _, err := client.DropletActions.PowerOn(id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// PasswordReset resets the droplet root password.
func PasswordReset(client *godo.Client, id int) (*godo.Action, error) {
	r, _, err := client.DropletActions.PasswordReset(id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// EnableIPv6 enables IPv6 for a droplet.
func EnableIPv6(client *godo.Client, id int) (*godo.Action, error) {
	r, _, err := client.DropletActions.EnableIPv6(id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// EnablePrivateNetworking enables private networking for a droplet.
func EnablePrivateNetworking(client *godo.Client, id int) (*godo.Action, error) {
	r, _, err := client.DropletActions.EnablePrivateNetworking(id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Upgrade upgrades a droplet.
func Upgrade(client *godo.Client, id int) (*godo.Action, error) {
	r, _, err := client.DropletActions.Upgrade(id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Restore restores a droplet using an image id.
func Restore(client *godo.Client, id, image int) (*godo.Action, error) {
	r, _, err := client.DropletActions.Restore(id, image)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Resize resizesx a droplet giving a size slug and optionally expands the disk.
func Resize(client *godo.Client, id int, size string, disk bool) (*godo.Action, error) {
	r, _, err := client.DropletActions.Resize(id, size, disk)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Rebuild rebuilds a droplet using an image id or slug.
func Rebuild(client *godo.Client, id int, image string) (*godo.Action, error) {
	var r *godo.Action
	var err error
	if i, aerr := strconv.Atoi(image); aerr != nil {
		r, _, err = client.DropletActions.RebuildByImageID(id, i)
	} else {
		r, _, err = client.DropletActions.RebuildByImageSlug(id, image)
	}

	if err != nil {
		return nil, err
	}

	return r, nil
}

// Rename renames a droplet.
func Rename(client *godo.Client, id int, name string) (*godo.Action, error) {
	r, _, err := client.DropletActions.Rename(id, name)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// ChangeKernel changes the kernel for a droplet.
func ChangeKernel(client *godo.Client, id int, kernel int) (*godo.Action, error) {
	r, _, err := client.DropletActions.ChangeKernel(id, kernel)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Snapshot creates a snapshot for a droplet.
func Snapshot(client *godo.Client, id int, name string) (*godo.Action, error) {
	r, _, err := client.DropletActions.Snapshot(id, name)
	if err != nil {
		return nil, err
	}

	return r, nil
}
