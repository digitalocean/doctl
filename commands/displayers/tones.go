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

package displayers

import (
	"strings"
	"unicode"

	"github.com/digitalocean/doctl/internal/ui"
)

// stateWords are the column names that hold the state of a resource. Tone needs
// both a state-named column and a value the vocabulary knows: state words also
// appear in prose columns like Message and Failure Reason, which stay plain.
var stateWords = []string{"status", "state", "phase", "health", "verdict"}

// painter styles one cell of a row, after it is measured and truncated.
type painter func(col int, cell string) string

// headerPainter emphasizes the header row, separating the labels from the data.
func headerPainter(env ui.Env) painter {
	if !env.Style {
		return nil
	}

	header := env.NewStyle().Bold(true)

	return func(_ int, cell string) string {
		return env.Sprint(header, cell)
	}
}

// paintCell applies paint when there is any, so callers repeat no nil check.
func paintCell(paint painter, col int, cell string) string {
	if paint == nil {
		return cell
	}

	return paint(col, cell)
}

// toneTable is one classification of a table, empty when styling is forbidden.
type toneTable struct {
	env   ui.Env
	cells [][]ui.Tone

	// state is the column a record's state is read from, or -1 where none.
	state int
}

// newToneTable classifies the table, unless nothing is going to be painted.
func newToneTable(env ui.Env, item Displayable, cols []string, kv []map[string]any) toneTable {
	table := toneTable{env: env, state: -1}
	if !env.Style {
		return table
	}

	table.cells = tableTones(item, cols, kv)
	for i, col := range cols {
		if isStateColumn(col) {
			table.state = i
			break
		}
	}

	return table
}

// painter returns a painter for one row. fallback tones the unclassified cells:
// ui.ToneNone leaves them plain, ui.ToneMuted sinks them behind a headline.
func (t toneTable) painter(row int, fallback ui.Tone) painter {
	if t.cells == nil {
		return nil
	}

	return func(col int, cell string) string {
		tone := t.cells[row][col]
		if tone == ui.ToneNone {
			tone = fallback
		}

		return t.env.SprintTone(tone, cell)
	}
}

// rowTone is the tone summarizing one row, taken from its state column. It is
// what the glyph opening a record carries.
func (t toneTable) rowTone(row int) ui.Tone {
	if t.cells == nil || t.state < 0 {
		return ui.ToneNone
	}

	return t.cells[row][t.state]
}

// tableTones classifies every cell, giving the displayer the final say.
func tableTones(item Displayable, cols []string, kv []map[string]any) [][]ui.Tone {
	toned, overrides := item.(Toned)

	// What a column holds is a property of the table, so decide it once.
	stateCols := make([]bool, len(cols))
	identifierCols := make([]bool, len(cols))
	for i, col := range cols {
		stateCols[i] = isStateColumn(col)
		identifierCols[i] = isIdentifierColumn(col)
	}

	tones := make([][]ui.Tone, 0, len(kv))
	for _, r := range kv {
		row := make([]ui.Tone, len(cols))
		for i, col := range cols {
			if overrides {
				if tone, decided := toned.ColTone(col, r[col]); decided {
					row[i] = tone
					continue
				}
			}

			// An empty cell is left alone, having no name in it to find.
			if identifierCols[i] {
				if s, ok := r[col].(string); ok && s != "" {
					row[i] = ui.ToneIdentifier
				}

				continue
			}

			if !stateCols[i] {
				continue
			}

			// Booleans are left alone: polarity belongs to the column, not the
			// value - true is healthy under Advertised, unhealthy under Disabled.
			if s, ok := r[col].(string); ok {
				row[i], _ = ui.ToneFor(s)
			}
		}
		tones = append(tones, row)
	}

	return tones
}

// isStateColumn reports whether col names the state of a resource.
func isStateColumn(col string) bool {
	normalized := normalizeColumn(col)

	for _, word := range stateWords {
		if normalized == word || strings.HasSuffix(normalized, word) {
			return true
		}
	}

	return false
}

// isIdentifierColumn reports whether col holds the name of the resource the row
// is about. The match is exact where isStateColumn takes a suffix, since a
// trailing "name" usually belongs to another resource.
func isIdentifierColumn(col string) bool {
	return normalizeColumn(col) == "name"
}

// normalizeColumn drops separators and case, matching "Health Status" to "HealthStatus".
func normalizeColumn(col string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '_', '-':
			return -1
		default:
			return unicode.ToLower(r)
		}
	}, col)
}
