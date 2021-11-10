/*
Copyright 2018 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/doctl"
	"github.com/fatih/color"
	"github.com/shiena/ansicolor"
	"github.com/spf13/viper"
)

var (
	errOperationAborted = fmt.Errorf("Operation aborted.")

	colorErr    = color.RedString("Error")
	colorWarn   = color.YellowString("Warning")
	colorNotice = color.GreenString("Notice")

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

func checkErr(err error) {
	if err == nil {
		return
	}

	output := viper.GetString("output")

	switch output {
	default:
		fmt.Fprintf(color.Output, "%s: %v\n", colorErr, err)
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

func ensureOneArg(c *CmdConfig) error {
	switch count := len(c.Args); {
	case count == 0:
		return doctl.NewMissingArgsErr(c.NS)
	case count > 1:
		return doctl.NewTooManyArgsErr(c.NS)
	default:
		return nil
	}
}

func warn(msg string, args ...interface{}) {
	fmt.Fprintf(color.Output, "%s: %s\n", colorWarn, fmt.Sprintf(msg, args...))
}
func warnConfirm(msg string, args ...interface{}) {
	fmt.Fprintf(color.Output, "%s: %s", colorWarn, fmt.Sprintf(msg, args...))
}

func notice(msg string, args ...interface{}) {
	fmt.Fprintf(color.Output, "%s: %s\n", colorNotice, fmt.Sprintf(msg, args...))
}
