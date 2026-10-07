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

package displayers

import (
	"io"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl/do"
)

// SignalsSession displays Signals session records.
type SignalsSession struct {
	Sessions do.SignalsSessions
}

var _ Displayable = &SignalsSession{}

func (d *SignalsSession) JSON(out io.Writer) error {
	return writeJSON(d.Sessions, out)
}

func (d *SignalsSession) Cols() []string {
	return []string{
		"SessionID",
		"TotalTurns",
		"StartedAt",
		"DurationSeconds",
		"SignalCount",
	}
}

func (d *SignalsSession) ColMap() map[string]string {
	return map[string]string{
		"SessionID":       "Session ID",
		"TotalTurns":      "Total Turns",
		"StartedAt":       "Started At",
		"DurationSeconds": "Duration (s)",
		"SignalCount":     "Signal Count",
	}
}

func (d *SignalsSession) KV() []map[string]any {
	out := make([]map[string]any, len(d.Sessions))
	for i, s := range d.Sessions {
		out[i] = map[string]any{
			"SessionID":       s.SessionID,
			"TotalTurns":      s.TotalTurns,
			"StartedAt":       s.StartedAt,
			"DurationSeconds": s.DurationSeconds,
			"SignalCount":     s.SignalCount,
		}
	}
	return out
}

// SignalsSessionDialogue displays Signals session dialogue records.
type SignalsSessionDialogue struct {
	Dialogues do.SignalsSessionDialogues
}

var _ Displayable = &SignalsSessionDialogue{}

func (d *SignalsSessionDialogue) JSON(out io.Writer) error {
	return writeJSON(d.Dialogues, out)
}

func (d *SignalsSessionDialogue) Cols() []string {
	return []string{
		"ID",
		"SegmentID",
		"Sequence",
		"RunStatus",
		"UserMessage",
		"Signals",
	}
}

func (d *SignalsSessionDialogue) ColMap() map[string]string {
	return map[string]string{
		"ID":          "ID",
		"SegmentID":   "Segment ID",
		"Sequence":    "Sequence",
		"RunStatus":   "Run Status",
		"UserMessage": "User Message",
		"Signals":     "Signals",
	}
}

func (d *SignalsSessionDialogue) KV() []map[string]any {
	out := make([]map[string]any, len(d.Dialogues))
	for i, dl := range d.Dialogues {
		signalTypes := make([]string, len(dl.Signals))
		for j, sig := range dl.Signals {
			signalTypes[j] = sig.SignalType
		}
		msg := dl.UserMessage
		if len(msg) > 80 {
			msg = msg[:77] + "..."
		}
		out[i] = map[string]any{
			"ID":          strconv.FormatInt(dl.ID, 10),
			"SegmentID":   dl.SegmentID,
			"Sequence":    dl.Sequence,
			"RunStatus":   dl.RunStatus,
			"UserMessage": msg,
			"Signals":     strings.Join(signalTypes, ", "),
		}
	}
	return out
}
