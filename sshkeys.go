package docli

import (
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func KeyList(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	opts := LoadOpts(c)

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

	si, err := PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list keys")
	}

	list := make([]godo.Key, len(si))
	for i := range si {
		list[i] = si[i].(godo.Key)
	}

	err = WriteJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

func KeyGet(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	rawKey := c.String(ArgKey)

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

	err = WriteJSON(key, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Create uploads a SSH key.
func KeyCreate(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)

	kcr := &godo.KeyCreateRequest{
		Name:      c.String(ArgKeyName),
		PublicKey: c.String(ArgKeyPublicKey),
	}

	r, _, err := client.Keys.Create(kcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create key")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

func KeyDelete(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	rawKey := c.String(ArgKey)

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

func KeyUpdate(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	rawKey := c.String(ArgKey)

	req := &godo.KeyUpdateRequest{
		Name: c.String(ArgKeyName),
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

	err = WriteJSON(key, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}
