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

package displayers

import (
	"io"

	"github.com/digitalocean/doctl/do"
)

// Secrets is the displayer for listing secret containers.
type Secrets struct {
	Secrets do.Secrets
}

var _ Displayable = &Secrets{}

func (s *Secrets) JSON(out io.Writer) error {
	return writeJSON(s.Secrets, out)
}

func (s *Secrets) Cols() []string {
	return []string{"Name", "Region", "Version", "Created At", "Updated At", "Delete Requested At"}
}

func (s *Secrets) ColMap() map[string]string {
	return map[string]string{
		"Name":                "Name",
		"Region":              "Region",
		"Version":             "Version",
		"Created At":          "Created At",
		"Updated At":          "Updated At",
		"Delete Requested At": "Delete Requested At",
	}
}

func (s *Secrets) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.Secrets))

	for _, secret := range s.Secrets {
		row := map[string]any{
			"Name":       secret.Name,
			"Region":     secret.Region,
			"Version":    secret.Version,
			"Created At": secret.CreatedAt,
			"Updated At": secret.UpdatedAt,
		}
		if secret.DeleteRequestedAt != nil {
			row["Delete Requested At"] = *secret.DeleteRequestedAt
		}
		out = append(out, row)
	}

	return out
}

// Secret is the displayer for a single secret and its key-value pairs.
type Secret struct {
	Secret do.Secret
}

var _ Displayable = &Secret{}

func (s *Secret) JSON(out io.Writer) error {
	return writeJSON(s.Secret, out)
}

func (s *Secret) Cols() []string {
	return []string{"Key", "Value"}
}

func (s *Secret) ColMap() map[string]string {
	return map[string]string{
		"Key":   "Key",
		"Value": "Value",
	}
}

func (s *Secret) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.Secret.Values))

	for key, value := range s.Secret.Values {
		out = append(out, map[string]any{
			"Key":   key,
			"Value": value,
		})
	}

	return out
}

// SecretVersions is the displayer for secret version history.
type SecretVersions struct {
	Versions do.SecretVersions
}

var _ Displayable = &SecretVersions{}

func (s *SecretVersions) JSON(out io.Writer) error {
	return writeJSON(s.Versions, out)
}

func (s *SecretVersions) Cols() []string {
	return []string{"Version", "Created At", "Updated At"}
}

func (s *SecretVersions) ColMap() map[string]string {
	return map[string]string{
		"Version":    "Version",
		"Created At": "Created At",
		"Updated At": "Updated At",
	}
}

func (s *SecretVersions) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.Versions))

	for _, version := range s.Versions {
		out = append(out, map[string]any{
			"Version":    version.Version,
			"Created At": version.CreatedAt,
			"Updated At": version.UpdatedAt,
		})
	}

	return out
}

// SecretWriteResult is the displayer for create/update responses.
type SecretWriteResult struct {
	Result do.SecretWriteResult
}

var _ Displayable = &SecretWriteResult{}

func (s *SecretWriteResult) JSON(out io.Writer) error {
	return writeJSON(s.Result, out)
}

func (s *SecretWriteResult) Cols() []string {
	return []string{"Name", "Region", "Version"}
}

func (s *SecretWriteResult) ColMap() map[string]string {
	return map[string]string{
		"Name":    "Name",
		"Region":  "Region",
		"Version": "Version",
	}
}

func (s *SecretWriteResult) KV() []map[string]any {
	return []map[string]any{{
		"Name":    s.Result.Name,
		"Region":  s.Result.Region,
		"Version": s.Result.Version,
	}}
}
