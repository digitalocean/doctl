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

	"github.com/digitalocean/doctl/internal/ui"
	"github.com/muesli/termenv"
)

// TestStyleUsesTerminalColorSlots pins the palette to the terminal's own 16
// colours. The profile is forced to TrueColor precisely because that is where
// a fixed hex would survive: on a 16-colour terminal lipgloss would downsample
// it and the difference would not show. A truecolor sequence appearing here
// means doctl has started dictating a value instead of naming a slot, and the
// user's theme no longer decides whether the result is readable.
func TestStyleUsesTerminalColorSlots(t *testing.T) {
	env := ui.Detect(io.Discard, io.Discard, ui.WithProfile(termenv.TrueColor), ui.WithASCII(false))
	style := ui.NewStyle(env)

	label := style.ErrorLabel()
	if !strings.Contains(label, "\x1b[31m") {
		t.Fatalf("error label not painted in the red slot; got %q", label)
	}
	if !strings.Contains(label, "Error:") {
		t.Fatalf("error label missing text; got %q", label)
	}

	// Colour is never combined with weight. Bold plus a base ANSI colour
	// renders bright on most terminals, so a bold label would be a different
	// red from a table cell painted in the same slot.
	if strings.Contains(label, "\x1b[1;") {
		t.Fatalf("error label combines bold with the colour slot; got %q", label)
	}

	dim := style.Dim("hint")
	if !strings.Contains(dim, "\x1b[90m") {
		t.Fatalf("dim not painted in the bright black slot; got %q", dim)
	}

	for _, got := range []string{label, style.Dim("hint"), style.Bold("flag")} {
		if strings.Contains(got, "38;2;") {
			t.Fatalf("palette emitted a fixed truecolor value instead of a slot; got %q", got)
		}
	}
}

// TestErrorLabelGlyphFollowsTheStream guards the two halves of the label
// separately: the glyph belongs to a screen and the colour belongs to a stream
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

	// --color=never on a terminal drops the ANSI codes and nothing else: the
	// symbol is still legible, and it is half of what carries the meaning once
	// the colour is gone.
	t.Run("colour and glyph are decided separately", func(t *testing.T) {
		label := ui.NewStyle(ui.Env{ErrTTY: true, ErrStyle: false}).ErrorLabel()

		if !strings.Contains(label, ui.GlyphFailure) {
			t.Fatalf("unstyled terminal lost the glyph; got %q", label)
		}
		if strings.Contains(label, "\x1b[") {
			t.Fatalf("unstyled terminal gained escape sequences; got %q", label)
		}
	})
}
