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
