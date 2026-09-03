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

	"github.com/charmbracelet/x/ansi"

	"github.com/digitalocean/doctl/internal/ui"
)

// columnGap is the number of spaces separating text table columns.
const columnGap = 4

// Displayable is a displayable entity. These are used for printing results.
type Displayable interface {
	Cols() []string
	ColMap() map[string]string
	KV() []map[string]any
	JSON(io.Writer) error
}

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

	widths := columnWidths(headers, rows, env.DataWidth)
	ellipsis := env.Glyphs().Ellipsis

	var buf bytes.Buffer
	if headers != nil {
		writeRow(&buf, headers, widths, ellipsis)
	}
	for _, row := range rows {
		writeRow(&buf, row, widths, ellipsis)
	}

	_, err := buf.WriteTo(out)
	return err
}

// formatCell renders a column value as it appears in text output.
func formatCell(v any) string {
	if f, ok := v.(float64); ok {
		return fmt.Sprintf("%f", f)
	}
	return fmt.Sprint(v)
}

// columnWidths measures how wide each column needs to be to hold its header
// and values. When maxWidth is positive, the widest column is repeatedly
// narrowed until the row fits, so the column with the most slack gives up
// space first. A column is never narrowed past its header, which means an
// unavoidably wide table still overflows rather than becoming unreadable.
func columnWidths(headers []string, rows [][]string, maxWidth int) []int {
	count := len(headers)
	for _, row := range rows {
		if len(row) > count {
			count = len(row)
		}
	}
	if count == 0 {
		return nil
	}

	widths := make([]int, count)
	floors := make([]int, count)
	for i, header := range headers {
		widths[i] = ansi.StringWidth(header)
		floors[i] = widths[i]
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := ansi.StringWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	if maxWidth <= 0 {
		return widths
	}

	total := columnGap * (count - 1)
	for _, w := range widths {
		total += w
	}

	for total > maxWidth {
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
func writeRow(buf *bytes.Buffer, cells []string, widths []int, ellipsis string) {
	for i, cell := range cells {
		if ansi.StringWidth(cell) > widths[i] {
			cell = ansi.Truncate(cell, widths[i], ellipsis)
		}
		buf.WriteString(cell)

		if i < len(cells)-1 {
			padding := widths[i] - ansi.StringWidth(cell) + columnGap
			if padding > 0 {
				buf.WriteString(strings.Repeat(" ", padding))
			}
		}
	}
	buf.WriteString("\n")
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
