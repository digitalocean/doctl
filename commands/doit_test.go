package commands

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/digitalocean/doctl/internal/ui"
)

func TestFlagName(t *testing.T) {
	var flag = "thing"
	testFn := func(c *CmdConfig) error {
		return nil
	}
	parent := &Command{
		Command: &cobra.Command{
			Use:   "doit",
			Short: "Do the thing",
		},
	}

	tests := []struct {
		name     string
		cmd      *Command
		expected string
	}{
		{
			name:     "default",
			cmd:      CmdBuilder(parent, testFn, "run", "Run it", "", Writer),
			expected: "doit.run.thing",
		},
		{
			name:     "top-level",
			cmd:      parent,
			expected: "doit.thing",
		},
		{
			name:     "overrideCmdNS",
			cmd:      CmdBuilder(parent, testFn, "run", "Run it", "", Writer, overrideCmdNS("doctl")),
			expected: "doctl.run.thing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddStringFlag(tt.cmd, flag, "", "", "the thing")

			assert.Equal(t, tt.expected, flagName(tt.cmd, flag))
		})
	}
}

// TestIsDetailNS covers the verb check that decides which commands are shown
// as a card. It reads the namespace rather than the command, so the nesting
// depth must not matter.
func TestIsDetailNS(t *testing.T) {
	for _, ns := range []string{"get", "droplet.get", "databases.pool.get", "droplet.create"} {
		assert.True(t, isDetailNS(ns), "%q reports one resource", ns)
	}

	for _, ns := range []string{
		"droplet.list", "droplet.get-backup-policy", "droplet.create-snapshot", "", "droplet.",
	} {
		assert.False(t, isDetailNS(ns), "%q does not report one resource", ns)
	}
}

// stubDisplayable is a resource with whatever fields a test needs, so that
// nextStep can be exercised without building a full API type.
type stubDisplayable struct {
	rows []map[string]any
}

func (s stubDisplayable) Cols() []string { return []string{"ID", "Name"} }
func (s stubDisplayable) ColMap() map[string]string {
	return map[string]string{"ID": "ID", "Name": "Name"}
}
func (s stubDisplayable) KV() []map[string]any   { return s.rows }
func (s stubDisplayable) JSON(w io.Writer) error { return nil }

func TestNextStep(t *testing.T) {
	// A resource whose commands are the shape doctl uses: a create and a get
	// under one parent.
	tree := func(withGet bool) *cobra.Command {
		parent := &cobra.Command{Use: "droplet"}
		root := &cobra.Command{Use: "doctl"}
		compute := &cobra.Command{Use: "compute"}
		root.AddCommand(compute)
		compute.AddCommand(parent)

		create := &cobra.Command{Use: "create"}
		parent.AddCommand(create)
		if withGet {
			parent.AddCommand(&cobra.Command{Use: "get"})
		}

		return create
	}

	t.Run("suggests the get for the resource", func(t *testing.T) {
		item := stubDisplayable{rows: []map[string]any{{"ID": 42, "Name": "web-01"}}}

		assert.Equal(t, "doctl compute droplet get 42", nextStep(tree(true), item))
	})

	t.Run("falls back to the help when there is no get", func(t *testing.T) {
		item := stubDisplayable{rows: []map[string]any{{"ID": 42, "Name": "web-01"}}}

		assert.Equal(t, "doctl --help", nextStep(tree(false), item))
	})

	t.Run("falls back to the help without a single resource to name", func(t *testing.T) {
		item := stubDisplayable{}

		assert.Equal(t, "doctl --help", nextStep(tree(true), item))
	})

	t.Run("a get suggests nothing rather than itself", func(t *testing.T) {
		item := stubDisplayable{rows: []map[string]any{{"ID": 42, "Name": "web-01"}}}
		get := &cobra.Command{Use: "get"}
		(&cobra.Command{Use: "droplet"}).AddCommand(get)

		assert.Empty(t, nextStep(get, item))
	})
}

func TestCmdNS(t *testing.T) {
	testFn := func(c *CmdConfig) error {
		return nil
	}
	parent := &Command{
		Command: &cobra.Command{
			Use:   "doit",
			Short: "Do the thing",
		},
	}

	tests := []struct {
		name     string
		cmd      *Command
		expected string
	}{
		{
			name:     "default",
			cmd:      CmdBuilder(parent, testFn, "run", "Run it", "", Writer),
			expected: "doit.run",
		},
		{
			name:     "top-level",
			cmd:      parent,
			expected: "doit",
		},
		{
			name:     "overrideCmdNS",
			cmd:      CmdBuilder(parent, testFn, "run", "Run it", "", Writer, overrideCmdNS("doctl")),
			expected: "doctl.run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, cmdNS(tt.cmd))
		})
	}
}

func TestReportingWriter(t *testing.T) {
	t.Run("notes a write", func(t *testing.T) {
		var buf bytes.Buffer
		w := &reportingWriter{out: &buf}

		fmt.Fprint(w, "something")

		assert.True(t, w.wrote.Load())
		assert.Equal(t, "something", buf.String(), "the write still reaches the stream")
	})

	t.Run("an untouched writer reports nothing", func(t *testing.T) {
		w := &reportingWriter{out: &bytes.Buffer{}}

		assert.False(t, w.wrote.Load())
	})

	t.Run("an empty write is not a report", func(t *testing.T) {
		var buf bytes.Buffer
		w := &reportingWriter{out: &buf}

		n, err := w.Write(nil)

		require.NoError(t, err)
		assert.Zero(t, n)
		assert.False(t, w.wrote.Load(), "a command that wrote no bytes still said nothing")
	})
}

// TestNoticeRecordsTheOutcome guards against the default closing line
// following a command that already reported. Chrome goes to stderr rather
// than through the command's writer, so the writer alone cannot answer
// whether the user heard anything.
func TestNoticeRecordsTheOutcome(t *testing.T) {
	t.Cleanup(func() { reportedOutcome = false })

	t.Run("a notice is an outcome", func(t *testing.T) {
		reportedOutcome = false

		notice("Context deleted successfully")

		assert.True(t, reportedOutcome, "a notice tells the user how the command went")
	})

	t.Run("a warning is not", func(t *testing.T) {
		reportedOutcome = false

		warn("that flag is deprecated")

		assert.False(t, reportedOutcome,
			"a warning says something went oddly, not that the command finished")
	})
}

// TestOwesClosingLine covers every reason doctl stays quiet about a command
// that finished. The stream test is the load-bearing one: 66 integration
// tests assert that a piped doctl says nothing on success.
func TestOwesClosingLine(t *testing.T) {
	t.Cleanup(func() { reportedOutcome = false })

	tests := []struct {
		name     string
		wrote    bool
		reported bool
		optedOut bool
		errTTY   bool
		owes     bool
	}{
		{
			name:   "a silent command on a terminal is closed",
			errTTY: true, owes: true,
		},
		{
			name:  "a command that reported is left alone",
			wrote: true, errTTY: true,
		},
		{
			name:     "a command that already said how it went is left alone",
			reported: true, errTTY: true,
		},
		{
			name:     "a command that opted out is left alone",
			optedOut: true, errTTY: true,
		},
		{
			name:   "a redirected stderr keeps the silence a script expects",
			errTTY: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reportedOutcome = tt.reported
			cmd := &Command{noDefaultSuccess: tt.optedOut}

			assert.Equal(t, tt.owes,
				cmd.owesClosingLine(tt.wrote, ui.Env{ErrTTY: tt.errTTY}))
		})
	}
}
