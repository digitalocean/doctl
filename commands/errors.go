package commands

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	colorErr = color.New(color.FgRed).SprintFunc()("Error")

	// errAction specifies what should happen when an error occurs
	errAction = func() {
		os.Exit(1)
	}
)

func checkErr(err error, cmd ...*cobra.Command) {
	if err == nil {
		return
	}

	if len(cmd) > 0 {
		cmd[0].Help()
	}

	fmt.Fprintf(color.Output, "\n%s: %v\n", colorErr, err)

	errAction()
}
