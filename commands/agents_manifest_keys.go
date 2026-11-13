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
	"fmt"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// This file holds doctl's copy of the flat agent manifest key tree and the
// check that reports field names the server would reject.
//
// The server decodes the flat manifest strictly, so a misspelled key such as
// `egresss` is rejected with an HTTP 400 at session create. doctl used to look
// at only `agent` and `name`, so `validate` and `create --dry-run` passed it.
//
// Source of truth: the published flat agent manifest JSON schema. A copy
// (as of 2026-11-02) lives in testdata/agent.flat.schema.json, and the key tree
// is generated from it (see the go:generate line below), so a schema refresh
// is one command and TestManifestKeyTreeIsGenerated fails if it was skipped.
// Key order is the schema's property order; it breaks ties between equally
// close candidates.
//
// Only closed objects (additionalProperties: false) are listed. Open objects
// (config, labels, env, secrets slot names, permissions.rules[].match,
// skills[].metadata, tools catalog keys) are marked as plain leaves and are
// never inspected. The egress object is also a leaf: validateEgressObject
// already errors on its unknown keys with format-specific hints.

// keyKind says what a known key holds, which decides whether the check
// descends into its value.
type keyKind int

const (
	// keyLeaf is a scalar, a list of scalars, an open map, or the egress
	// object. Never inspected.
	keyLeaf keyKind = iota
	// keyObject holds one closed object (child). A non-map value (the string
	// shorthand of model and memory) is left to the other validators.
	keyObject
	// keyObjectList holds a list of closed objects (child).
	keyObjectList
	// keyMapOfObjects holds a map with user-chosen keys whose object values
	// are closed (child). Used by secrets: slot names are free, the long
	// form of a slot is closed.
	keyMapOfObjects
	// keyTools is the tools list, which has its own per-entry forms.
	keyTools
)

type keyDef struct {
	name  string
	kind  keyKind
	child *keyNode
}

// keyNode is one closed object: its known keys in schema order.
type keyNode struct {
	keys []keyDef
}

func (n *keyNode) lookup(name string) (keyDef, bool) {
	for _, k := range n.keys {
		if k.name == name {
			return k, true
		}
	}
	return keyDef{}, false
}

func (n *keyNode) names() []string {
	out := make([]string, len(n.keys))
	for i, k := range n.keys {
		out[i] = k.name
	}
	return out
}

// The tree itself (manifestRootKeys and the nodes under it) lives in
// agents_manifest_keys_tree.go, generated from testdata/agent.flat.schema.json.
// To follow a schema change, replace the copy and run `go generate ./commands`.
//
//go:generate go run ./internal/manifestkeys/cmd

// editDistance is the Levenshtein distance between a and b, counted in runes.
// The server counts bytes. Every known key is ASCII, and a candidate within
// distance 2 must be ASCII-adjacent to the typed key anyway, so for any key a
// person plausibly mistypes the two counts agree; runes just avoid splitting a
// multi-byte character into several "edits".
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = minInt3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func minInt3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// nearMissKey returns the known key an unknown key most plausibly misspells.
// The unknown key is lower-cased first. A candidate counts only when the
// distance is at most 2 and smaller than the candidate's length, so a short
// key such as "id" is never "close" to everything. The smallest distance wins;
// the first-listed key wins a tie. This is the server's closestFlatField rule.
func nearMissKey(unknown string, known []string) (string, bool) {
	lower := strings.ToLower(unknown)
	best, bestDist := "", -1
	for _, cand := range known {
		d := editDistance(lower, cand)
		if d > 2 || d >= len([]rune(cand)) {
			continue
		}
		if bestDist < 0 || d < bestDist {
			best, bestDist = cand, d
		}
	}
	return best, bestDist >= 0
}

// manifestMap is yamlMap that tolerates non-string keys (for example
// `5: x`): yamlMap rejects the whole map for one such key, which would silently
// switch checking off for every sibling. Keys are stringified with Sprint.
func manifestMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			if ks, ok := k.(string); ok {
				out[ks] = val
			} else {
				out[fmt.Sprint(k)] = val
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func joinManifestPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

// unknownManifestKeys reports every key in a flat manifest that no closed
// object declares. A near-miss of a known key in the same object is an error,
// a key that differs from a known key only in letter case is a warning, and
// any other unknown key is a warning (the server stays the authority for names
// doctl does not know). Messages carry the full path, for example
// permissions.network.<key> or mounts[0].<key>.
//
// The warning text must not contain "matches a declared", "looks like a
// credential" or "name suggests a credential": config create promotes
// warnings containing them to errors.
func unknownManifestKeys(doc map[string]any) (errs, warns []string) {
	w := &keyWalker{}
	w.object(doc, manifestRootKeys, "")
	return w.errs, w.warns
}

type keyWalker struct {
	errs, warns []string
}

func (w *keyWalker) object(m map[string]any, node *keyNode, path string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		def, known := node.lookup(k)
		if !known {
			w.unknown(k, node, path)
			continue
		}
		w.value(m[k], def, joinManifestPath(path, k))
	}
}

func (w *keyWalker) unknown(key string, node *keyNode, path string) {
	// An unexpanded ${VAR} used as a key is a templating reference, not a field
	// name; the early check runs on raw bytes before expansion.
	if strings.Contains(key, "${") {
		return
	}
	if key == "" {
		w.warns = append(w.warns, fmt.Sprintf(
			"%s: unknown field with an empty name; the server rejects unknown fields when the session is created",
			joinManifestPath(path, "(empty key)")))
		return
	}
	full := joinManifestPath(path, key)
	names := node.names()
	for _, known := range names {
		if strings.EqualFold(key, known) {
			w.warns = append(w.warns, fmt.Sprintf(
				"%s: unknown field %q differs from %q only in letter case; the server may reject it when the session is created (use %q)",
				full, key, known, known))
			return
		}
	}
	if cand, ok := nearMissKey(key, names); ok {
		w.errs = append(w.errs, fmt.Sprintf(
			"%s: unknown field %q (did you mean %q? if this is a newer field, upgrade doctl)",
			full, key, cand))
		return
	}
	w.warns = append(w.warns, fmt.Sprintf(
		"%s: unknown field %q; the server rejects unknown fields when the session is created (if this is a newer field, upgrade doctl)",
		full, key))
}

func (w *keyWalker) value(v any, def keyDef, path string) {
	switch def.kind {
	case keyObject:
		if m, ok := manifestMap(v); ok {
			w.object(m, def.child, path)
		}
	case keyObjectList:
		list, ok := yamlList(v)
		if !ok {
			return
		}
		for i, item := range list {
			if m, ok := manifestMap(item); ok {
				w.object(m, def.child, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	case keyMapOfObjects:
		m, ok := manifestMap(v)
		if !ok {
			return
		}
		slots := make([]string, 0, len(m))
		for k := range m {
			slots = append(slots, k)
		}
		sort.Strings(slots)
		for _, slot := range slots {
			if sm, ok := manifestMap(m[slot]); ok {
				w.object(sm, def.child, joinManifestPath(path, slot))
			}
		}
	case keyTools:
		list, ok := yamlList(v)
		if !ok {
			return
		}
		for i, item := range list {
			if m, ok := manifestMap(item); ok {
				w.tool(m, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
}

// tool checks one tools[] entry that is a map. A map with any "do." key is a
// catalog selection ({do.actions: [...]} or {do.actions: {tools, preload_tools}}):
// the do.* keys are catalog names (open) and an object value allows only tools
// and preload_tools. Any other map is an inline server object; keys beside a
// do.* key are checked against the inline-server keys. String entries and list
// values are left to the other validators and the server.
func (w *keyWalker) tool(m map[string]any, path string) {
	catalog := false
	for k := range m {
		if strings.HasPrefix(k, "do.") {
			catalog = true
			break
		}
	}
	if !catalog {
		w.object(m, manifestToolInlineKeys, path)
		return
	}
	keys := make([]string, 0, len(m))
	others := map[string]any{}
	for k, v := range m {
		if strings.HasPrefix(k, "do.") {
			keys = append(keys, k)
		} else {
			others[k] = v
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		if vm, ok := manifestMap(m[k]); ok {
			w.object(vm, manifestToolCatalogKeys, joinManifestPath(path, k))
		}
	}
	// A catalog entry that also carries other keys is neither form; check those
	// against the inline-server keys so a misspelled inline-server key is still caught.
	if len(others) > 0 {
		w.object(others, manifestToolInlineKeys, path)
	}
}

// earlyUnknownKeyCheck runs only the unknown-key pass on raw manifest bytes.
// create --dry-run and real create call it before any side effect or ${VAR}
// expansion. It returns nil, nil when raw does not parse as a YAML map (the
// full validator reports parse errors later) or is a legacy envelope. With
// strict, warnings are returned as part of the error instead. With
// allowUnknown, near-miss errors are returned as warnings.
func earlyUnknownKeyCheck(raw []byte, strict, allowUnknown bool) (warnings []string, err error) {
	var doc map[string]any
	if yerr := yaml.Unmarshal(raw, &doc); yerr != nil || len(doc) == 0 {
		return nil, nil
	}
	if hasYAMLKey(doc, "apiVersion") {
		return nil, nil
	}
	errs, warns := unknownManifestKeys(doc)
	if allowUnknown {
		warns = append(warns, errs...)
		errs = nil
	}
	if strict {
		errs = append(errs, warns...)
		warns = nil
	}
	if len(errs) > 0 {
		return warns, fmt.Errorf("manifest validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return warns, nil
}
