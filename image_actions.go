package doit

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

// Get retrieves an action for an image.
func ImageActionsGet(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	imageID := c.Int(ArgImageID)
	actionID := c.Int(ArgActionID)

	action, _, err := client.ImageActions.Get(imageID, actionID)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not get action for image")
	}

	err = displayOutput(c, action)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// Tranfer an image.
func ImageActionsTransfer(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgImageID)
	req := &godo.ActionRequest{
		"region": c.String(ArgRegionSlug),
	}

	action, _, err := client.ImageActions.Transfer(id, req)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not transfer image")
	}

	err = displayOutput(c, action)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}
