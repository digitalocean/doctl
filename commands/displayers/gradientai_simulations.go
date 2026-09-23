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
	"strings"

	"github.com/digitalocean/doctl/do"
)

// ScenarioSet displays Gradient AI scenario sets.
type ScenarioSet struct {
	ScenarioSets do.ScenarioSets
}

var _ Displayable = &ScenarioSet{}

func (v *ScenarioSet) JSON(out io.Writer) error {
	return writeJSON(v.ScenarioSets, out)
}

func (v *ScenarioSet) Cols() []string {
	return []string{
		"UUID",
		"Name",
		"Status",
		"SourceKind",
		"ScenarioCount",
		"CreatedAt",
		"UpdatedAt",
	}
}

func (v *ScenarioSet) ColMap() map[string]string {
	return map[string]string{
		"UUID":          "UUID",
		"Name":          "Name",
		"Status":        "Status",
		"SourceKind":    "Source Kind",
		"ScenarioCount": "Scenario Count",
		"CreatedAt":     "Created At",
		"UpdatedAt":     "Updated At",
	}
}

func (v *ScenarioSet) KV() []map[string]any {
	if v == nil || v.ScenarioSets == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.ScenarioSets))
	for _, set := range v.ScenarioSets {
		out = append(out, map[string]any{
			"UUID":          set.ScenarioSetUUID,
			"Name":          set.Name,
			"Status":        set.Status,
			"SourceKind":    set.SourceKind,
			"ScenarioCount": set.ScenarioCount,
			"CreatedAt":     set.CreatedAt,
			"UpdatedAt":     set.UpdatedAt,
		})
	}
	return out
}

// Scenario displays scenarios within a scenario set or library entry.
type Scenario struct {
	Scenarios do.Scenarios
}

var _ Displayable = &Scenario{}

func (v *Scenario) JSON(out io.Writer) error {
	return writeJSON(v.Scenarios, out)
}

func (v *Scenario) Cols() []string {
	return []string{
		"UUID",
		"Name",
		"Description",
		"UserPersona",
		"MaxTurns",
		"ExplorationBudget",
	}
}

func (v *Scenario) ColMap() map[string]string {
	return map[string]string{
		"UUID":              "UUID",
		"Name":              "Name",
		"Description":       "Description",
		"UserPersona":       "User Persona",
		"MaxTurns":          "Max Turns",
		"ExplorationBudget": "Exploration Budget",
	}
}

func (v *Scenario) KV() []map[string]any {
	if v == nil || v.Scenarios == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.Scenarios))
	for _, scenario := range v.Scenarios {
		out = append(out, map[string]any{
			"UUID":              scenario.ScenarioUUID,
			"Name":              scenario.Name,
			"Description":       scenario.Description,
			"UserPersona":       scenario.UserPersona,
			"MaxTurns":          scenario.MaxTurns,
			"ExplorationBudget": scenario.ExplorationBudget,
		})
	}
	return out
}

// ScenarioLibraryEntry displays entries from the platform scenario library.
type ScenarioLibraryEntry struct {
	Entries do.ScenarioLibraryEntries
}

var _ Displayable = &ScenarioLibraryEntry{}

func (v *ScenarioLibraryEntry) JSON(out io.Writer) error {
	return writeJSON(v.Entries, out)
}

func (v *ScenarioLibraryEntry) Cols() []string {
	return []string{
		"UUID",
		"Name",
		"Category",
		"Status",
		"ScenarioCount",
		"CreatedAt",
	}
}

func (v *ScenarioLibraryEntry) ColMap() map[string]string {
	return map[string]string{
		"UUID":          "UUID",
		"Name":          "Name",
		"Category":      "Category",
		"Status":        "Status",
		"ScenarioCount": "Scenario Count",
		"CreatedAt":     "Created At",
	}
}

func (v *ScenarioLibraryEntry) KV() []map[string]any {
	if v == nil || v.Entries == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.Entries))
	for _, entry := range v.Entries {
		out = append(out, map[string]any{
			"UUID":          entry.LibraryScenarioUUID,
			"Name":          entry.Name,
			"Category":      entry.Category,
			"Status":        entry.Status,
			"ScenarioCount": entry.ScenarioCount,
			"CreatedAt":     entry.CreatedAt,
		})
	}
	return out
}

// GenAIDownloadURL displays a presigned download URL for a Gradient AI file,
// such as a scenario set file or a journey trajectory.
type GenAIDownloadURL struct {
	DownloadURL *do.GenAIDownloadURL
}

var _ Displayable = &GenAIDownloadURL{}

func (v *GenAIDownloadURL) JSON(out io.Writer) error {
	return writeJSON(v.DownloadURL, out)
}

func (v *GenAIDownloadURL) Cols() []string {
	return []string{
		"DownloadURL",
		"ExpiresAt",
	}
}

func (v *GenAIDownloadURL) ColMap() map[string]string {
	return map[string]string{
		"DownloadURL": "Download URL",
		"ExpiresAt":   "Expires At",
	}
}

func (v *GenAIDownloadURL) KV() []map[string]any {
	if v == nil || v.DownloadURL == nil {
		return []map[string]any{}
	}
	return []map[string]any{{
		"DownloadURL": v.DownloadURL.DownloadURL,
		"ExpiresAt":   v.DownloadURL.ExpiresAt,
	}}
}

// SimulationRun displays Gradient AI simulation runs.
type SimulationRun struct {
	SimulationRuns do.SimulationRuns
}

var _ Displayable = &SimulationRun{}

func (v *SimulationRun) JSON(out io.Writer) error {
	return writeJSON(v.SimulationRuns, out)
}

func (v *SimulationRun) Cols() []string {
	return []string{
		"UUID",
		"Name",
		"Status",
		"ScenarioSetUUID",
		"AgentUUID",
		"TotalJourneys",
		"JourneysFinished",
		"CreatedAt",
	}
}

func (v *SimulationRun) ColMap() map[string]string {
	return map[string]string{
		"UUID":             "UUID",
		"Name":             "Name",
		"Status":           "Status",
		"ScenarioSetUUID":  "Scenario Set UUID",
		"AgentUUID":        "Agent UUID",
		"TotalJourneys":    "Total Journeys",
		"JourneysFinished": "Journeys Finished",
		"CreatedAt":        "Created At",
	}
}

func (v *SimulationRun) KV() []map[string]any {
	if v == nil || v.SimulationRuns == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.SimulationRuns))
	for _, run := range v.SimulationRuns {
		agentUUID := ""
		if run.AgentConfig != nil {
			agentUUID = run.AgentConfig.AgentUUID
		}
		out = append(out, map[string]any{
			"UUID":             run.RunUUID,
			"Name":             run.Name,
			"Status":           run.Status,
			"ScenarioSetUUID":  run.ScenarioSetUUID,
			"AgentUUID":        agentUUID,
			"TotalJourneys":    run.TotalJourneys,
			"JourneysFinished": run.JourneysFinished,
			"CreatedAt":        run.CreatedAt,
		})
	}
	return out
}

// SimulationRunDetail displays a simulation run alongside its verdict rollups.
// The per-scenario results are only included in the JSON output.
type SimulationRunDetail struct {
	Detail *do.SimulationRunDetail
}

var _ Displayable = &SimulationRunDetail{}

func (v *SimulationRunDetail) JSON(out io.Writer) error {
	return writeJSON(v.Detail, out)
}

// verdictCols are the columns, in display order, that a single run adds on top
// of the list columns.
var verdictCols = []struct {
	col   string
	label string
}{
	{"Success", "Success"},
	{"Failure", "Failure"},
	{"Inconclusive", "Inconclusive"},
}

func (v *SimulationRunDetail) Cols() []string {
	cols := (&SimulationRun{}).Cols()
	for _, verdict := range verdictCols {
		cols = append(cols, verdict.col)
	}
	return cols
}

func (v *SimulationRunDetail) ColMap() map[string]string {
	colMap := (&SimulationRun{}).ColMap()
	for _, verdict := range verdictCols {
		colMap[verdict.col] = verdict.label
	}
	return colMap
}

func (v *SimulationRunDetail) KV() []map[string]any {
	if v == nil || v.Detail == nil || v.Detail.SimulationRunGetResponse == nil || v.Detail.SimulationRun == nil {
		return []map[string]any{}
	}
	run := v.Detail.SimulationRun

	kv := (&SimulationRun{SimulationRuns: do.SimulationRuns{{SimulationRun: run}}}).KV()[0]
	kv["Success"], kv["Failure"], kv["Inconclusive"] = uint32(0), uint32(0), uint32(0)
	if run.ResultSummary != nil && run.ResultSummary.VerdictCounts != nil {
		kv["Success"] = run.ResultSummary.VerdictCounts.SuccessCount
		kv["Failure"] = run.ResultSummary.VerdictCounts.FailureCount
		kv["Inconclusive"] = run.ResultSummary.VerdictCounts.InconclusiveCount
	}

	return []map[string]any{kv}
}

// SimulationJourney displays journeys within a simulation run.
type SimulationJourney struct {
	Journeys do.SimulationJourneys
}

var _ Displayable = &SimulationJourney{}

func (v *SimulationJourney) JSON(out io.Writer) error {
	return writeJSON(v.Journeys, out)
}

func (v *SimulationJourney) Cols() []string {
	return []string{
		"UUID",
		"ScenarioUUID",
		"JourneyIndex",
		"Status",
		"Verdict",
		"DurationSec",
		"CreatedAt",
	}
}

func (v *SimulationJourney) ColMap() map[string]string {
	return map[string]string{
		"UUID":         "UUID",
		"ScenarioUUID": "Scenario UUID",
		"JourneyIndex": "Journey Index",
		"Status":       "Status",
		"Verdict":      "Verdict",
		"DurationSec":  "Duration (s)",
		"CreatedAt":    "Created At",
	}
}

func (v *SimulationJourney) KV() []map[string]any {
	if v == nil || v.Journeys == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.Journeys))
	for _, journey := range v.Journeys {
		out = append(out, map[string]any{
			"UUID":         journey.JourneyUUID,
			"ScenarioUUID": journey.ScenarioUUID,
			"JourneyIndex": journey.JourneyIndex,
			"Status":       journey.Status,
			"Verdict":      journey.Verdict,
			"DurationSec":  journey.DurationSec,
			"CreatedAt":    journey.CreatedAt,
		})
	}
	return out
}

// SimulationTrajectory displays the trajectory of a single journey. The full
// message history is only included in the JSON output.
type SimulationTrajectory struct {
	Trajectory *do.SimulationTrajectory
}

var _ Displayable = &SimulationTrajectory{}

func (v *SimulationTrajectory) JSON(out io.Writer) error {
	return writeJSON(v.Trajectory, out)
}

func (v *SimulationTrajectory) Cols() []string {
	return []string{
		"JourneyUUID",
		"Status",
		"Verdict",
		"TurnCount",
		"DurationSec",
		"TotalTokens",
		"JudgeReasoning",
	}
}

func (v *SimulationTrajectory) ColMap() map[string]string {
	return map[string]string{
		"JourneyUUID":    "Journey UUID",
		"Status":         "Status",
		"Verdict":        "Verdict",
		"TurnCount":      "Turn Count",
		"DurationSec":    "Duration (s)",
		"TotalTokens":    "Total Tokens",
		"JudgeReasoning": "Judge Reasoning",
	}
}

func (v *SimulationTrajectory) KV() []map[string]any {
	if v == nil || v.Trajectory == nil || v.Trajectory.SimulationTrajectory == nil {
		return []map[string]any{}
	}
	trajectory := v.Trajectory

	totalTokens := ""
	if trajectory.TokenUsage != nil {
		totalTokens = trajectory.TokenUsage.TotalTokens
	}

	reasoning := ""
	if trajectory.Judge != nil {
		reasoning = summarize(trajectory.Judge.Reasoning)
	}

	return []map[string]any{{
		"JourneyUUID":    trajectory.JourneyUUID,
		"Status":         trajectory.Status,
		"Verdict":        trajectory.Verdict,
		"TurnCount":      trajectory.TurnCount,
		"DurationSec":    trajectory.DurationSec,
		"TotalTokens":    totalTokens,
		"JudgeReasoning": reasoning,
	}}
}

// summarize collapses multi-line text into a single truncated line so it stays
// readable in a table. The untruncated value is available in JSON output.
func summarize(text string) string {
	const maxLen = 80

	flat := strings.Join(strings.Fields(text), " ")
	if len(flat) <= maxLen {
		return flat
	}
	return flat[:maxLen-3] + "..."
}
