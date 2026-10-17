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

	"github.com/charmbracelet/x/ansi"

	"github.com/digitalocean/doctl/internal/ui"
)

// nextStepHeading titles the card's footer. It is a heading rather than a field
// label so the command below it can run the width of the card.
const nextStepHeading = "Next step"

// box draws the table inside box rules, with the header separated from the
// values by a rule of its own. Cells are measured and truncated exactly as in
// the plain layout, so a boxed table and a piped one show the same values.
func (r renderer) box(headers []string, rows [][]string, widths []int, head painter, rowPaint func(int) painter) {
	if len(widths) == 0 {
		return
	}

	frame := frameStyle(r.env)
	vertical := r.vertical()

	rule := func(left, join, right string) {
		segments := make([]string, len(widths))
		for i, w := range widths {
			segments[i] = strings.Repeat(r.border.Top, w+2*cellPad)
		}

		r.buf.WriteString(r.env.Sprint(frame, left+strings.Join(segments, join)+right))
		r.buf.WriteString("\n")
	}

	row := func(cells []string, paint painter) {
		pad := strings.Repeat(" ", cellPad)

		r.buf.WriteString(vertical)
		for i, width := range widths {
			var cell string
			if i < len(cells) {
				cell = cells[i]
			}

			cell = r.fit(cell, width)
			fill := strings.Repeat(" ", width-ansi.StringWidth(cell))

			r.buf.WriteString(pad + paintCell(paint, i, cell) + fill + pad + vertical)
		}
		r.buf.WriteString("\n")
	}

	rule(r.border.TopLeft, r.border.MiddleTop, r.border.TopRight)
	if headers != nil {
		row(headers, head)
		rule(r.border.MiddleLeft, r.border.Middle, r.border.MiddleRight)
	}
	for i, cells := range rows {
		row(cells, rowPaint(i))
	}
	rule(r.border.BottomLeft, r.border.MiddleBottom, r.border.BottomRight)
}

// card draws one resource as a card, a label and a value to a line, which keeps
// a resource with more fields than the terminal has columns readable. Labels
// take the identifier color and bold, so the eye can run down them to find a
// field. Empty values are dropped; the table layout and every pipe list them.
func (r renderer) card(labels, values []string, next string, paint painter) {
	fields := cardFields(labels, values)
	if len(fields) == 0 {
		return
	}

	labelWidth := 0
	for _, f := range fields {
		if w := ansi.StringWidth(f.label); w > labelWidth {
			labelWidth = w
		}
	}

	// Everything but the values: two rules, both margins, and the gap.
	chrome := 2 + 2*cardPad + cardGap + labelWidth
	valueWidth := r.cardValueWidth(fields, next, chrome, labelWidth)

	frame := frameStyle(r.env)
	labelStyle := r.env.NewStyle().Foreground(ui.ColorInfo).Bold(true)
	margin := strings.Repeat(" ", cardPad)
	gap := strings.Repeat(" ", cardGap)
	vertical := r.vertical()
	inner := chrome - 2 + valueWidth

	rule := func(left, fill, right string) {
		r.buf.WriteString(r.env.Sprint(frame, left+strings.Repeat(fill, inner)+right))
		r.buf.WriteString("\n")
	}

	rule(r.border.TopLeft, r.border.Top, r.border.TopRight)
	for _, f := range fields {
		// A value too wide for the column is wrapped onto further lines rather
		// than cut, its label left on the first so the pair still reads as one.
		for i, value := range wrapValue(f.value, valueWidth) {
			label := f.label
			if i > 0 {
				label = ""
			}

			// Both cells are measured and filled before anything is painted, so
			// that escape sequences stay out of the width arithmetic.
			valueFill := strings.Repeat(" ", valueWidth-ansi.StringWidth(value))
			labelFill := strings.Repeat(" ", labelWidth-ansi.StringWidth(label))

			r.buf.WriteString(vertical + margin + r.env.Sprint(labelStyle, label) + labelFill +
				gap + paintCell(paint, f.col, value) + valueFill + margin + vertical)
			r.buf.WriteString("\n")
		}
	}

	r.cardFooter(next, inner-2*cardPad)

	rule(r.border.BottomLeft, r.border.Bottom, r.border.BottomRight)
}

// cardValueWidth is the width the card gives its value column: what the values
// want, clamped to what the terminal has. Nothing needs to be held back for a
// value that will not fit, because wrapValue breaks it rather than cutting it.
func (r renderer) cardValueWidth(fields []cardField, next string, chrome, labelWidth int) int {
	width := 0
	for _, f := range fields {
		if w := ansi.StringWidth(f.value); w > width {
			width = w
		}
	}

	// The footer spans both columns, so the card has to be wide enough for it.
	// Widening before the clamp leaves a narrow terminal free to truncate it.
	if next != "" {
		if w := nextIndent + ansi.StringWidth(next) - cardGap - labelWidth; w > width {
			width = w
		}
	}

	if r.env.DataWidth > 0 {
		budget := r.env.DataWidth - chrome
		if budget < minCardValue {
			budget = minCardValue
		}
		if width > budget {
			width = budget
		}
	}

	return width
}

// wrapValue lays value out in lines no wider than width, preferring to break
// where a reader would - between words, after a comma - and breaking mid-run
// only when a run has no such point of its own. Either way the whole value is
// shown, which is what a card owes a value too long for one line.
func wrapValue(value string, width int) []string {
	if width <= 0 {
		return []string{value}
	}

	return strings.Split(ansi.Wrap(value, width, ","), "\n")
}

// cardFooter suggests the command to run next, below the resource's fields. It
// is never wrapped: a command broken across two lines cannot be copied, so a
// command too long for the card is truncated like any other value.
func (r renderer) cardFooter(next string, width int) {
	if next == "" {
		return
	}

	margin := strings.Repeat(" ", cardPad)
	muted := r.env.NewStyle().Foreground(ui.ColorMuted)
	vertical := r.vertical()

	// Measured plain and painted separately, so escape sequences stay out of the
	// width arithmetic. The fill is clamped so a narrow card cannot panic.
	line := func(plain, painted string) {
		fill := width - ansi.StringWidth(plain)
		if fill < 0 {
			fill = 0
		}

		r.buf.WriteString(vertical + margin + painted +
			strings.Repeat(" ", fill) + margin + vertical)
		r.buf.WriteString("\n")
	}

	command := r.fit(next, width-nextIndent)
	indent := strings.Repeat(" ", nextIndent)

	line("", "")
	line(nextStepHeading, r.env.Sprint(muted, nextStepHeading))
	line(indent+command, indent+command)
}

// cardField is one line of a card. It keeps the column it came from so that
// the row's painter still finds the cell's tone after empties are dropped.
type cardField struct {
	col   int
	label string
	value string
}

// cardFields pairs each labelled value with its column, dropping the empties.
func cardFields(labels, values []string) []cardField {
	fields := make([]cardField, 0, len(values))
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}

		var label string
		if i < len(labels) {
			label = labels[i]
		}

		fields = append(fields, cardField{col: i, label: label, value: value})
	}

	return fields
}
