package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
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
