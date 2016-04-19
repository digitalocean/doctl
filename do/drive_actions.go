package do

import "github.com/digitalocean/godo"

// DriveActionsService is an interface for interacting with DigitalOcean's account api.
type DriveActionsService interface {
	Attach(string, int) (*Action, error)
	Detach(string) (*Action, error)
}

type driveActionsService struct {
	client *godo.Client
}

var _ DriveActionsService = &driveActionsService{}

// NewAccountService builds an DriveActionsService instance.
func NewDriveActionsService(godoClient *godo.Client) DriveActionsService {
	return &driveActionsService{
		client: godoClient,
	}

}

func (das *driveActionsService) handleActionResponse(a *godo.Action, err error) (*Action, error) {
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}

func (das *driveActionsService) Attach(driveID string, dropletID int) (*Action, error) {
	a, _, err := das.client.StorageActions.Attach(driveID, dropletID)
	return das.handleActionResponse(a, err)

}

func (das *driveActionsService) Detach(driveID string) (*Action, error) {
	a, _, err := das.client.StorageActions.Detach(driveID)
	return das.handleActionResponse(a, err)

}
