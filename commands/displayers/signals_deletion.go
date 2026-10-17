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

	"github.com/digitalocean/doctl/do"
)

// SignalsDeletion displays Signals deletion jobs.
type SignalsDeletion struct {
	Deletions do.SignalsDeletions
}

var _ Displayable = &SignalsDeletion{}

func (d *SignalsDeletion) JSON(out io.Writer) error {
	return writeJSON(d.Deletions, out)
}

func (d *SignalsDeletion) Cols() []string {
	return []string{
		"DeletionID",
		"Type",
		"AgentID",
		"Status",
		"CreatedAt",
		"StartedAt",
		"CompletedAt",
	}
}

func (d *SignalsDeletion) ColMap() map[string]string {
	return map[string]string{
		"DeletionID":   "Deletion ID",
		"TeamID":       "Team ID",
		"Type":         "Type",
		"AgentID":      "Agent ID",
		"Status":       "Status",
		"ErrorMessage": "Error Message",
		"CreatedAt":    "Created At",
		"StartedAt":    "Started At",
		"CompletedAt":  "Completed At",
	}
}

func (d *SignalsDeletion) KV() []map[string]any {
	out := make([]map[string]any, len(d.Deletions))
	for i, j := range d.Deletions {
		startedAt := ""
		if j.StartedAt != nil {
			startedAt = strconv.FormatInt(*j.StartedAt, 10)
		}
		completedAt := ""
		if j.CompletedAt != nil {
			completedAt = strconv.FormatInt(*j.CompletedAt, 10)
		}
		errMsg := ""
		if j.ErrorMessage != nil {
			errMsg = *j.ErrorMessage
		}
		out[i] = map[string]any{
			"DeletionID":   j.DeletionID,
			"TeamID":       j.TeamID,
			"Type":         j.Type,
			"AgentID":      j.AgentID,
			"Status":       j.Status,
			"ErrorMessage": errMsg,
			"CreatedAt":    j.CreatedAt,
			"StartedAt":    startedAt,
			"CompletedAt":  completedAt,
		}
	}
	return out
}
