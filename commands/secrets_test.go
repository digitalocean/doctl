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
	"os"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testSecretWriteResult = do.SecretWriteResult{
		SecretWriteResult: &godo.SecretWriteResult{
			Name:    "prod-db-creds",
			Region:  "nyc3",
			Version: 1,
		},
	}

	testSecret = do.Secret{
		Secret: &godo.Secret{
			Name:    "prod-db-creds",
			Region:  "nyc3",
			Version: 1,
			Values: map[string]string{
				"password": "super-secret",
			},
			CreatedAt: "2026-06-08T12:00:00Z",
		},
	}

	testSecretsList = do.SecretsList{
		Secrets:            do.Secrets{testSecret},
		UnavailableRegions: []string{"sfo3"},
	}

	testSecretVersions = do.SecretVersions{
		{SecretVersion: &godo.SecretVersion{Version: 1, CreatedAt: "2026-06-08T12:00:00Z"}},
	}
)

func TestSecretsCommand(t *testing.T) {
	cmd := Secrets()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list", "list-versions", "restore", "update")
}

func TestParseSecretValues(t *testing.T) {
	t.Run("valid pairs", func(t *testing.T) {
		values, err := parseSecretValues([]string{"password=secret", "api_key=abc"})
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"password": "secret", "api_key": "abc"}, values)
	})

	t.Run("value contains equals", func(t *testing.T) {
		values, err := parseSecretValues([]string{"url=https://example.com?a=b"})
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com?a=b", values["url"])
	})

	t.Run("duplicate key", func(t *testing.T) {
		_, err := parseSecretValues([]string{"password=a", "password=b"})
		assert.Error(t, err)
	})

	t.Run("empty input", func(t *testing.T) {
		_, err := parseSecretValues(nil)
		assert.Error(t, err)
	})

	t.Run("invalid pair", func(t *testing.T) {
		_, err := parseSecretValues([]string{"invalid"})
		assert.Error(t, err)
	})
}

func TestParseSecretValuesFromFile(t *testing.T) {
	path := t.TempDir() + "/pw.txt"
	err := os.WriteFile(path, []byte("super-secret\n"), 0o600)
	assert.NoError(t, err)

	values, err := parseSecretValues([]string{"password=@" + path})
	assert.NoError(t, err)
	assert.Equal(t, "super-secret", values["password"])
}

func TestParseSecretValuesFromStdin(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdin = r

	_, err = w.WriteString("from-stdin")
	assert.NoError(t, err)
	assert.NoError(t, w.Close())

	values, err := parseSecretValues([]string{"password=-"})
	assert.NoError(t, err)
	assert.Equal(t, "from-stdin", values["password"])
}

func TestLoadSecretValuesFromEnvFile(t *testing.T) {
	path := t.TempDir() + "/.env"
	err := os.WriteFile(path, []byte("password=secret\napi_key=abc\n"), 0o600)
	assert.NoError(t, err)

	values, err := loadSecretValuesFromEnvFile(path)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"password": "secret", "api_key": "abc"}, values)
}

func TestMaskSecretValues(t *testing.T) {
	masked := maskSecretValues(testSecret)
	assert.Equal(t, secretMaskedValue, masked.Values["password"])
}

func TestReadSecretValuesInteractive(t *testing.T) {
	keys := []string{"password", "api_key", ""}
	values := []string{"secret", "abc"}
	keyIdx, valIdx := 0, 0

	oldPromptKey := promptSecretKeyFunc
	oldPromptValue := promptSecretValueFunc
	defer func() {
		promptSecretKeyFunc = oldPromptKey
		promptSecretValueFunc = oldPromptValue
	}()

	promptSecretKeyFunc = func() (string, error) {
		k := keys[keyIdx]
		keyIdx++
		return k, nil
	}
	promptSecretValueFunc = func(key string) (string, error) {
		assert.Contains(t, []string{"password", "api_key"}, key)
		v := values[valIdx]
		valIdx++
		return v, nil
	}

	out := &bytes.Buffer{}
	result, err := readSecretValuesInteractive(out, "prompt")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"password": "secret", "api_key": "abc"}, result)
	assert.Contains(t, out.String(), "prompt")
}

func TestReadSecretValuesInteractiveDuplicateKey(t *testing.T) {
	oldPromptKey := promptSecretKeyFunc
	oldPromptValue := promptSecretValueFunc
	defer func() {
		promptSecretKeyFunc = oldPromptKey
		promptSecretValueFunc = oldPromptValue
	}()

	keyCalls := 0
	promptSecretKeyFunc = func() (string, error) {
		keyCalls++
		if keyCalls == 1 {
			return "password", nil
		}
		return "password", nil
	}
	promptSecretValueFunc = func(key string) (string, error) {
		return "secret", nil
	}

	out := &bytes.Buffer{}
	_, err := readSecretValuesInteractive(out, "prompt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key")
}

func TestReadSecretValuesInteractiveEmpty(t *testing.T) {
	oldPromptKey := promptSecretKeyFunc
	defer func() {
		promptSecretKeyFunc = oldPromptKey
	}()

	promptSecretKeyFunc = func() (string, error) {
		return "", nil
	}

	out := &bytes.Buffer{}
	_, err := readSecretValuesInteractive(out, "prompt")
	assert.Error(t, err)
}

func TestRunCmdSecretsCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		request := &godo.SecretCreateRequest{
			Name:   "prod-db-creds",
			Region: "nyc3",
			Values: map[string]string{"password": "secret"},
		}
		tm.secrets.EXPECT().Create(request).Return(&testSecretWriteResult, nil)

		config.Args = []string{"prod-db-creds"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgSecretValue, []string{"password=secret"})

		err := RunCmdSecretsCreate(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.secrets.EXPECT().Get("prod-db-creds", "nyc3").Return(&testSecret, nil)

		config.Args = []string{"prod-db-creds"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		err := RunCmdSecretsGet(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsGetRaw(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.secrets.EXPECT().Get("prod-db-creds", "nyc3").Return(&testSecret, nil)

		config.Args = []string{"prod-db-creds"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgKey, "password")
		config.Doit.Set(config.NS, doctl.ArgSecretRaw, true)
		config.Out = &bytes.Buffer{}

		err := RunCmdSecretsGet(config)
		assert.NoError(t, err)
		assert.Equal(t, "super-secret", config.Out.(*bytes.Buffer).String())
	})
}

func TestRunCmdSecretsGetRawRequiresKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"prod-db-creds"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgSecretRaw, true)

		err := RunCmdSecretsGet(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), doctl.ArgKey)
	})
}

func TestRunCmdSecretsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.secrets.EXPECT().List().Return(&testSecretsList, nil)

		err := RunCmdSecretsList(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsListVersions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.secrets.EXPECT().ListVersions("prod-db-creds", "nyc3").Return(testSecretVersions, nil)

		config.Args = []string{"prod-db-creds"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		err := RunCmdSecretsListVersions(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		request := &godo.SecretUpdateRequest{
			Region:  "nyc3",
			Version: 1,
			Values:  map[string]string{"password": "new-secret"},
		}
		tm.secrets.EXPECT().Update("prod-db-creds", request).Return(&testSecretWriteResult, nil)

		config.Args = []string{"prod-db-creds"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgSecretVersion, 1)
		config.Doit.Set(config.NS, doctl.ArgSecretValue, []string{"password=new-secret"})

		err := RunCmdSecretsUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsUpdateUsesCurrentVersion(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		request := &godo.SecretUpdateRequest{
			Region:  "nyc3",
			Version: 1,
			Values:  map[string]string{"password": "new-secret"},
		}
		tm.secrets.EXPECT().Get("prod-db-creds", "nyc3").Return(&testSecret, nil)
		tm.secrets.EXPECT().Update("prod-db-creds", request).Return(&testSecretWriteResult, nil)

		config.Args = []string{"prod-db-creds"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgSecretValue, []string{"password=new-secret"})

		err := RunCmdSecretsUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.secrets.EXPECT().Delete("prod-db-creds", "nyc3").Return(nil)

		config.Args = []string{"prod-db-creds"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunCmdSecretsDelete(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsRestore(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.secrets.EXPECT().Restore("prod-db-creds", "nyc3").Return(nil)

		config.Args = []string{"prod-db-creds"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		err := RunCmdSecretsRestore(config)
		assert.NoError(t, err)
	})
}

func TestResolveSecretRegionUsesFlag(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		region, err := resolveSecretRegion(config)
		assert.NoError(t, err)
		assert.Equal(t, "nyc3", region)
	})
}

func TestResolveSecretRegionRequiresFlagWhenNonInteractive(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		defer withInteractive(false)()

		_, err := resolveSecretRegion(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), doctl.ArgRegionSlug)
	})
}

func TestResolveSecretNameUsesArg(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Args = []string{"prod-db-creds"}

		name, err := resolveSecretName(config)
		assert.NoError(t, err)
		assert.Equal(t, "prod-db-creds", name)
	})
}

func TestResolveSecretVersionUsesFlag(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSecretVersion, 3)

		version, err := resolveSecretVersion(config, "prod-db-creds", "nyc3")
		assert.NoError(t, err)
		assert.Equal(t, 3, version)
	})
}

func TestResolveSecretVersionRequiresFlagWhenNonInteractive(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		defer withInteractive(false)()

		tm.secrets.EXPECT().Get("prod-db-creds", "nyc3").Return(nil, assert.AnError)

		_, err := resolveSecretVersion(config, "prod-db-creds", "nyc3")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), doctl.ArgSecretVersion)
	})
}

func TestCollectSecretValuesRequiresFlagsWhenNonInteractive(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		defer withInteractive(false)()

		config.Args = []string{"prod-db-creds"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		_, err := collectSecretValues(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), doctl.ArgSecretValue)
		assert.Contains(t, err.Error(), doctl.ArgSecretFromEnvFile)
	})
}

func TestLoadSecretValuesFromFlagsEnvFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		path := t.TempDir() + "/.env"
		err := os.WriteFile(path, []byte("password=secret\n"), 0o600)
		assert.NoError(t, err)

		config.Doit.Set(config.NS, doctl.ArgSecretFromEnvFile, path)

		values, err := loadSecretValuesFromFlags(config)
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"password": "secret"}, values)
	})
}

func withInteractive(enabled bool) func() {
	prev := Interactive
	Interactive = enabled
	return func() {
		Interactive = prev
	}
}
