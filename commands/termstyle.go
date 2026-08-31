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
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

// Next-Gen terminal palette from doctl-nextgen-design.html (Kraken code-block tokens).
var (
	termColorRed    = lipgloss.Color("#d74623") // primary-terracotta · error
	termColorGreen  = lipgloss.Color("#00c483") // primary-green-200 · success
	termColorYellow = lipgloss.Color("#daa549") // tentacle-500 · warning
	termColorCyan   = lipgloss.Color("#6FE0ED") // foam-200 · info / spinner
	termColorDim    = lipgloss.Color("#8090a0") // muted / secondary
)

// terminalStyle implements the doctl Next-Gen terminal design system:
// semantic slots (error/success/warning/info/dim/bold), glyphs with ASCII
// fallback, and color stripped for pipes, NO_COLOR, and CI.
type terminalStyle struct {
	color  bool
	glyphs bool
}

func newTerminalStyle() terminalStyle {
	noColor := os.Getenv("NO_COLOR") != "" || color.NoColor
	ascii := os.Getenv("DOCTL_ASCII") != ""
	ci := os.Getenv("CI") != "" ||
		os.Getenv("CONTINUOUS_INTEGRATION") != "" ||
		os.Getenv("GITHUB_ACTIONS") != "" ||
		os.Getenv("GITLAB_CI") != "" ||
		os.Getenv("CIRCLECI") != ""
	tty := isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())

	useColor := !noColor && !ci && tty
	if useColor {
		// Force 24-bit so design-system hex tokens (#d74623, etc.) render as intended.
		lipgloss.SetColorProfile(termenv.TrueColor)
	} else {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	return terminalStyle{
		color:  useColor,
		glyphs: !ascii, // design: keep symbols when color is stripped
	}
}

func (s terminalStyle) paint(text string, c lipgloss.Color, bold bool) string {
	if !s.color {
		return text
	}
	st := lipgloss.NewStyle().Foreground(c)
	if bold {
		st = st.Bold(true)
	}
	return st.Render(text)
}

func (s terminalStyle) errorGlyph() string {
	if s.glyphs {
		return "✗"
	}
	return "X"
}

func (s terminalStyle) successGlyph() string {
	if s.glyphs {
		return "✓"
	}
	return "OK"
}

func (s terminalStyle) errorLabel() string {
	return s.paint(s.errorGlyph()+" Error:", termColorRed, true)
}

func (s terminalStyle) successLabel() string {
	return s.paint(s.successGlyph(), termColorGreen, false)
}

func (s terminalStyle) warningLabel() string {
	glyph := "!"
	if !s.glyphs {
		glyph = "Warning:"
		return s.paint(glyph, termColorYellow, true)
	}
	return s.paint(glyph+" Warning:", termColorYellow, true)
}

func (s terminalStyle) bold(text string) string {
	if !s.color {
		return text
	}
	return lipgloss.NewStyle().Bold(true).Render(text)
}

func (s terminalStyle) dim(text string) string {
	return s.paint(text, termColorDim, false)
}

func (s terminalStyle) cyan(text string) string {
	return s.paint(text, termColorCyan, false)
}

func (s terminalStyle) green(text string) string {
	return s.paint(text, termColorGreen, false)
}

func (s terminalStyle) yellow(text string) string {
	return s.paint(text, termColorYellow, false)
}

func (s terminalStyle) red(text string) string {
	return s.paint(text, termColorRed, false)
}

func (s terminalStyle) paintCommand(line string) string {
	// Bold doctl command snippets after "run " / "→ run ".
	lower := strings.ToLower(line)
	for _, prefix := range []string{"run ", "→ run "} {
		if strings.HasPrefix(lower, prefix) {
			cmd := strings.TrimSpace(line[len(prefix):])
			return s.dim(line[:len(prefix)]) + s.bold(cmd)
		}
	}
	return s.dim(line)
}
