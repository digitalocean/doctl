/*
Copyright 2017 The Doctl Authors All rights reserved.
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
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/pkg/runner"
	"github.com/digitalocean/doctl/pkg/runner/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestSCPComand(t *testing.T) {
	parent := &Command{
		Command: &cobra.Command{
			Use:   "compute",
			Short: "compute commands",
			Long:  "compute commands are for controlling and managing infrastructure",
		},
	}
	cmd := SCP(parent)
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd)
}

func TestSCP_NotEnoughArgs(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID))

		err := RunSCP(config)
		assert.Error(t, err)
	})
}

func TestSCP_TooManyArgs(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID))
		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID))
		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID))

		err := RunSCP(config)
		assert.Error(t, err)
	})
}

func TestSCP_FullID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Get", testDroplet.ID).Return(&testDroplet, nil)
		tm.droplets.On("Get", anotherTestDroplet.ID).Return(&anotherTestDroplet, nil)

		config.Args = append(config.Args, "test@"+strconv.Itoa(testDroplet.ID)+":test.txt")
		config.Args = append(config.Args, "test@"+strconv.Itoa(anotherTestDroplet.ID)+":test.txt")

		err := RunSCP(config)
		assert.NoError(t, err)
	})
}

func TestSCP_FullName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("List").Return(testDropletList, nil)

		config.Args = append(config.Args, "test@"+testDroplet.Name+":test.txt")
		config.Args = append(config.Args, "test@"+anotherTestDroplet.Name+":test.txt")

		err := RunSCP(config)
		assert.NoError(t, err)
	})
}

func TestSCP_FullLocal(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Get", testDroplet.ID).Return(&testDroplet, nil)

		config.Args = append(config.Args, "test@"+strconv.Itoa(testDroplet.ID)+":test.txt")
		config.Args = append(config.Args, "./test.txt")

		err := RunSCP(config)
		assert.NoError(t, err)
	})
}

func TestSCP_HostID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.On("Get", testDroplet.ID).Return(&testDroplet, nil)

		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID)+":test.txt")
		config.Args = append(config.Args, "./test.txt")

		err := RunSCP(config)
		assert.NoError(t, err)
	})
}

func TestSCP_DropletMissing(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, ":test.txt")
		config.Args = append(config.Args, "./test.txt")

		err := RunSCP(config)
		assert.Error(t, err)
	})
}

func TestSCP_DropletMissing2(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "user@:test.txt")
		config.Args = append(config.Args, "./test.txt")

		err := RunSCP(config)
		assert.Error(t, err)
	})
}

func TestSCP_FileMissing(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, strconv.Itoa(testDroplet.ID)+":")
		config.Args = append(config.Args, "./test.txt")

		err := RunSCP(config)
		assert.Error(t, err)
	})
}

func TestSCP_CustomPort(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		rm := &mocks.Runner{}
		rm.On("Run").Return(nil)

		tc := config.Doit.(*TestConfig)
		tc.SCPFn = func(f1, f2, keyPath string, port int) runner.Runner {
			assert.Equal(t, 2222, port)
			return rm
		}

		tm.droplets.On("List").Return(testDropletList, nil)

		config.Doit.Set(config.NS, doctl.ArgsSSHPort, "2222")
		config.Args = append(config.Args, "test@"+testDroplet.Name+":test.txt")
		config.Args = append(config.Args, "test@"+anotherTestDroplet.Name+":test.txt")

		err := RunSCP(config)
		assert.NoError(t, err)
	})
}

func Test_extractArgument(t *testing.T) {
	cases := []struct {
		s   string
		e   *scpHostInfo
		err error
	}{
		{s: "test@example.com:~/abc", e: &scpHostInfo{username: "test", host: "example.com", file: "~/abc"}, err: nil},
	}

	for _, c := range cases {
		i, err := extractArgument(c.s)
		assert.Equal(t, c.e, i)
		assert.Equal(t, c.err, err)
	}
}

func Test_formatSCPArgument(t *testing.T) {
	cases := []struct {
		s *scpHostInfo
		e string
	}{
		{s: &scpHostInfo{username: "test", host: "example.com", file: "~/abc"}, e: "test@example.com:~/abc"},
	}

	for _, c := range cases {
		i := formatSCPArgument(c.s)
		assert.Equal(t, c.e, i)
	}
}
