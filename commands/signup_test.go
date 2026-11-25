/*
Copyright 2026 The Doctl Authors All rights reserved.
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
	"bytes"
	"errors"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignupCommand(t *testing.T) {
	cmd := Signup()
	assert.NotNil(t, cmd)
	assert.Equal(t, "signup", cmd.Name())
	assert.Equal(t, configureDoctlGroup, cmd.GroupID)
	assert.Contains(t, cmd.Long, "doctl auth login")
	assert.Contains(t, cmd.Long, "--no-browser")
	assert.Contains(t, cmd.Example, "doctl signup")
}

func TestSignupOpensBrowserAndPrintsLoginHelper(t *testing.T) {
	var opened string
	prev := openSignupURL
	openSignupURL = func(url string) error {
		opened = url
		return nil
	}
	t.Cleanup(func() { openSignupURL = prev })

	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		var out bytes.Buffer
		config.Out = &out

		err := RunSignup(config)
		require.NoError(t, err)

		assert.Equal(t, digitalOceanSignupURL, opened)
		assert.Contains(t, out.String(), digitalOceanSignupURL)
		assert.Contains(t, out.String(), "doctl auth login")
		assert.Contains(t, out.String(), "doctl auth init")
	})
}

func TestSignupNoBrowserSkipsOpen(t *testing.T) {
	openCalled := false
	prev := openSignupURL
	openSignupURL = func(url string) error {
		openCalled = true
		return nil
	}
	t.Cleanup(func() { openSignupURL = prev })

	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgOAuthNoBrowser, true)
		var out bytes.Buffer
		config.Out = &out

		err := RunSignup(config)
		require.NoError(t, err)

		assert.False(t, openCalled)
		assert.Contains(t, out.String(), digitalOceanSignupURL)
		assert.Contains(t, out.String(), "doctl auth login")
	})
}

func TestSignupContinuesWhenBrowserOpenFails(t *testing.T) {
	prev := openSignupURL
	openSignupURL = func(url string) error {
		return errors.New("no display")
	}
	t.Cleanup(func() { openSignupURL = prev })

	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		var out bytes.Buffer
		config.Out = &out

		err := RunSignup(config)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "Could not open a browser automatically")
		assert.Contains(t, out.String(), "doctl auth login")
	})
}
