package apiv2

import (
	"fmt"
)

const (
	ActionReboot                  = "reboot"
	ActionPowerCycle              = "power_cycle"
	ActionShutdown                = "shutdown"
	ActionPowerOff                = "power_off"
	ActionPowerOn                 = "power_on"
	ActionPasswordReset           = "password_reset"
	ActionReaction                = "reaction"
	ActionRestore                 = "restore"
	ActionRebuild                 = "rebuild"
	ActionRename                  = "rename"
	ActionChangeKernel            = "change_kernel"
	ActionEnableIPv6              = "enable_ipv6"
	ActionDisableBackups          = "disable_backups"
	ActionEnablePrivateNetworking = "enable_private_networking"
	ActionSnapshot                = "snapshot"
)

// id				number	A unique identifier for each Droplet action event. This is used to reference a specific action that was requested.
// status			string	The current status of the action. The value of this attribute will be "in-progress", "completed", or "errored".
// type				string	The type of action that the event is executing (reboot, power_off, etc.).
// started_at		string	A time value given in ISO8601 combined date and time format that represents when the action was initiated.
// completed_at		string	A time value given in ISO8601 combined date and time format that represents when the action was completed.
// resource_id		number	A unique identifier for the resource that the action is associated with.
// resource_type	string	The type of resource that the action is associated with.
// region			string	(deprecated) A slug representing the region where the action occurred.
// region_slug		string A slug representing the region where the action occurred.
type Action struct {
	ID           int    `json:"id,omitempty"`
	Status       string `json:"status"`
	Type         string `json:"type"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at"`
	ResourceID   int    `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	RegionSlug   string `json:"region_slug"`
	client       *Client
}

type ActionListResponse struct {
	Actions []*Action `json:"actions"`
	Meta    struct {
		Total int `json:"total"`
	} `json:"meta"`
}

type ActionResponse struct {
	Action *Action `json:"action"`
}

func (c *Client) NewAction() *Action {
	return &Action{
		client: c,
	}
}

func (c *Client) LoadAction(id int) (*Action, error) {
	var action ActionResponse

	err := c.Get(fmt.Sprintf("actions/%d", id), nil, &action, nil)
	if err != nil {
		return nil, fmt.Errorf("API Error: %s", err.Message)
	}

	return action.Action, nil
}

func (c *Client) ListAllActions() (*ActionListResponse, error) {
	var actionList *ActionListResponse

	err := c.Get("actions", nil, &actionList, nil)
	if err != nil {
		return nil, fmt.Errorf("API Error: %s", err.Message)
	}

	return actionList, nil
}
