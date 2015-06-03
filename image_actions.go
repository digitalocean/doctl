package docli

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

// Get retrieves an action for an image.
func ImageActionsGet(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	imageID := c.Int(argImageID)
	actionID := c.Int(argActionID)

	action, _, err := client.ImageActions.Get(imageID, actionID)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not get action for image")
	}

	err = WriteJSON(action, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Tranfer an image.
func ImageActionsTransfer(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argImageID)
	req := &godo.ActionRequest{
		"region": c.String(argRegionSlug),
	}

	action, _, err := client.ImageActions.Transfer(id, req)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not transfer image")
	}

	err = WriteJSON(action, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}

}
