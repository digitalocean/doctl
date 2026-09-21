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

	// SuppressNextStepOnRetryExhaustion omits NextStep once godo's retry
	// client has already given up (the API error carries Attempts > 0).
	// Re-running the identical command immediately after exhausted retries
	// is unlikely to fare any better, so the canned "run it again" hint
	// would be actively misleading rather than just unhelpful.
	SuppressNextStepOnRetryExhaustion bool
}

// errorCodeTable covers the status codes doctl users hit most often. It is
// deliberately short: an entry earns its place by having something specific
// to say. An entry may set Reason without NextStep, which means there is no
// useful command to suggest for that status - genericStructuredError.NextStep
// takes that at face value. Codes absent from the table get no suggestion
// either, since an API failure is not something --help explains.
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
		// No canned next step: a bad ID and a wrong auth context both surface
		// as 404, and the ID typo is by far the more common of the two, so
		// there is no single suggestion that fits most of the time.
		Reason: "the requested resource does not exist, or not in this account/context",
	},
	http.StatusConflict: {
		// No canned next step: re-running the identical command right after
		// a conflict does not reliably help, since whatever state caused it
		// may well still be there.
		Reason: "the resource is in a state that conflicts with this request",
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
		Reason:                            "the API had an internal error",
		NextStep:                          "run %s",
		SuppressNextStepOnRetryExhaustion: true,
	},
	http.StatusBadGateway: {
		Reason:                            "the API gateway could not reach the upstream service",
		NextStep:                          "run %s",
		SuppressNextStepOnRetryExhaustion: true,
	},
	http.StatusServiceUnavailable: {
		Reason:                            "the API is temporarily unavailable",
		NextStep:                          "run %s",
		SuppressNextStepOnRetryExhaustion: true,
	},
	http.StatusGatewayTimeout: {
		Reason:                            "the API did not respond in time",
		NextStep:                          "run %s",
		SuppressNextStepOnRetryExhaustion: true,
	},
}

// apiError unwraps err to the godo API error it carries, if any.
func apiError(err error) (*godo.ErrorResponse, bool) {
	var gerr *godo.ErrorResponse
	if !errors.As(err, &gerr) {
		return nil, false
	}
	return gerr, true
}

// statusFor returns the HTTP status code carried by err, if it wraps a godo
// API error.
func statusFor(err error) int {
	gerr, ok := apiError(err)
	if !ok || gerr.Response == nil {
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
// attempts is the retry count godo attached to the error, if any; entries
// marked SuppressNextStepOnRetryExhaustion return "" once retries have
// already been exhausted rather than suggesting the same command again.
func resolvedNextStep(entry errorCodeEntry, attempts int) string {
	if entry.SuppressNextStepOnRetryExhaustion && attempts > 0 {
		return ""
	}
	if !strings.Contains(entry.NextStep, "%s") {
		return entry.NextStep
	}

	cmdPath := "doctl"
	if activeCommand != nil {
		cmdPath = activeCommand.CommandPath()
	}

	return fmt.Sprintf(entry.NextStep, cmdPath)
}
