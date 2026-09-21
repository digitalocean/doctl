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

package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Tone is the meaning a value carries, separate from how it is painted.
type Tone int

const (
	// ToneNone carries no meaning and is rendered unstyled.
	ToneNone Tone = iota
	// ToneSuccess is a resource that has reached its desired state.
	ToneSuccess
	// TonePending is a resource still moving towards it, or worth attention.
	TonePending
	// ToneError is a resource that failed or is unusable.
	ToneError
	// ToneMuted is a resource that is intentionally inert: off, deleted, unknown.
	ToneMuted
	// ToneIdentifier is the value that names the resource a row is about.
	// ToneFor never returns it: a name belongs to no vocabulary, so only a
	// caller that knows which column identifies the resource can ask for it.
	ToneIdentifier
)

// toneVocabulary maps state words to a meaning. doctl receives state in
// several encodings, so ToneFor normalizes before looking a value up here.
var toneVocabulary = map[string]Tone{
	// Reached the desired state.
	"active":     ToneSuccess,
	"available":  ToneSuccess,
	"complete":   ToneSuccess,
	"completed":  ToneSuccess,
	"finished":   ToneSuccess,
	"green":      ToneSuccess,
	"healthy":    ToneSuccess,
	"ok":         ToneSuccess,
	"online":     ToneSuccess,
	"ready":      ToneSuccess,
	"running":    ToneSuccess,
	"succeeded":  ToneSuccess,
	"success":    ToneSuccess,
	"successful": ToneSuccess,
	"verified":   ToneSuccess,

	// On the way there, or worth a look.
	"building":     TonePending,
	"configuring":  TonePending,
	"creating":     TonePending,
	"deploying":    TonePending,
	"evaluating":   TonePending,
	"forking":      TonePending,
	"generating":   TonePending,
	"in-progress":  TonePending,
	"inconclusive": TonePending,
	"migrating":    TonePending,
	"new":          TonePending,
	"paused":       TonePending,
	"pending":      TonePending,
	"preparing":    TonePending,
	"progress":     TonePending,
	"provisioning": TonePending,
	"queued":       TonePending,
	"requested":    TonePending,
	"resizing":     TonePending,
	"restoring":    TonePending,
	"upgrading":    TonePending,
	"waiting":      TonePending,
	"warning":      TonePending,
	"yellow":       TonePending,

	// Failed or unusable.
	"degraded":  ToneError,
	"error":     ToneError,
	"errored":   ToneError,
	"failed":    ToneError,
	"failing":   ToneError,
	"failure":   ToneError,
	"invalid":   ToneError,
	"red":       ToneError,
	"unhealthy": ToneError,

	// Intentionally inert.
	"archive":    ToneMuted,
	"archived":   ToneMuted,
	"canceled":   ToneMuted,
	"cancelled":  ToneMuted,
	"deleted":    ToneMuted,
	"disabled":   ToneMuted,
	"expired":    ToneMuted,
	"inactive":   ToneMuted,
	"off":        ToneMuted,
	"skipped":    ToneMuted,
	"superseded": ToneMuted,
	"unknown":    ToneMuted,
}

// ToneFor classifies a state value, reporting false when it is not recognized
// so that callers leave unfamiliar text alone. The whole value is tried first,
// then each of its words: "SCENARIO_SET_STATUS_READY" is a success on its last.
func ToneFor(value string) (Tone, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ToneNone, false
	}

	if tone, ok := toneVocabulary[normalized]; ok {
		return tone, true
	}

	for _, word := range strings.FieldsFunc(normalized, isToneWordSeparator) {
		if tone, ok := toneVocabulary[word]; ok {
			return tone, true
		}
	}

	return ToneNone, false
}

// isToneWordSeparator splits on underscores and whitespace but not hyphens,
// because "in-progress" is one word in the vocabulary.
func isToneWordSeparator(r rune) bool {
	return r == '_' || r == ' ' || r == '\t'
}

// toneColor maps a tone onto the palette, and is the only place the two meet.
func toneColor(tone Tone) (lipgloss.TerminalColor, bool) {
	switch tone {
	case ToneSuccess:
		return ColorSuccess, true
	case TonePending:
		return ColorWarning, true
	case ToneError:
		return ColorError, true
	case ToneMuted:
		return ColorMuted, true
	case ToneIdentifier:
		return ColorInfo, true
	default:
		return lipgloss.NoColor{}, false
	}
}

// toneBold reports whether a tone is drawn bold. Only the identifier is.
func toneBold(tone Tone) bool {
	return tone == ToneIdentifier
}

// SprintTone renders s in the color for tone when styling is permitted on Out,
// and returns s unchanged otherwise.
func (e Env) SprintTone(tone Tone, s string) string {
	color, ok := toneColor(tone)
	if !ok || !e.Style {
		return s
	}

	style := e.NewStyle().Foreground(color)
	if toneBold(tone) {
		style = style.Bold(true)
	}

	return e.Sprint(style, s)
}
