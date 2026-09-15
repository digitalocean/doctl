/*
Copyright 2026 The Doctl Authors All rights reserved.
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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/digitalocean/godo"
)

// errorCodeEntry is the canned guidance doctl gives for an API status code
// when the failing request itself did not supply anything more specific.
type errorCodeEntry struct {
	// Reason is a short, human gloss on what the status code means in a
	// doctl context - shown ahead of the raw API message, not instead of it.
	Reason string

	// NextStep is the suggested command, fed to checkErr the same way a
	// NextStepper's override is. Always of the form "run <command>" so
	// Style.Hint bolds the whole suggestion consistently with every other
	// hint in doctl. A "%s" placeholder is substituted with the path of the
	// command that failed (see resolvedNextStep).
	NextStep string
}

// errorCodeTable covers the status codes doctl users hit most often. It is
// deliberately short: an entry earns its place by being common enough that a
// canned next step beats sending everyone to --help. Codes not listed here
// fall back to defaultNextStep.
var errorCodeTable = map[int]errorCodeEntry{
	http.StatusBadRequest: {
		Reason:   "the request was malformed or failed validation",
		NextStep: "run %s --help",
	},
	http.StatusUnauthorized: {
		Reason:   "your API token is missing, invalid, or expired",
		NextStep: "run doctl auth init",
	},
	http.StatusForbidden: {
		Reason:   "your token does not have permission for this action",
		NextStep: "run doctl auth list",
	},
	http.StatusNotFound: {
		Reason:   "the requested resource does not exist, or not in this account/context",
		NextStep: "run doctl auth list",
	},
	http.StatusConflict: {
		Reason:   "the resource is in a state that conflicts with this request",
		NextStep: "run %s",
	},
	http.StatusUnprocessableEntity: {
		Reason:   "the request was well-formed but semantically invalid",
		NextStep: "run %s --help",
	},
	http.StatusTooManyRequests: {
		Reason:   "you have hit the API rate limit",
		NextStep: "run doctl account ratelimit",
	},
	http.StatusInternalServerError: {
		Reason:   "the API had an internal error",
		NextStep: "run %s",
	},
	http.StatusBadGateway: {
		Reason:   "the API gateway could not reach the upstream service",
		NextStep: "run %s",
	},
	http.StatusServiceUnavailable: {
		Reason:   "the API is temporarily unavailable",
		NextStep: "run %s",
	},
	http.StatusGatewayTimeout: {
		Reason:   "the API did not respond in time",
		NextStep: "run %s",
	},
}

// statusFor returns the HTTP status code carried by err, if it wraps a godo
// API error.
func statusFor(err error) int {
	var gerr *godo.ErrorResponse
	if !errors.As(err, &gerr) || gerr.Response == nil {
		return 0
	}
	return gerr.Response.StatusCode
}

// lookupErrorCode returns the canned entry for a godo API error, if any.
func lookupErrorCode(err error) (errorCodeEntry, bool) {
	status := statusFor(err)
	if status == 0 {
		return errorCodeEntry{}, false
	}

	entry, ok := errorCodeTable[status]
	return entry, ok
}

// resolvedNextStep fills entry.NextStep's "%s" placeholder, if it has one,
// with the failed command's path. Falls back to "doctl" when no command is
// active, which only happens for errors raised outside a command's Run.
func resolvedNextStep(entry errorCodeEntry) string {
	if !strings.Contains(entry.NextStep, "%s") {
		return entry.NextStep
	}

	cmdPath := "doctl"
	if activeCommand != nil {
		cmdPath = activeCommand.CommandPath()
	}

	return fmt.Sprintf(entry.NextStep, cmdPath)
}
