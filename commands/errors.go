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
	"net/http"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/spf13/cobra"
)

const (
	// exitGeneralError is doctl's default failure code: a command ran and
	// returned an error, or startup failed for a reason unrelated to how
	// doctl was invoked (e.g. an unreadable config file).
	exitGeneralError = 1

	// exitUsageError is reserved for failures that never reached a command's
	// handler - an unknown subcommand or a flag Cobra itself rejected before
	// Run. Kept distinct from exitGeneralError so the two are intentional,
	// not whichever branch happened to catch the error.
	exitUsageError = 255
)

var (
	errOperationAborted = fmt.Errorf("Operation aborted.")

	// errAction specifies what should happen when an error occurs
	errAction = func() {
		os.Exit(exitGeneralError)
	}

	// ErrExitSilently instructs doctl to exit silently with a bad status code. This can be used to fail a command
	// without printing an error message to the screen.
	//
	// IMPORTANT! Make sure to print your own error message if you use this! It is important for users to know
	// what caused the failure.
	ErrExitSilently = fmt.Errorf("")

	// activeCommand is the cobra command currently running, set by
	// cmdBuilderWithInit's Run before the handler is invoked. It exists so
	// checkErr can default the next step to that command's --help without
	// every call site having to pass a *cobra.Command through.
	activeCommand *cobra.Command
)

// NextStepper lets an error supply its own "next step" suggestion, overriding
// the default `<command> --help` hint checkErr otherwise prints. It is the
// narrow interface for an error that only wants to change one field; a type
// wanting to control the whole block implements StructuredError instead.
type NextStepper interface {
	NextStep() string
}

// StructuredError lets an error supply the whole Title → Reason → Status →
// Request ID → Next step block checkErr renders, and the same fields
// mirrored into the JSON envelope. Every error checkErr sees is resolved to
// one via resolveStructured, so checkErr and the JSON path always have a
// single source to read from, whether or not err implements this itself.
type StructuredError interface {
	error

	// Title is the one-line summary shown after the "Error:" label. Falls
	// back to err.Error() when empty, so a plain error renders exactly as it
	// always has.
	Title() string

	// Reason expands on Title with what the failure means. Empty suppresses
	// the line.
	Reason() string

	// Status is the API status code the failure carries, or 0 if none.
	Status() int

	// RequestID is the identifier the API assigned the request, for support
	// to look up. Empty suppresses the line.
	RequestID() string

	// NextStep is the suggested command. Empty suppresses the line.
	NextStep() string
}

// genericStructuredError is the StructuredError synthesized for an error
// that doesn't implement the interface itself. It is what makes an
// unannotated godo API error render with the same Title/Reason/Status/
// NextStep shape as a purpose-built one, by reading the status-code table.
type genericStructuredError struct {
	err error
}

func (e genericStructuredError) Error() string { return e.err.Error() }
func (e genericStructuredError) Unwrap() error { return e.err }

// Title is left blank for anything the status-code table doesn't recognize,
// so checkErr falls back to printing err.Error() as it always has.
func (e genericStructuredError) Title() string {
	if status := statusFor(e.err); status != 0 {
		return http.StatusText(status)
	}
	return ""
}

// Reason prefers the message the API itself returned, since it is specific
// to the request that failed; the status-code table's line is a fallback for
// when the API had nothing more to say than the status code.
func (e genericStructuredError) Reason() string {
	if gerr, ok := apiError(e.err); ok && gerr.Message != "" {
		return gerr.Message
	}
	if entry, ok := lookupErrorCode(e.err); ok {
		return entry.Reason
	}
	return ""
}

func (e genericStructuredError) Status() int {
	return statusFor(e.err)
}

// RequestID is the identifier godo's API client attached to the request, so
// that reporting an issue to support can reference it without digging it out
// of Error()'s full text.
func (e genericStructuredError) RequestID() string {
	if gerr, ok := apiError(e.err); ok {
		return gerr.RequestID
	}
	return ""
}

// NextStep prefers an explicit NextStepper override on the wrapped error,
// then the status-code table, then the generic `<command> --help` fallback.
func (e genericStructuredError) NextStep() string {
	var ns NextStepper
	if errors.As(e.err, &ns) {
		if step := ns.NextStep(); step != "" {
			return step
		}
	}
	if entry, ok := lookupErrorCode(e.err); ok {
		if step := resolvedNextStep(entry); step != "" {
			return step
		}
	}
	return defaultNextStep()
}

// defaultNextStep suggests the active command's help text. Empty if no
// command is active (e.g. errors raised during early config bootstrap).
func defaultNextStep() string {
	if activeCommand == nil {
		return ""
	}
	return fmt.Sprintf("run %s --help", activeCommand.CommandPath())
}

// resolveStructured returns err's own StructuredError if it implements one,
// otherwise a genericStructuredError wrapping it.
func resolveStructured(err error) StructuredError {
	var se StructuredError
	if errors.As(err, &se) {
		return se
	}
	return genericStructuredError{err: err}
}

type outputErrors struct {
	Errors []outputError `json:"errors"`
}

type outputError struct {
	Detail    string `json:"detail"`
	Title     string `json:"title,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Status    int    `json:"status,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	NextStep  string `json:"next_step,omitempty"`
}

func checkErr(err error) {
	if err == nil {
		return
	}

	if errors.Is(err, ErrExitSilently) {
		errAction()
		return
	}

	se := resolveStructured(err)

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
		// that a validation error and an API error read as one voice. Title
		// falls back to the raw error text, which is what keeps a plain
		// error's rendering exactly as it always was.
		style := ui.NewStyle(env)
		title := se.Title()
		if title == "" {
			title = err.Error()
		}
		fmt.Fprintf(env.ErrWriter(), "%s %s\n", style.ErrorLabel(), title)

		if reason := se.Reason(); reason != "" {
			fmt.Fprintf(env.ErrWriter(), "%s\n", style.Dim(reason))
		}
		// The label is dimmed to recede behind the value that names the actual
		// status or request. The value is painted in ColorInfo rather than
		// left in the default foreground, which read too close to the bold
		// default-colored command path in the hint below it.
		if status := se.Status(); status != 0 {
			fmt.Fprintf(env.ErrWriter(), "%s %s\n", style.Dim("status"), paintValue(env, fmt.Sprintf("%d", status)))
		}
		if reqID := se.RequestID(); reqID != "" {
			fmt.Fprintf(env.ErrWriter(), "%s %s\n", style.Dim("request"), paintValue(env, reqID))
		}
		if step := se.NextStep(); step != "" {
			fmt.Fprintf(env.ErrWriter(), "%s\n", style.Hint(step))
		}
	case "json":
		// Always keep the stable {"errors":[{"detail":...}]} envelope so
		// automation parsing --output json is not broken by richer flag
		// validation. Plain Error() text (no ANSI) goes in detail; title,
		// reason, status, request_id and next_step are additive and omitted
		// when empty.
		payload := outputErrors{
			Errors: []outputError{
				{
					Detail:    err.Error(),
					Title:     se.Title(),
					Reason:    se.Reason(),
					Status:    se.Status(),
					RequestID: se.RequestID(),
					NextStep:  se.NextStep(),
				},
			},
		}

		b, _ := json.Marshal(payload)
		fmt.Println(string(b))
	}

	errAction()
}

// errorValueColor is a light neutral grey for a status/request value: lighter
// than the muted label introducing it, but not a new hue like ui.ColorInfo,
// so it stays quieter than an identifier while still reading apart from the
// bold default-colored command path in the hint line beneath it. Kept local
// to this file rather than added to the shared palette in internal/ui.
const errorValueColor lipgloss.Color = "252"

// paintValue colors a status/request value in errorValueColor.
func paintValue(env ui.Env, s string) string {
	return env.SprintErr(env.NewErrStyle().Foreground(errorValueColor), s)
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
