package commands

import (
	"fmt"
	"os"

	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/fatih/color"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/cobra"
)

var (
	colorErr = color.New(color.FgRed).SprintFunc()("Error")
)

// MissingArgsErr is an error returned when their are too few arguments for a command.
type MissingArgsErr struct {
	Command string
}

var _ error = &MissingArgsErr{}

// NewMissingArgsErr creates a MissingArgsErr instance.
func NewMissingArgsErr(cmd string) *MissingArgsErr {
	return &MissingArgsErr{Command: cmd}
}

func (e *MissingArgsErr) Error() string {
	return fmt.Sprintf("(%s) command is missing required arguments", e.Command)
}

func checkErr(err error, cmd ...*cobra.Command) {
	if err == nil {
		return
	}

	if len(cmd) > 0 {
		cmd[0].Help()
	}

	fmt.Fprintf(color.Output, "\n%s: %v\n", colorErr, err)
	os.Exit(1)
}
