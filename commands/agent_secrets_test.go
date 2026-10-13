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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestExpandNestedSecretValueURLFromEnv(t *testing.T) {
	raw := []byte(`name: syed-nested
agent: opencode
secrets:
  OPENAI_API_KEY:
    value: ${OPENAI_API_KEY}
    url: https://api.openai.com/v1
`)
	t.Setenv("OPENAI_API_KEY", "sk-from-env")

	all := manifestSecretValues(raw)
	assert.Equal(t, "${OPENAI_API_KEY}", all["OPENAI_API_KEY"])

	resolved := resolvedManifestSecretValues(raw)
	_, has := resolved["OPENAI_API_KEY"]
	assert.False(t, has, "placeholder must not overlay the real env value")

	lookup := envLookupWithOverlay(withServerProvidedOverlay(resolved, nil))
	out, err := expandManifestEnvCollect(raw, lookup)
	require.NoError(t, err)
	assert.Contains(t, string(out), "sk-from-env")
	assert.NotContains(t, string(out), "${OPENAI_API_KEY}")

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))
	secrets, ok := yamlMap(doc["secrets"])
	require.True(t, ok)
	slot, ok := yamlMap(secrets["OPENAI_API_KEY"])
	require.True(t, ok, "object form must be preserved")
	assert.Equal(t, "sk-from-env", slot["value"])
	assert.Equal(t, "https://api.openai.com/v1", slot["url"])
}

func TestRedactNestedSecretValueURLPreservesURL(t *testing.T) {
	in := []byte(`name: demo
secrets:
  OPENAI_API_KEY:
    value: sk-secret
    url: https://api.openai.com/v1
`)
	out := redactManifestSecrets(in)
	assert.NotContains(t, string(out), "sk-secret")
	assert.Contains(t, string(out), redactedSecretValue)
	assert.Contains(t, string(out), "https://api.openai.com/v1")

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))
	secrets, ok := yamlMap(doc["secrets"])
	require.True(t, ok)
	slot, ok := yamlMap(secrets["OPENAI_API_KEY"])
	require.True(t, ok)
	assert.Equal(t, redactedSecretValue, slot["value"])
	assert.Equal(t, "https://api.openai.com/v1", slot["url"])
}

func TestInjectSecretIntoNestedValueURL(t *testing.T) {
	in := []byte(`name: demo
agent: opencode
secrets:
  OPENAI_API_KEY:
    value: ${OPENAI_API_KEY}
    url: https://api.openai.com/v1
`)
	out, err := injectManifestSecrets(in, map[string]string{"OPENAI_API_KEY": "sk-from-flag"})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))
	secrets, ok := yamlMap(doc["secrets"])
	require.True(t, ok)
	slot, ok := yamlMap(secrets["OPENAI_API_KEY"])
	require.True(t, ok)
	assert.Equal(t, "sk-from-flag", slot["value"])
	assert.Equal(t, "https://api.openai.com/v1", slot["url"])
}
