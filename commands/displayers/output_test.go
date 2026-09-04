package displayers

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/ui"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// terminalEnv returns an Env that reports Out as an interactive terminal, so
// that the boxed layout can be exercised against a buffer.
func terminalEnv(out, errOut *bytes.Buffer, opts ...ui.Option) ui.Env {
	env := ui.Detect(out, errOut, opts...)
	env.DataTTY = true

	return env
}

func TestDisplayTextBoxesTerminalOutput(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"id", "name"},
		colMap: map[string]string{"id": "ID", "name": "Name"},
		kv: []map[string]any{
			{"id": 1, "name": "web"},
			{"id": 22, "name": "database"},
		},
	}

	tests := []struct {
		name      string
		opts      []ui.Option
		noHeaders bool
		expected  string
	}{
		{
			name: "a table on a terminal is drawn inside rules",
			expected: "┌────┬──────────┐\n" +
				"│ ID │ Name     │\n" +
				"├────┼──────────┤\n" +
				"│ 1  │ web      │\n" +
				"│ 22 │ database │\n" +
				"└────┴──────────┘\n",
		},
		{
			name: "the ascii fallback draws the same table without box characters",
			opts: []ui.Option{ui.WithASCII(true)},
			expected: "+----+----------+\n" +
				"| ID | Name     |\n" +
				"+----+----------+\n" +
				"| 1  | web      |\n" +
				"| 22 | database |\n" +
				"+----+----------+\n",
		},
		{
			// --no-header drops the labels and the rule that separated them,
			// leaving the values framed by the same box.
			name:      "dropping the headers drops the rule below them",
			noHeaders: true,
			expected: "┌────┬──────────┐\n" +
				"│ 1  │ web      │\n" +
				"│ 22 │ database │\n" +
				"└────┴──────────┘\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			env := terminalEnv(&out, &errOut, tt.opts...)

			require.NoError(t, DisplayText(item, &out, tt.noHeaders, nil, env))
			assert.Equal(t, tt.expected, out.String())
		})
	}
}

// TestDisplayTextDoesNotBoxRedirectedOutput is the other half of the boxed
// layout: the rules are chrome, so a script reading the table must never have
// to strip them.
func TestDisplayTextDoesNotBoxRedirectedOutput(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"id", "name"},
		colMap: map[string]string{"id": "ID", "name": "Name"},
		kv:     []map[string]any{{"id": 1, "name": "web"}},
	}

	var out, errOut bytes.Buffer

	require.NoError(t, DisplayText(item, &out, false, nil, ui.Detect(&out, &errOut)))
	assert.Equal(t, "ID    Name\n1     web\n", out.String())
}

func TestDisplayTextBoxFitsTerminalWidth(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"id", "name", "desc"},
		colMap: map[string]string{"id": "ID", "name": "Name", "desc": "Description"},
		kv: []map[string]any{
			{"id": 1, "name": "a-fairly-long-droplet-name", "desc": "a description that runs well past the edge"},
		},
	}

	const width = 46

	var out, errOut bytes.Buffer
	env := terminalEnv(&out, &errOut, ui.WithWidth(width))

	require.NoError(t, DisplayText(item, &out, false, nil, env))

	// The rules are part of the width the terminal has to spend, so the values
	// are narrowed further than they would be in the plain layout.
	assert.Equal(t, "┌────┬───────────────────┬───────────────────┐\n"+
		"│ ID │ Name              │ Description       │\n"+
		"├────┼───────────────────┼───────────────────┤\n"+
		"│ 1  │ a-fairly-long-dr… │ a description th… │\n"+
		"└────┴───────────────────┴───────────────────┘\n", out.String())

	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), width, "line %q exceeds the width", line)
	}
}

// TestDisplayTextBoxStylingDoesNotMoveColumns holds the boxed layout to the
// same guarantee as the plain one: colour may only add escape sequences.
func TestDisplayTextBoxStylingDoesNotMoveColumns(t *testing.T) {
	var plainOut, styledOut, errOut bytes.Buffer

	require.NoError(t, DisplayText(stateTable(), &plainOut, false, nil,
		terminalEnv(&plainOut, &errOut, ui.WithProfile(termenv.Ascii))))
	require.NoError(t, DisplayText(stateTable(), &styledOut, false, nil,
		terminalEnv(&styledOut, &errOut, ui.WithProfile(termenv.TrueColor))))

	assert.Equal(t, plainOut.String(), ansi.Strip(styledOut.String()))
	assert.Contains(t, styledOut.String(), "\x1b[", "a styled table should carry colour")
}

// tonedDisplayable overrides the shared classification for one column.
type tonedDisplayable struct {
	testDisplayable
}

func (t *tonedDisplayable) ColTone(col string, value any) (ui.Tone, bool) {
	if col == "state" {
		return ui.ToneNone, true
	}

	return ui.ToneNone, false
}

func stateTable() *testDisplayable {
	return &testDisplayable{
		cols:   []string{"id", "name", "status", "message"},
		colMap: map[string]string{"id": "ID", "name": "Name", "status": "Status", "message": "Message"},
		kv: []map[string]any{
			{"id": 1, "name": "web", "status": "active", "message": "deploy failed earlier"},
			{"id": 2, "name": "db", "status": "errored", "message": "n/a"},
			{"id": 3, "name": "cache", "status": "PENDING_DEPLOY", "message": "n/a"},
			{"id": 4, "name": "queue", "status": "somewhere-new-entirely", "message": "n/a"},
		},
	}
}

func TestDisplayTextStylesStateColumns(t *testing.T) {
	var out, errOut bytes.Buffer
	env := ui.Detect(&out, &errOut, ui.WithProfile(termenv.TrueColor))

	require.NoError(t, DisplayText(stateTable(), &out, false, nil, env))
	rendered := out.String()

	assert.Contains(t, rendered, env.Sprint(env.NewStyle().Bold(true), "Status"),
		"the header row should be emphasised")
	assert.Contains(t, rendered, env.SprintTone(ui.ToneSuccess, "active"))
	assert.Contains(t, rendered, env.SprintTone(ui.ToneError, "errored"))
	assert.Contains(t, rendered, env.SprintTone(ui.TonePending, "PENDING_DEPLOY"))

	// A value the vocabulary does not know stays plain rather than being
	// guessed at, and prose in a non-state column is never painted even when
	// it contains a state word.
	assert.NotContains(t, rendered, env.SprintTone(ui.ToneSuccess, "somewhere-new-entirely"))
	assert.NotContains(t, rendered, env.SprintTone(ui.ToneError, "deploy failed earlier"))
	assert.Contains(t, rendered, "deploy failed earlier")
}

// TestDisplayTextStylingDoesNotMoveColumns is the guarantee that makes colour
// safe to turn on: styling may only add escape sequences, never shift a cell.
func TestDisplayTextStylingDoesNotMoveColumns(t *testing.T) {
	var plainOut, styledOut, errOut bytes.Buffer

	require.NoError(t, DisplayText(stateTable(), &plainOut, false, nil, ui.Plain(&plainOut, &errOut)))

	styledEnv := ui.Detect(&styledOut, &errOut, ui.WithProfile(termenv.TrueColor))
	require.NoError(t, DisplayText(stateTable(), &styledOut, false, nil, styledEnv))

	assert.Equal(t, plainOut.String(), ansi.Strip(styledOut.String()))
}

// TestDisplayTextStylingNarrowTerminal covers styling and truncation together,
// since a truncated cell is styled after its ellipsis is applied.
func TestDisplayTextStylingNarrowTerminal(t *testing.T) {
	var plainOut, styledOut, errOut bytes.Buffer

	require.NoError(t, DisplayText(stateTable(), &plainOut, false, nil,
		ui.Detect(&plainOut, &errOut, ui.WithWidth(40), ui.WithProfile(termenv.Ascii))))
	require.NoError(t, DisplayText(stateTable(), &styledOut, false, nil,
		ui.Detect(&styledOut, &errOut, ui.WithWidth(40), ui.WithProfile(termenv.TrueColor))))

	assert.Equal(t, plainOut.String(), ansi.Strip(styledOut.String()))
	for _, line := range strings.Split(strings.TrimRight(styledOut.String(), "\n"), "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), 40, "line %q exceeds the width", line)
	}
}

func TestDisplayTextToneOverride(t *testing.T) {
	item := &tonedDisplayable{testDisplayable{
		cols:   []string{"state", "status"},
		colMap: map[string]string{"state": "State", "status": "Status"},
		kv:     []map[string]any{{"state": "active", "status": "active"}},
	}}

	var out, errOut bytes.Buffer
	env := ui.Detect(&out, &errOut, ui.WithProfile(termenv.TrueColor))

	require.NoError(t, DisplayText(item, &out, false, nil, env))

	// The displayer declined State authoritatively and had no opinion on
	// Status, which therefore falls through to the shared vocabulary. Both
	// cells hold the same value, so only the styling tells them apart.
	cells := strings.Fields(out.String())
	assert.Equal(t, "active", cells[2], "the declined column stays plain")
	assert.Equal(t, env.SprintTone(ui.ToneSuccess, "active"), cells[3],
		"the column with no opinion is classified by the vocabulary")
}

func TestIsStateColumn(t *testing.T) {
	for _, col := range []string{
		"Status", "status", "State", "Phase", "Health", "Health Status",
		"health_status", "Verdict", "Restore Status",
	} {
		assert.True(t, isStateColumn(col), "%q holds state", col)
	}

	for _, col := range []string{
		"ID", "Name", "Message", "Error", "FailureReason", "Unhealthy Reason",
		"PendingChanges", "HealthCheck", "Progress", "Severity",
	} {
		assert.False(t, isStateColumn(col), "%q does not hold state", col)
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
		budget   int
		expected []int
	}{
		{
			name:     "unconstrained uses the widest value in each column",
			headers:  headers,
			rows:     rows,
			budget:   0,
			expected: []int{2, 4, 32},
		},
		{
			name:     "a budget larger than the table changes nothing",
			headers:  headers,
			rows:     rows,
			budget:   200,
			expected: []int{2, 4, 32},
		},
		{
			name:     "the column with the most slack gives up space first",
			headers:  headers,
			rows:     rows,
			budget:   22,
			expected: []int{2, 4, 16},
		},
		{
			name:     "columns are never narrowed past their header",
			headers:  []string{"Description", "Name"},
			rows:     [][]string{{"ab", "cdef"}},
			budget:   1,
			expected: []int{11, 4},
		},
		{
			// Without a header to hold the column open, the ellipsis width is
			// the floor: a column cut narrower than its own truncation mark
			// has nothing left to show.
			name:     "an unheadered column is never narrowed past the ellipsis",
			rows:     [][]string{{"a-long-value", "another-long-value"}},
			budget:   1,
			expected: []int{1, 1},
		},
		{
			name:     "double-width runes are measured in terminal cells",
			headers:  []string{"名前"},
			rows:     [][]string{{"ab"}},
			budget:   0,
			expected: []int{4},
		},
		{
			name:     "no columns yields no widths",
			budget:   80,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, columnWidths(tt.headers, tt.rows, tt.budget, 1))
		})
	}
}

func TestContentBudget(t *testing.T) {
	tests := []struct {
		name     string
		maxWidth int
		count    int
		boxed    bool
		expected int
	}{
		{
			name:     "an unconstrained width leaves the table unconstrained",
			maxWidth: 0,
			count:    3,
			expected: 0,
		},
		{
			name:     "plain columns pay for the gaps between them",
			maxWidth: 30,
			count:    3,
			expected: 22,
		},
		{
			name:     "boxed columns pay for their rules and pads",
			maxWidth: 30,
			count:    3,
			boxed:    true,
			expected: 20,
		},
		{
			// A budget of zero would read as unconstrained, which is the one
			// thing a terminal too narrow for its table must not do.
			name:     "a width smaller than the chrome still asks for narrowing",
			maxWidth: 4,
			count:    3,
			boxed:    true,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, contentBudget(tt.maxWidth, tt.count, tt.boxed))
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
			writeRow(&buf, tt.cells, tt.widths, "…", nil)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}
