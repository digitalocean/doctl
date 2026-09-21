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
	"bytes"
	"io"
	"testing"

	"errors"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	yaml "gopkg.in/yaml.v2"
)

func TestAuthCommand(t *testing.T) {
	cmd := Auth()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "init", "list", "remove", "switch", "token")
}

func TestAuthInit(t *testing.T) {
	viper.Set(doctl.ArgAccessToken, nil)

	retrieveUserTokenFunc := func() (string, error) {
		return "valid-token", nil
	}

	withStubConfigFile(t, "")

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.NoError(t, err)
	})
}

// init saves the token it just validated, and nothing else. doctl resolves its
// settings from defaults, flags, and the environment as well as the file, and
// writing that merged view back is what buried an environment token and every
// bound flag in the user's config.
func TestAuthInitWritesOnlyAuthSettings(t *testing.T) {
	defer withStubConfig(t, map[string]any{"context": doctl.ArgDefaultContext})()
	defer withContext(t, "")()

	retrieveUserTokenFunc := func() (string, error) {
		return "valid-token", nil
	}

	cfg := withStubConfigFile(t, "unrelated: keep-me\n")

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		require.NoError(t, err)

		written := cfg.settings(t)

		assert.Equal(t, "valid-token", written[doctl.ArgAccessToken])
		assert.Equal(t, "keep-me", written["unrelated"], "a key doctl did not touch must survive")

		for _, key := range []string{"config", "dev", "http-retry-max", "output", "interactive"} {
			assert.NotContains(t, written, key,
				"%q is a resolved setting, not something the user asked to save", key)
		}
	})
}

func TestAuthInitWithProvidedToken(t *testing.T) {
	defer withStubConfig(t, map[string]any{
		"context":            doctl.ArgDefaultContext,
		doctl.ArgAccessToken: "valid-token",
	})()
	defer withContext(t, "")()

	retrieveUserTokenFunc := func() (string, error) {
		return "", errors.New("should not have called this")
	}

	cfg := withStubConfigFile(t, "")

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		require.NoError(t, err)

		// A token supplied by --access-token or the environment is still saved
		// when init is what the user ran: saving a token is the whole job of
		// the command. It is the other auth commands that must leave it alone.
		assert.Equal(t, "valid-token", cfg.settings(t)[doctl.ArgAccessToken])
	})
}

// Switching contexts changes which context is current. It must not also write
// out the token doctl resolved for this invocation, which is how a token
// exported for one CI job became a durable plaintext file.
func TestAuthSwitchPersistsOnlyTheContext(t *testing.T) {
	cfg := withStubConfigFile(t, `access-token: saved-token
auth-contexts:
  work: work-token
context: default
`)

	// The token doctl resolved for this invocation, as --access-token or
	// DIGITALOCEAN_ACCESS_TOKEN would supply it.
	defer withStubConfig(t, map[string]any{
		"context":            doctl.ArgDefaultContext,
		doctl.ArgAccessToken: "ambient-token",
	})()
	defer withContext(t, "work")()

	require.NoError(t, RunAuthSwitch(&CmdConfig{Out: io.Discard}))

	written := cfg.settings(t)

	assert.Equal(t, "work", written[doctl.ArgContext])
	assert.Equal(t, "saved-token", written[doctl.ArgAccessToken], "the stored token must be left as it was")
	assert.NotContains(t, cfg.String(), "ambient-token")
}

func TestAuthForcesLowercase(t *testing.T) {
	viper.Set(doctl.ArgAccessToken, "valid-token")
	defer viper.Set(doctl.ArgAccessToken, nil)

	retrieveUserTokenFunc := func() (string, error) {
		return "", errors.New("should not have called this")
	}

	withStubConfigFile(t, "")

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		viper.Set("context", "TestCapitalCase")

		err := RunAuthInit(retrieveUserTokenFunc)(config)
		assert.NoError(t, err)

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

// stubConfigFile stands in for the config file on disk. Writes are visible to
// later reads, as they would be through a real file, so a test can run one
// auth command and then assert what the next one sees.
type stubConfigFile struct {
	contents []byte
}

var _ io.WriteCloser = (*stubConfigFile)(nil)

func (s *stubConfigFile) Write(p []byte) (int, error) {
	s.contents = append(s.contents, p...)
	return len(p), nil
}

func (s *stubConfigFile) Close() error { return nil }

func (s *stubConfigFile) String() string { return string(s.contents) }

// settings decodes what has been written so far.
func (s *stubConfigFile) settings(t *testing.T) testConfig {
	t.Helper()

	var cfg testConfig
	require.NoError(t, yaml.Unmarshal(s.contents, &cfg))

	return cfg
}

// withStubConfigFile points config reads and writes at memory, so tests
// neither depend on nor overwrite the config of whoever is running them.
func withStubConfigFile(t *testing.T, contents string) *stubConfigFile {
	t.Helper()

	stub := &stubConfigFile{contents: []byte(contents)}

	reader, writer := cfgFileReader, cfgFileWriter
	t.Cleanup(func() {
		cfgFileReader, cfgFileWriter = reader, writer
	})

	cfgFileReader = func() ([]byte, error) { return stub.contents, nil }
	cfgFileWriter = func() (io.WriteCloser, error) {
		// A real write truncates.
		stub.contents = nil
		return stub, nil
	}

	return stub
}
