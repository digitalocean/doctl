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
	return []string{
		"ID",
		"Name",
		"Secret",
		"CreatedAt",
		"ExpiresAt",
	}
}

// ColMap implements Displayable.
func (ak *AccessKeys) ColMap() map[string]string {
	return map[string]string{
		"ID":        "ID",
		"Name":      "Name",
		"Secret":    "Secret",
		"CreatedAt": "Created At",
		"ExpiresAt": "Expires At",
	}
}

// KV implements Displayable.
func (ak *AccessKeys) KV() []map[string]any {
	out := make([]map[string]any, 0, len(ak.AccessKeys))

	for _, key := range ak.AccessKeys {
		// Show full secret during creation, hidden otherwise
		secret := "<hidden>"
		if key.Secret != "" && ak.ShowFullSecret {
			// During creation: show the full secret
			secret = key.Secret
		}
		// For all other cases (listing, etc.): always show "<hidden>"

		// Format optional timestamp fields
		expiresAt := ""
		if key.ExpiresAt != nil {
			expiresAt = key.ExpiresAt.Format("2006-01-02 15:04:05 UTC")
		}

		// Truncate long IDs for display
		displayID := key.ID
		if len(displayID) > 12 {
			displayID = displayID[:12] + "..."
		}

		m := map[string]any{
			"ID":        displayID,
			"Name":      key.Name,
			"Secret":    secret,
			"CreatedAt": key.CreatedAt.Format("2006-01-02 15:04:05 UTC"),
			"ExpiresAt": expiresAt,
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
