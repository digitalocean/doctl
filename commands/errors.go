/*
Copyright 2016 The Doctl Authors All rights reserved.
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

package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/shiena/ansicolor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	colorErr  = color.RedString("Error")
	colorWarn = color.YellowString("Warning")

	// errAction specifies what should happen when an error occurs
	errAction = func() {
		os.Exit(1)
	}
)

func init() {
	color.Output = ansicolor.NewAnsiColorWriter(os.Stderr)
}

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

func warn(msg string) {
	fmt.Fprintf(color.Output, "%s: %s\n\n", colorWarn, msg)
}
