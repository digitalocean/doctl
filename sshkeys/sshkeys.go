package sshkeys

import (
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

const (
	argKey          = "key"
	argKeyName      = "key-name"
	argKeyPublicKey = "public-key"
)

func List(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	opts := docli.LoadOpts(c)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Keys.List(opt)
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
		logrus.WithField("err", err).Fatal("could not list keys")
	}

	list := make([]godo.Key, len(si))
	for i := range si {
		list[i] = si[i].(godo.Key)
	}

	err = docli.WriteJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

func Get(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	rawKey := c.String(argKey)

	var err error
	var key *godo.Key
	if i, aerr := strconv.Atoi(rawKey); aerr == nil {
		key, _, err = client.Keys.GetByID(i)
	} else {
		key, _, err = client.Keys.GetByFingerprint(rawKey)
	}

	if err != nil {
		logrus.WithField("err", err).Fatal("could not retrieve key")
	}

	err = docli.WriteJSON(key, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Create uploads a SSH key.
func Create(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)

	kcr := &godo.KeyCreateRequest{
		Name:      c.String(argKeyName),
		PublicKey: c.String(argKeyPublicKey),
	}

	r, _, err := client.Keys.Create(kcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create key")
	}

	err = docli.WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

func Delete(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	rawKey := c.String(argKey)

	var err error
	if i, aerr := strconv.Atoi(rawKey); aerr == nil {
		_, err = client.Keys.DeleteByID(i)
	} else {
		_, err = client.Keys.DeleteByFingerprint(rawKey)
	}

	if err != nil {
		logrus.WithField("err", err).Fatal("could not retrieve key")
	}
}

func Update(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	rawKey := c.String(argKey)

	req := &godo.KeyUpdateRequest{
		Name: c.String(argKeyName),
	}

	var err error
	var key *godo.Key
	if i, aerr := strconv.Atoi(rawKey); aerr == nil {
		key, _, err = client.Keys.UpdateByID(i, req)
	} else {
		key, _, err = client.Keys.UpdateByFingerprint(rawKey, req)
	}

	if err != nil {
		logrus.WithField("err", err).Fatal("could not update key")
	}

	err = docli.WriteJSON(key, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}
