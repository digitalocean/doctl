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
	"errors"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// load parses contents through the same seam the auth commands use.
func loadStubbedConfig(t *testing.T, contents string) *configFile {
	t.Helper()

	withStubConfigFile(t, contents)

	cfg, err := loadConfigFile()
	require.NoError(t, err)

	return cfg
}

// A machine that has never run doctl has no config file, which is exactly when
// `auth init` needs to work.
func TestLoadConfigFileTreatsAMissingFileAsEmpty(t *testing.T) {
	reader := cfgFileReader
	t.Cleanup(func() { cfgFileReader = reader })

	cfgFileReader = func() ([]byte, error) { return nil, nil }

	cfg, err := loadConfigFile()

	require.NoError(t, err)
	assert.Empty(t, cfg.doc)
	assert.False(t, cfg.hasContext("work"))
	assert.True(t, cfg.hasContext(doctl.ArgDefaultContext), "the default context is always available")
}

func TestLoadConfigFileReportsUnreadableAndUnparsableFiles(t *testing.T) {
	reader := cfgFileReader
	t.Cleanup(func() { cfgFileReader = reader })

	cfgFileReader = func() ([]byte, error) { return nil, errors.New("permission denied") }
	_, err := loadConfigFile()
	assert.ErrorContains(t, err, "permission denied")

	cfgFileReader = func() ([]byte, error) { return []byte("\tnot: valid: yaml:"), nil }
	_, err = loadConfigFile()
	assert.ErrorContains(t, err, "Unable to parse configuration")
}

func TestConfigFileSetToken(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		context  string
		assert   func(t *testing.T, cfg *configFile)
	}{
		{
			name:    "the default context is stored at the top level",
			context: doctl.ArgDefaultContext,
			assert: func(t *testing.T, cfg *configFile) {
				assert.Equal(t, "new-token", cfg.doc[doctl.ArgAccessToken])
				assert.NotContains(t, cfg.doc, authContextsKey)
			},
		},
		{
			name:    "a named context is stored alongside the others",
			context: "work",
			contents: `auth-contexts:
  other: other-token
`,
			assert: func(t *testing.T, cfg *configFile) {
				assert.Equal(t, "new-token", cfg.contexts()["work"])
				assert.Equal(t, "other-token", cfg.contexts()["other"], "other contexts are untouched")
			},
		},
		{
			name:     "a context is created when the file has none",
			context:  "work",
			contents: "context: default\n",
			assert: func(t *testing.T, cfg *configFile) {
				assert.Equal(t, "new-token", cfg.contexts()["work"])
			},
		},
		{
			name:    "an existing token is replaced rather than duplicated",
			context: "work",
			contents: `auth-contexts:
  work: old-token
`,
			assert: func(t *testing.T, cfg *configFile) {
				assert.Equal(t, "new-token", cfg.contexts()["work"])
				assert.Len(t, cfg.contexts(), 1)
			},
		},
		{
			// doctl lowercases context names before storing them, but a file
			// written by an older version or by hand may not have.
			name:    "a name differing only in case updates the existing entry",
			context: "work",
			contents: `auth-contexts:
  Work: old-token
`,
			assert: func(t *testing.T, cfg *configFile) {
				assert.Equal(t, "new-token", cfg.contexts()["Work"])
				assert.Len(t, cfg.contexts(), 1, "a second entry must not appear")
			},
		},
		{
			// viper splits keys on "." and would turn this into a nested map,
			// which is what mangled contexts in issue #996.
			name:    "a context containing a period is stored verbatim",
			context: "test@example.com",
			assert: func(t *testing.T, cfg *configFile) {
				assert.Equal(t, "new-token", cfg.contexts()["test@example.com"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadStubbedConfig(t, tt.contents)

			cfg.setToken(tt.context, "new-token")

			tt.assert(t, cfg)
		})
	}
}

func TestConfigFileRemoveContext(t *testing.T) {
	t.Run("a named context is deleted", func(t *testing.T) {
		cfg := loadStubbedConfig(t, `access-token: default-token
auth-contexts:
  work: work-token
  other: other-token
`)

		require.NoError(t, cfg.removeContext("work"))

		assert.NotContains(t, cfg.contexts(), "work")
		assert.Equal(t, "other-token", cfg.contexts()["other"])
		assert.Equal(t, "default-token", cfg.doc[doctl.ArgAccessToken], "the default context is unaffected")
	})

	t.Run("the default context has its token blanked", func(t *testing.T) {
		cfg := loadStubbedConfig(t, "access-token: default-token\n")

		require.NoError(t, cfg.removeContext(doctl.ArgDefaultContext))

		assert.Equal(t, "", cfg.doc[doctl.ArgAccessToken])
	})

	t.Run("an unknown context is an error", func(t *testing.T) {
		cfg := loadStubbedConfig(t, "auth-contexts:\n  work: work-token\n")

		assert.ErrorContains(t, cfg.removeContext("nope"), "Context not found")
		assert.Equal(t, "work-token", cfg.contexts()["work"], "nothing is removed on failure")
	})
}

// The file belongs to the user. doctl owns three keys in it and must return
// everything else, including keys this version knows nothing about, unchanged.
func TestConfigFileWritePreservesUnrelatedSettings(t *testing.T) {
	stub := withStubConfigFile(t, `access-token: saved-token
api-url: https://api.example.com
auth-contexts:
  test@example.com: dotted-token
output: json
some-future-key:
  nested: value
`)

	cfg, err := loadConfigFile()
	require.NoError(t, err)

	cfg.setCurrentContext("test@example.com")
	require.NoError(t, cfg.write())

	reloaded, err := loadConfigFile()
	require.NoError(t, err)

	assert.Equal(t, "test@example.com", reloaded.doc[doctl.ArgContext])
	assert.Equal(t, "saved-token", reloaded.doc[doctl.ArgAccessToken])
	assert.Equal(t, "https://api.example.com", reloaded.doc["api-url"])
	assert.Equal(t, "json", reloaded.doc["output"])
	assert.Equal(t, map[any]any{"nested": "value"}, reloaded.doc["some-future-key"])
	assert.Equal(t, "dotted-token", reloaded.contexts()["test@example.com"],
		"a context name containing a period survives a round trip")

	assert.NotContains(t, stub.String(), "http-retry-max",
		"only the file's own keys are written back")
}
