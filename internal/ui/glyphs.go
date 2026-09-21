/*
Copyright 2018 The Doctl Authors All rights reserved.
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

// Glyph characters are the one definition of doctl's symbol vocabulary, in the
// same way that style.go owns the palette.
const (
	GlyphSuccess   = "✓"
	GlyphFailure   = "✗"
	GlyphWarning   = "!"
	GlyphInfo      = "i"
	GlyphCancelled = "·"
	GlyphPending   = "⟳"
	GlyphBullet    = "•"
	GlyphArrow     = "❯"
	GlyphHint      = "→"
	GlyphEllipsis  = "…"
	GlyphSeparator = "·"
	GlyphNone      = "—"

	// GlyphAsterisk marks a required field on a prompt. It is decoration rather
	// than state, which is why it is not part of Glyphs.
	GlyphAsterisk = "✱"
)

// Glyphs is the symbol vocabulary used to convey state. It exists so that
// meaning survives when color does not, as when output is piped.
type Glyphs struct {
	Success   string
	Failure   string
	Warning   string
	Info      string
	Cancelled string
	Pending   string
	Bullet    string
	Arrow     string
	Ellipsis  string

	// Hint leads the suggestion that follows a diagnostic, pointing at what to
	// do about it. It is a different arrow from Arrow, which prefixes prompts.
	Hint string

	// Separator and None carry no state, but both need an ASCII fallback.
	Separator string
	None      string

	// Spinner holds the animation frames, which must all be the same display
	// width or the line jitters as it cycles.
	Spinner []string
}

// Several glyphs are already ASCII and therefore need no fallback.
var (
	unicodeGlyphs = Glyphs{
		Success:   GlyphSuccess,
		Failure:   GlyphFailure,
		Warning:   GlyphWarning,
		Info:      GlyphInfo,
		Cancelled: GlyphCancelled,
		Pending:   GlyphPending,
		Bullet:    GlyphBullet,
		Arrow:     GlyphArrow,
		Hint:      GlyphHint,
		Ellipsis:  GlyphEllipsis,
		Separator: GlyphSeparator,
		None:      GlyphNone,
		Spinner:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}

	asciiGlyphs = Glyphs{
		Success:   "OK",
		Failure:   "X",
		Warning:   GlyphWarning,
		Info:      GlyphInfo,
		Cancelled: "-",
		Pending:   "o",
		Bullet:    "*",
		Arrow:     ">",
		Hint:      "->",
		Ellipsis:  "...",
		Separator: "|",
		None:      "-",
		Spinner:   []string{"-", "\\", "|", "/"},
	}
)

// Glyphs returns the symbol vocabulary appropriate to the environment.
func (e Env) Glyphs() Glyphs {
	if e.ASCII {
		return asciiGlyphs
	}

	return unicodeGlyphs
}
