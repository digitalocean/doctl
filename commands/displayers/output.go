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

package displayers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/digitalocean/doctl/internal/ui"
)

const (
	// columnGap is the number of spaces separating plain text table columns.
	columnGap = 4

	// cellPad is the space between a boxed cell's value and the rules on
	// either side of it.
	cellPad = 1
)

// Displayable is a displayable entity. These are used for printing results.
type Displayable interface {
	Cols() []string
	ColMap() map[string]string
	KV() []map[string]any
	JSON(io.Writer) error
}

// Toned is an optional interface for displayers whose columns the shared state
// vocabulary would classify wrongly.
//
// Returning false means no opinion, leaving the column to the default
// classification. Returning true is authoritative, including ui.ToneNone to
// force a column to stay plain.
type Toned interface {
	ColTone(col string, value any) (ui.Tone, bool)
}

// stateWords are the column names that hold the state of a resource, as
// opposed to describing or identifying it.
//
// Tone is applied only when a column is named for state and its value is a
// word the vocabulary knows. Both halves matter: state words appear in prose
// that lives in columns like Message and Failure Reason, which must not be
// painted just because a resource happens to be failing.
var stateWords = []string{"status", "state", "phase", "health", "verdict"}

// Displayer has the display options, the item to display, and where to display to
type Displayer struct {
	OutputType string
	ColumnList string
	NoHeaders  bool

	Item Displayable
	Out  io.Writer

	// UI carries the terminal capabilities of Out. Its zero value renders
	// plain, unconstrained text, which is what tests and pipelines want.
	UI ui.Env
}

// Display ends up rendering the content in one of two formats (text|json)
func (d *Displayer) Display() error {
	switch d.OutputType {
	case "json":
		if containsOnlyNilSlice(d.Item) {
			_, err := d.Out.Write([]byte("[]"))
			return err
		}
		return d.Item.JSON(d.Out)
	case "text":
		var cols []string
		for _, c := range strings.Split(strings.Join(strings.Fields(d.ColumnList), ""), ",") {
			if c != "" {
				cols = append(cols, c)
			}
		}

		return DisplayText(d.Item, d.Out, d.NoHeaders, cols, d.UI)
	default:
		return fmt.Errorf("unknown output type")
	}
}

// DisplayText writes column-aligned content to the passed in io.Writer while
// potentially adding or removing headers. Columns are narrowed to fit env's
// data width, which is unconstrained unless out is a terminal.
//
// A terminal gets the table drawn inside box rules, which is what makes a wide
// row readable at a glance and a truncated cell obviously truncated. Anything
// else - a pipe, a file, a test - gets the same space-separated columns doctl
// has always written, because the rules are chrome and a script reading the
// table must not have to strip them.
func DisplayText(item Displayable, out io.Writer, noHeaders bool, includeCols []string, env ui.Env) error {
	cols := item.Cols()
	if len(includeCols) > 0 && includeCols[0] != "" {
		cols = includeCols
	}

	var headers []string
	if !noHeaders {
		headers = make([]string, 0, len(cols))
		for _, k := range cols {
			col := item.ColMap()[k]
			if col == "" {
				return fmt.Errorf("unknown column %q", k)
			}

			headers = append(headers, col)
		}
	}

	kv := item.KV()
	rows := make([][]string, 0, len(kv))
	for _, r := range kv {
		row := make([]string, 0, len(cols))
		for _, col := range cols {
			row = append(row, formatCell(r[col]))
		}
		rows = append(rows, row)
	}

	ellipsis := env.Glyphs().Ellipsis
	count := columnCount(headers, rows)
	widths := columnWidths(headers, rows, contentBudget(env.DataWidth, count, env.DataTTY), ansi.StringWidth(ellipsis))
	rowPainter := tonePainters(env, item, cols, kv)

	var buf bytes.Buffer
	if env.DataTTY {
		writeBox(&buf, headers, rows, widths, ellipsis, env, headerPainter(env), rowPainter)
	} else {
		if headers != nil {
			writeRow(&buf, headers, widths, ellipsis, headerPainter(env))
		}
		for i, row := range rows {
			writeRow(&buf, row, widths, ellipsis, rowPainter(i))
		}
	}

	_, err := buf.WriteTo(out)
	return err
}

// painter styles one cell of a row. It runs after the cell has been measured
// and truncated so that escape sequences never enter width arithmetic.
type painter func(col int, cell string) string

// headerPainter emphasises the header row, which is what separates the labels
// from the data when a table is long enough to scroll.
func headerPainter(env ui.Env) painter {
	if !env.Style {
		return nil
	}

	header := env.NewStyle().Bold(true)

	return func(_ int, cell string) string {
		return env.Sprint(header, cell)
	}
}

// tonePainters classifies the table once and returns a painter per row. When
// the env forbids styling it classifies nothing at all, so redirected output
// costs no more than it did before tones existed.
func tonePainters(env ui.Env, item Displayable, cols []string, kv []map[string]any) func(row int) painter {
	if !env.Style {
		return func(int) painter { return nil }
	}

	tones := tableTones(item, cols, kv)

	return func(row int) painter {
		return func(col int, cell string) string {
			return env.SprintTone(tones[row][col], cell)
		}
	}
}

// tableTones classifies every cell, giving the displayer the final say.
func tableTones(item Displayable, cols []string, kv []map[string]any) [][]ui.Tone {
	toned, overrides := item.(Toned)

	// Whether a column holds state is a property of the table, so it is
	// decided once rather than for every row.
	stateCols := make([]bool, len(cols))
	for i, col := range cols {
		stateCols[i] = isStateColumn(col)
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

			if !stateCols[i] {
				continue
			}

			// Only strings are classified. Booleans are left alone because
			// their polarity belongs to the column rather than the value: true
			// is healthy under Advertised and unhealthy under Disabled.
			if s, ok := r[col].(string); ok {
				row[i], _ = ui.ToneFor(s)
			}
		}
		tones = append(tones, row)
	}

	return tones
}

// isStateColumn reports whether col names the state of a resource. Separators
// are dropped before matching so that "Health Status" and "HealthStatus" are
// treated alike.
func isStateColumn(col string) bool {
	normalized := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '_', '-':
			return -1
		default:
			return unicode.ToLower(r)
		}
	}, col)

	for _, word := range stateWords {
		if normalized == word || strings.HasSuffix(normalized, word) {
			return true
		}
	}

	return false
}

// formatCell renders a column value as it appears in text output.
func formatCell(v any) string {
	if f, ok := v.(float64); ok {
		return fmt.Sprintf("%f", f)
	}
	return fmt.Sprint(v)
}

// columnCount reports how many columns the table has.
func columnCount(headers []string, rows [][]string) int {
	count := len(headers)
	for _, row := range rows {
		if len(row) > count {
			count = len(row)
		}
	}

	return count
}

// contentBudget is the width left for values once the chrome separating the
// columns is paid for. It returns 0 when the table is unconstrained, and never
// less than 1 otherwise, so that a budget too small to honour still asks for
// as much narrowing as the floors allow rather than reading as no limit.
func contentBudget(maxWidth, count int, boxed bool) int {
	if maxWidth <= 0 || count == 0 {
		return 0
	}

	chrome := columnGap * (count - 1)
	if boxed {
		// A rule to the left of every column plus one closing the row, and a
		// pad either side of every value.
		chrome = count + 1 + 2*cellPad*count
	}

	if budget := maxWidth - chrome; budget > 1 {
		return budget
	}

	return 1
}

// columnWidths measures how wide each column needs to be to hold its header
// and values. When budget is positive, the widest column is repeatedly
// narrowed until the values fit within it, so the column with the most slack
// gives up space first.
//
// A column is never narrowed past its header, nor past floor, which is the
// width of the ellipsis a truncated cell ends in: a column cut narrower than
// that has nothing left to show but the mark saying it was cut. An unavoidably
// wide table therefore overflows rather than becoming unreadable.
func columnWidths(headers []string, rows [][]string, budget, floor int) []int {
	count := columnCount(headers, rows)
	if count == 0 {
		return nil
	}

	widths := make([]int, count)
	floors := make([]int, count)
	for i := range floors {
		floors[i] = floor
	}
	for i, header := range headers {
		widths[i] = ansi.StringWidth(header)
		if widths[i] > floors[i] {
			floors[i] = widths[i]
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := ansi.StringWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	if budget <= 0 {
		return widths
	}

	total := 0
	for _, w := range widths {
		total += w
	}

	for total > budget {
		widest, idx := 0, -1
		for i, w := range widths {
			if w > floors[i] && w > widest {
				widest, idx = w, i
			}
		}
		if idx < 0 {
			break
		}
		widths[idx]--
		total--
	}

	return widths
}

// writeRow writes one row, truncating cells that exceed their column with
// ellipsis and padding the rest. Widths are measured in terminal cells rather
// than bytes so that styled and double-width values stay aligned.
//
// Styling is applied last, once a cell has been measured and padded for, so
// that a coloured table lays out identically to a plain one.
func writeRow(buf *bytes.Buffer, cells []string, widths []int, ellipsis string, paint painter) {
	for i, cell := range cells {
		if ansi.StringWidth(cell) > widths[i] {
			cell = ansi.Truncate(cell, widths[i], ellipsis)
		}

		padding := widths[i] - ansi.StringWidth(cell) + columnGap

		if paint != nil {
			cell = paint(i, cell)
		}
		buf.WriteString(cell)

		if i < len(cells)-1 && padding > 0 {
			buf.WriteString(strings.Repeat(" ", padding))
		}
	}
	buf.WriteString("\n")
}

// writeBox draws the table inside box rules, with the header separated from
// the values by a rule of its own.
//
// The rules are drawn from lipgloss's border sets rather than literals so that
// the ASCII fallback is the same one the rest of doctl's chrome falls back to,
// and they are painted as muted chrome so that the values keep the reader's
// eye. Cells are measured and truncated exactly as they are in the plain
// layout, which is what keeps a boxed table and a piped one showing the same
// values.
func writeBox(buf *bytes.Buffer, headers []string, rows [][]string, widths []int, ellipsis string, env ui.Env, head painter, rowPaint func(int) painter) {
	if len(widths) == 0 {
		return
	}

	border := lipgloss.NormalBorder()
	if env.ASCII {
		border = lipgloss.ASCIIBorder()
	}

	muted := env.NewStyle().Foreground(ui.ColorMuted)
	vertical := env.Sprint(muted, border.Left)

	rule := func(left, join, right string) {
		segments := make([]string, len(widths))
		for i, w := range widths {
			segments[i] = strings.Repeat(border.Top, w+2*cellPad)
		}

		buf.WriteString(env.Sprint(muted, left+strings.Join(segments, join)+right))
		buf.WriteString("\n")
	}

	row := func(cells []string, paint painter) {
		pad := strings.Repeat(" ", cellPad)

		buf.WriteString(vertical)
		for i, width := range widths {
			var cell string
			if i < len(cells) {
				cell = cells[i]
			}

			if ansi.StringWidth(cell) > width {
				cell = ansi.Truncate(cell, width, ellipsis)
			}
			fill := strings.Repeat(" ", width-ansi.StringWidth(cell))

			if paint != nil {
				cell = paint(i, cell)
			}

			buf.WriteString(pad + cell + fill + pad + vertical)
		}
		buf.WriteString("\n")
	}

	rule(border.TopLeft, border.MiddleTop, border.TopRight)
	if headers != nil {
		row(headers, head)
		rule(border.MiddleLeft, border.Middle, border.MiddleRight)
	}
	for i, cells := range rows {
		row(cells, rowPaint(i))
	}
	rule(border.BottomLeft, border.MiddleBottom, border.BottomRight)
}

func writeJSON(item any, w io.Writer) error {
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}

	var out bytes.Buffer
	err = json.Indent(&out, b, "", "  ")
	if err != nil {
		return err
	}
	_, err = out.WriteTo(w)

	return err
}

// containsOnlyNiSlice returns true if the given interface's concrete type is
// a pointer to a struct that contains a single nil slice field.
func containsOnlyNilSlice(i any) bool {
	if reflect.TypeOf(i).Kind() != reflect.Ptr {
		return false
	}

	element := reflect.ValueOf(i).Elem()
	if element.NumField() != 1 {
		return false
	}

	slice := element.Field(0)
	if slice.Kind() != reflect.Slice {
		return false
	}

	if slice.Cap() != 0 {
		return false
	}
	if slice.Len() != 0 {
		return false
	}
	if slice.Pointer() != 0 {
		return false
	}

	return true
}
