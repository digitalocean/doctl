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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// testdata/agent.flat.schema.json is a copy of the flat agent manifest JSON
// schema (free-text descriptions are removed; only the structure is compared).
// Keys for server features doctl does not support yet are left out of both the
// copy and the tree on purpose: a manifest using one gets the "unknown field"
// warning, which says it may be a newer field. To refresh the copy, copy the
// new schema over, drop those keys, then update
// manifestRootKeys and friends in agents_manifest_keys.go until this test
// passes.

const rootKeyPath = ""

func flatKeyPath(parent, key string) string {
	if parent == rootKeyPath {
		return key
	}
	return parent + "." + key
}

func displayKeyPath(path string) string {
	if path == rootKeyPath {
		return "(root)"
	}
	return path
}

// flattenKeyTree lists, per closed object, its known keys in order. Paths use
// the same convention as the schema walker: "a.b" for a nested object, "a[]"
// for list items, "a.*" for the object value of a free-keyed map, and
// "tools[].{catalog}" for the value object of a catalog selection.
func flattenKeyTree() map[string][]string {
	out := map[string][]string{}
	var walk func(n *keyNode, path string)
	walk = func(n *keyNode, path string) {
		out[path] = n.names()
		for _, k := range n.keys {
			p := flatKeyPath(path, k.name)
			switch k.kind {
			case keyObject:
				walk(k.child, p)
			case keyObjectList:
				walk(k.child, p+"[]")
			case keyMapOfObjects:
				walk(k.child, p+".*")
			case keyTools:
				walk(manifestToolInlineKeys, p+"[]")
				out[p+"[]."+"{catalog}"] = manifestToolCatalogKeys.names()
			}
		}
	}
	walk(manifestRootKeys, rootKeyPath)
	// The egress object is checked by validateEgressObject, not this tree.
	return out
}

// schemaKeyOrder returns the keys of the JSON object raw, in document order.
func schemaKeyOrder(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	require.NoError(t, err)
	require.Equal(t, json.Delim('{'), tok)
	var keys []string
	for dec.More() {
		kt, err := dec.Token()
		require.NoError(t, err)
		keys = append(keys, kt.(string))
		var skip json.RawMessage
		require.NoError(t, dec.Decode(&skip))
	}
	return keys
}

type schemaNode struct {
	Properties           json.RawMessage            `json:"properties"`
	AdditionalProperties json.RawMessage            `json:"additionalProperties"`
	PatternProperties    map[string]json.RawMessage `json:"patternProperties"`
	Items                json.RawMessage            `json:"items"`
	OneOf                []json.RawMessage          `json:"oneOf"`
	AnyOf                []json.RawMessage          `json:"anyOf"`
	AllOf                []json.RawMessage          `json:"allOf"`
	Ref                  string                     `json:"$ref"`
	Then                 json.RawMessage            `json:"then"`
	Else                 json.RawMessage            `json:"else"`
}

// closedKeysFromSchema walks the schema and returns the property names of
// every closed object (additionalProperties: false with properties), keyed by
// path. It follows properties, items, oneOf/anyOf/allOf branches, schema-valued
// additionalProperties, patternProperties and local $ref targets. if/then/else
// only constrain values, so they are not followed; instead the walker asserts that no then/else branch
// declares a closed object (properties + additionalProperties: false): if one
// ever does, its keys would be accepted by the server but invisible here, and
// this test fails so the walker gets extended.
// The egress object is skipped (see agents_manifest_keys.go).
func closedKeysFromSchema(t *testing.T, root, raw json.RawMessage, path string, out map[string][]string) {
	t.Helper()
	if path == "egress" {
		return
	}
	var n schemaNode
	require.NoError(t, json.Unmarshal(raw, &n), "path %q", path)

	for _, branch := range []json.RawMessage{n.Then, n.Else} {
		if len(branch) == 0 || string(bytes.TrimSpace(branch)) == "false" {
			continue
		}
		var b schemaNode
		if err := json.Unmarshal(branch, &b); err != nil {
			continue
		}
		require.False(t, len(b.Properties) > 0 && string(bytes.TrimSpace(b.AdditionalProperties)) == "false",
			"a then/else branch at %q declares a closed object; extend closedKeysFromSchema to follow it", displayKeyPath(path))
	}
	if n.Ref != "" {
		// A $ref target is walked at this path (crewai.framework is the
		// shared $defs/framework object); scalar targets such as the duration
		// def and the agent -> harness alias have no properties and add nothing.
		closedKeysFromSchema(t, root, resolveSchemaRef(t, root, n.Ref), path, out)
	}
	if len(n.Properties) > 0 {
		closed := string(bytes.TrimSpace(n.AdditionalProperties)) == "false"
		var props map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(n.Properties, &props))
		order := schemaKeyOrder(t, n.Properties)
		if closed {
			_, dup := out[path]
			require.False(t, dup, "two closed objects at %q", displayKeyPath(path))
			out[path] = order
		}
		for _, k := range order {
			closedKeysFromSchema(t, root, props[k], flatKeyPath(path, k), out)
		}
	}
	if len(n.Items) > 0 {
		closedKeysFromSchema(t, root, n.Items, path+"[]", out)
	}
	if ap := bytes.TrimSpace(n.AdditionalProperties); len(ap) > 0 && ap[0] == '{' {
		closedKeysFromSchema(t, root, n.AdditionalProperties, path+".*", out)
	}
	if len(n.PatternProperties) > 0 {
		require.Len(t, n.PatternProperties, 1, "path %q", path)
		for pattern, sub := range n.PatternProperties {
			require.True(t, strings.HasPrefix(pattern, `^do\.`), "unexpected pattern %q at %q", pattern, path)
			closedKeysFromSchema(t, root, sub, path+".{catalog}", out)
		}
	}
	for _, branches := range [][]json.RawMessage{n.OneOf, n.AnyOf, n.AllOf} {
		for _, b := range branches {
			closedKeysFromSchema(t, root, b, path, out)
		}
	}
}

// resolveSchemaRef resolves a local "#/a/b" reference against the schema root.
func resolveSchemaRef(t *testing.T, root json.RawMessage, ref string) json.RawMessage {
	t.Helper()
	require.True(t, strings.HasPrefix(ref, "#/"), "only local $ref is supported, got %q", ref)
	cur := root
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(cur, &m), "resolving %q", ref)
		next, ok := m[part]
		require.True(t, ok, "$ref %q: no %q", ref, part)
		cur = next
	}
	return cur
}

func TestManifestKeyTreeMatchesSchemaCopy(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "agent.flat.schema.json"))
	require.NoError(t, err)

	want := map[string][]string{}
	closedKeysFromSchema(t, raw, raw, rootKeyPath, want)

	got := flattenKeyTree()

	paths := map[string]bool{}
	for p := range want {
		paths[p] = true
	}
	for p := range got {
		paths[p] = true
	}
	var sorted []string
	for p := range paths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)

	var diffs []string
	for _, p := range sorted {
		w, g := want[p], got[p]
		switch {
		case w == nil:
			diffs = append(diffs, fmt.Sprintf("%s: in doctl's tree but not a closed object in the schema: %v", displayKeyPath(p), g))
		case g == nil:
			diffs = append(diffs, fmt.Sprintf("%s: closed object in the schema but missing from doctl's tree: %v", displayKeyPath(p), w))
		case strings.Join(w, ",") != strings.Join(g, ","):
			diffs = append(diffs, fmt.Sprintf("%s:\n    schema: %v\n    doctl:  %v\n    missing from doctl: %v\n    extra in doctl:     %v",
				displayKeyPath(p), w, g, subtractKeys(w, g), subtractKeys(g, w)))
		}
	}
	require.Empty(t, diffs, "doctl's manifest key tree differs from testdata/agent.flat.schema.json; update agents_manifest_keys.go (keep the schema's key order):\n  %s", strings.Join(diffs, "\n  "))

	// Sanity check only, after the readable diff: this tree holds 22 closed
	// objects (the schema's others are egress, which is skipped, and the tools
	// catalog map itself, which has no named keys).
	require.Len(t, want, 22)
}

func subtractKeys(a, b []string) []string {
	in := map[string]bool{}
	for _, k := range b {
		in[k] = true
	}
	var out []string
	for _, k := range a {
		if !in[k] {
			out = append(out, k)
		}
	}
	return out
}

// The egress object is the one closed object this tree skips; its keys must
// still agree with the schema so validateEgressObject does not drift either.
func TestEgressKeysMatchSchemaCopy(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "agent.flat.schema.json"))
	require.NoError(t, err)
	var schema struct {
		Properties struct {
			Egress struct {
				OneOf []json.RawMessage `json:"oneOf"`
			} `json:"egress"`
		} `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(raw, &schema))
	var objectForm json.RawMessage
	for _, b := range schema.Properties.Egress.OneOf {
		var probe struct {
			Type string `json:"type"`
		}
		require.NoError(t, json.Unmarshal(b, &probe))
		if probe.Type == "object" {
			objectForm = b
		}
	}
	require.NotNil(t, objectForm)
	var obj struct {
		Properties json.RawMessage `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(objectForm, &obj))
	schemaKeys := schemaKeyOrder(t, obj.Properties)

	var known []string
	for k := range egressKnownKeysForFormat(false) {
		known = append(known, k)
	}
	sort.Strings(known)
	sorted := append([]string(nil), schemaKeys...)
	sort.Strings(sorted)
	require.Equal(t, sorted, known)
}
