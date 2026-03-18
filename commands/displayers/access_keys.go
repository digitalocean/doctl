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

type AccessKeys struct {
	AccessKeys     []do.AccessKey
	ShowFullSecret bool // When true, shows full secret (for creation), otherwise truncates/hides
}

var _ Displayable = &AccessKeys{}

// JSON implements Displayable.
func (ak *AccessKeys) JSON(out io.Writer) error {
	return writeJSON(ak.AccessKeys, out)
}

// Cols implements Displayable.
func (ak *AccessKeys) Cols() []string {
	cols := []string{
		"ID",
		"Name",
	}
	// Only show Secret during creation (when ShowFullSecret is true)
	if ak.ShowFullSecret {
		cols = append(cols, "Secret")
	}
	cols = append(cols, "CreatedAt", "ExpiresAt")
	return cols
}

// ColMap implements Displayable.
func (ak *AccessKeys) ColMap() map[string]string {
	colMap := map[string]string{
		"ID":        "ID",
		"Name":      "Name",
		"CreatedAt": "Created At",
		"ExpiresAt": "Expires At",
	}
	// Only include Secret column during creation
	if ak.ShowFullSecret {
		colMap["Secret"] = "Secret"
	}
	return colMap
}

// KV implements Displayable.
func (ak *AccessKeys) KV() []map[string]any {
	out := make([]map[string]any, 0, len(ak.AccessKeys))

	for _, key := range ak.AccessKeys {
		// Format optional timestamp fields
		expiresAt := ""
		if key.ExpiresAt != nil {
			expiresAt = key.ExpiresAt.Format("2006-01-02 15:04:05 UTC")
		}

		m := map[string]any{
			"ID":        key.ID,
			"Name":      key.Name,
			"CreatedAt": key.CreatedAt.Format("2006-01-02 15:04:05 UTC"),
			"ExpiresAt": expiresAt,
		}

		// Only include Secret field during creation (when API returns it)
		if ak.ShowFullSecret && key.Secret != "" {
			m["Secret"] = key.Secret
		}

		out = append(out, m)
	}

	return out
}

// ForCreate returns a displayer optimized for showing newly created access keys
// This version shows the full secret since it's only displayed once
func (ak *AccessKeys) ForCreate() *AccessKeys {
	return &AccessKeys{AccessKeys: ak.AccessKeys, ShowFullSecret: true}
}
