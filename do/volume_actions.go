package do

import "github.com/digitalocean/godo"

// VolumeActionsService is an interface for interacting with DigitalOcean's account api.
type VolumeActionsService interface {
	Attach(string, int) (*Action, error)
	Detach(string) (*Action, error)
}

type volumeActionsService struct {
	client *godo.Client
}

var _ VolumeActionsService = &volumeActionsService{}

// NewAccountService builds an VolumeActionsService instance.
func NewVolumeActionsService(godoClient *godo.Client) VolumeActionsService {
	return &volumeActionsService{
		client: godoClient,
	}

}

func (das *volumeActionsService) handleActionResponse(a *godo.Action, err error) (*Action, error) {
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}

func (das *volumeActionsService) Attach(volumeID string, dropletID int) (*Action, error) {
	a, _, err := das.client.StorageActions.Attach(volumeID, dropletID)
	return das.handleActionResponse(a, err)

}

func (das *volumeActionsService) Detach(volumeID string) (*Action, error) {
	a, _, err := das.client.StorageActions.Detach(volumeID)
	return das.handleActionResponse(a, err)

}
