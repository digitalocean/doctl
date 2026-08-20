/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    15|See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

// godoErrorLine matches godo.ErrorResponse.Error() output:
//
//	METHOD URL: 400 message
//	METHOD URL: 400 (request "id") message
var godoErrorLine = regexp.MustCompile(`(?i)^(?:GET|POST|PUT|PATCH|DELETE|HEAD)\s+\S+:\s+\d{3}(?:\s+\(request "[^"]*"\))?\s+(.+)$`)

// agentPrettyError is a user-facing agent CLI error: human title + reason,
// without METHOD/URL/status noise from raw godo ErrorResponse strings.
type agentPrettyError struct {
	title  string
	reason string
	status int
	tips   []string
	cause  error
}

func (e *agentPrettyError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.reason) != "" {
		return e.reason
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.title
}

func (e *agentPrettyError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// DisplayError returns the styled multi-line form printed by checkErr for
// agent commands (instead of "Error: METHOD URL: 400 …").
func (e *agentPrettyError) DisplayError() string {
	if e == nil {
		return ""
	}
	prev := stylingEnabled
	stylingEnabled = agentErrorStyling()
	defer func() { stylingEnabled = prev }()

	var b strings.Builder
	title := strings.TrimSpace(e.title)
	if title == "" {
		title = "Something went wrong"
	}
	fmt.Fprintf(&b, "%s %s\n", colorize("✗", colError), boldColor(title, colError))

	reason := strings.TrimSpace(e.reason)
	if reason == "" && e.cause != nil {
		reason = strings.TrimSpace(e.cause.Error())
	}
	if reason != "" && !strings.EqualFold(reason, title) {
		b.WriteString(cardRow("Reason", reason))
	}
	if e.status > 0 {
		b.WriteString(cardRow("Status", colorize(httpStatusLabel(e.status), colMuted)))
	}
	if len(e.tips) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, colorize("Next step", colMuted))
		for _, tip := range e.tips {
			tip = strings.TrimSpace(tip)
			if tip == "" {
				continue
			}
			b.WriteString(cardRow("•", tip))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func agentErrorStyling() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stderr.Fd()))
}

func httpStatusLabel(code int) string {
	text := http.StatusText(code)
	if text == "" {
		return fmt.Sprintf("%d", code)
	}
	return fmt.Sprintf("%d %s", code, text)
}

// beautifyAgentError rewrites bare API/local errors into agentPrettyError so
// checkErr can render a card instead of a raw REST dump.
func beautifyAgentError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrExitSilently) {
		return err
	}
	var already *agentPrettyError
	if errors.As(err, &already) {
		return err
	}

	if msg, status, ok := agentAPIError(err); ok {
		title, tips := agentErrorTitleAndTips(msg, status)
		reason := strings.TrimSpace(msg)
		if reason == "" {
			reason = strings.TrimSpace(err.Error())
		}
		reason = stripGodoTransportNoise(reason)
		return &agentPrettyError{
			title:  title,
			reason: reason,
			status: status,
			tips:   tips,
			cause:  err,
		}
	}

	// Local validation / wrapped copy — still present as a clean card.
	reason := strings.TrimSpace(stripGodoTransportNoise(err.Error()))
	title := "Couldn't complete that request"
	tips := []string{}
	lower := strings.ToLower(reason)
	switch {
	case strings.Contains(lower, "limit of") && strings.Contains(lower, "active sessions"):
		title = "Session limit reached"
		tips = []string{"doctl open-harness-runtime list", "doctl open-harness-runtime remove <session>"}
	case strings.Contains(lower, "active sessions"):
		title = "Config still has active sessions"
		tips = []string{"doctl open-harness-runtime config list-sessions <config-id>", "doctl open-harness-runtime remove <session>"}
	case strings.Contains(lower, "mutually exclusive") || strings.Contains(lower, "is required") || strings.Contains(lower, "invalid --"):
		title = "Invalid arguments"
	case strings.Contains(lower, "not set locally") || strings.Contains(lower, "environment variable"):
		title = "Missing environment value"
		tips = []string{"Set the variable in your shell, or re-run in a terminal to be prompted"}
	case strings.Contains(lower, "openai"):
		title = "OpenAI Agents request failed"
		tips = []string{"Check $OPENAI_API_KEY", "Retry in a moment"}
	case strings.Contains(lower, "timed out"):
		title = "Timed out"
		tips = []string{"Retry with a longer --wait-timeout", "doctl open-harness-runtime show <session>"}
	case strings.Contains(lower, "no agent session goes by the name"):
		title = "Session not found"
		tips = []string{"doctl open-harness-runtime list"}
	case strings.Contains(lower, "many agent sessions go by the name"):
		title = "Ambiguous session name"
		tips = []string{"Pass the session ID instead", "doctl open-harness-runtime list"}
	}

	return &agentPrettyError{
		title:  title,
		reason: reason,
		tips:   tips,
		cause:  err,
	}
}

func agentErrorTitleAndTips(msg string, status int) (title string, tips []string) {
	lower := strings.ToLower(msg)
	switch status {
	case http.StatusUnauthorized:
		return "Authentication failed", []string{"doctl auth init"}
	case http.StatusForbidden:
		return "Access denied", []string{"Check your token scopes or team permissions"}
	case http.StatusNotFound:
		return "Not found", []string{"doctl open-harness-runtime list", "Confirm the ID or name and try again"}
	case http.StatusConflict:
		switch {
		case strings.Contains(lower, "limit of") && strings.Contains(lower, "active sessions"):
			return "Session limit reached", []string{"doctl open-harness-runtime list", "doctl open-harness-runtime remove <session>"}
		case strings.Contains(lower, "active sessions"):
			return "Config still has active sessions", []string{"doctl open-harness-runtime config list-sessions <config-id>", "doctl open-harness-runtime remove <session>"}
		case strings.Contains(lower, "run is terminal"):
			return "Session run has ended", []string{"doctl open-harness-runtime remove <session>", "doctl open-harness-runtime run --harness opencode --name new-session"}
		case strings.Contains(lower, "already attached") || strings.Contains(lower, "another device"):
			return "Session already attached elsewhere", []string{"Detach on the other device, then re-run doctl open-harness-runtime attach"}
		default:
			return "Conflict", nil
		}
	case http.StatusTooManyRequests:
		return "Rate limited", []string{"Wait a moment and retry"}
	case http.StatusBadRequest:
		return "Invalid request", nil
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return "Service temporarily unavailable", []string{"Retry in a moment"}
	default:
		if status >= 500 {
			return "Server error", []string{"Retry in a moment"}
		}
		if status >= 400 {
			return "Request failed", nil
		}
		return "Something went wrong", nil
	}
}

// stripGodoTransportNoise removes "METHOD URL: status" prefixes from error
// strings so users see the human message, not the REST wire format.
func stripGodoTransportNoise(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if m := godoErrorLine.FindStringSubmatch(s); len(m) == 2 {
		if msg := strings.TrimSpace(m[1]); msg != "" {
			return msg
		}
	}
	return s
}

// displayableError is implemented by agentPrettyError for checkErr.
type displayableError interface {
	DisplayError() string
}
