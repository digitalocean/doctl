/*
Copyright 2016 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Get", testDroplet.ID).Return(&testDroplet, nil)

		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID))

		err := RunSSH(config)
		assert.NoError(t, err)
	})
}

func TestSSH_InvalidID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunSSH(config)
		assert.Error(t, err)
	})
}

func TestSSH_UnknownDroplet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("List").Return(testDropletList, nil)

		config.Args = append(config.Args, "missing")

		err := RunSSH(config)
		assert.EqualError(t, err, "could not find droplet")
	})
}

func TestSSH_DropletWithNoPublic(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("List").Return(testPrivateDropletList, nil)

		config.Args = append(config.Args, testPrivateDroplet.Name)

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
		{s: "dokku@simple-task-02efb9c544", e: sshHostInfo{host: "simple-task-02efb9c544", user: "dokku"}},
	}

	for _, c := range cases {
		i := extractHostInfo(c.s)
		assert.Equal(t, c.e, i)
	}
}
