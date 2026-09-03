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
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/ui"
)

var (
	errOperationAborted = fmt.Errorf("Operation aborted.")

	// errAction specifies what should happen when an error occurs
	errAction = func() {
		os.Exit(1)
	}

	// ErrExitSilently instructs doctl to exit silently with a bad status code. This can be used to fail a command
	// without printing an error message to the screen.
	//
	// IMPORTANT! Make sure to print your own error message if you use this! It is important for users to know
	// what caused the failure.
	ErrExitSilently = fmt.Errorf("")
)

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

	if errors.Is(err, ErrExitSilently) {
		errAction()
		return
	}

	switch outputFormat() {
	default:
		env := uiEnv()

		var fv *FlagValidationError
		if errors.As(err, &fv) {
			// The validation block renders its own label, so it is printed
			// as-is rather than prefixed a second time.
			fmt.Fprintln(env.ErrWriter(), fv.format(ui.NewStyle(env)))
			errAction()
			return
		}

		// Every failure carries the same label, whatever produced it, so
		// that a validation error and an API error read as one voice.
		fmt.Fprintf(env.ErrWriter(), "%s %v\n", ui.NewStyle(env).ErrorLabel(), err)
	case "json":
		// Always keep the stable {"errors":[{"detail":...}]} envelope so
		// automation parsing --output json is not broken by richer flag
		// validation. Plain Error() text (no ANSI) goes in detail.
		payload := outputErrors{
			Errors: []outputError{
				{Detail: err.Error()},
			},
		}

		b, _ := json.Marshal(payload)
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

func warn(msg string, args ...any) {
	writeChrome("Warning", ui.ColorWarning, "\n", msg, args...)
}

func notice(msg string, args ...any) {
	writeChrome("Notice", ui.ColorSuccess, "\n", msg, args...)
}

// writeChrome renders a labelled diagnostic on stderr.
//
// The colour decision comes from ui.Env, which resolves it per stream. The
// package-init decision it replaced looked at whether *stdout* was a terminal,
// so `doctl ... 2>log` wrote escape sequences into the log file and
// `doctl ... > data` stripped colour from a terminal well able to show it.
func writeChrome(label string, color lipgloss.Color, suffix, msg string, args ...any) {
	env := uiEnv()
	label = env.SprintErr(env.NewErrStyle().Foreground(color).Bold(true), label)

	fmt.Fprintf(env.ErrWriter(), "%s: %s%s", label, fmt.Sprintf(msg, args...), suffix)
}
