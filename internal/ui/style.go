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

package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Next-Gen terminal palette from doctl-nextgen-design.html (Kraken tokens).
// commands/charm/colors.go should mirror these so error/success chrome shares
// one source of truth across interactive and CLI error paths.
var (
	ColorError   = lipgloss.Color("#d74623") // primary-terracotta
	ColorSuccess = lipgloss.Color("#00c483") // primary-green-200
	ColorWarning = lipgloss.Color("#daa549") // tentacle-500
	ColorInfo    = lipgloss.Color("#6FE0ED") // foam-200
	ColorMuted   = lipgloss.Color("#8090a0") // secondary / dim
)

// Style is a presentation helper bound to an Env. Build chrome (errors,
// notices) through Style so colour/glyph policy stays consistent with the
// rest of doctl.
type Style struct {
	env Env
}

// NewStyle wraps env for semantic chrome rendering on Err.
func NewStyle(env Env) Style {
	return Style{env: env}
}

func (s Style) paint(text string, c lipgloss.Color, bold bool) string {
	st := s.env.NewErrStyle().Foreground(c)
	if bold {
		st = st.Bold(true)
	}
	return s.env.SprintErr(st, text)
}

// ErrorLabel returns the leading failure glyph plus "Error:" (ASCII-safe via Glyphs).
func (s Style) ErrorLabel() string {
	return s.paint(s.env.Glyphs().Failure+" Error:", ColorError, true)
}

// Bold emphasises flag names and command paths when Err styling is on.
func (s Style) Bold(text string) string {
	if !s.env.ErrStyle {
		return text
	}
	return s.env.SprintErr(s.env.NewErrStyle().Bold(true), text)
}

// Dim renders secondary hint text.
func (s Style) Dim(text string) string {
	return s.paint(text, ColorMuted, false)
}

// PaintCommand dims the "→ run " / "run " prefix and bolds the command path.
func (s Style) PaintCommand(line string) string {
	lower := strings.ToLower(line)
	for _, prefix := range []string{"run ", "→ run ", "> run "} {
		if strings.HasPrefix(lower, prefix) {
			cmd := strings.TrimSpace(line[len(prefix):])
			return s.Dim(line[:len(prefix)]) + s.Bold(cmd)
		}
	}
	return s.Dim(line)
}
