package doit

import (
	"io/ioutil"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/digitalocean/godo/util"
)

// Actions returns a list of actions for a droplet.
func DropletActions(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgDropletID)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Actions(id, opt)
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
		logrus.WithField("err", err).Fatal("could not list actions for droplet")
	}

	list := make([]godo.Action, len(si))
	for i := range si {
		list[i] = si[i].(godo.Action)
	}

	err = DisplayOutput(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// Backups returns a list of backup images for a droplet.
func DropletBackups(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgDropletID)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Backups(id, opt)
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
		logrus.WithField("err", err).Fatal("could not list backups for droplet")
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	err = DisplayOutput(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// Create creates a droplet.
func DropletCreate(c *cli.Context) {
	client := NewClient(c, DefaultConfig)

	sshKeys := []godo.DropletCreateSSHKey{}
	for _, rawKey := range c.StringSlice(ArgSSHKeys) {
		if i, err := strconv.Atoi(rawKey); err == nil {
			sshKeys = append(sshKeys, godo.DropletCreateSSHKey{ID: i})
			continue
		}

		sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: rawKey})
	}

	userData := c.String(ArgUserData)
	if userData == "" && c.String(ArgUserDataFile) != "" {
		data, err := ioutil.ReadFile(c.String(ArgUserDataFile))
		if err != nil {
			logrus.WithField("err", err).Fatal("could not read user-data file")
		}
		userData = string(data)
	}

	wait := c.Bool(ArgDropletWait)

	dcr := &godo.DropletCreateRequest{
		Name:              c.String(ArgDropletName),
		Region:            c.String(ArgRegionSlug),
		Size:              c.String(ArgSizeSlug),
		Backups:           c.Bool(ArgBackups),
		IPv6:              c.Bool(ArgIPv6),
		PrivateNetworking: c.Bool(ArgPrivateNetworking),
		SSHKeys:           sshKeys,
		UserData:          userData,
	}

	imageStr := c.String(ArgImage)
	if i, err := strconv.Atoi(imageStr); err == nil {
		dcr.Image = godo.DropletCreateImage{ID: i}
	} else {
		dcr.Image = godo.DropletCreateImage{Slug: imageStr}
	}

	r, resp, err := client.Droplets.Create(dcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create droplet")
	}

	var action *godo.LinkAction

	if wait {
		for _, a := range resp.Links.Actions {
			if a.Rel == "create" {
				action = &a
			}
		}
	}

	if action != nil {
		err = util.WaitForActive(client, action.HREF)
		if err != nil {
			logrus.WithField("err", err).Fatal("error waiting for droplet to become active")
		}

		r, err = getDropletByID(client, r.ID)
		if err != nil {
			Bail(err, "could not get droplet")
		}
	}

	err = DisplayOutput(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// Delete destroy a droplet by id.
func DropletDelete(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgDropletID)

	_, err := client.Droplets.Delete(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not delete droplet")
	}
}

// Get returns a droplet.
func DropletGet(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgDropletID)

	droplet, err := getDropletByID(client, id)
	if err != nil {
		Bail(err, "could not get droplet")
	}

	err = DisplayOutput(droplet, c.App.Writer)
	if err != nil {
		Bail(err, "could not write output")
	}
}

// Kernels returns a list of available kernels for a droplet.
func DropletKernels(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgDropletID)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Kernels(id, opt)
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
		logrus.WithField("err", err).Fatal("could not list kernels for droplet")
	}

	list := make([]godo.Kernel, len(si))
	for i := range si {
		list[i] = si[i].(godo.Kernel)
	}

	err = DisplayOutput(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// List returns a list of droplets.
func DropletList(c *cli.Context) {
	client := NewClient(c, DefaultConfig)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.List(opt)
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
		logrus.WithField("err", err).Fatal("could not list droplets")
	}

	list := make([]godo.Droplet, len(si))
	for i := range si {
		list[i] = si[i].(godo.Droplet)
	}

	err = DisplayOutput(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// Neighbors returns a list of droplet neighbors.
func DropletNeighbors(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgDropletID)

	list, _, err := client.Droplets.Neighbors(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list neighbors for droplet")
	}

	err = DisplayOutput(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// Snapshots returns a list of available kernels for a droplet.
func DropletSnapshots(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgDropletID)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Snapshots(id, opt)
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
		logrus.WithField("err", err).Fatal("could not list snapshots for droplet")
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	err = DisplayOutput(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}
