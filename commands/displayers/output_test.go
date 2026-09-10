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

// TestDisplayTextPlainLayoutIgnoresWidth pins the plain layout to the values
// it was given. It is what a pipe, a file and a test read, so a width the
// stream happens to report must not shorten a value: the reader is a program,
// and a program that receives two thirds of a description has received nothing.
func TestDisplayTextPlainLayoutIgnoresWidth(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"id", "name", "desc"},
		colMap: map[string]string{"id": "ID", "name": "Name", "desc": "Description"},
		kv: []map[string]any{
			{"id": 1, "name": "web", "desc": "a description that runs well past the edge"},
		},
	}

	const expected = "ID    Name    Description\n" +
		"1     web     a description that runs well past the edge\n"

	for _, opts := range [][]ui.Option{
		{ui.WithWidth(46)},
		{ui.WithWidth(46), ui.WithASCII(true)},
		{},
	} {
		var out, errOut bytes.Buffer

		assert.NoError(t, DisplayText(item, &out, false, nil, ui.Detect(&out, &errOut, opts...)))
		assert.Equal(t, expected, out.String())
	}
}

// TestDisplayTextKeepsUnbreakableValuesWhole is the rule that decides which
// tables get fitted at all: an ID, an address or a name is the value a user
// came to the table for and usually the value they are about to copy out of
// it, so it is shown whole even when that costs the fit and the rules that
// would have been drawn around it.
//
// Stacking is how it is shown whole without also overrunning the terminal,
// which is what the full-width wrap this replaced had to do.
func TestDisplayTextKeepsUnbreakableValuesWhole(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"id", "name", "ipv4", "tags"},
		colMap: map[string]string{"id": "ID", "name": "Name", "ipv4": "Public IPv4", "tags": "Tags"},
		kv: []map[string]any{
			{"id": 504211456, "name": "a-fairly-long-droplet-name", "ipv4": "167.71.255.12", "tags": "web,production"},
		},
	}

	var out, errOut bytes.Buffer
	env := terminalEnv(&out, &errOut, ui.WithWidth(46))

	require.NoError(t, DisplayText(item, &out, false, nil, env))

	assert.Equal(t, "• a-fairly-long-droplet-name · 504211456\n"+
		"  public ipv4 167.71.255.12\n"+
		"  tags web,production\n", out.String())
	assert.NotContains(t, out.String(), "…", "no value should have been cut")
	assert.NotContains(t, out.String(), "│", "a table the terminal cannot hold is not boxed")

	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), 46,
			"records should have kept %q inside the terminal", line)
	}
}

// TestDisplayTextRecordsListEveryField guards the promise the layout is built
// on: a record carries every column, including the ones it has no value for,
// so that what a resource is missing is as legible as what it has.
func TestDisplayTextRecordsListEveryField(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"id", "name", "ipv6", "region", "tags"},
		colMap: map[string]string{"id": "ID", "name": "Name", "ipv6": "Public IPv6", "region": "Region", "tags": "Tags"},
		kv: []map[string]any{
			{"id": 1, "name": "web", "ipv6": "", "region": "nyc3", "tags": ""},
		},
	}

	var out, errOut bytes.Buffer
	env := terminalEnv(&out, &errOut, ui.WithWidth(20))

	require.NoError(t, DisplayText(item, &out, false, nil, env))

	assert.Equal(t, "• web · 1\n"+
		"  public ipv6 —\n"+
		"  region nyc3\n"+
		"  tags —\n", out.String())
	assert.Equal(t, 2, strings.Count(out.String(), ui.GlyphNone),
		"the two empty fields should each stand in a placeholder")
}

// TestDisplayTextRecordsHeadlineTheIdentifiers covers what a record is read
// by: its ID and its name lead it and carry the accent, every other field sits
// beneath in the muted run, and the ID of some other resource is a field like
// any other rather than part of this record's heading.
func TestDisplayTextRecordsHeadlineTheIdentifiers(t *testing.T) {
	item := &testDisplayable{
		cols: []string{"id", "vpcid", "name", "status"},
		colMap: map[string]string{
			"id": "ID", "vpcid": "VPC ID", "name": "Name", "status": "Status",
		},
		kv: []map[string]any{
			{"id": "504211456", "vpcid": "9f1c2e", "name": "web", "status": "active"},
			{"id": "504211457", "vpcid": "9f1c2e", "name": "db", "status": "off"},
		},
	}

	var out, errOut bytes.Buffer
	env := terminalEnv(&out, &errOut, ui.WithWidth(36), ui.WithProfile(termenv.TrueColor))

	require.NoError(t, DisplayText(item, &out, false, nil, env))
	rendered := out.String()

	for _, identifier := range []string{"504211456", "web", "504211457", "db"} {
		assert.Contains(t, rendered, env.SprintTone(ui.ToneIdentifier, identifier),
			"%q names the resource, so it should carry the accent", identifier)
	}
	assert.NotContains(t, rendered, env.SprintTone(ui.ToneIdentifier, "9f1c2e"),
		"another resource's ID is a field, not part of the heading")

	// The glyph reports the state the record is in, which is the only place a
	// headline says anything beyond naming the resource.
	assert.Contains(t, rendered, env.SprintTone(ui.ToneSuccess, ui.GlyphBullet))
	assert.Contains(t, rendered, env.SprintTone(ui.ToneMuted, ui.GlyphBullet))

	plain := ansi.Strip(rendered)
	assert.Equal(t, "• web · 504211456\n"+
		"  vpc id 9f1c2e · status active\n"+
		"\n"+
		"• db · 504211457\n"+
		"  vpc id 9f1c2e · status off\n", plain,
		"the name should lead the ID, and a blank line should separate records")
}

// TestDisplayTextRecordsHeadlineNameFirst guards the order of the two halves
// of a heading against the order they happen to be declared in. Most
// displayers put ID first and a few do not, and a reader scanning the page
// should not have to find the name in a different place for each.
func TestDisplayTextRecordsHeadlineNameFirst(t *testing.T) {
	tests := []struct {
		name string
		cols []string
	}{
		{name: "id declared first", cols: []string{"id", "name", "region", "tags"}},
		{name: "name declared first", cols: []string{"name", "id", "region", "tags"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &testDisplayable{
				cols: tt.cols,
				colMap: map[string]string{
					"id": "ID", "name": "Name", "region": "Region", "tags": "Tags",
				},
				kv: []map[string]any{
					{"id": "9f1c2e", "name": "web", "region": "nyc3", "tags": "prod"},
				},
			}

			var out, errOut bytes.Buffer
			env := terminalEnv(&out, &errOut, ui.WithWidth(24))

			require.NoError(t, DisplayText(item, &out, false, nil, env))

			assert.Equal(t, "• web · 9f1c2e\n"+
				"  region nyc3\n"+
				"  tags prod\n", out.String())
		})
	}
}

// TestDisplayTextRecordsHeadlineWithoutIdentifiers guards the fallback for a
// resource that declares neither an ID nor a name: its first column heads the
// record, so that no record is left without a heading.
func TestDisplayTextRecordsHeadlineWithoutIdentifiers(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"slug", "region", "tags"},
		colMap: map[string]string{"slug": "Slug", "region": "Region", "tags": "Tags"},
		kv: []map[string]any{
			{"slug": "1.36.3-do.2", "region": "nyc3", "tags": "current"},
		},
	}

	var out, errOut bytes.Buffer
	env := terminalEnv(&out, &errOut, ui.WithWidth(20), ui.WithProfile(termenv.TrueColor))

	require.NoError(t, DisplayText(item, &out, false, nil, env))

	assert.Contains(t, out.String(), env.SprintTone(ui.ToneIdentifier, "1.36.3-do.2"),
		"the first column should head the record and carry the accent")
	assert.Equal(t, "• 1.36.3-do.2\n"+
		"  region nyc3\n"+
		"  tags current\n", ansi.Strip(out.String()))
}

// TestDisplayTextRecordsNeedHeaders guards the fallback: --no-header leaves a
// record's fields with no labels to say what they are, which is the unlabelled
// run the layout exists to avoid.
func TestDisplayTextRecordsNeedHeaders(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"id", "name", "ipv4", "tags"},
		colMap: map[string]string{"id": "ID", "name": "Name", "ipv4": "Public IPv4", "tags": "Tags"},
		kv: []map[string]any{
			{"id": 504211456, "name": "a-fairly-long-droplet-name", "ipv4": "167.71.255.12", "tags": "web,production"},
		},
	}

	var out, errOut bytes.Buffer
	env := terminalEnv(&out, &errOut, ui.WithWidth(46))

	require.NoError(t, DisplayText(item, &out, true, nil, env))

	assert.Equal(t, "504211456    a-fairly-long-droplet-name    167.71.255.12    web,production\n",
		out.String(), "without headers it falls back to the full-width wrap")
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
			expected: "╭────┬──────────╮\n" +
				"│ ID │ Name     │\n" +
				"├────┼──────────┤\n" +
				"│ 1  │ web      │\n" +
				"│ 22 │ database │\n" +
				"╰────┴──────────╯\n",
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
			expected: "╭────┬──────────╮\n" +
				"│ 1  │ web      │\n" +
				"│ 22 │ database │\n" +
				"╰────┴──────────╯\n",
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

// TestDisplayTextBoxOnlyWhenItFits covers the choice the rules depend on. They
// are chrome, and chrome is only worth drawing around a table that fits inside
// it: the moment it does not, the layout changes rather than the values, since
// the width the rules take is not the user's to give up.
func TestDisplayTextBoxOnlyWhenItFits(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"id", "name", "desc"},
		colMap: map[string]string{"id": "ID", "name": "Name", "desc": "Description"},
		kv: []map[string]any{
			{"id": 1, "name": "web", "desc": "a description that runs well past the edge"},
		},
	}

	t.Run("a table within the width is boxed", func(t *testing.T) {
		const width = 80

		var out, errOut bytes.Buffer
		env := terminalEnv(&out, &errOut, ui.WithWidth(width))

		require.NoError(t, DisplayText(item, &out, false, nil, env))

		assert.Equal(t, "╭────┬──────┬────────────────────────────────────────────╮\n"+
			"│ ID │ Name │ Description                                │\n"+
			"├────┼──────┼────────────────────────────────────────────┤\n"+
			"│ 1  │ web  │ a description that runs well past the edge │\n"+
			"╰────┴──────┴────────────────────────────────────────────╯\n", out.String())

		for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
			assert.LessOrEqual(t, ansi.StringWidth(line), width, "line %q exceeds the width", line)
		}
	})

	t.Run("a table the rules would not fit becomes records", func(t *testing.T) {
		const width = 46

		var out, errOut bytes.Buffer
		env := terminalEnv(&out, &errOut, ui.WithWidth(width))

		require.NoError(t, DisplayText(item, &out, false, nil, env))

		assert.NotContains(t, out.String(), "│", "the rules should have been given up first")
		assert.NotContains(t, out.String(), "…", "no value should have been cut")
		// Wrapped over two lines, so the record is compared by its words.
		assert.Equal(t,
			strings.Fields("• web · 1 description a description that runs well past the edge"),
			strings.Fields(out.String()),
			"the value should have wrapped rather than lost its tail")
	})
}

// TestDisplayTextBoxStylingDoesNotMoveColumns holds the boxed layout to the
// same guarantee as the plain one: color may only add escape sequences.
func TestDisplayTextBoxStylingDoesNotMoveColumns(t *testing.T) {
	var plainOut, styledOut, errOut bytes.Buffer

	require.NoError(t, DisplayText(stateTable(), &plainOut, false, nil,
		terminalEnv(&plainOut, &errOut, ui.WithProfile(termenv.Ascii))))
	require.NoError(t, DisplayText(stateTable(), &styledOut, false, nil,
		terminalEnv(&styledOut, &errOut, ui.WithProfile(termenv.TrueColor))))

	assert.Equal(t, plainOut.String(), ansi.Strip(styledOut.String()))
	assert.Contains(t, styledOut.String(), "\x1b[", "a styled table should carry color")
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
		"the header row should be emphasized")
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

// TestDisplayTextStylingDoesNotMoveColumns is the guarantee that makes color
// safe to turn on: styling may only add escape sequences, never shift a cell.
func TestDisplayTextStylingDoesNotMoveColumns(t *testing.T) {
	var plainOut, styledOut, errOut bytes.Buffer

	require.NoError(t, DisplayText(stateTable(), &plainOut, false, nil, ui.Plain(&plainOut, &errOut)))

	styledEnv := ui.Detect(&styledOut, &errOut, ui.WithProfile(termenv.TrueColor))
	require.NoError(t, DisplayText(stateTable(), &styledOut, false, nil, styledEnv))

	assert.Equal(t, plainOut.String(), ansi.Strip(styledOut.String()))
}

// TestDisplayTextStylingNarrowTerminal covers styling where the table does not
// fit, which is where a layout that reflows and a layout that paints have to
// agree: the record layout decides its line breaks from measured widths, so an
// escape sequence leaking into that arithmetic would break them apart.
func TestDisplayTextStylingNarrowTerminal(t *testing.T) {
	var plainOut, styledOut, errOut bytes.Buffer

	const width = 56

	require.NoError(t, DisplayText(stateTable(), &plainOut, false, nil,
		terminalEnv(&plainOut, &errOut, ui.WithWidth(width), ui.WithProfile(termenv.Ascii))))
	require.NoError(t, DisplayText(stateTable(), &styledOut, false, nil,
		terminalEnv(&styledOut, &errOut, ui.WithWidth(width), ui.WithProfile(termenv.TrueColor))))

	assert.NotContains(t, plainOut.String(), "…", "no value should have been cut")
	assert.Equal(t, plainOut.String(), ansi.Strip(styledOut.String()))
	for _, line := range strings.Split(strings.TrimRight(styledOut.String(), "\n"), "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), width, "line %q exceeds the width", line)
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

// TestIsIdentifierColumn pins the exact match that isStateColumn does not use.
// A trailing "name" generally names some other resource - a row's Model Name
// or Droplet Name describes its subject rather than being it - and accenting
// those would put several highlights in a row that has one subject.
func TestIsIdentifierColumn(t *testing.T) {
	for _, col := range []string{"Name", "name", "NAME"} {
		assert.True(t, isIdentifierColumn(col), "%q identifies the resource", col)
	}

	for _, col := range []string{
		"ID", "ModelName", "Model Name", "DropletName", "Droplet Name",
		"Spec Name", "UserName", "Status", "Message",
	} {
		assert.False(t, isIdentifierColumn(col), "%q does not identify the resource", col)
	}
}

// TestDisplayTextAccentsTheIdentifier covers the one tone decided by the
// column rather than by the value.
func TestDisplayTextAccentsTheIdentifier(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"name", "status"},
		colMap: map[string]string{"name": "Name", "status": "Status"},
		kv: []map[string]any{
			{"name": "web", "status": "active"},
			{"name": "", "status": "active"},
		},
	}

	var out, errOut bytes.Buffer
	env := ui.Detect(&out, &errOut, ui.WithProfile(termenv.TrueColor))

	require.NoError(t, DisplayText(item, &out, false, nil, env))
	rendered := out.String()

	assert.Contains(t, rendered, env.SprintTone(ui.ToneIdentifier, "web"),
		"the name should carry the identifier accent")

	// An empty name has nothing to find, so it is left without escapes rather
	// than wrapped in a pair that paints nothing.
	assert.NotContains(t, rendered, env.SprintTone(ui.ToneIdentifier, ""))
}

// tabularDisplayable opts out of the card layout.
type tabularDisplayable struct {
	testDisplayable
}

func (t *tabularDisplayable) Tabular() bool { return true }

// oneResource is the table a get command produces: a single row.
func oneResource() testDisplayable {
	return testDisplayable{
		cols:   []string{"id", "name", "status", "ipv6"},
		colMap: map[string]string{"id": "ID", "name": "Name", "status": "Status", "ipv6": "Public IPv6"},
		kv:     []map[string]any{{"id": 1, "name": "web", "status": "active", "ipv6": ""}},
	}
}

func detailItem() *testDisplayable {
	item := oneResource()

	return &item
}

// TestDisplayTextCardsASingleResource covers the layout a get command is given
// on a terminal, where a row with more fields than the window has columns is
// turned on its side instead of being wrapped.
func TestDisplayTextCardsASingleResource(t *testing.T) {
	var out, errOut bytes.Buffer
	env := terminalEnv(&out, &errOut, ui.WithProfile(termenv.TrueColor))

	require.NoError(t, DisplayText(detailItem(), &out, false, nil, env, WithDetail(true)))
	rendered := out.String()

	assert.Contains(t, rendered, "╭", "a card is drawn in rules")
	assert.NotContains(t, rendered, "┬", "a card has no interior columns to join")
	assert.Contains(t, rendered, env.Sprint(env.NewStyle().Foreground(ui.ColorInfo).Bold(true), "Name"),
		"a card's labels are emphasized")
	// The top rule is painted as one span, so it is compared whole rather than
	// by its corner, which keeps the assertion free of the card's width.
	top := strings.SplitN(rendered, "\n", 2)[0]
	assert.Equal(t, env.Sprint(env.NewStyle().Foreground(ui.ColorSuccess), ansi.Strip(top)), top,
		"a card's rules are drawn in the frame color")

	// Values keep the tones they carry in a table, so the identifier and the
	// state are still found by eye.
	assert.Contains(t, rendered, env.SprintTone(ui.ToneIdentifier, "web"))
	assert.Contains(t, rendered, env.SprintTone(ui.ToneSuccess, "active"))

	// Two rules around one line per non-empty field.
	lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	assert.Len(t, lines, 5, "three fields between two rules")
	assert.NotContains(t, rendered, "Public IPv6", "an empty value is dropped")
}

func TestDisplayTextCardFooter(t *testing.T) {
	next := "doctl compute droplet get 42"

	t.Run("a next step is set below the fields", func(t *testing.T) {
		var out, errOut bytes.Buffer
		env := terminalEnv(&out, &errOut, ui.WithProfile(termenv.TrueColor))

		require.NoError(t, DisplayText(detailItem(), &out, false, nil, env,
			WithDetail(true), WithNextStep(next)))
		rendered := out.String()

		assert.Contains(t, rendered, env.Sprint(env.NewStyle().Foreground(ui.ColorMuted), nextStepHeading),
			"the heading is muted chrome")
		assert.Contains(t, rendered, next, "the command is left whole for copying")

		lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
		assert.Len(t, lines, 8, "a blank line, the heading and the command follow the fields")

		// The rules still meet the corners once the footer has widened it.
		assert.Equal(t, ansi.StringWidth(lines[0]), ansi.StringWidth(lines[len(lines)-2]),
			"the footer sits inside the card")
	})

	t.Run("no next step leaves the card as it was", func(t *testing.T) {
		var out, errOut bytes.Buffer
		env := terminalEnv(&out, &errOut, ui.WithProfile(termenv.TrueColor))

		require.NoError(t, DisplayText(detailItem(), &out, false, nil, env, WithDetail(true)))

		assert.NotContains(t, out.String(), nextStepHeading)
	})

	t.Run("a table is given no footer", func(t *testing.T) {
		var out, errOut bytes.Buffer
		env := terminalEnv(&out, &errOut, ui.WithProfile(termenv.TrueColor))

		// Without WithDetail the same resource is a table, which is a list of
		// resources and has no single one to suggest a command for.
		require.NoError(t, DisplayText(detailItem(), &out, false, nil, env, WithNextStep(next)))

		assert.NotContains(t, out.String(), nextStepHeading)
	})
}

// TestDisplayTextCardWidth covers the two directions the value column is
// sized in: narrowed to the terminal, and left alone when it already fits. A
// narrowed card wraps its values; the width it gives up is never taken out of
// them.
func TestDisplayTextCardWidth(t *testing.T) {
	item := &testDisplayable{
		cols:   []string{"name", "note"},
		colMap: map[string]string{"name": "Name", "note": "Note"},
		kv: []map[string]any{{
			"name": "web",
			"note": strings.Repeat("wide-", 40),
		}},
	}

	t.Run("a card is narrowed to the terminal", func(t *testing.T) {
		const width = 48

		var out, errOut bytes.Buffer
		require.NoError(t, DisplayText(item, &out, false, nil,
			terminalEnv(&out, &errOut, ui.WithWidth(width)), WithDetail(true)))

		assert.NotContains(t, out.String(), "…", "the long value should have wrapped, not been cut")
		for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
			assert.LessOrEqual(t, ansi.StringWidth(line), width, "line %q is too wide", line)
		}

	})

	// A short card keeps its natural width rather than being padded out to
	// the floor the narrowing stops at.
	t.Run("a card that fits is not padded", func(t *testing.T) {
		short := &testDisplayable{
			cols:   []string{"status"},
			colMap: map[string]string{"status": "Status"},
			kv:     []map[string]any{{"status": "up"}},
		}

		var out, errOut bytes.Buffer
		require.NoError(t, DisplayText(short, &out, false, nil,
			terminalEnv(&out, &errOut, ui.WithWidth(80)), WithDetail(true)))

		lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
		require.Len(t, lines, 3)

		// Two rules, both margins, the gap, and just enough for "up".
		want := 2 + 2*cardPad + len("Status") + cardGap + len("up")
		assert.Equal(t, want, ansi.StringWidth(lines[0]),
			"the value column should not be widened to minCardValue; got %q", lines[0])
	})
}

// TestDisplayTextCardIsTerminalOnly is the guard that keeps scripts working. A
// card is chrome, and `doctl ... get x --format ID,Name` is parsed by splitting
// on whitespace, so a pipe has to keep the columns doctl has always written.
func TestDisplayTextCardIsTerminalOnly(t *testing.T) {
	var out, errOut bytes.Buffer

	require.NoError(t, DisplayText(detailItem(), &out, false, nil, ui.Plain(&out, &errOut),
		WithDetail(true)))

	assert.NotContains(t, out.String(), "│", "a pipe gets no rules of any kind")
	assert.Len(t, strings.Split(strings.TrimRight(out.String(), "\n"), "\n"), 2,
		"a pipe keeps the header and its single row")
}

// TestDisplayTextCardFallsBackToTheTable pins the cases that stay tabular even
// on a terminal.
func TestDisplayTextCardFallsBackToTheTable(t *testing.T) {
	tests := []struct {
		name      string
		item      Displayable
		detail    bool
		noHeaders bool
	}{
		{
			// Without headers there are no labels, and a card of bare values
			// is just an indented column.
			name:      "no headers were asked for",
			item:      detailItem(),
			detail:    true,
			noHeaders: true,
		},
		{
			name:   "the command was not a get",
			item:   detailItem(),
			detail: false,
		},
		{
			name:   "the displayer opted out",
			item:   &tabularDisplayable{oneResource()},
			detail: true,
		},
		{
			name:   "more than one resource is a list",
			detail: true,
			item: func() Displayable {
				item := detailItem()
				item.kv = append(item.kv, map[string]any{
					"id": 2, "name": "db", "status": "active", "ipv6": "",
				})

				return item
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer

			require.NoError(t, DisplayText(tt.item, &out, tt.noHeaders, nil,
				terminalEnv(&out, &errOut), WithDetail(tt.detail)))

			// Both layouts are drawn in the same rounded rules, so what tells
			// them apart is that only a table joins columns.
			assert.Contains(t, out.String(), "┬", "should have stayed a table")
		})
	}
}

func TestFormatCell(t *testing.T) {
	for _, human := range []bool{true, false} {
		assert.Equal(t, "1.500000", formatCell(float64(1.5), human))
		assert.Equal(t, "42", formatCell(42, human))
		assert.Equal(t, "true", formatCell(true, human))
		assert.Equal(t, "abc", formatCell("abc", human))

		// An empty list is the displayer's statement that there is a list and
		// it is empty, which is not the same as having none.
		assert.Equal(t, "[]", formatCell([]string{}, human))
	}
}

// TestFormatCellSpellsNilPerStream pins the promise that gates the empty cell:
// a terminal is spared Go's "<nil>", and a redirected stream keeps it, so a
// script parsing doctl's output sees the bytes it was written against.
func TestFormatCellSpellsNilPerStream(t *testing.T) {
	tests := []struct {
		name  string
		value any
		human string
		piped string
	}{
		{
			name:  "an unset key holds the untyped nil",
			value: nil,
			human: "", piped: "<nil>",
		},
		{
			name:  "an absent field holds a typed nil",
			value: (*string)(nil),
			human: "", piped: "<nil>",
		},
		{
			name:  "a compound cell carries the nil inside it",
			value: "protocol:http,proxy_protocol:<nil>",
			human: "protocol:http,proxy_protocol:", piped: "protocol:http,proxy_protocol:<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.human, formatCell(tt.value, true), "a terminal")
			assert.Equal(t, tt.piped, formatCell(tt.value, false), "a pipe")
		})
	}
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
		expected []int
	}{
		{
			name:     "each column takes the width of its widest value",
			headers:  headers,
			rows:     rows,
			expected: []int{2, 4, 32},
		},
		{
			name:     "a header wider than every value holds the column open",
			headers:  []string{"Description", "Name"},
			rows:     [][]string{{"short", "cdef"}},
			expected: []int{11, 4},
		},
		{
			name:     "a column with no header is measured by its values alone",
			rows:     [][]string{{"a b c d e"}},
			expected: []int{9},
		},
		{
			name:     "double-width runes are measured in terminal cells",
			headers:  []string{"名前"},
			rows:     [][]string{{"ab"}},
			expected: []int{4},
		},
		{
			name:     "no columns yields no widths",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, columnWidths(tt.headers, tt.rows))
		})
	}
}

// TestWrapValue is the guarantee the card rests on: a value too wide for its
// column is laid out over more lines, never shortened. It breaks where a
// reader would when the value gives it somewhere, and mid-run when it does not.
func TestWrapValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		width    int
		expected []string
	}{
		{
			name:     "a value that fits is left on one line",
			value:    "web-01",
			width:    10,
			expected: []string{"web-01"},
		},
		{
			name:     "prose breaks between words",
			value:    "a description that runs past the edge",
			width:    16,
			expected: []string{"a description", "that runs past", "the edge"},
		},
		{
			name:     "a list breaks after a comma",
			value:    "web,production,europe",
			width:    12,
			expected: []string{"web,", "production,", "europe"},
		},
		{
			name:     "a run with nowhere to break is broken anyway",
			value:    "504211456504211456",
			width:    8,
			expected: []string{"50421145", "65042114", "56"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := wrapValue(tt.value, tt.width)

			assert.Equal(t, tt.expected, lines)
			for _, line := range lines {
				assert.LessOrEqual(t, ansi.StringWidth(line), tt.width, "line %q is too wide", line)
			}
		})
	}
}

func TestFitsBudget(t *testing.T) {
	widths := []int{2, 4, 32}

	assert.True(t, fitsBudget(widths, 0), "an unconstrained stream fits anything")
	assert.True(t, fitsBudget(widths, 38), "a budget the table exactly fills")
	assert.True(t, fitsBudget(widths, 200))
	assert.False(t, fitsBudget(widths, 37), "one cell short is still too narrow")
	assert.True(t, fitsBudget(nil, 1), "a table with no columns always fits")
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
			r := renderer{buf: &buf, ellipsis: "…"}

			r.row(tt.cells, tt.widths, nil)

			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

// TestBreakValueRestoresSeparators guards the fidelity the record layout
// promises: a value laid out on one line must read exactly as the displayer
// wrote it, separators and all. Values joined with ", " lost the space when
// the joiner was replaced rather than accumulated.
func TestBreakValueRestoresSeparators(t *testing.T) {
	for _, value := range []string{
		"web,production",
		"Ubuntu 22.04 x64",
		"vpc-1, vpc-2, vpc-3",
		"protocol:http,port:80,path:/",
		"a  b",
		"",
	} {
		var got string
		for _, part := range breakValue(value) {
			got += part.joiner + part.text
		}

		assert.Equal(t, value, got, "breakValue must rejoin to the value it split")
	}
}
