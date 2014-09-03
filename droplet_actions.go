package godo

import (
	"fmt"
	"net/url"
)

// DropletActionsService handles communication with the droplet action related
// methods of the DigitalOcean API.
type DropletActionsService struct {
	client *Client
}

// Shutdown a Droplet
func (s *DropletActionsService) Shutdown(id int) (*Action, *Response, error) {
	request := &ActionRequest{Type: "shutdown"}
	return s.doAction(id, request)
}

// PowerOff a Droplet
func (s *DropletActionsService) PowerOff(id int) (*Action, *Response, error) {
	request := &ActionRequest{Type: "power_off"}
	return s.doAction(id, request)
}

// PowerCycle a Droplet
func (s *DropletActionsService) PowerCycle(id int) (*Action, *Response, error) {
	request := &ActionRequest{Type: "power_cycle"}
	return s.doAction(id, request)
}

// Reboot a Droplet
func (s *DropletActionsService) Reboot(id int) (*Action, *Response, error) {
	request := &ActionRequest{Type: "reboot"}
	return s.doAction(id, request)
}

// Restore an image to a Droplet
func (s *DropletActionsService) Restore(id, imageID int) (*Action, *Response, error) {
	options := map[string]interface{}{
		"image": float64(imageID),
	}

	requestType := "restore"
	request := &ActionRequest{
		Type:   requestType,
		Params: options,
	}
	return s.doAction(id, request)
}

// Resize a Droplet
func (s *DropletActionsService) Resize(id int, sizeSlug string) (*Action, *Response, error) {
	options := map[string]interface{}{
		"size": sizeSlug,
	}

	requestType := "resize"
	request := &ActionRequest{
		Type:   requestType,
		Params: options,
	}
	return s.doAction(id, request)
}

// Rename a Droplet
func (s *DropletActionsService) Rename(id int, name string) (*Action, *Response, error) {
	options := map[string]interface{}{
		"name": name,
	}

	requestType := "rename"
	request := &ActionRequest{
		Type:   requestType,
		Params: options,
	}
	return s.doAction(id, request)
}

func (s *DropletActionsService) doAction(id int, request *ActionRequest) (*Action, *Response, error) {
	path := dropletActionPath(id)

	req, err := s.client.NewRequest("POST", path, request)
	if err != nil {
		return nil, nil, err
	}

	root := new(actionRoot)
	resp, err := s.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.Event, resp, err
}

// Get an action for a particular droplet by id.
func (s *DropletActionsService) Get(dropletID, actionID int) (*Action, *Response, error) {
	path := fmt.Sprintf("%s/%d", dropletActionPath(dropletID), actionID)
	return s.get(path)
}

// GetByURI gets an action for a particular droplet by id.
func (s *DropletActionsService) GetByURI(rawurl string) (*Action, *Response, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, nil, err
	}

	return s.get(u.Path)

}

func (s *DropletActionsService) get(path string) (*Action, *Response, error) {
	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(actionRoot)
	resp, err := s.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.Event, resp, err

}

func dropletActionPath(dropletID int) string {
	return fmt.Sprintf("v2/droplets/%d/actions", dropletID)
}
