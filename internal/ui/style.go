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

// The semantic palette, named as slots in the terminal's own 16 colours
// rather than as exact values. This is the one definition of doctl's colours:
// commands/charm/colors.go reads them from here rather than restating them, so
// error and success chrome cannot drift between the interactive and CLI paths.
//
// Slots rather than hex is what the design system asks for, and the reason is
// contrast: every terminal renders green a little differently, and a fixed
// value gambles that it stays readable against whatever background the user
// picked. Deferring to the slot means the user's own theme decides, which is
// the only way doctl can be legible on all of them. It also keeps DigitalOcean
// blue out of the palette, which is deliberate for the same reason.
//
// The values are the ANSI slots, not brand tokens: red, green, yellow, cyan,
// and bright black for dim.
var (
	ColorError   lipgloss.TerminalColor = lipgloss.ANSIColor(1)
	ColorSuccess lipgloss.TerminalColor = lipgloss.ANSIColor(2)
	ColorWarning lipgloss.TerminalColor = lipgloss.ANSIColor(3)
	ColorInfo    lipgloss.TerminalColor = lipgloss.ANSIColor(6)
	ColorMuted   lipgloss.TerminalColor = lipgloss.ANSIColor(8)
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

// paint colours text without touching its weight.
//
// Weight is deliberately not combined with the palette. The palette names
// slots in the terminal's own 16 colours, and most terminals render bold plus
// one of the eight base colours as that colour's bright variant - Terminal.app
// and iTerm2 both do by default. A bold label and an unbolded value in the
// same colour would then be two visibly different reds. The design system asks
// for neither: bold is its own meaning, Highlight, which carries no colour.
func (s Style) paint(text string, c lipgloss.TerminalColor) string {
	return s.env.SprintErr(s.env.NewErrStyle().Foreground(c), text)
}

// ErrorLabel returns the failure label, led by its glyph.
//
// The glyph is kept even when the stream is redirected. Stripping colour and
// keeping the symbol and the word is what the design system asks for, on the
// grounds that the symbol is half of what carries the meaning once the colour
// is gone; only the ANSI codes are dropped. Terminals that cannot render the
// symbol get the ASCII fallback rather than nothing.
func (s Style) ErrorLabel() string {
	return s.paint(s.env.Glyphs().Failure+" Error:", ColorError)
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
	return s.paint(text, ColorMuted)
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
