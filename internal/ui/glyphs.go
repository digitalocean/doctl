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

// Glyphs is the symbol vocabulary used to convey state. It exists so that
// meaning survives when colour does not: a status rendered as a glyph plus a
// word still reads correctly when piped, when NO_COLOR is set, or on a
// terminal that cannot render the palette.
type Glyphs struct {
	Success  string
	Failure  string
	Warning  string
	Pending  string
	Bullet   string
	Arrow    string
	Ellipsis string

	// Spinner holds the animation frames, which must all be the same display
	// width or the line will jitter as it cycles.
	Spinner []string
}

var (
	unicodeGlyphs = Glyphs{
		Success:  "✔",
		Failure:  "✘",
		Warning:  "✱",
		Pending:  "◌",
		Bullet:   "●",
		Arrow:    "❯",
		Ellipsis: "…",
		Spinner:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}

	asciiGlyphs = Glyphs{
		Success:  "+",
		Failure:  "x",
		Warning:  "!",
		Pending:  "o",
		Bullet:   "*",
		Arrow:    ">",
		Ellipsis: "...",
		Spinner:  []string{"|", "/", "-", "\\"},
	}
)

// Glyphs returns the symbol vocabulary appropriate to the environment.
func (e Env) Glyphs() Glyphs {
	if e.ASCII {
		return asciiGlyphs
	}

	return unicodeGlyphs
}
