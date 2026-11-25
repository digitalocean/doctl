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
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

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
				walk(manifestKeys().toolInline, p+"[]")
				out[p+"[]."+"{catalog}"] = manifestKeys().toolCatalog.names()
			}
		}
	}
	walk(manifestKeys().root, rootKeyPath)
	// The egress object is checked by validateEgressObject, not this tree.
	return out
}

// The egress object is the one closed object this tree skips; its keys must
// still agree with the schema so validateEgressObject does not drift either.
func TestEgressKeysMatchSchemaCopy(t *testing.T) {
	var schema struct {
		Properties struct {
			Egress struct {
				OneOf []json.RawMessage `json:"oneOf"`
			} `json:"egress"`
		} `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(agentFlatSchema, &schema))
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
	schemaKeys, err := schemaKeyOrder(obj.Properties)
	require.NoError(t, err)

	var known []string
	for k := range egressKnownKeysForFormat(false) {
		known = append(known, k)
	}
	sort.Strings(known)
	sorted := append([]string(nil), schemaKeys...)
	sort.Strings(sorted)
	require.Equal(t, sorted, known)
}
