package doit

import (
	"flag"
	"fmt"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

type sshMock struct {
	user    string
	host    string
	didRun  bool
	isError bool
}

func (s *sshMock) cmd() func(u, h string) error {
	return func(u, h string) error {
		s.didRun = true
		s.user = u
		s.host = h

		if s.isError {
			return fmt.Errorf("ssh forced failure")
		}

		return nil
	}
}

func TestSSH_ID(t *testing.T) {
	didFetchDroplet := false

	client := &godo.Client{
		Droplets: &DropletsServiceMock{
			GetFn: func(id int) (*godo.Droplet, *godo.Response, error) {
				assert.Equal(t, id, testDroplet.ID, "droplet ids did not match")
				didFetchDroplet = true
				return &testDroplet, nil, nil
			},
		},
	}

	ms := &sshMock{}
	cs := NewTestConfig(client)
	cs.SSHFn = ms.cmd()

	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, testDroplet.ID, ArgDropletID)

	withinTest(cs, fs, func(c *cli.Context) {
		SSH(c)
		assert.True(t, didFetchDroplet)
		assert.True(t, ms.didRun)
		assert.Equal(t, "root", ms.user)
		assert.Equal(t, testDroplet.Networks.V4[0].IPAddress, ms.host)
	})
}

func TestSSH_InvalidID(t *testing.T) {
	didFetchDroplet := false

	client := &godo.Client{
		Droplets: &DropletsServiceMock{
			GetFn: func(id int) (*godo.Droplet, *godo.Response, error) {
				didFetchDroplet = true
				return nil, nil, fmt.Errorf("not here")
			},
		},
	}

	ms := &sshMock{}
	cs := NewTestConfig(client)
	cs.SSHFn = ms.cmd()

	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, testDroplet.ID, ArgDropletID)

	withinTest(cs, fs, func(c *cli.Context) {
		SSH(c)
		assert.True(t, didFetchDroplet)
		assert.False(t, ms.didRun)
	})
}

func TestSSH_Name(t *testing.T) {
	didFetchDroplet := false

	client := &godo.Client{
		Droplets: &DropletsServiceMock{
			ListFn: func(*godo.ListOptions) ([]godo.Droplet, *godo.Response, error) {
				didFetchDroplet = true
				return testDropletList, nil, nil
			},
		},
	}

	ms := &sshMock{}
	cs := NewTestConfig(client)
	cs.SSHFn = ms.cmd()

	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgDropletName, testDroplet.Name, ArgDropletName)

	withinTest(cs, fs, func(c *cli.Context) {
		SSH(c)
		assert.True(t, didFetchDroplet)
		assert.True(t, ms.didRun)
		assert.Equal(t, "root", ms.user)
		assert.Equal(t, testDroplet.Networks.V4[0].IPAddress, ms.host)
	})
}

func TestSSH_InvalidName(t *testing.T) {
	didFetchDroplet := false

	client := &godo.Client{
		Droplets: &DropletsServiceMock{
			ListFn: func(*godo.ListOptions) ([]godo.Droplet, *godo.Response, error) {
				didFetchDroplet = true
				return nil, nil, fmt.Errorf("not here")
			},
		},
	}

	ms := &sshMock{}
	cs := NewTestConfig(client)
	cs.SSHFn = ms.cmd()

	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgDropletName, "nope", ArgDropletName)

	withinTest(cs, fs, func(c *cli.Context) {
		SSH(c)
		assert.True(t, didFetchDroplet)
		assert.False(t, ms.didRun)
	})
}

func TestSSH_InvalidOpts(t *testing.T) {

	client := &godo.Client{}

	ms := &sshMock{}
	cs := NewTestConfig(client)
	cs.SSHFn = ms.cmd()

	fs := flag.NewFlagSet("flag set", 0)

	withinTest(cs, fs, func(c *cli.Context) {
		SSH(c)
		assert.False(t, ms.didRun)
	})
}
