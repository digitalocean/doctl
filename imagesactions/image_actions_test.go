package imageactions

import (
	"flag"
	"testing"

	"github.com/bryanl/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testAction = godo.Action{ID: 1}
)

func TestImageActionsGet(t *testing.T) {
	client := &godo.Client{
		ImageActions: &docli.ImageActionsServiceMock{
			GetFn: func(imageID, actionID int) (*godo.Action, *godo.Response, error) {
				assert.Equal(t, imageID, 1)
				assert.Equal(t, actionID, 2)
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argImageID, 1, "image-id")
	fs.Int(argActionID, 2, "action-id")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Get(c)
	})
}

func TestImageActionsTransfer(t *testing.T) {
	client := &godo.Client{
		ImageActions: &docli.ImageActionsServiceMock{
			TransferFn: func(imageID int, req *godo.ActionRequest) (*godo.Action, *godo.Response, error) {
				assert.Equal(t, imageID, 1)

				region := (*req)["region"]
				assert.Equal(t, region, "dev0")

				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argImageID, 1, "image-id")
	fs.String(argRegionSlug, "dev0", "region")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Transfer(c)
	})
}
