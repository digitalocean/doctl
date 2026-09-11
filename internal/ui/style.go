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

// The semantic palette, and the one definition of doctl's colors. They are
// the colors doctl's interactive surfaces already use, so that a table, a
// prompt and an error all name the same state the same way. termenv
// downsamples them for terminals that cannot show truecolor.
const (
	// ColorError is the red a failure is named in.
	ColorError lipgloss.Color = "#ff6188"
	// ColorSuccess is the green of a resource that reached its desired state.
	ColorSuccess lipgloss.Color = "#04b575"
	// ColorWarning is the yellow that reads as attention without alarm.
	ColorWarning lipgloss.Color = "#ffd866"
	// ColorInfo is the blue that names a resource.
	ColorInfo lipgloss.Color = "#2ea0f9"
	// ColorMuted is xterm 241, dim enough to recede behind a value.
	ColorMuted lipgloss.Color = "241"
)

// Style is a presentation helper bound to an Env, for chrome written to Err.
type Style struct {
	env Env
}

// NewStyle wraps env for semantic chrome rendering on Err.
func NewStyle(env Env) Style {
	return Style{env: env}
}

func (s Style) paint(text string, c lipgloss.TerminalColor, bold bool) string {
	style := s.env.NewErrStyle().Foreground(c)
	if bold {
		style = style.Bold(true)
	}

	return s.env.SprintErr(style, text)
}

// ErrorLabel returns the failure label, led by its glyph on a terminal.
func (s Style) ErrorLabel() string {
	label := "Error:"
	if s.env.ErrTTY {
		label = s.env.Glyphs().Failure + " " + label
	}

	return s.paint(label, ColorError, true)
}

// Bold emphasizes flag names and command paths when Err styling is on.
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

// SuccessLine reports that a command finished, glyph and message painted
// together so that completion reads as one mark rather than a labelled
// diagnostic. The glyph leads only on a terminal, as ErrorLabel's does.
func (s Style) SuccessLine(msg string) string {
	if s.env.ErrTTY {
		msg = s.env.Glyphs().Success + " " + msg
	}

	return s.paint(msg, ColorSuccess, false)
}

// Hint renders the suggestion that follows a diagnostic, led by its arrow.
func (s Style) Hint(text string) string {
	return s.PaintCommand(s.env.Glyphs().Hint + " " + text)
}

// PaintCommand dims the lead-in of a suggestion and bolds the command path, so
// that the part worth copying stands out from the sentence around it.
func (s Style) PaintCommand(line string) string {
	for _, prefix := range []string{s.env.Glyphs().Hint + " run ", "run "} {
		if len(line) >= len(prefix) && strings.EqualFold(line[:len(prefix)], prefix) {
			cmd := strings.TrimSpace(line[len(prefix):])
			return s.Dim(line[:len(prefix)]) + s.Bold(cmd)
		}
	}

	return s.Dim(line)
}
