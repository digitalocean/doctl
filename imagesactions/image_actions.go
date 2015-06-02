package imageactions

import (
	"github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

const (
	argImageID    = "image-id"
	argActionID   = "action-id"
	argRegionSlug = "region"
)

// Get retrieves an action for an image.
func Get(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	imageID := c.Int(argImageID)
	actionID := c.Int(argActionID)

	action, _, err := client.ImageActions.Get(imageID, actionID)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not get action for image")
	}

	err = docli.WriteJSON(action, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Tranfer an image.
func Transfer(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.Int(argImageID)
	req := &godo.ActionRequest{
		"region": c.String(argRegionSlug),
	}

	action, _, err := client.ImageActions.Transfer(id, req)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not transfer image")
	}

	err = docli.WriteJSON(action, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}

}
