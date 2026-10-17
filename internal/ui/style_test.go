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

package ui_test

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/muesli/termenv"
)

// paintedIn is the foreground sequence the palette color c renders as, so that
// these tests pin chrome to the palette rather than to whatever value the
// palette currently holds.
func paintedIn(c lipgloss.Color) string {
	return termenv.TrueColor.Color(string(c)).Sequence(false)
}

// TestStyleUsesThePalette pins each piece of chrome to a palette color rather
// than to one of its own, and to the weight that distinguishes them: a label
// is bold, the secondary text under it is not. The profile is forced to
// TrueColor so that the exact color reaches the sequence, where drift shows.
func TestStyleUsesThePalette(t *testing.T) {
	env := ui.Detect(io.Discard, io.Discard, ui.WithProfile(termenv.TrueColor), ui.WithASCII(false))
	style := ui.NewStyle(env)

	label := style.ErrorLabel()
	if !strings.Contains(label, "\x1b[1;"+paintedIn(ui.ColorError)+"m") {
		t.Fatalf("error label not painted bold in the error color; got %q", label)
	}
	if !strings.Contains(label, "Error:") {
		t.Fatalf("error label missing text; got %q", label)
	}

	// Bold is what marks a label, so it stays off everything else. Dim is
	// secondary text: same palette, normal weight.
	dim := style.Dim("hint")
	if !strings.Contains(dim, "\x1b["+paintedIn(ui.ColorMuted)+"m") {
		t.Fatalf("dim not painted in the muted color; got %q", dim)
	}
	if strings.Contains(dim, "\x1b[1;") {
		t.Fatalf("dim combines bold with the color; got %q", dim)
	}
}

// TestSuccessLineIsPaintedWhole guards the closing line: the tick and the
// sentence are one green mark, not a label introducing a plain message, so no
// escape sequence may fall between them.
func TestSuccessLineIsPaintedWhole(t *testing.T) {
	env := ui.Detect(io.Discard, io.Discard, ui.WithProfile(termenv.TrueColor), ui.WithASCII(false))
	env.ErrTTY = true

	line := ui.NewStyle(env).SuccessLine("Command completed successfully")

	want := "\x1b[" + paintedIn(ui.ColorSuccess) + "m" +
		ui.GlyphSuccess + " Command completed successfully\x1b[0m"
	if line != want {
		t.Fatalf("got %q, want %q", line, want)
	}
}

// TestSuccessLineFollowsTheStream matches ErrorLabel: the glyph is for a
// screen, so a redirected stderr gets the sentence and nothing else.
func TestSuccessLineFollowsTheStream(t *testing.T) {
	line := ui.NewStyle(ui.Env{}).SuccessLine("Command completed successfully")

	if line != "Command completed successfully" {
		t.Fatalf("got %q, want the bare sentence", line)
	}
}

// TestErrorLabelGlyphFollowsTheStream guards the two halves of the label
// separately: the glyph belongs to a screen and the color belongs to a stream
// that permits styling. A redirected stderr therefore reads "Error:", which is
// what the scripts and tests matching doctl's errors have always matched on.
func TestErrorLabelGlyphFollowsTheStream(t *testing.T) {
	t.Run("a terminal is led by the glyph", func(t *testing.T) {
		label := ui.NewStyle(ui.Env{ErrTTY: true}).ErrorLabel()

		if want := ui.GlyphFailure + " Error:"; label != want {
			t.Fatalf("got %q, want %q", label, want)
		}
	})

	t.Run("a redirected stream gets the word alone", func(t *testing.T) {
		label := ui.NewStyle(ui.Env{}).ErrorLabel()

		if label != "Error:" {
			t.Fatalf("got %q, want %q", label, "Error:")
		}
	})

	// NO_COLOR on a terminal drops the ANSI codes and nothing else: the
	// symbol is still legible, and it is half of what carries the meaning once
	// the color is gone.
	t.Run("color and glyph are decided separately", func(t *testing.T) {
		label := ui.NewStyle(ui.Env{ErrTTY: true, ErrStyle: false}).ErrorLabel()

		if !strings.Contains(label, ui.GlyphFailure) {
			t.Fatalf("unstyled terminal lost the glyph; got %q", label)
		}
		if strings.Contains(label, "\x1b[") {
			t.Fatalf("unstyled terminal gained escape sequences; got %q", label)
		}
	})
}
