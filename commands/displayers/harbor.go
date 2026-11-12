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

// HarborPublish displays the outcome of publishing task folders.
type HarborPublish struct {
	Results []*do.HarborPublishResult
}

var _ Displayable = &HarborPublish{}

func (v *HarborPublish) JSON(out io.Writer) error {
	return harborJSON(v.Results, true, out)
}

func (v *HarborPublish) Cols() []string {
	return []string{"Name", "State", "Digest", "Files", "Size", "Lint", "UploadUUID"}
}

func (v *HarborPublish) ColMap() map[string]string {
	return map[string]string{
		"Name":       "Name",
		"State":      "State",
		"Digest":     "Digest",
		"Files":      "Files",
		"Size":       "Size",
		"Lint":       "Lint",
		"UploadUUID": "Upload UUID",
	}
}

func (v *HarborPublish) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.Results))
	for _, r := range v.Results {
		out = append(out, map[string]any{
			"Name":       r.Name,
			"State":      r.State,
			"Digest":     r.Digest,
			"Files":      r.Files,
			"Size":       harborSize(r.ArchiveSizeBytes),
			"Lint":       harborLintSummary(r.LintReport),
			"UploadUUID": r.UploadUUID,
		})
	}
	return out
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
		out = append(out, map[string]any{"Severity": f.Severity, "Rule": f.Rule, "Path": f.Path, "Message": f.Message})
	}
	return out
}

// HarborTasks displays published tasks.
type HarborTasks struct {
	Tasks  []do.HarborTask
	Single bool
}

var _ Displayable = &HarborTasks{}

func (v *HarborTasks) JSON(out io.Writer) error {
	return harborJSON(v.Tasks, v.Single, out)
}

func (v *HarborTasks) Cols() []string {
	if v.Single {
		return []string{"Name", "Digest", "Size", "Image", "Lint", "PublishedBy", "PublishedAt"}
	}
	return []string{"Name", "Digest", "Size", "PublishedAt"}
}

func (v *HarborTasks) ColMap() map[string]string {
	return map[string]string{
		"Name":        "Name",
		"Digest":      "Digest",
		"Size":        "Size",
		"Image":       "Image",
		"Lint":        "Lint",
		"PublishedBy": "Published By",
		"PublishedAt": "Published At",
	}
}

func (v *HarborTasks) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.Tasks))
	for _, t := range v.Tasks {
		image := t.ImageSource
		if t.ImageRef != "" {
			image = t.ImageRef
		}
		out = append(out, map[string]any{
			"Name":        t.Name,
			"Digest":      t.Digest,
			"Size":        harborSize(t.ArchiveSizeBytes),
			"Image":       image,
			"Lint":        harborLintSummary(t.LintReport),
			"PublishedBy": t.PublishedByUserEmail,
			"PublishedAt": harborTime(t.PublishedAt),
		})
	}
	return out
}

// HarborPull displays a downloaded task.
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
		return []string{"Name", "Description", "Tasks", "PublishedAt"}
	}
	return []string{"Name", "Description", "TaskCount", "PublishedAt"}
}

func (v *HarborDatasets) ColMap() map[string]string {
	return map[string]string{
		"Name":        "Name",
		"Description": "Description",
		"Tasks":       "Tasks",
		"TaskCount":   "Tasks",
		"PublishedAt": "Published At",
	}
}

func (v *HarborDatasets) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.Datasets))
	for _, d := range v.Datasets {
		names := make([]string, len(d.Tasks))
		for i, t := range d.Tasks {
			names[i] = t.Name
		}
		out = append(out, map[string]any{
			"Name":        d.Name,
			"Description": d.Description,
			"Tasks":       strings.Join(names, ","),
			"TaskCount":   len(d.Tasks),
			"PublishedAt": harborTime(d.PublishedAt),
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
			agents[i] = a.Name + "/" + a.Model
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

// HarborTrials displays the trials of a job.
type HarborTrials struct {
	Trials []do.HarborTrial
}

var _ Displayable = &HarborTrials{}

func (v *HarborTrials) JSON(out io.Writer) error {
	return harborJSON(v.Trials, false, out)
}

func (v *HarborTrials) Cols() []string {
	return []string{"Trial", "Task", "Agent", "Model", "Attempt", "Status", "Reward", "Tokens", "Error"}
}

func (v *HarborTrials) ColMap() map[string]string {
	return map[string]string{
		"Trial":   "Trial",
		"Task":    "Task",
		"Agent":   "Agent",
		"Model":   "Model",
		"Attempt": "Attempt",
		"Status":  "Status",
		"Reward":  "Reward",
		"Tokens":  "Tokens (in/out)",
		"Error":   "Error",
	}
}

func (v *HarborTrials) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.Trials))
	for _, t := range v.Trials {
		tokens := ""
		if t.NInputTokens != nil || t.NOutputTokens != nil {
			tokens = fmt.Sprintf("%d/%d", derefInt64(t.NInputTokens), derefInt64(t.NOutputTokens))
		}
		errText := t.ExceptionType
		if t.ExceptionMessage != "" {
			errText = strings.TrimSpace(errText + ": " + t.ExceptionMessage)
			errText = strings.TrimPrefix(errText, ": ")
		}
		out = append(out, map[string]any{
			"Trial":   t.TrialName,
			"Task":    t.TaskName,
			"Agent":   t.AgentName,
			"Model":   t.ModelSlug,
			"Attempt": t.Attempt,
			"Status":  t.Status,
			"Reward":  harborReward(t.Rewards),
			"Tokens":  tokens,
			"Error":   errText,
		})
	}
	return out
}

// harborReward shows the conventional "reward" key on its own, and any other
// reward keys as key=value.
func harborReward(rewards map[string]float64) string {
	if len(rewards) == 0 {
		return ""
	}
	if r, ok := rewards["reward"]; ok && len(rewards) == 1 {
		return fmt.Sprintf("%g", r)
	}
	keys := make([]string, 0, len(rewards))
	for k := range rewards {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=%g", k, rewards[k])
	}
	return strings.Join(parts, ",")
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
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

// HarborAgents displays the supported agents.
type HarborAgents struct {
	Agents []do.HarborAgent
}

var _ Displayable = &HarborAgents{}

func (v *HarborAgents) JSON(out io.Writer) error {
	return harborJSON(v.Agents, false, out)
}

func (v *HarborAgents) Cols() []string {
	return []string{"Name", "Version", "Protocol", "Description"}
}

func (v *HarborAgents) ColMap() map[string]string {
	return map[string]string{"Name": "Name", "Version": "Version", "Protocol": "Protocol", "Description": "Description"}
}

func (v *HarborAgents) KV() []map[string]any {
	out := make([]map[string]any, 0, len(v.Agents))
	for _, a := range v.Agents {
		out = append(out, map[string]any{"Name": a.Name, "Version": a.Version, "Protocol": a.Protocol, "Description": a.Description})
	}
	return out
}
