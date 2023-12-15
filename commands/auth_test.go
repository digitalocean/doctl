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
	"bufio"
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"errors"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	yaml "gopkg.in/yaml.v2"
)

func TestAuthCommand(t *testing.T) {
	cmd := Auth()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "init", "list", "remove", "switch")
}

func TestAuthInit(t *testing.T) {
	cfw := cfgFileWriter
	viper.Set(doctl.ArgAccessToken, nil)
	defer func() {
		cfgFileWriter = cfw
	}()

	retrieveUserTokenFunc := func() (string, error) {
		return "valid-token", nil
	}

	cfgFileWriter = func() (io.WriteCloser, error) { return &nopWriteCloser{Writer: io.Discard}, nil }

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.NoError(t, err)
	})
}

func TestAuthInitConfig(t *testing.T) {
	cfw := cfgFileWriter
	viper.Set(doctl.ArgAccessToken, nil)
	defer func() {
		cfgFileWriter = cfw
	}()

	retrieveUserTokenFunc := func() (string, error) {
		return "valid-token", nil
	}

	var buf bytes.Buffer
	cfgFileWriter = func() (io.WriteCloser, error) {
		return &nopWriteCloser{
			Writer: bufio.NewWriter(&buf),
		}, nil
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.NoError(t, err)

		var configFile testConfig
		err = yaml.Unmarshal(buf.Bytes(), &configFile)
		assert.NoError(t, err)
		defaultCfgFile := filepath.Join(defaultConfigHome(), defaultConfigName)
		assert.Equal(t, configFile["config"], defaultCfgFile, "unexpected setting for 'config'")

		// Ensure that the dev.config.set.dev-config setting is correct to prevent
		// a conflict with the base config setting.
		devConfig := configFile["dev"]
		devConfigSetting := devConfig.(map[any]any)["config"]
		expectedConfigSetting := map[any]any(
			map[any]any{
				"set":   map[any]any{"dev-config": ""},
				"unset": map[any]any{"dev-config": ""},
			},
		)
		assert.Equal(t, expectedConfigSetting, devConfigSetting, "unexpected setting for 'dev.config'")
	})
}

func TestAuthInitWithProvidedToken(t *testing.T) {
	cfw := cfgFileWriter
	viper.Set(doctl.ArgAccessToken, "valid-token")
	defer func() {
		cfgFileWriter = cfw
		viper.Set(doctl.ArgAccessToken, nil)
	}()

	retrieveUserTokenFunc := func() (string, error) {
		return "", errors.New("should not have called this")
	}

	cfgFileWriter = func() (io.WriteCloser, error) { return &nopWriteCloser{Writer: io.Discard}, nil }

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.NoError(t, err)
	})
}

func TestAuthForcesLowercase(t *testing.T) {
	cfw := cfgFileWriter
	viper.Set(doctl.ArgAccessToken, "valid-token")
	defer func() {
		cfgFileWriter = cfw
		viper.Set(doctl.ArgAccessToken, nil)
	}()

	retrieveUserTokenFunc := func() (string, error) {
		return "", errors.New("should not have called this")
	}

	cfgFileWriter = func() (io.WriteCloser, error) { return &nopWriteCloser{Writer: io.Discard}, nil }

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		contexts := map[string]any{doctl.ArgDefaultContext: true, "TestCapitalCase": true}
		context := "TestCapitalCase"
		viper.Set("auth-contexts", contexts)
		viper.Set("context", context)

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.NoError(t, err)

		contexts = map[string]any{doctl.ArgDefaultContext: true, "TestCapitalCase": true}
		viper.Set("auth-contexts", contexts)
		viper.Set("context", "contextDoesntExist")
		err = RunAuthSwitch(config)
		// should error because context doesn't exist
		assert.Error(t, err)

		viper.Set("context", "testcapitalcase")
		err = RunAuthSwitch(config)
		// should not error because context does exist
		assert.NoError(t, err)
	})
}

func TestAuthList(t *testing.T) {
	buf := &bytes.Buffer{}
	config := &CmdConfig{Out: buf}

	err := RunAuthList(config)
	assert.NoError(t, err)
}

func Test_displayAuthContexts(t *testing.T) {
	testCases := []struct {
		Name     string
		Out      *bytes.Buffer
		Context  string
		Contexts map[string]any
		Expected string
	}{
		{
			Name:    "default context only",
			Out:     &bytes.Buffer{},
			Context: doctl.ArgDefaultContext,
			Contexts: map[string]any{
				doctl.ArgDefaultContext: true,
			},
			Expected: "default (current)\n",
		},
		{
			Name:    "default context and additional context",
			Out:     &bytes.Buffer{},
			Context: doctl.ArgDefaultContext,
			Contexts: map[string]any{
				doctl.ArgDefaultContext: true,
				"test":                  true,
			},
			Expected: "default (current)\ntest\n",
		},
		{
			Name:    "default context and additional context set to additional context",
			Out:     &bytes.Buffer{},
			Context: "test",
			Contexts: map[string]any{
				doctl.ArgDefaultContext: true,
				"test":                  true,
			},
			Expected: "default\ntest (current)\n",
		},
		{
			Name:    "unset context",
			Out:     &bytes.Buffer{},
			Context: "missing",
			Contexts: map[string]any{
				doctl.ArgDefaultContext: true,
				"test":                  true,
			},
			Expected: "default\ntest\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			displayAuthContexts(tc.Out, tc.Context, tc.Contexts)
			assert.Equal(t, tc.Expected, tc.Out.String())
		})
	}
}

func TestTokenInputValidator(t *testing.T) {
	tests := []struct {
		name  string
		token string
		valid bool
	}{
		{
			name:  "valid legacy token",
			token: "53918d3cd735062ca6ea791427900af10cf595f18dc6016c1cb0c3a11adcae84",
			valid: true,
		},
		{
			name:  "valid v1 pat",
			token: "dop_v1_53918d3cd735062ca6ea791427900af10cf595f18dc6016c1cb0c3a11adcae84",
			valid: true,
		},
		{
			name:  "valid v1 oauth",
			token: "doo_v1_53918d3cd735062ca6ea791427900af10cf595f18dc6016c1cb0c3a11adcae84",
			valid: true,
		},
		{
			name:  "too short legacy token",
			token: "53918d3cd735062ca6ea791427900af10cf595f18dc6016c1cb0c3a11adca",
		},
		{
			name:  "too long legacy token",
			token: "53918d3cd735062ca6ea791427900af10cf595f18dc6016c1cb0c3a11adcae84a2d45",
		},
		{
			name:  "too short v1 pat",
			token: "dop_v1_53918d3cd735062ca6ea791427900af10cf595f18dc6016c1cb0c3a11adcae",
		},
		{
			name:  "too short v1 oauth",
			token: "doo_v1_53918d3cd735062ca6ea791427900af10cf595f18dc6016c1cb0c3a11adc84",
		},
		{
			name:  "too long v1 pat",
			token: "dop_v1_53918d3cd735062ca6ea791427900af10cf595f18dc6016c1cb0c3a11adcae84sdsd",
		},
		{
			name:  "too long v1 oauth",
			token: "doo_v1_53918d3cd735062ca6ea791427900af10cf595f18dc6016c1cb0c3a11adcae84sd",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.NoError(t, tokenInputValidator(tt.token))
			} else {
				assert.Error(t, tokenInputValidator(tt.name))
			}
		})
	}
}

type testConfig map[string]any

type nopWriteCloser struct {
	io.Writer
}

var _ io.WriteCloser = (*nopWriteCloser)(nil)

func (d *nopWriteCloser) Close() error {
	return nil
}
