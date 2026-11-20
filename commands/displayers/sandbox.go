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
	"strings"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// Sandbox displays sandboxes.
type Sandbox struct {
	Sandboxes do.Sandboxes
}

var _ Displayable = &Sandbox{}

func (s *Sandbox) JSON(out io.Writer) error {
	return writeJSON(s.Sandboxes, out)
}

func (s *Sandbox) Cols() []string {
	return []string{"ID", "Template", "State", "Size", "Created"}
}

func (s *Sandbox) ColMap() map[string]string {
	return map[string]string{
		"ID":       "ID",
		"Template": "Template",
		"State":    "State",
		"Size":     "Size",
		"Created":  "Created At",
	}
}

func (s *Sandbox) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.Sandboxes))
	for _, sb := range s.Sandboxes {
		out = append(out, map[string]any{
			"ID":       sb.ID,
			"Template": sb.Template,
			"State":    FormatSandboxState(sb.State),
			"Size":     sb.SizeSlug,
			"Created":  sb.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

// FormatSandboxState turns an API state such as SANDBOX_STATE_RUNNING into
// the short form shown in tables: running. JSON output keeps the API value.
func FormatSandboxState(state godo.SandboxState) string {
	return strings.ToLower(strings.TrimPrefix(string(state), "SANDBOX_STATE_"))
}
