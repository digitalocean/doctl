package doit

import (
	"fmt"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

// List images.
func ImagesList(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	opts := LoadOpts(c)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Images.List(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list images")
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	err = displayOutput(c, list)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// ListDistribution lists distributions that are available.
func ImagesListDistribution(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	opts := LoadOpts(c)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Images.ListDistribution(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list distributions")
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	err = writeJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// ListApplication lists application iamges.
func ImagesListApplication(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	opts := LoadOpts(c)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Images.ListApplication(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list application images")
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	err = writeJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// ListUser lists user images.
func ImagesListUser(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	opts := LoadOpts(c)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Images.ListUser(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list user images")
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	err = writeJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Get retrieves an image by id or slug.
func ImagesGet(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	rawID := c.String(ArgImage)

	var err error
	var image *godo.Image
	if id, cerr := strconv.Atoi(rawID); cerr == nil {
		image, _, err = client.Images.GetByID(id)
	} else {
		if len(rawID) > 0 {
			image, _, err = client.Images.GetBySlug(rawID)
		} else {
			err = fmt.Errorf("image identifier is required")
		}
	}

	if err != nil {
		Bail(err, "could not retrieve image")
	}

	err = writeJSON(image, c.App.Writer)
	if err != nil {
		Bail(err, "coult not write JSON")
	}
}

// Update updates an image.
func ImagesUpdate(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgImageID)

	req := &godo.ImageUpdateRequest{
		Name: c.String(ArgImageName),
	}

	image, _, err := client.Images.Update(id, req)
	if err != nil {
		Bail(err, "could not update image")
	}

	err = writeJSON(image, c.App.Writer)
	if err != nil {
		Bail(err, "could not write JSON")
	}
}

func ImagesDelete(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgImageID)

	_, err := client.Images.Delete(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not delete image")
	}
}
