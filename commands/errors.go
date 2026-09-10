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

// notice reports how a command went. It records that it has done so, which is
// what keeps the default closing line from following a command that already
// said its own piece.
func notice(msg string, args ...any) {
	reportedOutcome = true
	writeChrome("Notice", ui.ColorSuccess, "\n", msg, args...)
}

// reportSuccess is the closing line of a command that reported nothing itself.
// Unlike a notice it carries no label: there is nothing to introduce, so the
// tick and the sentence are painted together as one mark of completion.
func reportSuccess(msg string, args ...any) {
	reportedOutcome = true

	env := uiEnv()
	line := ui.NewStyle(env).SuccessLine(fmt.Sprintf(msg, args...))
	fmt.Fprintf(env.ErrWriter(), "%s\n", line)
}

// writeChrome renders a labelled diagnostic on stderr. The color decision
// comes from ui.Env, which resolves it per stream.
func writeChrome(label string, color lipgloss.TerminalColor, suffix, msg string, args ...any) {
	env := uiEnv()

	// Bolded as well as colored, for the reason ui.Style.paint gives: this is
	// a label, and weight is what sets it apart from the message it
	// introduces while the two share a color.
	label = env.SprintErr(env.NewErrStyle().Foreground(color).Bold(true), label)

	fmt.Fprintf(env.ErrWriter(), "%s: %s%s", label, fmt.Sprintf(msg, args...), suffix)
}
