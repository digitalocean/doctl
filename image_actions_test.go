package docli

import (
	"flag"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestImageActionsGet(t *testing.T) {
	client := &godo.Client{
		ImageActions: &ImageActionsServiceMock{
			GetFn: func(imageID, actionID int) (*godo.Action, *godo.Response, error) {
				assert.Equal(t, imageID, 1)
				assert.Equal(t, actionID, 2)
				return &testAction, nil, nil
			},
		},
	}

	cs := &TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgImageID, 1, "image-id")
	fs.Int(ArgActionID, 2, "action-id")

	WithinTest(cs, fs, func(c *cli.Context) {
		ImageActionsGet(c)
	})
}

func TestImageActionsTransfer(t *testing.T) {
	client := &godo.Client{
		ImageActions: &ImageActionsServiceMock{
			TransferFn: func(imageID int, req *godo.ActionRequest) (*godo.Action, *godo.Response, error) {
				assert.Equal(t, imageID, 1)

				region := (*req)["region"]
				assert.Equal(t, region, "dev0")

				return &testAction, nil, nil
			},
		},
	}

	cs := &TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgImageID, 1, "image-id")
	fs.String(ArgRegionSlug, "dev0", "region")

	WithinTest(cs, fs, func(c *cli.Context) {
		ImageActionsTransfer(c)
	})
}
