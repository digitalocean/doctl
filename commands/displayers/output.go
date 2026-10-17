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

	"github.com/digitalocean/doctl/internal/ui"
)

const (
	// columnGap is the number of spaces separating plain text table columns.
	columnGap = 4

	// cellPad is the space between a boxed cell's value and its rules.
	cellPad = 1

	// cardPad is the margin between a card's rules and its contents.
	cardPad = 2

	// cardGap is the space between a card's label and its value.
	cardGap = 1

	// nextIndent sets the command in the card's footer under its heading.
	nextIndent = 2

	// minCardValue floors a card's values; past it the card overruns instead.
	minCardValue = 8

	// recordIndent indents a record's fields under the headline naming it.
	recordIndent = 2
)

// Displayable is a displayable entity. These are used for printing results.
type Displayable interface {
	Cols() []string
	ColMap() map[string]string
	KV() []map[string]any
	JSON(io.Writer) error
}

// Toned is an optional interface for a displayer to tone its own columns.
// Returning false leaves a column to the default classification.
type Toned interface {
	ColTone(col string, value any) (ui.Tone, bool)
}

// Tabular is an optional interface that keeps a displayer in the table layout
// even when one resource is shown, for fields only meaningful side by side.
type Tabular interface {
	Tabular() bool
}

// Displayer has the display options, the item to display, and where to display to
type Displayer struct {
	OutputType string
	ColumnList string
	NoHeaders  bool

	// Detail reports a single-resource fetch, shown in a terminal as a card.
	Detail bool

	// NextStep is the command to run after this one, shown at a card's foot.
	NextStep string

	Item Displayable
	Out  io.Writer

	// UI carries the terminal capabilities of Out. Its zero value is plain text.
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

		return DisplayText(d.Item, d.Out, d.NoHeaders, cols, d.UI,
			WithDetail(d.Detail), WithNextStep(d.NextStep))
	default:
		return fmt.Errorf("unknown output type")
	}
}

// TextOption adjusts how DisplayText renders, mirroring ui.Option.
type TextOption func(*textConfig)

// textConfig holds the resolved options. Its zero value is the table layout.
type textConfig struct {
	detail   bool
	nextStep string
}

// WithDetail reports that item describes a single resource, shown as a card.
func WithDetail(v bool) TextOption {
	return func(c *textConfig) { c.detail = v }
}

// WithNextStep names the command to run after this one, shown at a card's foot.
func WithNextStep(v string) TextOption {
	return func(c *textConfig) { c.nextStep = v }
}

// DisplayText writes column-aligned content to out. Layout follows the stream,
// rules and cards being chrome, and no layout cuts a value:
//
//   - a pipe, file, or test gets space-separated columns;
//   - a terminal showing one resource gets a card, a field to a line;
//   - a terminal showing a table that fits gets it inside box rules;
//   - a table too wide to fit becomes one record per resource;
//   - --no-header leaves a record no labels, so it wraps at full width unruled;
//   - naming columns overrides the width rule, since they were picked by hand,
//     and the table overruns rather than dropping any of them.
func DisplayText(item Displayable, out io.Writer, noHeaders bool, includeCols []string, env ui.Env, opts ...TextOption) error {
	var cfg textConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	cols := item.Cols()
	explicitCols := len(includeCols) > 0 && includeCols[0] != ""
	if explicitCols {
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
			row = append(row, formatCell(r[col], env.DataTTY))
		}
		rows = append(rows, row)
	}

	count := columnCount(headers, rows)
	widths := columnWidths(headers, rows)
	fits := fitsBudget(widths, contentBudget(env.DataWidth, count, env.DataTTY))
	tones := newToneTable(env, item, cols, kv)
	plainRow := func(row int) painter { return tones.painter(row, ui.ToneNone) }

	var buf bytes.Buffer
	r := newRenderer(&buf, env)

	switch {
	case env.DataTTY && shouldCard(cfg, item, headers, len(rows)):
		r.card(headers, rows[0], cfg.nextStep, plainRow(0))
	case env.DataTTY && (fits || explicitCols):
		r.box(headers, rows, widths, headerPainter(env), plainRow)
	case env.DataTTY && headers != nil:
		r.records(headers, rows, tones)
	default:
		if headers != nil {
			r.row(headers, widths, headerPainter(env))
		}
		for i, row := range rows {
			r.row(row, widths, plainRow(i))
		}
	}

	_, err := buf.WriteTo(out)
	return err
}

// shouldCard reports whether to draw a card, out being known to be a terminal.
// rowCount is checked too, so a get returning several resources stays a table.
func shouldCard(cfg textConfig, item Displayable, headers []string, rowCount int) bool {
	if !cfg.detail || rowCount != 1 || headers == nil {
		return false
	}

	tabular, ok := item.(Tabular)

	return !ok || !tabular.Tabular()
}

// goNil is how fmt spells a nil value. Displayers build their cells with fmt,
// so an unset field reaches this package already spelled this way.
const goNil = "<nil>"

// formatCell renders a column value as it appears in text output.
//
// human asks for the reading a person wants of an unset field, which is
// nothing at all. A redirected stream keeps fmt's "<nil>" instead, because a
// script parsing doctl's output is entitled to the bytes it was written
// against.
func formatCell(v any, human bool) string {
	if human && isNil(v) {
		return ""
	}

	var cell string
	if f, ok := v.(float64); ok {
		cell = fmt.Sprintf("%f", f)
	} else {
		cell = fmt.Sprint(v)
	}

	// A compound cell is assembled by the displayer, so a field it left unset
	// arrives as "<nil>" inside an otherwise good value.
	if human {
		cell = strings.ReplaceAll(cell, goNil, "")
	}

	return cell
}

// isNil reports whether v holds nothing, including the typed nil an absent
// pointer field yields, which is not equal to the untyped nil of an unset key.
// Slices and maps are deliberately not treated as nil; that is the displayer's call.
func isNil(v any) bool {
	if v == nil {
		return true
	}

	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
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

// containsOnlyNilSlice reports whether i points to a struct holding one nil slice.
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
