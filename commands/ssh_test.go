/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"strconv"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/pkg/runner"
	"github.com/digitalocean/doctl/pkg/ssh"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestSSHComand(t *testing.T) {
	parent := &Command{
		Command: &cobra.Command{
			Use:   "compute",
			Short: "compute commands",
			Long:  "compute commands are for controlling and managing infrastructure",
		},
	}
	cmd := SSH(parent)
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd)
}

func TestSSH_ID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Get(testDroplet.ID).Return(&testDroplet, nil)

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
		tm.droplets.EXPECT().List().Return(testDropletList, nil)

		config.Args = append(config.Args, "missing")

		err := RunSSH(config)
		assert.EqualError(t, err, "Could not find Droplet")
	})
}

func TestSSH_DropletWithNoPublic(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().List().Return(testPrivateDropletList, nil)

		config.Args = append(config.Args, testPrivateDroplet.Name)

		err := RunSSH(config)
		assert.EqualError(t, err, "Could not find Droplet address")
	})
}

func TestSSH_CustomPort(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.sshRunner.EXPECT().Run().Return(nil)

		tc := config.Doit.(*doctl.TestConfig)
		tc.SSHFn = func(user, host, keyPath string, port int, opts ssh.Options) runner.Runner {
			assert.Equal(t, 2222, port)
			return tm.sshRunner
		}

		tm.droplets.EXPECT().List().Return(testDropletList, nil)

		config.Doit.Set(config.NS, doctl.ArgsSSHPort, "2222")
		config.Args = append(config.Args, testDroplet.Name)

		err := RunSSH(config)
		assert.NoError(t, err)
	})
}

func TestSSH_CustomUser(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.sshRunner.EXPECT().Run().Return(nil)

		tc := config.Doit.(*doctl.TestConfig)
		tc.SSHFn = func(user, host, keyPath string, port int, opts ssh.Options) runner.Runner {
			assert.Equal(t, "foobar", user)
			return tm.sshRunner
		}

		tm.droplets.EXPECT().List().Return(testDropletList, nil)

		config.Doit.Set(config.NS, doctl.ArgSSHUser, "foobar")
		config.Args = append(config.Args, testDroplet.Name)

		err := RunSSH(config)
		assert.NoError(t, err)
	})
}

func TestSSH_AgentForwarding(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.sshRunner.EXPECT().Run().Return(nil)

		tc := config.Doit.(*doctl.TestConfig)
		tc.SSHFn = func(user, host, keyPath string, port int, opts ssh.Options) runner.Runner {
			assert.Equal(t, true, opts[doctl.ArgsSSHAgentForwarding])
			return tm.sshRunner
		}

		tm.droplets.EXPECT().List().Return(testDropletList, nil)

		config.Doit.Set(config.NS, doctl.ArgsSSHAgentForwarding, true)
		config.Args = append(config.Args, testDroplet.Name)

		err := RunSSH(config)
		assert.NoError(t, err)
	})
}

func TestSSH_CommandExecuting(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.sshRunner.EXPECT().Run().Return(nil)

		tc := config.Doit.(*doctl.TestConfig)
		tc.SSHFn = func(user, host, keyPath string, port int, opts ssh.Options) runner.Runner {
			assert.Equal(t, "uptime", opts[doctl.ArgSSHCommand])
			return tm.sshRunner
		}

		tm.droplets.EXPECT().List().Return(testDropletList, nil)
		config.Doit.Set(config.NS, doctl.ArgSSHCommand, "uptime")
		config.Args = append(config.Args, testDroplet.Name)

		err := RunSSH(config)
		assert.NoError(t, err)
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
