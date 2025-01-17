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

package displayers

import (
	"io"

	"github.com/digitalocean/doctl/do"
)

type SpacesKey struct {
	SpacesKeys []do.SpacesKey
}

var _ Displayable = &SpacesKey{}

// ColMap implements Displayable.
func (s *SpacesKey) ColMap() map[string]string {
	return map[string]string{
		"Name":      "Name",
		"AccessKey": "Access Key",
		"SecretKey": "Secret Key",
		"Grants":    "Grants",
		"CreatedAt": "Created At",
	}
}

// Cols implements Displayable.
func (s *SpacesKey) Cols() []string {
	return []string{
		"Name",
		"AccessKey",
		"SecretKey",
		"Grants",
		"CreatedAt",
	}
}

// JSON implements Displayable.
func (s *SpacesKey) JSON(out io.Writer) error {
	return writeJSON(s.SpacesKeys, out)
}

// KV implements Displayable.
func (s *SpacesKey) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.SpacesKeys))

	for _, key := range s.SpacesKeys {
		m := map[string]any{
			"Name":      key.Name,
			"AccessKey": key.AccessKey,
			"SecretKey": key.SecretKey,
			"Grants":    key.GrantString(),
			"CreatedAt": key.CreatedAt,
		}

		out = append(out, m)
	}

	return out
}
