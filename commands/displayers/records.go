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
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/digitalocean/doctl/internal/ui"
)

// records draws each resource as a headline naming it, followed by its
// remaining fields labeled on lines indented beneath. It is the layout for a
// table the terminal cannot hold: every column is present, in the order the
// displayer declared it, and no value is cut. What gives way is alignment,
// which on a wide table spends the window on padding.
func (r renderer) records(headers []string, rows [][]string, tones toneTable) {
	none := r.env.Glyphs().None
	count := columnCount(headers, rows)
	headline := headlineColumns(headers)

	for i, row := range rows {
		if i > 0 {
			r.buf.WriteString("\n")
		}

		row = withPlaceholders(row, count, none)
		r.headline(row, headline, tones.rowTone(i))
		r.fields(headers, row, headline, tones.painter(i, ui.ToneMuted))
	}
}

// headline writes the line that opens a record: a glyph carrying the resource's
// state, then the values that name it, accented so the eye can find them.
func (r renderer) headline(row []string, headline []int, tone ui.Tone) {
	// A record with no state still needs its glyph, and muted claims nothing.
	if tone == ui.ToneNone {
		tone = ui.ToneMuted
	}

	named := make([]string, 0, len(headline))
	for _, col := range headline {
		if col < len(row) {
			named = append(named, r.env.SprintTone(ui.ToneIdentifier, row[col]))
		}
	}

	r.buf.WriteString(r.env.SprintTone(tone, r.env.Glyphs().Bullet) + " " +
		strings.Join(named, fieldSeparator(r.env)) + "\n")
}

// fields writes the fields the headline did not, as label and value pairs,
// packing as many onto each line as the terminal holds. A value too long for a
// line of its own is broken at the points isBreak allows, so compound values
// wrap where they can be read; a value with no such point overruns.
func (r renderer) fields(headers, row []string, headline []int, paint painter) {
	indent := strings.Repeat(" ", recordIndent)
	hanging := strings.Repeat(" ", 2*recordIndent)

	separator := fieldSeparator(r.env)
	separatorWidth := ansi.StringWidth(separator)

	prefix, line, used := indent, "", 0
	flush := func(next string) {
		if used > 0 {
			r.buf.WriteString(prefix + line + "\n")
		}
		prefix, line, used = next, "", 0
	}
	// room reports whether n more cells fit the line being built.
	room := func(n int) bool {
		return r.env.DataWidth <= 0 || len(prefix)+used+n <= r.env.DataWidth
	}
	add := func(joiner string, joinerWidth, width int, text string) {
		line += joiner + text
		used += joinerWidth + width
	}

	for _, field := range recordFields(headers, row, headline, r.env, paint) {
		joiner, joinerWidth := separator, separatorWidth
		if used == 0 {
			joiner, joinerWidth = "", 0
		}

		// Breaking between fields is preferred to breaking inside a value, so a
		// field that will not share this line is given one of its own.
		switch {
		case room(joinerWidth + field.width):
			add(joiner, joinerWidth, field.width, field.text)
		case r.env.DataWidth <= 0 || recordIndent+field.width <= r.env.DataWidth:
			flush(indent)
			add("", 0, field.width, field.text)
		default:
			flush(indent)
			for _, part := range field.parts {
				joiner, joinerWidth := part.joiner, ansi.StringWidth(part.joiner)
				if used == 0 {
					joiner, joinerWidth = "", 0
				}

				if used > 0 && !room(joinerWidth+part.width) {
					flush(hanging)
					joiner, joinerWidth = "", 0
				}

				add(joiner, joinerWidth, part.width, part.text)
			}

			// The hanging indent signals continuation and nothing else, so the
			// next field starts a fresh line.
			flush(indent)
		}
	}

	flush("")
}

// recordField is one labeled field of a record, held whole and in parts.
type recordField struct {
	text  string
	width int
	parts []recordPart
}

// recordPart is a piece of a field that can start a line, with its joiner.
type recordPart struct {
	text   string
	width  int
	joiner string
}

// recordFields labels and paints each field the headline did not take, breaking
// its value in case a line cannot hold it whole.
func recordFields(headers, row []string, headline []int, env ui.Env, paint painter) []recordField {
	label := labelStyle(env)

	var fields []recordField
	for col, cell := range row {
		if slices.Contains(headline, col) {
			continue
		}

		lead := ""
		if col < len(headers) {
			lead = strings.ToLower(headers[col]) + " "
		}

		painted := env.Sprint(label, lead)
		field := recordField{text: painted, width: ansi.StringWidth(lead)}

		for i, part := range breakValue(cell) {
			text := paintCell(paint, col, part.text)
			width := ansi.StringWidth(part.text)

			field.text += part.joiner + text
			field.width += ansi.StringWidth(part.joiner) + width

			// The label belongs to the part that opens the field.
			if i == 0 {
				field.parts = append(field.parts, recordPart{
					text:  painted + text,
					width: ansi.StringWidth(lead) + width,
				})

				continue
			}

			field.parts = append(field.parts, recordPart{
				text: text, width: width, joiner: part.joiner,
			})
		}

		fields = append(fields, field)
	}

	return fields
}

// valuePart is one break-delimited part of a value, with what rejoins it.
type valuePart struct {
	text   string
	joiner string
}

// breakValue splits a value into the parts isBreak allows it to be cut into.
func breakValue(value string) []valuePart {
	var parts []valuePart
	joiner, start := "", 0

	for i, r := range value {
		if !isBreak(r) {
			continue
		}

		// A comma is kept with its part, a trailing comma saying the value
		// carries on. A space is dropped into the joiner, which puts it back.
		end, dropped := i, value[i:i+utf8.RuneLen(r)]
		if r == ',' {
			end, dropped = i+utf8.RuneLen(r), ""
		}

		if text := value[start:end]; text != "" {
			parts = append(parts, valuePart{text: text, joiner: joiner})
			joiner = ""
		}

		// Accumulated rather than replaced, so a run of separators is restored
		// whole when the parts share a line.
		joiner += dropped
		start = i + utf8.RuneLen(r)
	}

	if start < len(value) {
		parts = append(parts, valuePart{text: value[start:], joiner: joiner})
	}

	if parts == nil {
		return []valuePart{{text: value}}
	}

	return parts
}

// headlineColumns are the columns that name a resource: its name and its ID,
// which are what a list is read to find. The name leads whichever order the
// displayer declared them in, being what a reader recognizes. Both are matched
// exactly, so an ID belonging to another resource stays among the fields. A
// resource declaring neither is headlined by its first column.
func headlineColumns(headers []string) []int {
	name, id := -1, -1
	for i, header := range headers {
		switch normalizeColumn(header) {
		case "name":
			if name < 0 {
				name = i
			}
		case "id":
			if id < 0 {
				id = i
			}
		}
	}

	var headline []int
	for _, col := range []int{name, id} {
		if col >= 0 {
			headline = append(headline, col)
		}
	}

	if headline == nil {
		return []int{0}
	}

	return headline
}

// labelStyle draws a field's label faint as well as muted, so it sits behind
// its own value.
func labelStyle(env ui.Env) lipgloss.Style {
	return env.NewStyle().Foreground(ui.ColorMuted).Faint(true)
}

// fieldSeparator divides the fields sharing a line, drawn as a label is.
func fieldSeparator(env ui.Env) string {
	return env.Sprint(labelStyle(env), " "+env.Glyphs().Separator+" ")
}

// withPlaceholders fills a row out to count cells, glyphing the fields with no value.
func withPlaceholders(row []string, count int, none string) []string {
	filled := make([]string, count)
	for i := range filled {
		filled[i] = none
		if i < len(row) && strings.TrimSpace(row[i]) != "" {
			filled[i] = row[i]
		}
	}

	return filled
}
