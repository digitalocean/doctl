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
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/digitalocean/doctl/do"
)

func harborTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// harborJSON writes one object when single is set and there is exactly one
// item, so commands about one resource print an object rather than an array.
func harborJSON[T any](items []T, single bool, out io.Writer) error {
	if single && len(items) == 1 {
		return writeJSON(items[0], out)
	}
	if items == nil {
		items = []T{}
	}
	return writeJSON(items, out)
}

func harborSize(n int64) string {
	if n <= 0 {
		return ""
	}
	return BytesToHumanReadableUnitBinary(uint64(n))
}

func harborLintSummary(r *do.HarborLintReport) string {
	if r == nil {
		return ""
	}
	counts := map[string]int{}
	for _, f := range r.Findings {
		sev := strings.ToLower(f.Severity)
		if sev == "" {
			sev = "finding"
		}
		counts[sev]++
	}
	var parts []string
	if r.Passed != nil {
		if *r.Passed {
			parts = append(parts, "passed")
		} else {
			parts = append(parts, "failed")
		}
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%d %s", counts[k], k))
	}
	return strings.Join(parts, ", ")
}

// HarborLintFindings displays lint findings.
type HarborLintFindings struct {
	Findings []do.HarborLintFinding
}

var _ Displayable = &HarborLintFindings{}

func (v *HarborLintFindings) JSON(out io.Writer) error {
	return harborJSON(v.Findings, false, out)
}

func (v *HarborLintFindings) Cols() []string {
	return []string{"Severity", "Rule", "Path", "Message"}
}

func (v *HarborLintFindings) ColMap() map[string]string {
	return map[string]string{"Severity": "Severity", "Rule": "Rule", "Path": "Path", "Message": "Message"}
}

func (v *HarborLintFindings) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.Findings))
	for _, f := range v.Findings {
		file := f.Path
		if file == "" {
			file = f.File
		}
		if f.Task != "" {
			file = f.Task + ": " + file
		}
		out = append(out, map[string]any{"Severity": f.Severity, "Rule": f.Rule, "Path": file, "Message": f.Message})
	}
	return out
}

// HarborPull displays a downloaded dataset.
type HarborPull struct {
	Result *do.HarborPullResult
}

var _ Displayable = &HarborPull{}

func (v *HarborPull) JSON(out io.Writer) error { return writeJSON(v.Result, out) }

func (v *HarborPull) Cols() []string { return []string{"Name", "Digest", "Files", "Path"} }

func (v *HarborPull) ColMap() map[string]string {
	return map[string]string{"Name": "Name", "Digest": "Digest", "Files": "Files", "Path": "Path"}
}

func (v *HarborPull) KV() []map[string]any {
	if v.Result == nil {
		return nil
	}
	r := v.Result
	return []map[string]any{{"Name": r.Name, "Digest": r.Digest, "Files": r.Files, "Path": r.Path}}
}

// HarborDatasets displays datasets.
type HarborDatasets struct {
	Datasets []do.HarborDataset
	Single   bool
}

var _ Displayable = &HarborDatasets{}

func (v *HarborDatasets) JSON(out io.Writer) error {
	return harborJSON(v.Datasets, v.Single, out)
}

func (v *HarborDatasets) Cols() []string {
	if v.Single {
		return []string{"Name", "Status", "Digest", "Tasks", "Updated"}
	}
	return []string{"Name", "Status", "Digest", "TaskCount", "Updated"}
}

func (v *HarborDatasets) ColMap() map[string]string {
	return map[string]string{
		"Name":        "Name",
		"Status":      "Status",
		"Digest":      "Digest",
		"Description": "Description",
		"Tasks":       "Tasks",
		"TaskCount":   "Tasks",
		"Updated":     "Updated",
	}
}

func (v *HarborDatasets) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.Datasets))
	for _, d := range v.Datasets {
		names := make([]string, len(d.Tasks))
		for i, t := range d.Tasks {
			names[i] = t.Name
		}
		updated := d.UpdatedAt
		if updated == nil {
			updated = d.PublishedAt
		}
		out = append(out, map[string]any{
			"Name":        d.Name,
			"Status":      d.Status,
			"Digest":      d.Digest,
			"Description": d.Description,
			"Tasks":       strings.Join(names, ","),
			"TaskCount":   len(d.Tasks),
			"Updated":     harborTime(updated),
		})
	}
	return out
}

// HarborJobs displays jobs.
type HarborJobs struct {
	Jobs   []do.HarborJob
	Single bool
}

var _ Displayable = &HarborJobs{}

func (v *HarborJobs) JSON(out io.Writer) error {
	return harborJSON(v.Jobs, v.Single, out)
}

func (v *HarborJobs) Cols() []string {
	if v.Single {
		return []string{"UUID", "Name", "Status", "Progress", "Errored", "Agents", "CreatedAt", "CompletedAt"}
	}
	return []string{"UUID", "Name", "Status", "Progress", "Errored", "CreatedAt"}
}

func (v *HarborJobs) ColMap() map[string]string {
	return map[string]string{
		"UUID":        "UUID",
		"Name":        "Name",
		"Status":      "Status",
		"Progress":    "Progress",
		"Errored":     "Errored",
		"Agents":      "Agents",
		"CreatedAt":   "Created At",
		"CompletedAt": "Completed At",
	}
}

func (v *HarborJobs) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.Jobs))
	for _, j := range v.Jobs {
		agents := make([]string, len(j.Agents))
		for i, a := range j.Agents {
			agents[i] = a.Name + "/" + a.ModelName
		}
		out = append(out, map[string]any{
			"UUID":        j.JobUUID,
			"Name":        j.Name,
			"Status":      j.Status,
			"Progress":    fmt.Sprintf("%d/%d", j.NCompleted, j.NTrials),
			"Errored":     j.NErrored,
			"Agents":      strings.Join(agents, ","),
			"CreatedAt":   harborTime(j.CreatedAt),
			"CompletedAt": harborTime(j.CompletedAt),
		})
	}
	return out
}

// HarborDownload displays a downloaded job.
type HarborDownload struct {
	Result *do.HarborDownloadResult
}

var _ Displayable = &HarborDownload{}

func (v *HarborDownload) JSON(out io.Writer) error { return writeJSON(v.Result, out) }

func (v *HarborDownload) Cols() []string {
	return []string{"JobUUID", "Downloaded", "Skipped", "Size", "Path"}
}

func (v *HarborDownload) ColMap() map[string]string {
	return map[string]string{"JobUUID": "Job UUID", "Downloaded": "Downloaded", "Skipped": "Skipped", "Size": "Size", "Path": "Path"}
}

func (v *HarborDownload) KV() []map[string]any {
	if v.Result == nil {
		return nil
	}
	r := v.Result
	return []map[string]any{{"JobUUID": r.JobUUID, "Downloaded": r.Downloaded, "Skipped": r.Skipped, "Size": harborSize(r.Bytes), "Path": r.Path}}
}
