package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// VolumeActionsService is an interface for interacting with DigitalOcean's volume-action api.
type VolumeActionsService interface {
	Attach(string, int) (*Action, error)
	Detach(string, int) (*Action, error)
	Get(string, int) (*Action, error)
	List(string) ([]Action, error)
	Resize(string, int, string) (*Action, error)
}

type volumeActionsService struct {
	client *godo.Client
}

var _ VolumeActionsService = &volumeActionsService{}

// NewVolumeActionsService builds an VolumeActionsService instance.
func NewVolumeActionsService(godoClient *godo.Client) VolumeActionsService {
	return &volumeActionsService{
		client: godoClient,
	}

}

func (vas *volumeActionsService) handleActionResponse(a *godo.Action, err error) (*Action, error) {
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}

func (vas *volumeActionsService) Attach(volumeID string, dropletID int) (*Action, error) {
	a, _, err := vas.client.StorageActions.Attach(context.TODO(), volumeID, dropletID)
	return vas.handleActionResponse(a, err)

}

func (vas *volumeActionsService) Detach(volumeID string, dropletID int) (*Action, error) {
	a, _, err := vas.client.StorageActions.DetachByDropletID(context.TODO(), volumeID, dropletID)
	return vas.handleActionResponse(a, err)

}

func (vas *volumeActionsService) Get(volumeID string, actionID int) (*Action, error) {
	a, _, err := vas.client.StorageActions.Get(context.TODO(), volumeID, actionID)
	return vas.handleActionResponse(a, err)
}

func (vas *volumeActionsService) List(volumeID string) ([]Action, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := vas.client.StorageActions.List(context.TODO(), volumeID, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(Actions, len(si))
	for i := range si {
		a := si[i].(godo.Action)
		list[i] = Action{Action: &a}
	}

	return list, nil
}

func (vas *volumeActionsService) Resize(volumeID string, sizeGigabytes int, regionSlug string) (*Action, error) {
	a, _, err := vas.client.StorageActions.Resize(context.TODO(), volumeID, sizeGigabytes, regionSlug)
	return vas.handleActionResponse(a, err)
}
