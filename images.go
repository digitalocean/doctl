package doit

import (
	"fmt"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

// ImagesList images.
func ImagesList(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	listImages(client.Images.List, c)
}

// ImagesListDistribution lists distributions that are available.
func ImagesListDistribution(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	listImages(client.Images.ListDistribution, c)
}

// ImagesListApplication lists application iamges.
func ImagesListApplication(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	listImages(client.Images.ListApplication, c)
}

// ImagesListUser lists user images.
func ImagesListUser(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	listImages(client.Images.ListUser, c)
}

// ImagesGet retrieves an image by id or slug.
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
		return
	}

	err = DisplayOutput(c, image)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// ImagesUpdate updates an image.
func ImagesUpdate(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgImageID)

	req := &godo.ImageUpdateRequest{
		Name: c.String(ArgImageName),
	}

	image, _, err := client.Images.Update(id, req)
	if err != nil {
		Bail(err, "could not update image")
		return
	}

	err = DisplayOutput(c, image)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// ImagesDelete deletes an image.
func ImagesDelete(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int(ArgImageID)

	_, err := client.Images.Delete(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not delete image")
	}
}

type listFn func(*godo.ListOptions) ([]godo.Image, *godo.Response, error)

func listImages(lFn listFn, c *cli.Context) {
	public := c.Bool(ArgImagePublic)

	fn := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := lFn(opt)
		if err != nil {
			return nil, nil, err
		}

		si := []interface{}{}
		for _, i := range list {
			if (public && i.Public) || !public {
				si = append(si, i)
			}
		}

		return si, resp, err
	}

	si, err := PaginateResp(fn)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list images")
	}

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	err = DisplayOutput(c, list)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}
