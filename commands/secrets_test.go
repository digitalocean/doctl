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
	"fmt"
	"os"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testSecretName         = "my-secret"
	testSecretKey          = "key"
	testSecretKey2         = "other-key"
	testSecretKey3         = "third-key"
	testSecretVal          = "value"
	testSecretVal2         = "value-2"
	testSecretVal3         = "value-3"
	testSecretValNew       = "value-new"
	testSecretValFromFile  = "file-value"
	testSecretValFromStdin = "stdin-value"

	testSecretWriteResult = do.SecretWriteResult{
		SecretWriteResult: &godo.SecretWriteResult{
			Name:    testSecretName,
			Region:  "nyc3",
			Version: 2,
		},
	}

	testSecret = do.Secret{
		Secret: &godo.Secret{
			Name:    testSecretName,
			Region:  "nyc3",
			Version: 1,
			Values: map[string]string{
				testSecretKey:  testSecretVal,
				testSecretKey2: testSecretVal2,
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
	assertCommandNames(t, cmd, "create", "delete", "get", "list", "list-versions", "restore", "set", "unset", "update")
}

func TestParseSecretValues(t *testing.T) {
	t.Run("valid pairs", func(t *testing.T) {
		line1 := fmt.Sprintf("%s=%s", testSecretKey, testSecretVal)
		line2 := fmt.Sprintf("%s=%s", testSecretKey2, testSecretVal2)
		values, err := parseSecretValues([]string{line1, line2})
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{testSecretKey: testSecretVal, testSecretKey2: testSecretVal2}, values)
	})

	t.Run("value contains equals", func(t *testing.T) {
		values, err := parseSecretValues([]string{"url=https://example.com?a=b"})
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com?a=b", values["url"])
	})

	t.Run("duplicate key", func(t *testing.T) {
		_, err := parseSecretValues([]string{
			fmt.Sprintf("%s=a", testSecretKey),
			fmt.Sprintf("%s=b", testSecretKey),
		})
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
	path := t.TempDir() + "/value.txt"
	err := os.WriteFile(path, []byte(testSecretValFromFile+"\n"), 0o600)
	assert.NoError(t, err)

	values, err := parseSecretValues([]string{fmt.Sprintf("%s=@%s", testSecretKey, path)})
	assert.NoError(t, err)
	assert.Equal(t, testSecretValFromFile, values[testSecretKey])
}

func TestParseSecretValuesFromStdin(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdin = r

	_, err = w.WriteString(testSecretValFromStdin)
	assert.NoError(t, err)
	assert.NoError(t, w.Close())

	values, err := parseSecretValues([]string{fmt.Sprintf("%s=-", testSecretKey)})
	assert.NoError(t, err)
	assert.Equal(t, testSecretValFromStdin, values[testSecretKey])
}

func TestLoadSecretValuesFromEnvFile(t *testing.T) {
	path := t.TempDir() + "/.env"
	err := os.WriteFile(path, []byte(
		fmt.Sprintf("%s=%s\n%s=%s\n", testSecretKey, testSecretVal, testSecretKey2, testSecretVal2),
	), 0o600)
	assert.NoError(t, err)

	values, err := loadSecretValuesFromEnvFile(path)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{testSecretKey: testSecretVal, testSecretKey2: testSecretVal2}, values)
}

func TestMaskSecretValues(t *testing.T) {
	secret := cloneSecret(testSecret)
	masked := maskSecretValues(secret)
	assert.Equal(t, secretMaskedValue, masked.Values[testSecretKey])
}

func TestMergeSecretValues(t *testing.T) {
	current := map[string]string{testSecretKey: testSecretVal, testSecretKey2: testSecretVal2}
	updates := map[string]string{testSecretKey: testSecretValNew, testSecretKey3: testSecretVal3}

	merged := mergeSecretValues(current, updates)
	assert.Equal(t, map[string]string{
		testSecretKey:  testSecretValNew,
		testSecretKey2: testSecretVal2,
		testSecretKey3: testSecretVal3,
	}, merged)
}

func TestUnsetSecretKeys(t *testing.T) {
	current := map[string]string{testSecretKey: testSecretVal, testSecretKey2: testSecretVal2}

	values, err := unsetSecretKeys(current, []string{testSecretKey2})
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{testSecretKey: testSecretVal}, values)
}

func TestUnsetSecretKeysMissingKey(t *testing.T) {
	current := map[string]string{testSecretKey: testSecretVal}

	_, err := unsetSecretKeys(current, []string{testSecretKey2})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUnsetSecretKeysRemovesAll(t *testing.T) {
	current := map[string]string{testSecretKey: testSecretVal}

	_, err := unsetSecretKeys(current, []string{testSecretKey})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one key-value pair")
}

func TestRemovedSecretKeys(t *testing.T) {
	current := map[string]string{
		testSecretKey:  testSecretVal,
		testSecretKey2: testSecretVal2,
		testSecretKey3: testSecretVal3,
	}
	next := map[string]string{testSecretKey: testSecretVal}

	removed := removedSecretKeys(current, next)
	assert.Equal(t, []string{testSecretKey2, testSecretKey3}, removed)
}

func TestReadSecretValuesInteractive(t *testing.T) {
	keys := []string{testSecretKey, testSecretKey2, ""}
	values := []string{testSecretVal, testSecretVal2}
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
		assert.Contains(t, []string{testSecretKey, testSecretKey2}, key)
		v := values[valIdx]
		valIdx++
		return v, nil
	}

	out := &bytes.Buffer{}
	result, err := readSecretValuesInteractive(out, "prompt")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{testSecretKey: testSecretVal, testSecretKey2: testSecretVal2}, result)
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
		return testSecretKey, nil
	}
	promptSecretValueFunc = func(key string) (string, error) {
		return testSecretVal, nil
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
			Name:   testSecretName,
			Region: "nyc3",
			Values: map[string]string{testSecretKey: testSecretVal},
		}
		tm.secrets.EXPECT().Create(request).Return(&testSecretWriteResult, nil)

		config.Args = []string{testSecretName}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgSecretValue, []string{fmt.Sprintf("%s=%s", testSecretKey, testSecretVal)})

		err := RunCmdSecretsCreate(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsSet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		secret := cloneSecret(testSecret)
		request := &godo.SecretUpdateRequest{
			Region:  "nyc3",
			Version: 1,
			Values: map[string]string{
				testSecretKey:  testSecretValNew,
				testSecretKey2: testSecretVal2,
			},
		}
		tm.secrets.EXPECT().Get(testSecretName, "nyc3").Return(&secret, nil)
		tm.secrets.EXPECT().Update(testSecretName, request).Return(&testSecretWriteResult, nil)

		config.Args = []string{testSecretName}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgSecretValue, []string{fmt.Sprintf("%s=%s", testSecretKey, testSecretValNew)})

		err := RunCmdSecretsSet(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsUnset(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		secret := cloneSecret(testSecret)
		request := &godo.SecretUpdateRequest{
			Region:  "nyc3",
			Version: 1,
			Values:  map[string]string{testSecretKey: testSecretVal},
		}
		tm.secrets.EXPECT().Get(testSecretName, "nyc3").Return(&secret, nil)
		tm.secrets.EXPECT().Update(testSecretName, request).Return(&testSecretWriteResult, nil)

		config.Args = []string{testSecretName}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgKey, []string{testSecretKey2})

		err := RunCmdSecretsUnset(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		secret := cloneSecret(testSecret)
		tm.secrets.EXPECT().Get(testSecretName, "nyc3").Return(&secret, nil)

		config.Args = []string{testSecretName}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		err := RunCmdSecretsGet(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsGetRaw(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		secret := cloneSecret(testSecret)
		tm.secrets.EXPECT().Get(testSecretName, "nyc3").Return(&secret, nil)

		config.Args = []string{testSecretName}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgKey, testSecretKey)
		config.Doit.Set(config.NS, doctl.ArgSecretRaw, true)
		config.Out = &bytes.Buffer{}

		err := RunCmdSecretsGet(config)
		assert.NoError(t, err)
		assert.Equal(t, testSecretVal, config.Out.(*bytes.Buffer).String())
	})
}

func TestRunCmdSecretsGetRawRequiresKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testSecretName}
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
		tm.secrets.EXPECT().ListVersions(testSecretName, "nyc3").Return(testSecretVersions, nil)

		config.Args = []string{testSecretName}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		err := RunCmdSecretsListVersions(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsUpdateRequiresReplace(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Args = []string{testSecretName}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgSecretValue, []string{fmt.Sprintf("%s=%s", testSecretKey, testSecretValNew)})

		err := RunCmdSecretsUpdate(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), doctl.ArgSecretReplace)
	})
}

func TestRunCmdSecretsUpdateReplace(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		secret := cloneSecret(testSecret)
		request := &godo.SecretUpdateRequest{
			Region:  "nyc3",
			Version: 1,
			Values:  map[string]string{testSecretKey: testSecretValNew},
		}
		tm.secrets.EXPECT().Get(testSecretName, "nyc3").Return(&secret, nil)
		tm.secrets.EXPECT().Update(testSecretName, request).Return(&testSecretWriteResult, nil)

		config.Args = []string{testSecretName}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgSecretReplace, true)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		config.Doit.Set(config.NS, doctl.ArgSecretValue, []string{fmt.Sprintf("%s=%s", testSecretKey, testSecretValNew)})

		err := RunCmdSecretsUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.secrets.EXPECT().Delete(testSecretName, "nyc3").Return(nil)

		config.Args = []string{testSecretName}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunCmdSecretsDelete(config)
		assert.NoError(t, err)
	})
}

func TestRunCmdSecretsRestore(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.secrets.EXPECT().Restore(testSecretName, "nyc3").Return(nil)

		config.Args = []string{testSecretName}
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
		config.Args = []string{testSecretName}

		name, err := resolveSecretName(config)
		assert.NoError(t, err)
		assert.Equal(t, testSecretName, name)
	})
}

func TestCollectSecretValuesRequiresFlagsWhenNonInteractive(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		defer withInteractive(false)()

		config.Args = []string{testSecretName}
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
		err := os.WriteFile(path, []byte(fmt.Sprintf("%s=%s\n", testSecretKey, testSecretVal)), 0o600)
		assert.NoError(t, err)

		config.Doit.Set(config.NS, doctl.ArgSecretFromEnvFile, path)

		values, err := loadSecretValuesFromFlags(config)
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{testSecretKey: testSecretVal}, values)
	})
}

func withInteractive(enabled bool) func() {
	prev := Interactive
	Interactive = enabled
	return func() {
		Interactive = prev
	}
}
