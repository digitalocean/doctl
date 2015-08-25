package commands

import (
	"io/ioutil"
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestImageActionsGet(t *testing.T) {
	client := &godo.Client{
		ImageActions: &doit.ImageActionsServiceMock{
			GetFn: func(imageID, actionID int) (*godo.Action, *godo.Response, error) {
				assert.Equal(t, imageID, 1)
				assert.Equal(t, actionID, 2)
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c doit.ViperConfig) {
		ns := "test"
		c.Set(ns, doit.ArgImageID, 1)
		c.Set(ns, doit.ArgActionID, 2)

		RunImageActionsGet(ns, ioutil.Discard)
	})

}

func TestImageActionsTransfer(t *testing.T) {
	client := &godo.Client{
		ImageActions: &doit.ImageActionsServiceMock{
			TransferFn: func(imageID int, req *godo.ActionRequest) (*godo.Action, *godo.Response, error) {
				assert.Equal(t, imageID, 1)

				region := (*req)["region"]
				assert.Equal(t, region, "dev0")

				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c doit.ViperConfig) {
		ns := "test"
		c.Set(ns, doit.ArgImageID, 1)
		c.Set(ns, doit.ArgRegionSlug, "dev0")

		RunImageActionsTransfer(ns, ioutil.Discard)
	})
}
