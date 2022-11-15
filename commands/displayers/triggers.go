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

	"github.com/digitalocean/doctl/do"
)

// Triggers is the type of the displayer for triggers list
type Triggers struct {
	List []do.ServerlessTrigger
}

var _ Displayable = &Triggers{}

// JSON is the displayer JSON method specialized for triggers list
func (i *Triggers) JSON(out io.Writer) error {
	return writeJSON(i.List, out)
}

// Cols is the displayer Cols method specialized for triggers list
func (i *Triggers) Cols() []string {
	return []string{"Name", "Cron", "Function", "Enabled", "LastRun"}
}

// ColMap is the displayer ColMap method specialized for triggers list
func (i *Triggers) ColMap() map[string]string {
	return map[string]string{
		"Name":     "Name",
		"Cron":     "Cron Expression",
		"Function": "Invokes",
		"Enabled":  "Enabled",
		"LastRun":  "Last Run At",
	}
}

// KV is the displayer KV method specialized for triggers list
func (i *Triggers) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(i.List))
	for _, ii := range i.List {
		lastRun := "_"
		if ii.ScheduledRuns != nil && ii.ScheduledRuns.LastRunAt != nil && !ii.ScheduledRuns.LastRunAt.IsZero() {
			lastRun = ii.ScheduledRuns.LastRunAt.String()
		}

		x := map[string]interface{}{
			"Name":     ii.Name,
			"Cron":     ii.ScheduledDetails.Cron,
			"Function": ii.Function,
			"Enabled":  ii.IsEnabled,
			"LastRun":  lastRun,
		}
		out = append(out, x)
	}

	return out
}
