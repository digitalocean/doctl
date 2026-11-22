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
	"fmt"
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

// SandboxCommand displays the commands running in a sandbox.
type SandboxCommand struct {
	Commands do.SandboxCommands
}

var _ Displayable = &SandboxCommand{}

func (s *SandboxCommand) JSON(out io.Writer) error {
	return writeJSON(s.Commands, out)
}

func (s *SandboxCommand) Cols() []string {
	return []string{"PID", "Command", "Cwd"}
}

func (s *SandboxCommand) ColMap() map[string]string {
	return map[string]string{
		"PID":     "PID",
		"Command": "Command",
		"Cwd":     "Working Directory",
	}
}

func (s *SandboxCommand) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.Commands))
	for _, c := range s.Commands {
		out = append(out, map[string]any{
			"PID":     c.PID,
			"Command": c.Cmd,
			"Cwd":     c.Cwd,
		})
	}
	return out
}

// SandboxTemplate displays sandbox templates.
type SandboxTemplate struct {
	Templates do.SandboxTemplates
}

var _ Displayable = &SandboxTemplate{}

func (s *SandboxTemplate) JSON(out io.Writer) error {
	return writeJSON(s.Templates, out)
}

func (s *SandboxTemplate) Cols() []string {
	return []string{"ID", "Name", "Status", "BaseTemplate", "Image", "Created", "Updated"}
}

func (s *SandboxTemplate) ColMap() map[string]string {
	return map[string]string{
		"ID":           "ID",
		"Name":         "Name",
		"Status":       "Status",
		"BaseTemplate": "Base Template",
		"Image":        "Image",
		"Created":      "Created At",
		"Updated":      "Updated At",
	}
}

func (s *SandboxTemplate) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.Templates))
	for _, t := range s.Templates {
		out = append(out, map[string]any{
			"ID":           t.ID,
			"Name":         t.Name,
			"Status":       formatSandboxStatus(string(t.Status), "TEMPLATE_STATUS_"),
			"BaseTemplate": baseTemplateOf(t.Spec),
			"Image":        FormatSandboxImage(imageOf(t.Spec)),
			"Created":      t.CreatedAt.UTC().Format(time.RFC3339),
			"Updated":      t.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

// SandboxTemplateBuild displays the builds of a sandbox template.
type SandboxTemplateBuild struct {
	Builds do.SandboxTemplateBuilds
}

var _ Displayable = &SandboxTemplateBuild{}

func (s *SandboxTemplateBuild) JSON(out io.Writer) error {
	return writeJSON(s.Builds, out)
}

func (s *SandboxTemplateBuild) Cols() []string {
	return []string{"ID", "TemplateID", "Status", "Error", "Created", "Updated"}
}

func (s *SandboxTemplateBuild) ColMap() map[string]string {
	return map[string]string{
		"ID":         "ID",
		"TemplateID": "Template ID",
		"Status":     "Status",
		"Error":      "Error",
		"Created":    "Created At",
		"Updated":    "Updated At",
	}
}

func (s *SandboxTemplateBuild) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.Builds))
	for _, b := range s.Builds {
		out = append(out, map[string]any{
			"ID":         b.ID,
			"TemplateID": b.TemplateID,
			"Status":     formatSandboxStatus(string(b.Status), "TEMPLATE_BUILD_STATUS_"),
			"Error":      b.Error,
			"Created":    b.CreatedAt.UTC().Format(time.RFC3339),
			"Updated":    b.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

// FormatSandboxImage writes an image as registry/repository:tag or
// registry/repository@digest. An empty or missing image is an empty string.
func FormatSandboxImage(img *godo.SandboxImageSource) string {
	if img == nil {
		return ""
	}
	ref := img.Repository
	if img.Registry != "" {
		ref = img.Registry + "/" + ref
	}
	switch {
	case img.Digest != "":
		return ref + "@" + img.Digest
	case img.Tag != "":
		return fmt.Sprintf("%s:%s", ref, img.Tag)
	}
	return ref
}

func imageOf(spec *godo.SandboxTemplateSpec) *godo.SandboxImageSource {
	if spec == nil {
		return nil
	}
	return spec.Image
}

func baseTemplateOf(spec *godo.SandboxTemplateSpec) string {
	if spec == nil {
		return ""
	}
	return spec.BaseTemplate
}

func formatSandboxStatus(status, prefix string) string {
	return strings.ToLower(strings.TrimPrefix(status, prefix))
}

// FormatSandboxTemplateStatus shortens an API status such as
// TEMPLATE_STATUS_READY to ready.
func FormatSandboxTemplateStatus(status godo.SandboxTemplateStatus) string {
	return formatSandboxStatus(string(status), "TEMPLATE_STATUS_")
}
