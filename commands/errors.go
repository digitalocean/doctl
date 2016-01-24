package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	colorErr = color.New(color.FgRed).SprintFunc()("Error")

	// errAction specifies what should happen when an error occurs
	errAction = func() {
		os.Exit(1)
	}
)

type outputErrors struct {
	Errors []outputError `json:"errors"`
}

type outputError struct {
	Detail string `json:"detail"`
}

func checkErr(err error, cmd ...*cobra.Command) {
	if err == nil {
		return
	}

	output := viper.GetString("output")

	switch output {
	default:
		if len(cmd) > 0 {
			cmd[0].Help()
		}
		fmt.Fprintf(color.Output, "\n%s: %v\n", colorErr, err)
	case "json":
		es := outputErrors{
			Errors: []outputError{
				{Detail: err.Error()},
			},
		}

		b, _ := json.Marshal(&es)
		fmt.Println(string(b))
	}

	errAction()
}
