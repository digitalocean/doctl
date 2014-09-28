package apiv2

import (
	"fmt"
)

type DropletAction struct {
	Type   string `json:"type"`
	Name   string `json:"name,moitempty"`
	Size   string `json:"size,omitempty"`
	Image  string `json:"image,omitempty"`
	Kernel int    `json:"kernel,omitempty"`
}

type DropletActionResponse struct {
	Action *Action `json:"action"`
}

func (d *Droplet) performAction(action *DropletAction) (*Action, error) {
	var response DropletActionResponse

	apiErr := d.Client.Post(fmt.Sprintf("droplets/%d/actions", d.ID), action, &response, nil)
	if apiErr != nil {
		return nil, fmt.Errorf("API Error: %s", apiErr.Message)
	}

	return response.Action, nil
}

func (d *Droplet) Reboot() (*Action, error) {
	action := &DropletAction{
		Type: "reboot",
	}
	return d.performAction(action)
}

func (d *Droplet) Powercycle() (*Action, error) {
	action := &DropletAction{
		Type: "power_cycle",
	}
	return d.performAction(action)
}

func (d *Droplet) Shutdown() (*Action, error) {
	action := &DropletAction{
		Type: "shutdown",
	}
	return d.performAction(action)
}

func (d *Droplet) Poweroff() (*Action, error) {
	action := &DropletAction{
		Type: "power_off",
	}
	return d.performAction(action)
}

func (d *Droplet) Poweron() (*Action, error) {
	action := &DropletAction{
		Type: "power_on",
	}
	return d.performAction(action)
}

func (d *Droplet) PasswordReset() (*Action, error) {
	action := &DropletAction{
		Type: "password_reset",
	}
	return d.performAction(action)
}

func (d *Droplet) Resize(size string) (*Action, error) {
	action := &DropletAction{
		Type: "resize",
		Size: size,
	}
	return d.performAction(action)
}

func (d *Droplet) Restore(image string) (*Action, error) {
	action := &DropletAction{
		Type:  "restore",
		Image: image,
	}
	return d.performAction(action)
}

func (d *Droplet) Rebuild(image string) (*Action, error) {
	action := &DropletAction{
		Type: "rebuild",
	}
	return d.performAction(action)
}

func (d *Droplet) Rename(name string) (*Action, error) {
	action := &DropletAction{
		Type: "rename",
		Name: name,
	}
	return d.performAction(action)
}

func (d *Droplet) ChangeKernel(kernel int) (*Action, error) {
	action := &DropletAction{
		Type:   "change_kernel",
		Kernel: kernel,
	}
	return d.performAction(action)
}

func (d *Droplet) EnableIPv6() (*Action, error) {
	action := &DropletAction{
		Type: "enable_ipv6",
	}
	return d.performAction(action)
}

func (d *Droplet) DisableBackups() (*Action, error) {
	action := &DropletAction{
		Type: "disable_backups",
	}
	return d.performAction(action)
}

func (d *Droplet) EnablePrivateNetworking() (*Action, error) {
	action := &DropletAction{
		Type: "enable_private_networking",
	}
	return d.performAction(action)
}

func (d *Droplet) SnapshotDroplet(name string) (*Action, error) {
	action := &DropletAction{
		Type: "snapshot",
		Name: name,
	}
	return d.performAction(action)
}
