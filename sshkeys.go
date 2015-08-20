package doit

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"golang.org/x/crypto/ssh"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

// KeyList lists keys.
func KeyList(c *cli.Context) {
	client := NewClient(c, DefaultConfig)

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

	si, err := PaginateResp(f)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list keys")
	}

	list := make([]godo.Key, len(si))
	for i := range si {
		list[i] = si[i].(godo.Key)
	}

	err = DisplayOutput(c, list)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// KeyGet retrieves a key.
func KeyGet(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	rawKey := c.String(ArgKey)

	var err error
	var key *godo.Key
	if i, aerr := strconv.Atoi(rawKey); aerr == nil {
		key, _, err = client.Keys.GetByID(i)
	} else {
		if len(rawKey) > 0 {
			key, _, err = client.Keys.GetByFingerprint(rawKey)
		} else {
			err = fmt.Errorf("missing key id or fingerprint")
		}
	}

	if err != nil {
		logrus.WithField("err", err).Fatal("could not retrieve key")
	}

	err = DisplayOutput(c, key)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}

}

// KeyCreate uploads a SSH key.
func KeyCreate(c *cli.Context) {
	client := NewClient(c, DefaultConfig)

	kcr := &godo.KeyCreateRequest{
		Name:      c.String(ArgKeyName),
		PublicKey: c.String(ArgKeyPublicKey),
	}

	r, _, err := client.Keys.Create(kcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create key")
	}

	err = DisplayOutput(c, r)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

// KeyImport imports a key from a file
func KeyImport(c *cli.Context) {
	client := NewClient(c, DefaultConfig)

	keyPath := c.String(ArgKeyPublicKeyFile)
	keyName := c.String(ArgKeyName)

	keyFile, err := ioutil.ReadFile(keyPath)
	if err != nil {
		Bail(err, "could not read the public key")
	}

	_, comment, _, _, err := ssh.ParseAuthorizedKey(keyFile)
	if err != nil {
		Bail(err, "could ot parse public key")
	}

	if len(keyName) < 1 {
		keyName = comment
	}

	kcr := &godo.KeyCreateRequest{
		Name:      keyName,
		PublicKey: string(keyFile),
	}

	r, _, err := client.Keys.Create(kcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create key")
	}

	err = DisplayOutput(c, r)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}

}

// KeyDelete deletes a key.
func KeyDelete(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
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

// KeyUpdate updates a key.
func KeyUpdate(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
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

	err = DisplayOutput(c, key)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}
