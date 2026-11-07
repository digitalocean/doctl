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
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	yaml "gopkg.in/yaml.v2"
)

func parseManifestDoc(t *testing.T, src string) map[string]any {
	t.Helper()
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(src), &doc))
	return doc
}

func TestEditDistance(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"egress", "egress", 0},
		{"egresss", "egress", 1},
		{"permisions", "permissions", 1},
		{"modl", "model", 1},
		{"modle", "model", 2},
		{"kitten", "sitting", 3},
		{"alow", "allow", 1},
	} {
		assert.Equal(t, tc.want, editDistance(tc.a, tc.b), "%q vs %q", tc.a, tc.b)
		assert.Equal(t, tc.want, editDistance(tc.b, tc.a), "%q vs %q (swapped)", tc.b, tc.a)
	}
}

func TestNearMissKey(t *testing.T) {
	root := manifestRootKeys.names()
	for _, tc := range []struct {
		name    string
		unknown string
		want    string // "" means no candidate
	}{
		{"egresss", "egresss", "egress"},
		{"permisions", "permisions", "permissions"},
		// `model` is listed before `mode`; both are distance 1 from `modl`.
		{"tie goes to first listed", "modl", "model"},
		{"smallest distance beats listing order", "mod", "mode"},
		{"case-insensitive on the unknown key", "EGRESSS", "egress"},
		{"far key has no candidate", "bogus_field", ""},
		{"distance 3 is too far", "xyz", ""},
		// "ab" is within distance 2 of nothing shorter than itself: a short
		// candidate must be longer than the distance.
		{"short unknown key", "id", ""},
		{"legacy camelCase within 2 of snake_case is a near-miss", "keepWarm", "keep_warm"},
		{"legacy camelCase too far from snake_case", "egressHosts", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := nearMissKey(tc.unknown, root)
			assert.Equal(t, tc.want != "", ok)
			assert.Equal(t, tc.want, got)
		})
	}

	// The distance must also be smaller than the candidate's length.
	got, ok := nearMissKey("en", []string{"env"})
	assert.True(t, ok)
	assert.Equal(t, "env", got)
	_, ok = nearMissKey("x", []string{"ab"})
	assert.False(t, ok, "distance 2 is not less than len(\"ab\") == 2")
}

func TestUnknownManifestKeys(t *testing.T) {
	for _, tc := range []struct {
		name     string
		manifest string
		wantErrs []string // substrings, one per expected error, in order
		wantWarn []string
	}{
		{
			name:     "CUJ1 root near-miss",
			manifest: "name: demo\nagent: opencode\negresss: [api.github.com]\n",
			wantErrs: []string{`egresss: unknown field "egresss" (did you mean "egress"? if this is a newer field, upgrade doctl)`},
		},
		{
			name:     "CUJ4 nested near-miss uses the full path",
			manifest: "agent: opencode\npermissions:\n  network:\n    alow: [api.github.com]\n",
			wantErrs: []string{`permissions.network.alow: unknown field "alow" (did you mean "allow"?`},
		},
		{
			name:     "CUJ5 far key warns",
			manifest: "agent: opencode\nbogus_field: 1\n",
			wantWarn: []string{`bogus_field: unknown field "bogus_field"; the server rejects unknown fields when the session is created (if this is a newer field, upgrade doctl)`},
		},
		{
			name:     "key differing only in case warns",
			manifest: "agent: opencode\nEgress: [a.com]\n",
			wantWarn: []string{`Egress: unknown field "Egress" differs from "egress" only in letter case`},
		},
		{
			name:     "uppercase misspelling is still a near-miss",
			manifest: "agent: opencode\nEGRESSS: [a.com]\n",
			wantErrs: []string{`did you mean "egress"?`},
		},
		{
			name:     "array item path",
			manifest: "agent: opencode\nmounts:\n  - type: s3fuse\n    bukket: b\n",
			wantErrs: []string{`mounts[0].bukket: unknown field "bukket" (did you mean "bucket"?`},
		},
		{
			name:     "second array item",
			manifest: "agent: opencode\nskills:\n  - name: a\n  - name: b\n    descripton: x\n",
			wantErrs: []string{`skills[1].descripton: unknown field "descripton" (did you mean "description"?`},
		},
		{
			name:     "permission rule item",
			manifest: "agent: opencode\npermissions:\n  rules:\n    - tool: bash\n      acton: allow\n",
			wantErrs: []string{`permissions.rules[0].acton: unknown field "acton" (did you mean "action"?`},
		},
		{
			name:     "secret long form",
			manifest: "agent: opencode\nsecrets:\n  KEY:\n    sorce: tenantSecret\n",
			wantErrs: []string{`secrets.KEY.sorce: unknown field "sorce" (did you mean "source"?`},
		},
		{
			name:     "model object",
			manifest: "agent: opencode\nmodel:\n  defualt: x\n",
			wantErrs: []string{`model.defualt: unknown field "defualt" (did you mean "default"?`},
		},
		{
			name:     "nested object two levels deep",
			manifest: "agent: opencode\nserving:\n  scale_to_zero:\n    idel: 5m\n",
			wantErrs: []string{`serving.scale_to_zero.idel: unknown field "idel" (did you mean "idle"?`},
		},
		{
			name:     "catalog tool selection object",
			manifest: "agent: opencode\ntools:\n  - do.actions:\n      tols: [a]\n",
			wantErrs: []string{`tools[0].do.actions.tols: unknown field "tols" (did you mean "tools"?`},
		},
		{
			name:     "inline tool server and auth",
			manifest: "agent: opencode\ntools:\n  - nam: x\n    url: https://m.example.com\n    auth:\n      secrt: S\n",
			wantErrs: []string{
				`tools[0].auth.secrt: unknown field "secrt" (did you mean "secret"?`,
				`tools[0].nam: unknown field "nam" (did you mean "name"?`,
			},
		},
		{
			name:     "string tool entries and catalog keys are not checked",
			manifest: "agent: opencode\ntools:\n  - do.actions\n  - do.whatever.you.like: [a, b]\n",
		},
		{
			name:     "errors and warnings together, sorted by key",
			manifest: "agent: opencode\nzzz_unknown: 1\negresss: [a.com]\naaa_unknown: 2\n",
			wantErrs: []string{`egresss:`},
			wantWarn: []string{`aaa_unknown:`, `zzz_unknown:`},
		},
		{
			name:     "non-map where an object is expected is left alone",
			manifest: "agent: opencode\nmodel: claude\nmemory: default\npermissions: allow\nmounts: nope\nserving: 3\n",
		},
		{
			name: "CUJ6 open maps are not checked",
			manifest: `agent: opencode
labels:
  anything_goes: x
  egresss: x
env:
  ANY_KEY: x
  egresss: x
config:
  egresss: 1
  nested:
    permisions: 1
secrets:
  ANY_SLOT: oauth/github
  egresss: oauth/github
permissions:
  rules:
    - tool: bash
      action: allow
      match:
        egresss: x
skills:
  - name: a
    description: Use when
    instructions: do
    metadata:
      egresss: x
tools:
  - do.actions:
      tools: [t]
`,
		},
		{
			name:     "non-string key does not disable checking of its siblings (permissions.network)",
			manifest: "permissions:\n  network:\n    alow: [a]\n    5: x\n",
			wantErrs: []string{`permissions.network.alow: unknown field "alow" (did you mean "allow"?`},
			wantWarn: []string{`permissions.network.5: unknown field "5"`},
		},
		{
			name:     "non-string key does not disable checking of its siblings (serving.scale_to_zero)",
			manifest: "serving:\n  scale_to_zero:\n    idel: 1\n    2: 3\n",
			wantErrs: []string{`serving.scale_to_zero.idel: unknown field "idel" (did you mean "idle"?`},
			wantWarn: []string{`serving.scale_to_zero.2: unknown field "2"`},
		},
		{
			name:     "non-string key does not disable checking of its siblings (permissions)",
			manifest: "permissions:\n  1: x\n  alow: y\n",
			// `alow` is within 2 of no permissions key ("allow" lives in
			// network), so both keys warn; the point is that `alow` is seen.
			wantWarn: []string{`permissions.1: unknown field "1"`, `permissions.alow: unknown field "alow"`},
		},
		{
			name:     "unexpanded ${VAR} keys are not reported",
			manifest: "agent: opencode\n${EXTRA_KEY}: 1\npermissions:\n  ${NET}: x\n",
		},
		{
			name:     "catalog entry mixed with other keys checks the others as inline-server keys",
			manifest: "agent: opencode\ntools:\n  - do.actions: [a]\n    nmae: x\n    bogus_thing: y\n",
			wantErrs: []string{`tools[0].nmae: unknown field "nmae" (did you mean "name"?`},
			wantWarn: []string{`tools[0].bogus_thing: unknown field "bogus_thing"`},
		},
		{
			name:     "catalog entry mixed with known inline keys is quiet",
			manifest: "agent: opencode\ntools:\n  - do.actions: [a]\n    name: x\n",
		},
		{
			name:     "empty key uses a placeholder",
			manifest: "agent: opencode\n\"\": 1\npermissions:\n  \"\": 2\n",
			wantWarn: []string{`(empty key): unknown field with an empty name`, `permissions.(empty key): unknown field with an empty name`},
		},
		{
			// Pins current behavior: a root key that only exists to hold a YAML
			// anchor is still a root key, and `base` is within 2 of `name`. The
			// server's strict decode rejects it too, so reporting it is correct.
			name:     "anchor-holder root key is a near-miss of name",
			manifest: "agent: opencode\nbase: &b {x: 1}\nenv: *b\n",
			wantErrs: []string{`base: unknown field "base" (did you mean "name"?`},
		},
		{
			name:     "CUJ7 egress object is skipped by this check",
			manifest: "agent: opencode\negress:\n  allow_hosts: [a.com]\n  bogus: 1\n  allow_hostz: [b.com]\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errs, warns := unknownManifestKeys(parseManifestDoc(t, tc.manifest))
			require.Len(t, errs, len(tc.wantErrs), "errors: %v", errs)
			for i, want := range tc.wantErrs {
				assert.Contains(t, errs[i], want)
			}
			require.Len(t, warns, len(tc.wantWarn), "warnings: %v", warns)
			for i, want := range tc.wantWarn {
				assert.Contains(t, warns[i], want)
			}
		})
	}
}

// config create promotes warnings containing these substrings to errors, so a
// far-key warning must never contain them.
func TestUnknownManifestKeys_MessagesAvoidPromotionSubstrings(t *testing.T) {
	_, warns := unknownManifestKeys(parseManifestDoc(t, "agent: opencode\nbogus_field: 1\nEgress: x\n"))
	require.Len(t, warns, 2)
	for _, w := range warns {
		for _, banned := range []string{"matches a declared", "looks like a credential", "name suggests a credential"} {
			assert.NotContains(t, w, banned)
		}
	}

	// And end to end: the durable reporter keeps them as warnings.
	v := validateAgentManifest([]byte("agent: opencode\nbogus_field: 1\n"))
	require.Len(t, v.Warnings, 1)
	require.NoError(t, reportDurableAgentManifestValidation(v))
}

func TestValidateAgentManifest_UnknownKeys(t *testing.T) {
	t.Run("near miss is an error", func(t *testing.T) {
		v := validateAgentManifest([]byte("name: demo\nagent: opencode\negresss: [api.github.com]\n"))
		require.False(t, v.ok())
		require.Len(t, v.Errors, 1)
		assert.Contains(t, v.Errors[0], `"egresss"`)
		assert.Contains(t, v.Errors[0], `"egress"`)
	})

	t.Run("far key is a warning", func(t *testing.T) {
		v := validateAgentManifest([]byte("agent: opencode\nbogus_field: 1\n"))
		assert.True(t, v.ok())
		require.Len(t, v.Warnings, 1)
		assert.Contains(t, v.Warnings[0], "bogus_field")
	})

	t.Run("egress keeps its own error exactly once", func(t *testing.T) {
		v := validateAgentManifest([]byte("agent: opencode\negress:\n  allow_hosts: [a.com]\n  bogus: 1\n"))
		var hits []string
		for _, e := range v.Errors {
			if strings.Contains(e, "egress.bogus") {
				hits = append(hits, e)
			}
		}
		require.Len(t, hits, 1, "errors: %v", v.Errors)
		assert.Equal(t, "egress.bogus: unknown field", hits[0])
		assert.Empty(t, v.Warnings)
	})

	t.Run("legacy envelope is untouched", func(t *testing.T) {
		v := validateAgentManifest([]byte(`apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
metadata:
  name: demo
spec:
  runtime:
    adapter: opencode
  egresss: [a.com]
  bogus_field: 1
`))
		assert.True(t, v.ok(), "errors: %v", v.Errors)
		for _, w := range v.Warnings {
			assert.NotContains(t, w, "unknown field")
		}
		warns, err := earlyUnknownKeyCheck([]byte("apiVersion: agents.digitalocean.com/v1alpha1\nkind: Agent\negresss: 1\n"), true)
		assert.NoError(t, err)
		assert.Nil(t, warns)
	})

	t.Run("skip variant reports no unknown keys", func(t *testing.T) {
		v := validateAgentManifestSkippingUnknownKeys([]byte("agent: opencode\negresss: [a.com]\nbogus_field: 1\n"))
		assert.True(t, v.ok(), "errors: %v", v.Errors)
		assert.Empty(t, v.Warnings)
	})

	t.Run("valid fixtures have no errors or warnings", func(t *testing.T) {
		for _, name := range []string{"valid_full.yaml", "valid_langgraph.yaml", "valid_agent_alias.yaml"} {
			raw, err := os.ReadFile(filepath.Join("testdata", name))
			require.NoError(t, err)
			v := validateAgentManifest(raw)
			assert.Empty(t, v.Errors, name)
			assert.Empty(t, v.Warnings, name)
			warns, err := earlyUnknownKeyCheck(raw, true)
			assert.NoError(t, err, name)
			assert.Empty(t, warns, name)
		}
	})

	t.Run("crewai fixture has no unknown-field reports", func(t *testing.T) {
		raw, err := os.ReadFile(filepath.Join("testdata", "valid_crewai.yaml"))
		require.NoError(t, err)
		warns, err := earlyUnknownKeyCheck(raw, true)
		assert.NoError(t, err)
		assert.Empty(t, warns)
	})
}

// --- harness / agent spellings -------------------------------------------------

func TestUnknownManifestKeys_HarnessSpellings(t *testing.T) {
	t.Run("harness is a known key and egresss is the only report (the ticket's repro)", func(t *testing.T) {
		v := validateAgentManifest([]byte("name: demo\nharness: opencode\negresss: [api.github.com]\n"))
		require.False(t, v.ok())
		require.Len(t, v.Errors, 1, "errors: %v", v.Errors)
		assert.Contains(t, v.Errors[0], `egresss: unknown field "egresss" (did you mean "egress"?`)
		assert.Empty(t, v.Warnings)
		for _, m := range append(append([]string{}, v.Errors...), v.Warnings...) {
			assert.NotContains(t, m, `"harness"`)
		}
	})

	t.Run("harness: opencode alone is clean even with --strict", func(t *testing.T) {
		warns, err := earlyUnknownKeyCheck([]byte("name: demo\nharness: opencode\n"), true)
		assert.NoError(t, err)
		assert.Empty(t, warns)
		assert.True(t, validateAgentManifest([]byte("name: demo\nharness: opencode\n")).ok())
	})

	t.Run("bare sandbox manifest with size and egress has no unknown-key report", func(t *testing.T) {
		const bare = "name: box\nharness: none\nsize: s-2vcpu-4gb\negress:\n  allow_hosts: [api.github.com]\n"
		errs, warns := unknownManifestKeys(parseManifestDoc(t, bare))
		assert.Empty(t, errs)
		assert.Empty(t, warns)
		v := validateAgentManifest([]byte(bare))
		assert.True(t, v.ok(), "errors: %v", v.Errors)
		assert.Empty(t, v.Warnings)
	})

	t.Run("both harness and agent still get the upstream error, and no unknown-field noise", func(t *testing.T) {
		v := validateAgentManifest([]byte("name: demo\nharness: opencode\nagent: opencode\n"))
		require.False(t, v.ok())
		require.Len(t, v.Errors, 1, "errors: %v", v.Errors)
		assert.Contains(t, v.Errors[0], "not both")
		assert.Empty(t, v.Warnings)
	})

	t.Run("crewai and langgraph blocks are checked", func(t *testing.T) {
		errs, warns := unknownManifestKeys(parseManifestDoc(t,
			"harness: crewai\ncrewai:\n  framework:\n    repoo: x\n  inputs:\n    prompt_kee: t\n    schema:\n      topic:\n        typ: string\nlanggraph:\n  framework:\n    subdr: x\n"))
		assert.Len(t, errs, 4, "errors: %v warnings: %v", errs, warns)
		joined := strings.Join(errs, "\n")
		for _, want := range []string{
			`crewai.framework.repoo: unknown field "repoo" (did you mean "repo"?`,
			`crewai.inputs.prompt_kee: unknown field "prompt_kee" (did you mean "prompt_key"?`,
			`crewai.inputs.schema.topic.typ: unknown field "typ" (did you mean "type"?`,
			`langgraph.framework.subdr: unknown field "subdr" (did you mean "subdir"?`,
		} {
			assert.Contains(t, joined, want)
		}
		// Free-form maps inside them are open.
		errs, warns = unknownManifestKeys(parseManifestDoc(t,
			"harness: crewai\ncrewai:\n  inputs:\n    defaults:\n      anything_goes: 1\n    schema:\n      any_input_name:\n        type: string\n"))
		assert.Empty(t, errs)
		assert.Empty(t, warns)
	})
}

// The valid_*.yaml fixtures are only useful regression fixtures while, between
// them, they use every key the tree knows; this fails when a key is added to
// the tree but to none of the files. (crewai, langgraph and the agent alias
// cannot share a manifest with harness: custom, hence several fixtures.)
func TestValidFixturesUseEveryKnownKey(t *testing.T) {
	fixtures := []string{"valid_full.yaml", "valid_crewai.yaml", "valid_langgraph.yaml", "valid_agent_alias.yaml"}

	used := map[string]map[string]bool{}
	var walk func(m map[string]any, node *keyNode, path string)
	mark := func(path, key string) {
		if used[path] == nil {
			used[path] = map[string]bool{}
		}
		used[path][key] = true
	}
	walk = func(m map[string]any, node *keyNode, path string) {
		for k, v := range m {
			def, ok := node.lookup(k)
			if !ok {
				continue
			}
			mark(path, k)
			switch def.kind {
			case keyObject:
				if cm, ok := yamlMap(v); ok {
					walk(cm, def.child, flatKeyPath(path, k))
				}
			case keyObjectList:
				list, _ := yamlList(v)
				for _, item := range list {
					if cm, ok := yamlMap(item); ok {
						walk(cm, def.child, flatKeyPath(path, k)+"[]")
					}
				}
			case keyMapOfObjects:
				slots, _ := yamlMap(v)
				for _, sv := range slots {
					if cm, ok := yamlMap(sv); ok {
						walk(cm, def.child, flatKeyPath(path, k)+".*")
					}
				}
			case keyTools:
				list, _ := yamlList(v)
				for _, item := range list {
					cm, ok := yamlMap(item)
					if !ok {
						continue
					}
					isCatalog := false
					for ck, cv := range cm {
						if strings.HasPrefix(ck, "do.") {
							isCatalog = true
							if vm, ok := yamlMap(cv); ok {
								walk(vm, manifestToolCatalogKeys, "tools[].{catalog}")
							}
						}
					}
					if !isCatalog {
						walk(cm, manifestToolInlineKeys, "tools[]")
					}
				}
			}
		}
	}
	for _, name := range fixtures {
		raw, err := os.ReadFile(filepath.Join("testdata", name))
		require.NoError(t, err)
		walk(parseManifestDoc(t, string(raw)), manifestRootKeys, "")
	}

	want := flattenKeyTree()
	for path, keys := range want {
		for _, k := range keys {
			if path == "" && k == "egress" {
				continue // present, but its inner keys belong to validateEgressObject
			}
			assert.True(t, used[path][k], "no valid_*.yaml fixture uses %q at %q", k, path)
		}
	}
}

func TestEarlyUnknownKeyCheck(t *testing.T) {
	t.Run("near miss errors in the create-path shape", func(t *testing.T) {
		warns, err := earlyUnknownKeyCheck([]byte("agent: opencode\negresss: [a.com]\n"), false)
		require.Error(t, err)
		assert.Empty(t, warns)
		assert.True(t, strings.HasPrefix(err.Error(), "manifest validation failed:\n  - egresss:"), err.Error())
	})

	t.Run("far key warns without strict", func(t *testing.T) {
		warns, err := earlyUnknownKeyCheck([]byte("agent: opencode\nbogus_field: 1\n"), false)
		require.NoError(t, err)
		require.Len(t, warns, 1)
		assert.Contains(t, warns[0], "bogus_field")
	})

	t.Run("far key fails with strict", func(t *testing.T) {
		warns, err := earlyUnknownKeyCheck([]byte("agent: opencode\nbogus_field: 1\n"), true)
		require.Error(t, err)
		assert.Empty(t, warns)
		assert.Contains(t, err.Error(), "bogus_field")
	})

	t.Run("unparseable or empty input is left to the validator", func(t *testing.T) {
		for _, raw := range []string{"", "{{{ not yaml", "- a\n- b\n"} {
			warns, err := earlyUnknownKeyCheck([]byte(raw), true)
			assert.NoError(t, err, raw)
			assert.Nil(t, warns, raw)
		}
	})

	t.Run("${VAR} placeholders do not trip the parser", func(t *testing.T) {
		warns, err := earlyUnknownKeyCheck([]byte("agent: opencode\nsecrets:\n  K: ${SOME_KEY}\nenv:\n  A: ${B}\n"), true)
		assert.NoError(t, err)
		assert.Empty(t, warns)
	})
}

func TestApplyStrict(t *testing.T) {
	v := &agentManifestValidation{Errors: []string{"e"}, Warnings: []string{"w1", "w2"}}
	applyStrict(v)
	assert.Equal(t, []string{"e", "w1", "w2"}, v.Errors)
	assert.Empty(t, v.Warnings)
	assert.Nil(t, applyStrict(nil))

	v = validateAgentManifest([]byte("agent: opencode\nbogus_field: 1\n"))
	require.True(t, v.ok())
	assert.False(t, applyStrict(v).ok())
}

func TestRunAgentsValidate_UnknownKeys(t *testing.T) {
	write := func(t *testing.T, manifest string) string {
		path := filepath.Join(t.TempDir(), "agent.yaml")
		require.NoError(t, os.WriteFile(path, []byte(manifest), 0o600))
		return path
	}

	t.Run("CUJ1 near miss fails validate and names the key on stdout", func(t *testing.T) {
		path := write(t, "name: demo\nagent: opencode\negresss: [api.github.com]\n")
		withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgAgentSpec, path)
			var buf bytes.Buffer
			config.Out = &buf
			require.ErrorIs(t, RunAgentsValidate(config), ErrExitSilently)
			assert.Contains(t, buf.String(), "egresss")
			assert.Contains(t, buf.String(), `"egress"`)
		})
	})

	t.Run("CUJ5 far key warns then --strict fails", func(t *testing.T) {
		path := write(t, "agent: opencode\nbogus_field: 1\n")
		withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgAgentSpec, path)
			var buf bytes.Buffer
			config.Out = &buf
			require.NoError(t, RunAgentsValidate(config))
			assert.Contains(t, buf.String(), "Warning")
			assert.Contains(t, buf.String(), "bogus_field")

			config.Doit.Set(config.NS, doctl.ArgAgentStrict, true)
			buf.Reset()
			require.ErrorIs(t, RunAgentsValidate(config), ErrExitSilently)
			assert.Contains(t, buf.String(), "bogus_field")
			assert.Contains(t, buf.String(), "Manifest is invalid")
		})
	})

	t.Run("CUJ3 valid fixture is clean even with --strict", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgAgentSpec, filepath.Join("testdata", "valid_full.yaml"))
			config.Doit.Set(config.NS, doctl.ArgAgentStrict, true)
			var buf bytes.Buffer
			config.Out = &buf
			require.NoError(t, RunAgentsValidate(config))
			assert.Contains(t, buf.String(), "Manifest looks valid")
			assert.NotContains(t, buf.String(), "Warning")
		})
	})
}

// --- create --dry-run --------------------------------------------------------

func dryRunCreate(t *testing.T, manifest string, strict bool) (stdout, stderr string, err error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agents.yaml")
	require.NoError(t, os.WriteFile(path, []byte(manifest), 0o600))
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{path}
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)
		if strict {
			config.Doit.Set(config.NS, doctl.ArgAgentStrict, true)
		}
		stderr = captureProcessStderr(t, func() { err = RunAgentsCreate(config) })
		stdout = buf.String()
	})
	return stdout, stderr, err
}

func TestRunAgentsCreate_DryRunUnknownKeys(t *testing.T) {
	t.Run("CUJ2 near miss fails and prints no manifest", func(t *testing.T) {
		stdout, _, err := dryRunCreate(t, "name: demo\nagent: opencode\negresss: [api.github.com]\n", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "manifest validation failed:")
		assert.Contains(t, err.Error(), `"egresss"`)
		assert.Contains(t, err.Error(), `"egress"`)
		assert.Empty(t, stdout)
	})

	t.Run("CUJ5 far key warns on stderr and still prints the manifest", func(t *testing.T) {
		stdout, stderr, err := dryRunCreate(t, "name: demo\nagent: opencode\nbogus_field: 1\n", false)
		require.NoError(t, err)
		assert.Contains(t, stderr, "bogus_field")
		assert.Contains(t, stdout, "name: demo")
		assert.Contains(t, stdout, "bogus_field")
		assert.NotContains(t, stdout, "Warning")
	})

	t.Run("CUJ5 --strict turns the warning into a failure", func(t *testing.T) {
		stdout, _, err := dryRunCreate(t, "name: demo\nagent: opencode\nbogus_field: 1\n", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bogus_field")
		assert.Empty(t, stdout)
	})

	t.Run("CUJ3 valid fixture prints the previous release's output and nothing on stderr", func(t *testing.T) {
		raw, err := os.ReadFile(filepath.Join("testdata", "valid_full.yaml"))
		require.NoError(t, err)
		// valid_full.dryrun.golden.yaml was generated by the dry-run of
		// origin/main before the unknown-key check existed. Git may check the
		// file out with CRLF line endings on Windows, so compare with LF.
		goldenBytes, err := os.ReadFile(filepath.Join("testdata", "valid_full.dryrun.golden.yaml"))
		require.NoError(t, err)
		golden := strings.ReplaceAll(string(goldenBytes), "\r\n", "\n")
		for _, strict := range []bool{false, true} {
			stdout, stderr, runErr := dryRunCreate(t, string(raw), strict)
			require.NoError(t, runErr)
			assert.Equal(t, golden, stdout)
			assert.Empty(t, stderr)
		}
	})

	t.Run("a value that is not an unknown-key problem still prints", func(t *testing.T) {
		// Dry-run runs only the unknown-key pass, not the whole validator.
		stdout, stderr, err := dryRunCreate(t, "name: demo\nagent: not-a-real-adapter\n", false)
		require.NoError(t, err)
		assert.Contains(t, stdout, "not-a-real-adapter")
		assert.Empty(t, strings.TrimSpace(stderr))
	})
}

// --- real create -------------------------------------------------------------

// No API expectation is registered, so gomock fails the test if the session is
// created. codex-agentapi would otherwise start OpenAI environment setup, so
// the error being the unknown-key one proves the check runs first.
func TestRunAgentsCreate_UnknownKeyFailsBeforeSideEffects(t *testing.T) {
	// Without these, codex-agentapi could fail on a missing key for an
	// unrelated reason and still look like it "failed before side effects".
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	for _, agent := range []string{"opencode", "codex-agentapi"} {
		t.Run(agent, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "agents.yaml")
			require.NoError(t, os.WriteFile(path, []byte("name: demo\nagent: "+agent+"\negresss: [api.github.com]\n"), 0o600))
			prevInteractive := Interactive
			Interactive = false
			t.Cleanup(func() { Interactive = prevInteractive })

			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				var buf bytes.Buffer
				config.Out = &buf
				config.Args = []string{path}
				err := RunAgentsCreate(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), `"egresss"`)
				assert.Contains(t, err.Error(), `did you mean "egress"`)
			})
		})
	}
}

func TestRunAgentsCreate_FarKeyStrictFailsBeforeSideEffects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agents.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: demo\nagent: opencode\nbogus_field: 1\n"), 0o600))
	prevInteractive := Interactive
	Interactive = false
	t.Cleanup(func() { Interactive = prevInteractive })

	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{path}
		config.Doit.Set(config.NS, doctl.ArgAgentStrict, true)
		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bogus_field")
	})
}

// --- sibling pairs -----------------------------------------------------------

// siblingPairs lists every pair of known keys in the same closed object that
// are within the near-miss rule of each other. A user typing one of the pair
// can never be told it is a misspelling of the other, but a NEW server field
// that lands next to an old key would be rejected as a typo by an older doctl,
// so this list is the review point when the schema grows.
func siblingPairs() []string {
	var pairs []string
	for path, keys := range flattenKeyTree() {
		for i := range keys {
			for j := i + 1; j < len(keys); j++ {
				a, b := keys[i], keys[j]
				if a > b {
					a, b = b, a
				}
				if d := editDistance(a, b); d <= 2 && (d < len([]rune(a)) || d < len([]rune(b))) {
					pairs = append(pairs, displayKeyPath(path)+": "+a+"/"+b)
				}
			}
		}
	}
	sort.Strings(pairs)
	return pairs
}

func TestManifestKeySiblingPairs(t *testing.T) {
	assert.Equal(t, []string{
		"(root): mode/model",
		"serving: max/min",
	}, siblingPairs(), "a new key landed within edit distance 2 of an existing sibling; review it, then update this list")
}

// --- ordering, strict wiring, flags --------------------------------------------

// A near-miss must fail before maybeOfferGitHubAuth, which calls the API
// (StartProviderAuth) and may start an interactive OAuth flow. No expectation
// is registered for it, so gomock fails the test if it is called.
func TestRunAgentsCreate_NearMissWithGhRepoFailsBeforeGitHubAuth(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agents.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: demo\nagent: opencode\negresss: [api.github.com]\n"), 0o600))
	prevInteractive := Interactive
	Interactive = false
	t.Cleanup(func() { Interactive = prevInteractive })

	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{path}
		config.Doit.Set(config.NS, doctl.ArgAgentRepo, "digitalocean/doctl")
		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `did you mean "egress"`)
	})
}

func TestRunAgentsCreate_StrictRejectedWithFromConfig(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentStrict, true)
		_, err := resolveAgentCreationSource(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--strict cannot be used with --from-config")
	})
}

func TestLaunchExistingSession_RejectsStrict(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentStrict, true)
		err := launchExistingSession(config, "sess_1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--strict only applies when creating a new session")
	})
}

// A far key warns exactly once on a real create: the early check reports it and
// the later validator skips the unknown-key pass.
func TestRunAgentsCreate_FarKeyWarnsExactlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agents.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: demo\nagent: opencode\nbogus_field: 1\nenv:\n  LITERAL_FLAG: on\n"), 0o600))
	prevInteractive := Interactive
	Interactive = false
	t.Cleanup(func() { Interactive = prevInteractive })
	prev := sessionReadyPollInterval
	sessionReadyPollInterval = time.Millisecond
	t.Cleanup(func() { sessionReadyPollInterval = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			Return(&do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
				SessionID: "sess_once", Name: "demo", Status: godo.HostedAgentSessionStatusProvisioning,
			}}, nil)
		tm.hostedAgents.EXPECT().
			GetSession("sess_once").
			Return(&do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
				SessionID: "sess_once", Name: "demo", Status: godo.HostedAgentSessionStatusReady,
			}}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{path}
		var err error
		stderr := captureProcessStderr(t, func() { err = RunAgentsCreate(config) })
		require.NoError(t, err)
		assert.Equal(t, 1, strings.Count(stderr, `unknown field "bogus_field"`), "stderr: %s", stderr)
	})
}

func TestStrictFlagRegisteredEverywhereItApplies(t *testing.T) {
	flagOn := func(t *testing.T, parent *Command, name string) {
		t.Helper()
		for _, c := range parent.Commands() {
			if c.Name() == name {
				f := c.Flags().Lookup(doctl.ArgAgentStrict)
				require.NotNil(t, f, "%s is missing --%s", name, doctl.ArgAgentStrict)
				assert.Equal(t, "bool", f.Value.Type())
				return
			}
		}
		t.Fatalf("command %q not found", name)
	}
	agents := Agents()
	for _, name := range []string{"validate", "create", "launch"} {
		t.Run(name, func(t *testing.T) { flagOn(t, agents, name) })
	}
	t.Run("config create", func(t *testing.T) { flagOn(t, AgentConfigs(), "create") })
	t.Run("trigger create", func(t *testing.T) { flagOn(t, AgentTriggers(), "create") })
	t.Run("trigger update", func(t *testing.T) { flagOn(t, AgentTriggers(), "update") })
}

// --- config create and trigger create ----------------------------------------

func writeSpec(t *testing.T, manifest string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent.yaml")
	require.NoError(t, os.WriteFile(path, []byte(manifest), 0o600))
	return path
}

func TestRunAgentsConfigCreate_UnknownKeys(t *testing.T) {
	run := func(t *testing.T, manifest string, strict, expectCreate bool) (stderr string, err error) {
		spec := writeSpec(t, manifest)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			if expectCreate {
				tm.hostedAgents.EXPECT().
					CreateAgentConfig(gomock.Any()).
					Return(&godo.HostedAgentConfig{ID: "cfg_1", Name: "my-config"}, nil)
			} // otherwise any CreateAgentConfig call fails the test
			var buf bytes.Buffer
			config.Out = &buf
			config.Doit.Set(config.NS, doctl.ArgAgentSpec, spec)
			config.Doit.Set(config.NS, doctl.ArgAgentName, "my-config")
			if strict {
				config.Doit.Set(config.NS, doctl.ArgAgentStrict, true)
			}
			stderr = captureProcessStderr(t, func() { err = RunAgentsConfigCreate(config) })
		})
		return stderr, err
	}

	t.Run("near miss fails", func(t *testing.T) {
		_, err := run(t, "agent: opencode\negresss: [a.com]\n", false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `did you mean "egress"`)
	})
	t.Run("far key with --strict fails", func(t *testing.T) {
		_, err := run(t, "agent: opencode\nbogus_field: 1\n", true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bogus_field")
	})
	t.Run("far key without --strict is only a warning, not promoted to an error", func(t *testing.T) {
		stderr, err := run(t, "agent: opencode\nbogus_field: 1\n", false, true)
		require.NoError(t, err)
		assert.Contains(t, stderr, "bogus_field")
	})
}

func TestRunAgentTriggersCreate_UnknownKeys(t *testing.T) {
	run := func(t *testing.T, manifest string, strict, expectCreate bool) (stderr string, err error) {
		spec := writeSpec(t, manifest)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			if expectCreate {
				tm.hostedAgentTriggers.EXPECT().
					Create(gomock.Any()).
					Return(&do.HostedAgentTriggerCreateResult{
						WebhookSecret: "sec",
						Trigger: &do.HostedAgentTrigger{HostedAgentTrigger: &godo.HostedAgentTrigger{
							TriggerID: "tr_1", Name: "gh-ci", Kind: godo.HostedAgentTriggerKindWebhook,
						}},
					}, nil)
			} // otherwise any Create call fails the test
			var buf bytes.Buffer
			config.Out = &buf
			config.Doit.Set(config.NS, doctl.ArgAgentTriggerKind, "webhook")
			config.Doit.Set(config.NS, doctl.ArgAgentName, "gh-ci")
			config.Doit.Set(config.NS, doctl.ArgAgentTriggerSessionMode, "fresh")
			config.Doit.Set(config.NS, doctl.ArgAgentTriggerPrompt, "run it")
			config.Doit.Set(config.NS, doctl.ArgAgentTriggerOutputMode, "none")
			config.Doit.Set(config.NS, doctl.ArgAgentSpec, spec)
			config.Doit.Set(config.NS, doctl.ArgAgentTriggerProvider, "github")
			if strict {
				config.Doit.Set(config.NS, doctl.ArgAgentStrict, true)
			}
			stderr = captureProcessStderr(t, func() { err = RunAgentTriggersCreate(config) })
		})
		return stderr, err
	}

	t.Run("near miss fails", func(t *testing.T) {
		_, err := run(t, "agent: opencode\negresss: [a.com]\n", false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `did you mean "egress"`)
	})
	t.Run("far key with --strict fails", func(t *testing.T) {
		_, err := run(t, "agent: opencode\nbogus_field: 1\n", true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bogus_field")
	})
	t.Run("far key without --strict warns and creates", func(t *testing.T) {
		stderr, err := run(t, "agent: opencode\nbogus_field: 1\n", false, true)
		require.NoError(t, err)
		assert.Contains(t, stderr, "bogus_field")
	})
}
