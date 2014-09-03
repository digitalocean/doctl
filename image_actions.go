package godo

import "fmt"

// ImageActionsService handles communition with the image action related methods of the
// DigitalOcean API.
type ImageActionsService struct {
	client *Client
}

// Transfer an image
func (i *ImageActionsService) Transfer(imageID int, transferRequest *ActionRequest) (*Action, *Response, error) {
	path := fmt.Sprintf("v2/images/%d/actions", imageID)

	req, err := i.client.NewRequest("POST", path, transferRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(actionRoot)
	resp, err := i.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.Event, resp, err
}

// Get an action for a particular image by id.
func (i *ImageActionsService) Get(imageID, actionID int) (*Action, *Response, error) {
	path := fmt.Sprintf("v2/images/%d/actions/%d", imageID, actionID)

	req, err := i.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(actionRoot)
	resp, err := i.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.Event, resp, err
}
