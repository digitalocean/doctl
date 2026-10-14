/*
Copyright 2018-2019 The Doctl Authors All rights reserved.
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

package doctl

import "fmt"

// MissingArgsErr is returned when there are too few arguments for a command.
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

// MissingAccessTokenErr is returned when doctl needs an API token to
// initialize a client and none was configured.
type MissingAccessTokenErr struct{}

var _ error = &MissingAccessTokenErr{}

// NewMissingAccessTokenErr creates a MissingAccessTokenErr instance.
func NewMissingAccessTokenErr() *MissingAccessTokenErr {
	return &MissingAccessTokenErr{}
}

func (e *MissingAccessTokenErr) Error() string {
	return "access token is required. (hint: run 'doctl auth init')"
}

// NextStep is empty on purpose: Error() already names the command to run,
// so commands.checkErr's generic `<command> --help` fallback would only
// contradict it. Structurally satisfies commands.NextStepper without an
// import (commands already depends on this package, not the reverse).
func (e *MissingAccessTokenErr) NextStep() string { return "" }

// TooManyArgsErr is returned when there are too many arguments for a command.
type TooManyArgsErr struct {
	Command string
}

var _ error = &TooManyArgsErr{}

// NewTooManyArgsErr creates a TooManyArgsErr instance.
func NewTooManyArgsErr(cmd string) *TooManyArgsErr {
	return &TooManyArgsErr{Command: cmd}
}

func (e *TooManyArgsErr) Error() string {
	return fmt.Sprintf("(%s) command contains unsupported arguments", e.Command)
}
