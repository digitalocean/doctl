package commands

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

type sshMock struct {
	user    string
	host    string
	didRun  bool
	isError bool
}

func (s *sshMock) cmd() func(u, h, kp string, p int) doit.Runner {
	return func(u, h, kp string, p int) doit.Runner {
		s.didRun = true
		s.user = u
		s.host = h

		r := &doit.MockRunner{}

		if s.isError {
			r.Err = errors.New("ssh forced failure")
		}

		return r
	}
}

func TestSSH_ID(t *testing.T) {
	didFetchDroplet := false

	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
			GetFn: func(id int) (*godo.Droplet, *godo.Response, error) {
				assert.Equal(t, id, testDroplet.ID, "droplet ids did not match")
				didFetchDroplet = true
				return &testDroplet, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ms := &sshMock{}
		c.SSHFn = ms.cmd()

		ns := "test"
		err := RunSSH(ns, c, ioutil.Discard, []string{strconv.Itoa(testDroplet.ID)})
		assert.NoError(t, err)
		assert.True(t, didFetchDroplet)
		assert.True(t, ms.didRun)
		assert.Equal(t, "root", ms.user)
		assert.Equal(t, testDroplet.Networks.V4[0].IPAddress, ms.host)
	})
}

func TestSSH_InvalidID(t *testing.T) {
	didFetchDroplet := false

	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
			GetFn: func(id int) (*godo.Droplet, *godo.Response, error) {
				didFetchDroplet = true
				return nil, nil, fmt.Errorf("not here")
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ms := &sshMock{}
		c.SSHFn = ms.cmd()

		ns := "test"
		c.Set(ns, doit.ArgDropletID, testDroplet.ID)

		err := RunSSH(ns, c, ioutil.Discard, []string{})
		assert.Error(t, err)
	})
}

func TestSSH_Name(t *testing.T) {
	didFetchDroplet := false

	client := &godo.Client{
		Droplets: &doit.DropletsServiceMock{
			ListFn: func(*godo.ListOptions) ([]godo.Droplet, *godo.Response, error) {
				didFetchDroplet = true
				return testDropletList, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		ms := &sshMock{}
		c.SSHFn = ms.cmd()

		ns := "test"

		err := RunSSH(ns, c, ioutil.Discard, []string{testDroplet.Name})
		assert.NoError(t, err)

		assert.Equal(t, "root", ms.user)
		assert.Equal(t, testDroplet.Networks.V4[0].IPAddress, ms.host)
	})
}

func Test_extractHostInfo(t *testing.T) {
	cases := []struct {
		s string
		e sshHostInfo
	}{
		{s: "host", e: sshHostInfo{host: "host"}},
		{s: "root@host", e: sshHostInfo{user: "root", host: "host"}},
		{s: "root@host:22", e: sshHostInfo{user: "root", host: "host", port: "22"}},
		{s: "host:22", e: sshHostInfo{host: "host", port: "22"}},
		{s: "dokku@simple-task-02efb9c4", e: sshHostInfo{host: "simple-task-02efb9c4", user: "dokku"}},
	}

	for _, c := range cases {
		i := extractHostInfo(c.s)
		assert.Equal(t, c.e, i)
	}
}
