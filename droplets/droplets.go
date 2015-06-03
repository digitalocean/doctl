package droplets

import (
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

const (
	argBackups           = "enable-backups"
	argDropletID         = "droplet-id"
	argDropletName       = "dropletj-name"
	argImage             = "image"
	argIPv6              = "enable-ipv6"
	argPrivateNetworking = "enable-private-networking"
	argRegionSlug        = "region"
	argSizeSlug          = "size"
	argSSHKeys           = "ssh-keys"
	argUserData          = "user-data"
)

// Actions returns a list of actions for a droplet.
func Actions(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.Int(argDropletID)
	opts := docli.LoadOpts(c)

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

	si, err := docli.PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list actions for droplet")
	}

	list := make([]godo.Action, len(si))
	for i := range si {
		list[i] = si[i].(godo.Action)
	}

	err = docli.WriteJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Backups returns a list of backup images for a droplet.
func Backups(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.Int(argDropletID)
	opts := docli.LoadOpts(c)

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

	si, err := docli.PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list backups for droplet")
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	err = docli.WriteJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Create creates a droplet.
func Create(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)

	sshKeys := []godo.DropletCreateSSHKey{}
	for _, rawKey := range c.StringSlice(argSSHKeys) {
		if i, err := strconv.Atoi(rawKey); err == nil {
			sshKeys = append(sshKeys, godo.DropletCreateSSHKey{ID: i})
			continue
		}

		sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: rawKey})
	}

	dcr := &godo.DropletCreateRequest{
		Name:              c.String(argDropletName),
		Region:            c.String(argRegionSlug),
		Size:              c.String(argSizeSlug),
		Backups:           c.Bool(argBackups),
		IPv6:              c.Bool(argIPv6),
		PrivateNetworking: c.Bool(argPrivateNetworking),
		SSHKeys:           sshKeys,
		UserData:          c.String(argUserData),
	}

	imageStr := c.String(argImage)
	if i, err := strconv.Atoi(imageStr); err == nil {
		dcr.Image = godo.DropletCreateImage{ID: i}
	} else {
		dcr.Image = godo.DropletCreateImage{Slug: imageStr}
	}

	r, _, err := client.Droplets.Create(dcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create droplet")
	}

	err = docli.WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Delete destroy a droplet by id.
func Delete(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.Int(argDropletID)

	_, err := client.Droplets.Delete(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not delete droplet")
	}
}

// Get returns a droplet.
func Get(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.Int(argDropletID)

	droplet, _, err := client.Droplets.Get(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not get droplet")
	}

	err = docli.WriteJSON(droplet, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Kernels returns a list of available kernels for a droplet.
func Kernels(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.Int(argDropletID)
	opts := docli.LoadOpts(c)

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

	si, err := docli.PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list kernels for droplet")
	}

	list := make([]godo.Kernel, len(si))
	for i := range si {
		list[i] = si[i].(godo.Kernel)
	}

	err = docli.WriteJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// List returns a list of droplets.
func List(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	opts := docli.LoadOpts(c)

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

	si, err := docli.PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list droplets")
	}

	list := make([]godo.Droplet, len(si))
	for i := range si {
		list[i] = si[i].(godo.Droplet)
	}

	err = docli.WriteJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Neighbors returns a list of droplet neighbors.
func Neighbors(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.Int(argDropletID)

	list, _, err := client.Droplets.Neighbors(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list neighbors for droplet")
	}

	err = docli.WriteJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Snapshots returns a list of available kernels for a droplet.
func Snapshots(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.Int(argDropletID)
	opts := docli.LoadOpts(c)

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

	si, err := docli.PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list snapshots for droplet")
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	err = docli.WriteJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}
