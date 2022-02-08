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
	"os/exec"
	"sort"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunctionsCommand(t *testing.T) {
	cmd := Functions()
	assert.NotNil(t, cmd)
	expected := []string{"get", "invoke", "list"}

	names := []string{}
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}

	sort.Strings(expected)
	sort.Strings(names)
	assert.Equal(t, expected, names)
}

func TestFunctionsInvoke(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fakeCmd := &exec.Cmd{
			// Path:   "/bin/true",
			// Args:   []string{"action/invoke", "hello", "--param", "name", "world"},
			Stdout: config.Out,
		}
		out := make(map[string]string)
		out["body"] = "Hello world!"
		tm.sandbox.EXPECT().Cmd("action/invoke", []string{"hello", "--param", "name", "world"}).Return(fakeCmd, nil)
		tm.sandbox.EXPECT().Exec(gomock.Eq(fakeCmd)).Return(do.SandboxOutput{
			Entity: out,
		}, nil)

		config.Args = append(config.Args, "hello")
		config.Doit.Set(config.NS, "param", "name world")

		err := RunFunctionsInvoke(config)
		require.NoError(t, err)
	})

}
