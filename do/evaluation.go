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

package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// ModelEvalDatasetFileUploads wraps a godo.CreateModelEvalDatasetUploadPresignedURLsResponse.
type ModelEvalDatasetFileUploads struct {
	*godo.CreateModelEvalDatasetUploadPresignedURLsResponse
}

// EvaluationDatasetCreate wraps a godo.CreateEvaluationDatasetResponse.
type EvaluationDatasetCreate struct {
	*godo.CreateEvaluationDatasetResponse
}

// EvaluationDataset wraps a godo.EvaluationDatasetInfo.
type EvaluationDataset struct {
	*godo.EvaluationDatasetInfo
}

// EvaluationDatasets is a slice of EvaluationDataset.
type EvaluationDatasets []EvaluationDataset

// ModelEvaluationRunCreate wraps a godo.ModelEvaluationRunCreateResponse.
type ModelEvaluationRunCreate struct {
	*godo.ModelEvaluationRunCreateResponse
}

// ModelEvaluationRun wraps a godo.ModelEvaluationRunSummary.
type ModelEvaluationRun struct {
	*godo.ModelEvaluationRunSummary
}

// ModelEvaluationRuns is a slice of ModelEvaluationRun.
type ModelEvaluationRuns []ModelEvaluationRun

// ModelEvaluationRunDetail wraps a godo.ModelEvaluationRunGetResponse, which
// carries a run together with its paginated per-prompt results.
type ModelEvaluationRunDetail struct {
	*godo.ModelEvaluationRunGetResponse
}

// ModelEvaluationPreset wraps a godo.ModelEvaluationPreset.
type ModelEvaluationPreset struct {
	*godo.ModelEvaluationPreset
}

// ModelEvaluationPresets is a slice of ModelEvaluationPreset.
type ModelEvaluationPresets []ModelEvaluationPreset

// EvaluationMetric wraps a godo.EvaluationMetric.
type EvaluationMetric struct {
	*godo.EvaluationMetric
}

// EvaluationMetrics is a slice of EvaluationMetric.
type EvaluationMetrics []EvaluationMetric

// CreateModelEvalDatasetUploadPresignedURLs creates presigned URLs for uploading
// model evaluation dataset files.
func (a *agentPlatformService) CreateModelEvalDatasetUploadPresignedURLs(req *godo.CreateModelEvalDatasetUploadPresignedURLsRequest) (*ModelEvalDatasetFileUploads, error) {
	uploads, _, err := a.client.AgentPlatform.CreateModelEvalDatasetUploadPresignedURLs(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &ModelEvalDatasetFileUploads{CreateModelEvalDatasetUploadPresignedURLsResponse: uploads}, nil
}

// CreateEvaluationDataset registers an evaluation dataset from a previously uploaded file.
func (a *agentPlatformService) CreateEvaluationDataset(req *godo.CreateEvaluationDatasetRequest) (*EvaluationDatasetCreate, error) {
	res, _, err := a.client.AgentPlatform.CreateEvaluationDataset(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &EvaluationDatasetCreate{CreateEvaluationDatasetResponse: res}, nil
}

// ListEvaluationDatasets lists evaluation datasets for the team.
func (a *agentPlatformService) ListEvaluationDatasets(opt *godo.EvaluationDatasetListOptions) (EvaluationDatasets, error) {
	res, _, err := a.client.AgentPlatform.ListEvaluationDatasets(context.TODO(), opt)
	if err != nil {
		return nil, err
	}

	list := make(EvaluationDatasets, len(res.EvaluationDatasets))
	for i := range res.EvaluationDatasets {
		list[i] = EvaluationDataset{EvaluationDatasetInfo: res.EvaluationDatasets[i]}
	}
	return list, nil
}

// DeleteEvaluationDataset deletes an evaluation dataset by its UUID.
func (a *agentPlatformService) DeleteEvaluationDataset(datasetUUID string) error {
	_, _, err := a.client.AgentPlatform.DeleteEvaluationDataset(context.TODO(), datasetUUID)
	return err
}

// CreateModelEvaluationRun creates a model evaluation run.
func (a *agentPlatformService) CreateModelEvaluationRun(req *godo.CreateModelEvaluationRunRequest) (*ModelEvaluationRunCreate, error) {
	res, _, err := a.client.AgentPlatform.CreateModelEvaluationRun(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &ModelEvaluationRunCreate{ModelEvaluationRunCreateResponse: res}, nil
}

// ListModelEvaluationRuns lists model evaluation runs for the team.
func (a *agentPlatformService) ListModelEvaluationRuns(opt *godo.ModelEvaluationRunListOptions) (ModelEvaluationRuns, error) {
	if opt == nil {
		opt = &godo.ModelEvaluationRunListOptions{}
	}

	return paginateGenAIList(func(listOpt *godo.ListOptions) ([]*godo.ModelEvaluationRunSummary, *godo.Response, error) {
		filters := *opt
		filters.ListOptions = *listOpt
		res, resp, err := a.client.AgentPlatform.ListModelEvaluationRuns(context.TODO(), &filters)
		if err != nil {
			return nil, nil, err
		}
		return res.Runs, resp, nil
	}, func(run *godo.ModelEvaluationRunSummary) ModelEvaluationRun {
		return ModelEvaluationRun{ModelEvaluationRunSummary: run}
	})
}

// GetModelEvaluationRun retrieves a model evaluation run by its UUID, including
// the first page of per-prompt results.
func (a *agentPlatformService) GetModelEvaluationRun(evalRunUUID string) (*ModelEvaluationRunDetail, error) {
	res, _, err := a.client.AgentPlatform.GetModelEvaluationRun(context.TODO(), evalRunUUID, nil)
	if err != nil {
		return nil, err
	}
	return &ModelEvaluationRunDetail{ModelEvaluationRunGetResponse: res}, nil
}

// UpdateModelEvaluationRun updates a model evaluation run by its UUID.
func (a *agentPlatformService) UpdateModelEvaluationRun(evalRunUUID string, req *godo.UpdateModelEvaluationRunRequest) (*ModelEvaluationRun, error) {
	res, _, err := a.client.AgentPlatform.UpdateModelEvaluationRun(context.TODO(), evalRunUUID, req)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Run == nil {
		return &ModelEvaluationRun{}, nil
	}
	return &ModelEvaluationRun{ModelEvaluationRunSummary: res.Run}, nil
}

// CancelModelEvaluationRun cancels an in-progress model evaluation run.
func (a *agentPlatformService) CancelModelEvaluationRun(evalRunUUID string) (*ModelEvaluationRun, error) {
	res, _, err := a.client.AgentPlatform.CancelModelEvaluationRun(context.TODO(), evalRunUUID)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Run == nil {
		return &ModelEvaluationRun{}, nil
	}
	return &ModelEvaluationRun{ModelEvaluationRunSummary: res.Run}, nil
}

// DeleteModelEvaluationRun deletes a model evaluation run by its UUID.
func (a *agentPlatformService) DeleteModelEvaluationRun(evalRunUUID string) error {
	_, _, err := a.client.AgentPlatform.DeleteModelEvaluationRun(context.TODO(), evalRunUUID)
	return err
}

// GetModelEvaluationRunResultsDownloadURL returns a presigned download URL for a
// model evaluation run's results.
func (a *agentPlatformService) GetModelEvaluationRunResultsDownloadURL(evalRunUUID string) (*GenAIDownloadURL, error) {
	res, _, err := a.client.AgentPlatform.GetModelEvaluationRunResultsDownloadURL(context.TODO(), evalRunUUID)
	if err != nil {
		return nil, err
	}
	return &GenAIDownloadURL{DownloadURL: res.DownloadURL, ExpiresAt: res.ExpiresAt}, nil
}

// ListModelEvaluationPresets lists saved model evaluation presets.
func (a *agentPlatformService) ListModelEvaluationPresets() (ModelEvaluationPresets, error) {
	res, _, err := a.client.AgentPlatform.ListModelEvaluationPresets(context.TODO())
	if err != nil {
		return nil, err
	}

	list := make(ModelEvaluationPresets, len(res.Presets))
	for i := range res.Presets {
		list[i] = ModelEvaluationPreset{ModelEvaluationPreset: res.Presets[i]}
	}
	return list, nil
}

// GetModelEvaluationPreset retrieves a saved model evaluation preset by UUID.
func (a *agentPlatformService) GetModelEvaluationPreset(evalPresetUUID string) (*ModelEvaluationPreset, error) {
	res, _, err := a.client.AgentPlatform.GetModelEvaluationPreset(context.TODO(), evalPresetUUID)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Preset == nil {
		return &ModelEvaluationPreset{}, nil
	}
	return &ModelEvaluationPreset{ModelEvaluationPreset: res.Preset}, nil
}

// DeleteModelEvaluationPreset deletes a saved model evaluation preset by UUID.
func (a *agentPlatformService) DeleteModelEvaluationPreset(evalPresetUUID string) error {
	_, _, err := a.client.AgentPlatform.DeleteModelEvaluationPreset(context.TODO(), evalPresetUUID)
	return err
}

// ListModelEvaluationMetrics lists available model evaluation metrics.
func (a *agentPlatformService) ListModelEvaluationMetrics() (EvaluationMetrics, error) {
	res, _, err := a.client.AgentPlatform.ListModelEvaluationMetrics(context.TODO())
	if err != nil {
		return nil, err
	}

	list := make(EvaluationMetrics, len(res.Metrics))
	for i := range res.Metrics {
		list[i] = EvaluationMetric{EvaluationMetric: res.Metrics[i]}
	}
	return list, nil
}

// CreateCustomEvaluationMetric creates a custom model evaluation metric.
func (a *agentPlatformService) CreateCustomEvaluationMetric(req *godo.CreateCustomEvaluationMetricRequest) (*EvaluationMetric, error) {
	metric, _, err := a.client.AgentPlatform.CreateCustomEvaluationMetric(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &EvaluationMetric{EvaluationMetric: metric}, nil
}

// UpdateCustomEvaluationMetric updates a custom model evaluation metric.
func (a *agentPlatformService) UpdateCustomEvaluationMetric(metricUUID string, req *godo.UpdateCustomEvaluationMetricRequest) (*EvaluationMetric, error) {
	metric, _, err := a.client.AgentPlatform.UpdateCustomEvaluationMetric(context.TODO(), metricUUID, req)
	if err != nil {
		return nil, err
	}
	return &EvaluationMetric{EvaluationMetric: metric}, nil
}

// DeleteCustomEvaluationMetric deletes a custom model evaluation metric.
func (a *agentPlatformService) DeleteCustomEvaluationMetric(metricUUID string) error {
	_, err := a.client.AgentPlatform.DeleteCustomEvaluationMetric(context.TODO(), metricUUID)
	return err
}
