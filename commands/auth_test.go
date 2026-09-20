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
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"errors"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	yaml "gopkg.in/yaml.v2"
)

func TestAuthCommand(t *testing.T) {
	cmd := Auth()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "init", "list", "remove", "switch", "token")
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

func TestAuthInitFlagOverridesNamedContextToken(t *testing.T) {
	cfw := cfgFileWriter
	origToken := Token
	// Simulate `doctl auth init --context team -t <flag-token>` against a
	// config that already holds a (stale) token for that context.
	Token = "flag-token"
	viper.Set("context", "team")
	defer func() {
		cfgFileWriter = cfw
		Token = origToken
		viper.Set("context", nil)
	}()

	retrieveUserTokenFunc := func() (string, error) {
		return "", errors.New("should not have prompted")
	}

	var out bytes.Buffer
	cfgFileWriter = func() (io.WriteCloser, error) { return &nopWriteCloser{Writer: io.Discard}, nil }

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = &out
		// The harness stubs the context accessors; stand in for a named
		// context that already holds a stale token and record what init saves.
		var saved string
		config.getContextAccessToken = func() string { return "stale-token" }
		config.setContextAccessToken = func(token string) { saved = token }
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.NoError(t, err)

		assert.Equal(t, "flag-token", saved,
			"the --access-token value should replace the stored token for the named context")
		assert.Contains(t, out.String(), "Using token from --access-token for context team")
	})
}

func TestAuthInitEnvTokenDefaultContext(t *testing.T) {
	cfw := cfgFileWriter
	// Mirror initConfig so that viper resolves DIGITALOCEAN_ACCESS_TOKEN.
	viper.SetEnvPrefix("DIGITALOCEAN")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	t.Setenv(accessTokenEnvVar, "env-token")
	viper.Set(doctl.ArgAccessToken, nil)
	viper.Set("context", doctl.ArgDefaultContext)
	defer func() {
		cfgFileWriter = cfw
		viper.Set("context", nil)
	}()

	retrieveUserTokenFunc := func() (string, error) {
		return "", errors.New("should not have prompted")
	}

	var out bytes.Buffer
	cfgFileWriter = func() (io.WriteCloser, error) { return &nopWriteCloser{Writer: io.Discard}, nil }

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = &out
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		var saved string
		config.setContextAccessToken = func(token string) { saved = token }

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.NoError(t, err)
		assert.Equal(t, "env-token", saved)
		assert.Contains(t, out.String(), "Using token from DIGITALOCEAN_ACCESS_TOKEN for context default")
	})
}

// unauthorizedErr mimics the error godo returns when the API answers 401.
func unauthorizedErr() error {
	return &godo.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusUnauthorized,
			Request:    &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/v1/oauth/token/info"}},
		},
		Message: "Unable to authenticate you",
	}
}

func TestAuthInitRepromptsWhenStoredTokenIsRejected(t *testing.T) {
	cfw := cfgFileWriter
	viper.Set(doctl.ArgAccessToken, "expired-token")
	defer func() {
		cfgFileWriter = cfw
		viper.Set(doctl.ArgAccessToken, nil)
	}()

	prompted := 0
	retrieveUserTokenFunc := func() (string, error) {
		prompted++
		return "fresh-token", nil
	}

	cfgFileWriter = func() (io.WriteCloser, error) { return &nopWriteCloser{Writer: io.Discard}, nil }

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		gomock.InOrder(
			tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(nil, unauthorizedErr()),
			tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil),
		)

		var saved string
		config.setContextAccessToken = func(token string) { saved = token }

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.NoError(t, err)
		assert.Equal(t, 1, prompted)
		assert.Equal(t, "fresh-token", saved)
	})
}

func TestAuthInitRejectedStoredTokenNonInteractive(t *testing.T) {
	cfw := cfgFileWriter
	viper.Set(doctl.ArgAccessToken, "expired-token")
	defer func() {
		cfgFileWriter = cfw
		viper.Set(doctl.ArgAccessToken, nil)
	}()

	retrieveUserTokenFunc := func() (string, error) {
		return "", ErrUnknownTerminal
	}

	cfgFileWriter = func() (io.WriteCloser, error) {
		return nil, errors.New("config must not be written when validation fails")
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(nil, unauthorizedErr())

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "401 Unable to authenticate you")
		assert.Contains(t, err.Error(), "--access-token")
	})
}

func TestAuthInitStoredTokenNetworkErrorDoesNotPrompt(t *testing.T) {
	cfw := cfgFileWriter
	viper.Set(doctl.ArgAccessToken, "stored-token")
	defer func() {
		cfgFileWriter = cfw
		viper.Set(doctl.ArgAccessToken, nil)
	}()

	retrieveUserTokenFunc := func() (string, error) {
		return "", errors.New("should not have prompted")
	}

	cfgFileWriter = func() (io.WriteCloser, error) {
		return nil, errors.New("config must not be written when validation fails")
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(nil, errors.New("dial tcp: connection refused"))

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.NotContains(t, err.Error(), "should not have prompted")
		assert.NotContains(t, err.Error(), "--access-token")
	})
}

func TestAuthInitRejectedFlagTokenDoesNotPrompt(t *testing.T) {
	cfw := cfgFileWriter
	origToken := Token
	Token = "bad-flag-token"
	defer func() {
		cfgFileWriter = cfw
		Token = origToken
		viper.Set(doctl.ArgAccessToken, nil)
	}()

	retrieveUserTokenFunc := func() (string, error) {
		return "", errors.New("should not have prompted")
	}

	cfgFileWriter = func() (io.WriteCloser, error) {
		return nil, errors.New("config must not be written when validation fails")
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(nil, errors.New("401 Unable to authenticate you"))

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "should not have prompted")
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
		Name         string
		Out          *bytes.Buffer
		Context      string
		Contexts     map[string]any
		Expected     string
		ExpectedJSON string
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

func Test_displayAuthContextsJSON(t *testing.T) {
	testCases := []struct {
		Name         string
		Out          *bytes.Buffer
		Context      string
		Contexts     map[string]any
		ExpectedJSON string
	}{
		{
			Name:    "default context only",
			Out:     &bytes.Buffer{},
			Context: doctl.ArgDefaultContext,
			Contexts: map[string]any{
				doctl.ArgDefaultContext: true,
			},
			ExpectedJSON: "[\n  {\n    \"name\": \"default\",\n    \"current\": true\n  }\n]\n",
		},
		{
			Name:    "default context and additional context",
			Out:     &bytes.Buffer{},
			Context: doctl.ArgDefaultContext,
			Contexts: map[string]any{
				doctl.ArgDefaultContext: true,
				"test":                  true,
			},
			ExpectedJSON: "[\n  {\n    \"name\": \"default\",\n    \"current\": true\n  },\n  {\n    \"name\": \"test\",\n    \"current\": false\n  }\n]\n",
		},
		{
			Name:    "default context and additional context set to additional context",
			Out:     &bytes.Buffer{},
			Context: "test",
			Contexts: map[string]any{
				doctl.ArgDefaultContext: true,
				"test":                  true,
			},
			ExpectedJSON: "[\n  {\n    \"name\": \"default\",\n    \"current\": false\n  },\n  {\n    \"name\": \"test\",\n    \"current\": true\n  }\n]\n",
		},
		{
			Name:    "unset context",
			Out:     &bytes.Buffer{},
			Context: "missing",
			Contexts: map[string]any{
				doctl.ArgDefaultContext: true,
				"test":                  true,
			},
			ExpectedJSON: "[\n  {\n    \"name\": \"default\",\n    \"current\": false\n  },\n  {\n    \"name\": \"test\",\n    \"current\": false\n  }\n]\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			displayAuthContextsJSON(tc.Out, tc.Context, tc.Contexts)
			assert.Equal(t, tc.ExpectedJSON, tc.Out.String())
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
