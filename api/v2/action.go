package apiv2

const (
	ActionReboot                  = "reboot"
	ActionPowerCycle              = "power_cycle"
	ActionShutdown                = "shutdown"
	ActionPowerOff                = "power_off"
	ActionPowerOn                 = "power_on"
	ActionPasswordReset           = "password_reset"
	ActionResize                  = "resize"
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
// region			string	A slug representing the region where the action occurred.
type Action struct {
	ID           int    `json:"id,omitempty"`
	Status       string `json:"status"`
	Type         string `json:"type"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at"`
	ResourceID   int    `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	Region       string `json:"region"`
	client       *Client
}

func (c *Client) NewAction() *Action {
	return &Action{
		client: c,
	}
}
