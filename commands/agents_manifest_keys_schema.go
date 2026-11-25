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
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// agentFlatSchema is a copy of the flat agent manifest JSON schema with the
// free-text descriptions removed. Keys for server features doctl does not
// support yet are left out on purpose: a manifest using one gets the "unknown
// field" warning, which says it may be a newer field. To follow a schema
// change, copy the new schema over and drop those keys.
//
//go:embed agent.flat.schema.json
var agentFlatSchema []byte

type manifestKeyTree struct {
	root, toolInline, toolCatalog *keyNode
}

// manifestKeys is the key tree built from agentFlatSchema. The schema is
// compiled into the binary, so any test that loads the tree proves it builds.
var manifestKeys = sync.OnceValue(func() manifestKeyTree {
	t, err := buildManifestKeyTree(agentFlatSchema)
	if err != nil {
		panic("agent.flat.schema.json: " + err.Error())
	}
	return t
})

func buildManifestKeyTree(schema []byte) (manifestKeyTree, error) {
	closed := map[string][]string{}
	if err := walkSchema(schema, schema, "", closed); err != nil {
		return manifestKeyTree{}, err
	}
	for _, p := range []string{"", "tools[]", "tools[].{catalog}"} {
		if _, ok := closed[p]; !ok {
			return manifestKeyTree{}, fmt.Errorf("schema has no closed object at %q", p)
		}
	}
	nodes := map[string]*keyNode{}
	var node func(path string) *keyNode
	node = func(path string) *keyNode {
		if n, ok := nodes[path]; ok {
			return n
		}
		n := &keyNode{}
		nodes[path] = n
		for _, k := range closed[path] {
			def := keyDef{name: k}
			var child string
			def.kind, child = classifyKey(closed, path, k)
			if child != "" {
				def.child = node(child)
			}
			n.keys = append(n.keys, def)
		}
		return n
	}
	return manifestKeyTree{root: node(""), toolInline: node("tools[]"), toolCatalog: node("tools[].{catalog}")}, nil
}

// classifyKey says what key k of the closed object at path holds, from which
// closed-object paths the schema walk found beneath it.
func classifyKey(closed map[string][]string, path, k string) (keyKind, string) {
	child := joinManifestPath(path, k)
	if _, ok := closed[child+"[].{catalog}"]; ok {
		return keyTools, ""
	}
	for _, c := range []struct {
		suffix string
		kind   keyKind
	}{
		{"", keyObject},
		{"[]", keyObjectList},
		{".*", keyMapOfObjects},
	} {
		if _, ok := closed[child+c.suffix]; ok {
			return c.kind, child + c.suffix
		}
	}
	return keyLeaf, ""
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

// walkSchema records the property names (in schema order) of every closed
// object (additionalProperties: false with properties), keyed by path: "a.b"
// for a nested object, "a[]" for list items, "a.*" for the object value of a
// free-keyed map and "tools[].{catalog}" for a catalog selection's value. It
// follows properties, items, oneOf/anyOf/allOf, schema-valued
// additionalProperties, patternProperties and local $ref targets. The egress
// object is skipped: validateEgressObject checks it.
func walkSchema(root, raw json.RawMessage, path string, out map[string][]string) error {
	if path == "egress" {
		return nil
	}
	var n schemaNode
	if err := json.Unmarshal(raw, &n); err != nil {
		return fmt.Errorf("path %q: %w", path, err)
	}
	// if/then/else only constrain values; a closed object there would hold
	// keys the server accepts but this tree cannot see, so refuse it.
	for _, branch := range []json.RawMessage{n.Then, n.Else} {
		if len(branch) == 0 || string(bytes.TrimSpace(branch)) == "false" {
			continue
		}
		var br schemaNode
		if json.Unmarshal(branch, &br) == nil && len(br.Properties) > 0 && string(bytes.TrimSpace(br.AdditionalProperties)) == "false" {
			return fmt.Errorf("a then/else branch at %q declares a closed object; extend walkSchema to follow it", path)
		}
	}
	if n.Ref != "" {
		target, err := resolveSchemaRef(root, n.Ref)
		if err != nil {
			return err
		}
		if err := walkSchema(root, target, path, out); err != nil {
			return err
		}
	}
	if len(n.Properties) > 0 {
		order, err := schemaKeyOrder(n.Properties)
		if err != nil {
			return fmt.Errorf("path %q: %w", path, err)
		}
		var props map[string]json.RawMessage
		if err := json.Unmarshal(n.Properties, &props); err != nil {
			return err
		}
		if string(bytes.TrimSpace(n.AdditionalProperties)) == "false" {
			if _, dup := out[path]; dup {
				return fmt.Errorf("two closed objects at %q", path)
			}
			out[path] = order
		}
		for _, k := range order {
			if err := walkSchema(root, props[k], joinManifestPath(path, k), out); err != nil {
				return err
			}
		}
	}
	if len(n.Items) > 0 {
		if err := walkSchema(root, n.Items, path+"[]", out); err != nil {
			return err
		}
	}
	if ap := bytes.TrimSpace(n.AdditionalProperties); len(ap) > 0 && ap[0] == '{' {
		if err := walkSchema(root, n.AdditionalProperties, path+".*", out); err != nil {
			return err
		}
	}
	if len(n.PatternProperties) > 0 {
		if len(n.PatternProperties) != 1 {
			return fmt.Errorf("path %q: expected one patternProperties entry", path)
		}
		for pattern, sub := range n.PatternProperties {
			if !strings.HasPrefix(pattern, `^do\.`) {
				return fmt.Errorf("unexpected pattern %q at %q", pattern, path)
			}
			if err := walkSchema(root, sub, path+".{catalog}", out); err != nil {
				return err
			}
		}
	}
	for _, branches := range [][]json.RawMessage{n.OneOf, n.AnyOf, n.AllOf} {
		for _, br := range branches {
			if err := walkSchema(root, br, path, out); err != nil {
				return err
			}
		}
	}
	return nil
}

// schemaKeyOrder returns the keys of the JSON object raw, in document order.
func schemaKeyOrder(raw json.RawMessage) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, fmt.Errorf("not a JSON object")
	}
	var keys []string
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keys = append(keys, kt.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

func resolveSchemaRef(root json.RawMessage, ref string) (json.RawMessage, error) {
	if !strings.HasPrefix(ref, "#/") {
		return nil, fmt.Errorf("only local $ref is supported, got %q", ref)
	}
	cur := root
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(cur, &m); err != nil {
			return nil, fmt.Errorf("resolving %q: %w", ref, err)
		}
		next, ok := m[part]
		if !ok {
			return nil, fmt.Errorf("$ref %q: no %q", ref, part)
		}
		cur = next
	}
	return cur, nil
}
