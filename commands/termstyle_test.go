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
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestTerminalStyleUsesDesignHexColors(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	style := terminalStyle{color: true, glyphs: true}

	label := style.errorLabel()
	// #d74623 => rgb(215, 70, 35)
	if !strings.Contains(label, "38;2;215;70;35") {
		t.Fatalf("error label missing design red #d74623 RGB; got %q", label)
	}
	if !strings.Contains(label, "✗ Error:") {
		t.Fatalf("error label missing glyph/text; got %q", label)
	}

	dim := style.dim("hint")
	// #8090a0 => rgb(128, 144, 160)
	if !strings.Contains(dim, "38;2;128;144;160") {
		t.Fatalf("dim missing design #8090a0 RGB; got %q", dim)
	}

	green := style.green("ok")
	// #00c483 ≈ rgb(0, 196, 131); termenv may round slightly.
	if !strings.Contains(green, "38;2;0;") || !strings.Contains(green, ";131m") {
		t.Fatalf("green missing design #00c483 RGB; got %q", green)
	}
}

func TestFlagValidationDisplayIncludesDesignRed(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	err := &FlagValidationError{
		Command: "doctl compute droplet create",
		Issues: []FlagIssue{
			{Flag: "size", Problem: "is required but was not set", Purpose: "Droplet size", Hint: "run doctl compute size list"},
		},
	}
	// Force styled path by calling format with color enabled.
	out := err.format(terminalStyle{color: true, glyphs: true})
	if !strings.Contains(out, "38;2;215;70;35") {
		t.Fatalf("expected design red in display output; got %q", out)
	}
}
