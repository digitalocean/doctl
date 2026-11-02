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

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// ModelEvaluationRun displays model evaluation run summaries.
type ModelEvaluationRun struct {
	ModelEvaluationRuns do.ModelEvaluationRuns
}

var _ Displayable = &ModelEvaluationRun{}

func (v *ModelEvaluationRun) JSON(out io.Writer) error {
	return writeJSON(v.ModelEvaluationRuns, out)
}

func (v *ModelEvaluationRun) Cols() []string {
	return []string{
		"UUID",
		"Name",
		"Status",
		"CandidateModelName",
		"DatasetName",
		"Progress",
		"CreatedAt",
	}
}

func (v *ModelEvaluationRun) ColMap() map[string]string {
	return map[string]string{
		"UUID":               "UUID",
		"Name":               "Name",
		"Status":             "Status",
		"CandidateModelName": "Candidate Model",
		"DatasetName":        "Dataset",
		"Progress":           "Progress",
		"CreatedAt":          "Created At",
	}
}

func (v *ModelEvaluationRun) KV() []map[string]any {
	if v == nil || v.ModelEvaluationRuns == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.ModelEvaluationRuns))
	for _, run := range v.ModelEvaluationRuns {
		out = append(out, map[string]any{
			"UUID":               run.EvalRunUuid,
			"Name":               run.Name,
			"Status":             run.Status,
			"CandidateModelName": run.CandidateModelName,
			"DatasetName":        run.DatasetName,
			"Progress":           formatEvaluationProgress(run.Progress),
			"CreatedAt":          run.CreatedAt,
		})
	}
	return out
}

// ModelEvaluationRunDetail displays a model evaluation run with status, overall
// score, and progress. Per-prompt results are only included in the JSON output.
type ModelEvaluationRunDetail struct {
	Detail *do.ModelEvaluationRunDetail
}

var _ Displayable = &ModelEvaluationRunDetail{}

func (v *ModelEvaluationRunDetail) JSON(out io.Writer) error {
	return writeJSON(v.Detail, out)
}

func (v *ModelEvaluationRunDetail) Cols() []string {
	return []string{
		"UUID",
		"Name",
		"Status",
		"OverallScore",
		"CandidateModelName",
		"DatasetName",
		"Progress",
		"CreatedAt",
		"CompletedAt",
	}
}

func (v *ModelEvaluationRunDetail) ColMap() map[string]string {
	return map[string]string{
		"UUID":               "UUID",
		"Name":               "Name",
		"Status":             "Status",
		"OverallScore":       "Overall Score",
		"CandidateModelName": "Candidate Model",
		"DatasetName":        "Dataset",
		"Progress":           "Progress",
		"CreatedAt":          "Created At",
		"CompletedAt":        "Completed At",
	}
}

func (v *ModelEvaluationRunDetail) KV() []map[string]any {
	if v == nil || v.Detail == nil || v.Detail.ModelEvaluationRunGetResponse == nil || v.Detail.Run == nil {
		return []map[string]any{}
	}
	run := v.Detail.Run

	overallScore := ""
	if run.ResultSummary != nil {
		overallScore = fmt.Sprintf("%.2f%%", run.ResultSummary.OverallScorePercent)
	}

	return []map[string]any{{
		"UUID":               run.EvalRunUuid,
		"Name":               run.Name,
		"Status":             run.Status,
		"OverallScore":       overallScore,
		"CandidateModelName": run.CandidateModelName,
		"DatasetName":        run.DatasetName,
		"Progress":           formatEvaluationProgress(run.Progress),
		"CreatedAt":          run.CreatedAt,
		"CompletedAt":        run.CompletedAt,
	}}
}

// ModelEvaluationRunCreate displays the UUID of a newly created evaluation run.
type ModelEvaluationRunCreate struct {
	Create *do.ModelEvaluationRunCreate
}

var _ Displayable = &ModelEvaluationRunCreate{}

func (v *ModelEvaluationRunCreate) JSON(out io.Writer) error {
	return writeJSON(v.Create, out)
}

func (v *ModelEvaluationRunCreate) Cols() []string {
	return []string{"UUID"}
}

func (v *ModelEvaluationRunCreate) ColMap() map[string]string {
	return map[string]string{"UUID": "UUID"}
}

func (v *ModelEvaluationRunCreate) KV() []map[string]any {
	if v == nil || v.Create == nil || v.Create.ModelEvaluationRunCreateResponse == nil {
		return []map[string]any{}
	}
	return []map[string]any{{"UUID": v.Create.EvalRunUuid}}
}

// EvaluationDataset displays evaluation datasets.
type EvaluationDataset struct {
	EvaluationDatasets do.EvaluationDatasets
}

var _ Displayable = &EvaluationDataset{}

func (v *EvaluationDataset) JSON(out io.Writer) error {
	return writeJSON(v.EvaluationDatasets, out)
}

func (v *EvaluationDataset) Cols() []string {
	return []string{
		"UUID",
		"Name",
		"Type",
		"Paradigm",
		"RowCount",
		"HasGroundTruth",
		"CreatedAt",
	}
}

func (v *EvaluationDataset) ColMap() map[string]string {
	return map[string]string{
		"UUID":           "UUID",
		"Name":           "Name",
		"Type":           "Type",
		"Paradigm":       "Paradigm",
		"RowCount":       "Row Count",
		"HasGroundTruth": "Has Ground Truth",
		"CreatedAt":      "Created At",
	}
}

func (v *EvaluationDataset) KV() []map[string]any {
	if v == nil || v.EvaluationDatasets == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.EvaluationDatasets))
	for _, dataset := range v.EvaluationDatasets {
		out = append(out, map[string]any{
			"UUID":           dataset.DatasetUUID,
			"Name":           dataset.DatasetName,
			"Type":           dataset.DatasetType,
			"Paradigm":       dataset.DatasetParadigm,
			"RowCount":       dataset.RowCount,
			"HasGroundTruth": dataset.HasGroundTruth,
			"CreatedAt":      dataset.CreatedAt,
		})
	}
	return out
}

// EvaluationDatasetCreate displays the UUID of a newly created evaluation dataset.
type EvaluationDatasetCreate struct {
	Create *do.EvaluationDatasetCreate
}

var _ Displayable = &EvaluationDatasetCreate{}

func (v *EvaluationDatasetCreate) JSON(out io.Writer) error {
	return writeJSON(v.Create, out)
}

func (v *EvaluationDatasetCreate) Cols() []string {
	return []string{"UUID"}
}

func (v *EvaluationDatasetCreate) ColMap() map[string]string {
	return map[string]string{"UUID": "UUID"}
}

func (v *EvaluationDatasetCreate) KV() []map[string]any {
	if v == nil || v.Create == nil || v.Create.CreateEvaluationDatasetResponse == nil {
		return []map[string]any{}
	}
	return []map[string]any{{"UUID": v.Create.EvaluationDatasetUUID}}
}

// ModelEvaluationPreset displays saved model evaluation presets.
type ModelEvaluationPreset struct {
	Presets do.ModelEvaluationPresets
}

var _ Displayable = &ModelEvaluationPreset{}

func (v *ModelEvaluationPreset) JSON(out io.Writer) error {
	return writeJSON(v.Presets, out)
}

func (v *ModelEvaluationPreset) Cols() []string {
	return []string{
		"UUID",
		"Name",
		"CandidateModelName",
		"DatasetName",
		"JudgeModelName",
		"CreatedAt",
	}
}

func (v *ModelEvaluationPreset) ColMap() map[string]string {
	return map[string]string{
		"UUID":               "UUID",
		"Name":               "Name",
		"CandidateModelName": "Candidate Model",
		"DatasetName":        "Dataset",
		"JudgeModelName":     "Judge Model",
		"CreatedAt":          "Created At",
	}
}

func (v *ModelEvaluationPreset) KV() []map[string]any {
	if v == nil || v.Presets == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.Presets))
	for _, preset := range v.Presets {
		out = append(out, map[string]any{
			"UUID":               preset.EvalPresetUuid,
			"Name":               preset.Name,
			"CandidateModelName": preset.CandidateModelName,
			"DatasetName":        preset.DatasetName,
			"JudgeModelName":     preset.JudgeModelName,
			"CreatedAt":          preset.CreatedAt,
		})
	}
	return out
}

// EvaluationMetric displays model evaluation metrics.
type EvaluationMetric struct {
	Metrics do.EvaluationMetrics
}

var _ Displayable = &EvaluationMetric{}

func (v *EvaluationMetric) JSON(out io.Writer) error {
	return writeJSON(v.Metrics, out)
}

func (v *EvaluationMetric) Cols() []string {
	return []string{
		"UUID",
		"Name",
		"Source",
		"Category",
		"ValueType",
		"Description",
	}
}

func (v *EvaluationMetric) ColMap() map[string]string {
	return map[string]string{
		"UUID":        "UUID",
		"Name":        "Name",
		"Source":      "Source",
		"Category":    "Category",
		"ValueType":   "Value Type",
		"Description": "Description",
	}
}

func (v *EvaluationMetric) KV() []map[string]any {
	if v == nil || v.Metrics == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(v.Metrics))
	for _, metric := range v.Metrics {
		out = append(out, map[string]any{
			"UUID":        metric.MetricUUID,
			"Name":        metric.MetricName,
			"Source":      metric.Source,
			"Category":    metric.Category,
			"ValueType":   metric.MetricValueType,
			"Description": metric.Description,
		})
	}
	return out
}

func formatEvaluationProgress(progress *godo.ModelEvaluationRunProgress) string {
	if progress == nil || progress.TotalRows == nil || *progress.TotalRows == 0 {
		return ""
	}
	candidate := int64(0)
	if progress.CandidateRowsEvaluated != nil {
		candidate = *progress.CandidateRowsEvaluated
	}
	judge := int64(0)
	if progress.JudgeRowsEvaluated != nil {
		judge = *progress.JudgeRowsEvaluated
	}
	return fmt.Sprintf("candidate %d/%d, judge %d/%d", candidate, *progress.TotalRows, judge, *progress.TotalRows)
}
