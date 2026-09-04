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

// Glyph characters are the one definition of doctl's symbol vocabulary, in
// the same way that style.go owns the palette. commands/charm/text renders
// these rather than restating them, so the glyph on an interactive prompt and
// the glyph closing a --wait cannot drift apart.
const (
	GlyphSuccess   = "✓"
	GlyphFailure   = "✗"
	GlyphWarning   = "!"
	GlyphInfo      = "i"
	GlyphCancelled = "·"
	GlyphPending   = "⟳"
	GlyphBullet    = "•"
	GlyphArrow     = "❯"
	GlyphEllipsis  = "…"

	// GlyphAsterisk marks a required field on an interactive prompt. It is
	// decoration rather than state, which is why it is not part of Glyphs and
	// why it did not follow GlyphWarning to the "!" the design system asks for.
	GlyphAsterisk = "✱"
)

// Glyphs is the symbol vocabulary used to convey state. It exists so that
// meaning survives when colour does not: a status rendered as a glyph plus a
// word still reads correctly when piped, when NO_COLOR is set, or on a
// terminal that cannot render the palette.
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

	// Spinner holds the animation frames, which must all be the same display
	// width or the line will jitter as it cycles.
	Spinner []string
}

// Several glyphs are already ASCII and therefore need no fallback: the design
// system leaves their ASCII column empty for exactly that reason.
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
		Ellipsis:  GlyphEllipsis,
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
		Ellipsis:  "...",
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
