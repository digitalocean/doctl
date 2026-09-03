package displayers

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/charmbracelet/x/ansi"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/ui"

	"github.com/stretchr/testify/assert"
)

type testDisplayable struct {
	cols   []string
	colMap map[string]string
	kv     []map[string]any
}

func (t *testDisplayable) Cols() []string            { return t.cols }
func (t *testDisplayable) ColMap() map[string]string { return t.colMap }
func (t *testDisplayable) KV() []map[string]any      { return t.kv }
func (t *testDisplayable) JSON(w io.Writer) error    { return writeJSON(t.kv, w) }

func TestDisplayerDisplay(t *testing.T) {
	emptyVolumes := make([]do.Volume, 0)
	var nilVolumes []do.Volume

	tests := []struct {
		name         string
		item         Displayable
		expectedJSON string
	}{
		{
			name:         "displaying a non-nil slice of Volumes should return an empty JSON array",
			item:         &Volume{Volumes: emptyVolumes},
			expectedJSON: `[]`,
		},
		{
			name:         "displaying a nil slice of Volumes should return an empty JSON array",
			item:         &Volume{Volumes: nilVolumes},
			expectedJSON: `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}

			displayer := Displayer{
				OutputType: "json",
				Item:       tt.item,
				Out:        out,
			}

			err := displayer.Display()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedJSON, out.String())
		})
	}
}

// TestDisplayTextMatchesTabwriter pins the layout of unconstrained output to
// what text/tabwriter produced, since scripts and the integration suite diff
// it verbatim.
func TestDisplayTextMatchesTabwriter(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"id", "name", "size"},
		colMap: map[string]string{"id": "ID", "name": "Name", "size": "Size"},
		kv: []map[string]any{
			{"id": 1, "name": "a-fairly-long-droplet-name", "size": "s-1vcpu-1gb"},
			{"id": 22, "name": "web", "size": "s-8vcpu-16gb"},
		},
	}

	var expected bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&expected, 0, 0, 4, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tSize")
	fmt.Fprintln(w, "1\ta-fairly-long-droplet-name\ts-1vcpu-1gb")
	fmt.Fprintln(w, "22\tweb\ts-8vcpu-16gb")
	assert.NoError(t, w.Flush())

	var out bytes.Buffer
	assert.NoError(t, DisplayText(item, &out, false, nil, ui.Env{}))
	assert.Equal(t, expected.String(), out.String())
}

func TestDisplayTextDoesNotTruncateRedirectedOutput(t *testing.T) {
	// A shell that exports COLUMNS must not cause piped values to be
	// truncated, since that is what scripts parse.
	t.Setenv("COLUMNS", "40")

	long := strings.Repeat("x", 500)
	item := &testDisplayable{
		cols:   []string{"desc"},
		colMap: map[string]string{"desc": "Description"},
		kv:     []map[string]any{{"desc": long}},
	}

	var out, errOut bytes.Buffer
	env := ui.Detect(&out, &errOut)

	assert.NoError(t, DisplayText(item, &out, true, nil, env))
	assert.Equal(t, long+"\n", out.String())
}

func TestDisplayTextFitsTerminalWidth(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"id", "name", "desc"},
		colMap: map[string]string{"id": "ID", "name": "Name", "desc": "Description"},
		kv: []map[string]any{
			{"id": 1, "name": "a-fairly-long-droplet-name", "desc": "a description that runs well past the edge"},
		},
	}

	tests := []struct {
		name     string
		opts     []ui.Option
		expected string
	}{
		{
			name: "columns are narrowed to the width",
			opts: []ui.Option{ui.WithWidth(46)},
			expected: "ID    Name                  Description\n" +
				"1     a-fairly-long-dro…    a description tha…\n",
		},
		{
			name: "the ascii fallback avoids a unicode ellipsis",
			opts: []ui.Option{ui.WithWidth(46), ui.WithASCII(true)},
			expected: "ID    Name                  Description\n" +
				"1     a-fairly-long-d...    a description t...\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			env := ui.Detect(&out, &errOut, tt.opts...)

			assert.NoError(t, DisplayText(item, &out, false, nil, env))
			assert.Equal(t, tt.expected, out.String())

			for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
				assert.LessOrEqual(t, ansi.StringWidth(line), 46, "line %q exceeds the width", line)
			}
		})
	}
}

func TestFormatCell(t *testing.T) {
	assert.Equal(t, "1.500000", formatCell(float64(1.5)))
	assert.Equal(t, "42", formatCell(42))
	assert.Equal(t, "true", formatCell(true))
	assert.Equal(t, "abc", formatCell("abc"))
	assert.Equal(t, "<nil>", formatCell(nil))
}

func TestColumnWidths(t *testing.T) {
	headers := []string{"ID", "Name", "Description"}
	rows := [][]string{
		{"1", "web", "a description that is quite long"},
		{"22", "db", "short"},
	}

	tests := []struct {
		name     string
		headers  []string
		rows     [][]string
		maxWidth int
		expected []int
	}{
		{
			name:     "unconstrained uses the widest value in each column",
			headers:  headers,
			rows:     rows,
			maxWidth: 0,
			expected: []int{2, 4, 32},
		},
		{
			name:     "a width larger than the table changes nothing",
			headers:  headers,
			rows:     rows,
			maxWidth: 200,
			expected: []int{2, 4, 32},
		},
		{
			name:     "the column with the most slack gives up space first",
			headers:  headers,
			rows:     rows,
			maxWidth: 30,
			expected: []int{2, 4, 16},
		},
		{
			name:     "columns are never narrowed past their header",
			headers:  []string{"Description", "Name"},
			rows:     [][]string{{"ab", "cdef"}},
			maxWidth: 5,
			expected: []int{11, 4},
		},
		{
			name:     "double-width runes are measured in terminal cells",
			headers:  []string{"名前"},
			rows:     [][]string{{"ab"}},
			maxWidth: 0,
			expected: []int{4},
		},
		{
			name:     "no columns yields no widths",
			maxWidth: 80,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, columnWidths(tt.headers, tt.rows, tt.maxWidth))
		})
	}
}

func TestWriteRow(t *testing.T) {
	tests := []struct {
		name     string
		cells    []string
		widths   []int
		expected string
	}{
		{
			name:     "cells are padded to their column",
			cells:    []string{"ab", "cd"},
			widths:   []int{5, 2},
			expected: "ab       cd\n",
		},
		{
			name:     "an oversized cell is truncated with an ellipsis",
			cells:    []string{"abcdefgh", "xy"},
			widths:   []int{5, 2},
			expected: "abcd…    xy\n",
		},
		{
			name:     "padding ignores ansi escapes so styled cells stay aligned",
			cells:    []string{"\x1b[32mabc\x1b[0m", "xy"},
			widths:   []int{5, 2},
			expected: "\x1b[32mabc\x1b[0m      xy\n",
		},
		{
			name:     "the final column is not padded",
			cells:    []string{"ab", "cd"},
			widths:   []int{2, 10},
			expected: "ab    cd\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeRow(&buf, tt.cells, tt.widths, "…")
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}
