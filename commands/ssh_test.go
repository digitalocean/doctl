package commands

import (
	"errors"
	"strconv"
	"testing"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/pkg/runner"
	"github.com/stretchr/testify/assert"
)

type sshMock struct {
	user    string
	host    string
	didRun  bool
	isError bool
}

func TestSSHComand(t *testing.T) {
	cmd := SSH()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd)
}

func (s *sshMock) cmd() func(u, h, kp string, p int) runner.Runner {
	return func(u, h, kp string, p int) runner.Runner {
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

	// client := &godo.Client{
	// 	Droplets: &doit.DropletsServiceMock{
	// 		GetFn: func(id int) (*godo.Droplet, *godo.Response, error) {
	// 			assert.Equal(t, id, testDroplet.ID, "droplet ids did not match")
	// 			didFetchDroplet = true
	// 			return &testDroplet, nil, nil
	// 		},
	// 	},
	// }

	withTestClient(func(config *cmdConfig) {
		config.args = append(config.args, strconv.Itoa(testDroplet.ID))

		err := RunSSH(config)
		assert.NoError(t, err)
		assert.True(t, didFetchDroplet)
	})
}

func TestSSH_InvalidID(t *testing.T) {

	// client := &godo.Client{
	// 	Droplets: &doit.DropletsServiceMock{
	// 		GetFn: func(id int) (*godo.Droplet, *godo.Response, error) {
	// 			return nil, nil, fmt.Errorf("not here")
	// 		},
	// 	},
	// }

	withTestClient(func(config *cmdConfig) {
		err := RunSSH(config)
		assert.Error(t, err)
	})
}

func TestSSH_UnknownDroplet(t *testing.T) {
	// client := &godo.Client{
	// 	Droplets: &doit.DropletsServiceMock{
	// 		ListFn: func(*godo.ListOptions) ([]godo.Droplet, *godo.Response, error) {
	// 			return testDropletList, nil, nil
	// 		},
	// 	},
	// }

	withTestClient(func(config *cmdConfig) {
		config.args = append(config.args, "missing")

		err := RunSSH(config)
		assert.EqualError(t, err, "could not find droplet")
	})
}

func TestSSH_DropletWithNoPublic(t *testing.T) {
	// client := &godo.Client{
	// 	Droplets: &doit.DropletsServiceMock{
	// 		ListFn: func(*godo.ListOptions) ([]godo.Droplet, *godo.Response, error) {
	// 			return testPrivateDropletList, nil, nil
	// 		},
	// 	},
	// }

	withTestClient(func(config *cmdConfig) {
		config.args = append(config.args, testPrivateDroplet.Name)

		err := RunSSH(config)
		assert.EqualError(t, err, "could not find droplet address")
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
