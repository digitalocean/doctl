package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testModelEvaluationRunUUID = "55555555-5555-4555-8555-555555555555"

	testModelEvaluationRun = do.ModelEvaluationRun{
		ModelEvaluationRunSummary: &godo.ModelEvaluationRunSummary{
			EvalRunUuid:        testModelEvaluationRunUUID,
			Name:               "Nightly Eval",
			Status:             godo.ModelEvaluationRunQueued,
			CandidateModelName: "llama-3",
			DatasetName:        "support-prompts",
		},
	}

	testModelEvaluationRunCreate = do.ModelEvaluationRunCreate{
		ModelEvaluationRunCreateResponse: &godo.ModelEvaluationRunCreateResponse{
			EvalRunUuid: testModelEvaluationRunUUID,
		},
	}
)

func TestModelEvaluationRunCommand(t *testing.T) {
	cmd := ModelEvaluationRunCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "get", "update", "cancel", "delete", "results-download-url")
}

func TestModelEvaluationRunCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Nightly Eval")
		config.Doit.Set(config.NS, doctl.ArgEvaluationCandidateModelUUID, "candidate-model-uuid")
		config.Doit.Set(config.NS, doctl.ArgEvaluationDatasetUUID, "dataset-uuid")
		config.Doit.Set(config.NS, doctl.ArgSimulationJudgeModelUUID, "judge-model-uuid")
		config.Doit.Set(config.NS, doctl.ArgSimulationMetricUUIDs, []string{"metric-uuid-1"})

		tm.gradientAI.EXPECT().CreateModelEvaluationRun(&godo.CreateModelEvaluationRunRequest{
			Name:               "Nightly Eval",
			CandidateModelUUID: "candidate-model-uuid",
			DatasetUUID:        "dataset-uuid",
			JudgeModelUUID:     "judge-model-uuid",
			MetricUUIDs:        []string{"metric-uuid-1"},
		}).Return(&testModelEvaluationRunCreate, nil)

		err := RunModelEvaluationRunCreate(config)
		assert.NoError(t, err)
	})
}

func TestModelEvaluationRunList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.gradientAI.EXPECT().ListModelEvaluationRuns(&godo.ModelEvaluationRunListOptions{}).
			Return(do.ModelEvaluationRuns{testModelEvaluationRun}, nil)

		err := RunModelEvaluationRunList(config)
		assert.NoError(t, err)
	})
}

func TestModelEvaluationRunListWithFilters(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgGenAIStatuses, []string{"queued", "successful"})
		config.Doit.Set(config.NS, doctl.ArgEvaluationEvalPresetUUID, "preset-uuid")
		config.Doit.Set(config.NS, doctl.ArgEvaluationCandidateTypes, []string{"serverless"})
		config.Doit.Set(config.NS, doctl.ArgGenAISortBy, "created_at")
		config.Doit.Set(config.NS, doctl.ArgGenAISortDirection, "desc")

		tm.gradientAI.EXPECT().ListModelEvaluationRuns(&godo.ModelEvaluationRunListOptions{
			EvalPresetUUID: "preset-uuid",
			Statuses: []godo.ModelEvaluationRunStatus{
				godo.ModelEvaluationRunQueued,
				godo.ModelEvaluationRunSuccessful,
			},
			CandidateTypes: []godo.CandidateModelSource{godo.CandidateModelSourceServerless},
			SortBy:         godo.ModelEvaluationRunSortFieldCreatedAt,
			SortDirection:  godo.ModelEvaluationRunSortDirectionDesc,
		}).Return(do.ModelEvaluationRuns{testModelEvaluationRun}, nil)

		err := RunModelEvaluationRunList(config)
		assert.NoError(t, err)
	})
}

func TestModelEvaluationRunGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testModelEvaluationRunUUID)

		tm.gradientAI.EXPECT().GetModelEvaluationRun(testModelEvaluationRunUUID).Return(&do.ModelEvaluationRunDetail{
			ModelEvaluationRunGetResponse: &godo.ModelEvaluationRunGetResponse{
				Run: &godo.ModelEvaluationRunDetail{
					EvalRunUuid: testModelEvaluationRunUUID,
					Name:        "Nightly Eval",
					Status:      godo.ModelEvaluationRunSuccessful,
					ResultSummary: &godo.ModelEvaluationRunResultSummary{
						OverallScorePercent: 92.5,
					},
				},
			},
		}, nil)

		err := RunModelEvaluationRunGet(config)
		assert.NoError(t, err)
	})
}

func TestModelEvaluationRunUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testModelEvaluationRunUUID)
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Nightly Eval v2")

		tm.gradientAI.EXPECT().UpdateModelEvaluationRun(testModelEvaluationRunUUID, &godo.UpdateModelEvaluationRunRequest{
			Name: "Nightly Eval v2",
		}).Return(&testModelEvaluationRun, nil)

		err := RunModelEvaluationRunUpdate(config)
		assert.NoError(t, err)
	})
}

func TestModelEvaluationRunCancel(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testModelEvaluationRunUUID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.gradientAI.EXPECT().CancelModelEvaluationRun(testModelEvaluationRunUUID).Return(&testModelEvaluationRun, nil)

		err := RunModelEvaluationRunCancel(config)
		assert.NoError(t, err)
	})
}

func TestModelEvaluationRunDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testModelEvaluationRunUUID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.gradientAI.EXPECT().DeleteModelEvaluationRun(testModelEvaluationRunUUID).Return(nil)

		err := RunModelEvaluationRunDelete(config)
		assert.NoError(t, err)
	})
}

func TestModelEvaluationRunResultsDownloadURL(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testModelEvaluationRunUUID)

		tm.gradientAI.EXPECT().GetModelEvaluationRunResultsDownloadURL(testModelEvaluationRunUUID).
			Return(&do.GenAIDownloadURL{DownloadURL: "https://example.com/results"}, nil)

		err := RunModelEvaluationRunResultsDownloadURL(config)
		assert.NoError(t, err)
	})
}
