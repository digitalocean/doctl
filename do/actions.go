package do

import (
	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
)

//Action is a wrapper for godo.Action
type Action struct {
	*godo.Action
}

// Actions is a slice of Action.
type Actions []Action

// ActionsService is an interface for interacting with DigitalOcean's action api.
type ActionsService interface {
	List() (Actions, error)
	Get(int) (*Action, error)
}

type actionsService struct {
	client *godo.Client
}

var _ ActionsService = &actionsService{}

// NewActionsService builds an ActionsService instance.
func NewActionsService(godoClient *godo.Client) ActionsService {
	return &actionsService{
		client: godoClient,
	}
}

func (as *actionsService) List() (Actions, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := as.client.Actions.List(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(f)
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

func (as *actionsService) Get(id int) (*Action, error) {
	a, _, err := as.client.Actions.Get(id)
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}
