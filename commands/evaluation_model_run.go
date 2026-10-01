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

package commands

import (
	"encoding/json"
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	modelEvaluationRunStatusPrefix    = "MODEL_EVALUATION_RUN_"
	modelEvaluationRunSortFieldPrefix = "MODEL_EVALUATION_RUN_SORT_FIELD_"
	candidateModelSourcePrefix        = "CANDIDATE_MODEL_SOURCE_"
	presetSaveSectionPrefix           = "PRESET_SAVE_SECTION_"
)

// ModelEvaluationRunCmd handles operations on model evaluation runs.
func ModelEvaluationRunCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "model-run",
			Aliases: []string{"model-runs", "mr"},
			Short:   "Display commands that manage GenAI model evaluation runs.",
			Long:    "The subcommands of `doctl evaluation model-run` create and inspect model evaluation runs.",
		},
	}

	modelRunDetails := `
		- The evaluation run UUID
		- The evaluation run name
		- The evaluation run status
		- The candidate model name
		- The dataset name
		- The evaluation run progress
		- The evaluation run creation timestamp
	`

	cmdModelRunCreate := CmdBuilder(
		cmd,
		RunModelEvaluationRunCreate,
		"create",
		"Create a model evaluation run",
		"Starts a model evaluation run and returns the run UUID.",
		Writer, aliasOpt("c"),
		displayerType(&displayers.ModelEvaluationRunCreate{}),
	)
	AddStringFlag(cmdModelRunCreate, doctl.ArgGenAIName, "", "", "The name of the evaluation run.", requiredOpt())
	AddStringFlag(cmdModelRunCreate, doctl.ArgEvaluationCandidateModelUUID, "", "", "The UUID of the candidate model under evaluation.")
	AddStringFlag(cmdModelRunCreate, doctl.ArgEvaluationCandidateModelName, "", "", "The display name of the candidate model.")
	AddStringFlag(cmdModelRunCreate, doctl.ArgEvaluationCandidateModelSource, "", "", "The source of the candidate model. One of: `serverless`, `dedicated`, `router`, `agent`")
	AddStringFlag(cmdModelRunCreate, doctl.ArgEvaluationDatasetUUID, "", "", "The UUID of the evaluation dataset.")
	AddStringFlag(cmdModelRunCreate, doctl.ArgSimulationJudgeModelUUID, "", "", "The UUID of the judge model.")
	AddStringSliceFlag(cmdModelRunCreate, doctl.ArgSimulationMetricUUIDs, "", []string{}, "The UUIDs of evaluation metrics to score the run with.")
	AddStringFlag(cmdModelRunCreate, doctl.ArgEvaluationEvalPresetUUID, "", "", "The UUID of a saved evaluation preset to use.")
	AddStringFlag(cmdModelRunCreate, doctl.ArgEvaluationPresetName, "", "", "The name to use when saving a new evaluation preset from this run.")
	AddStringSliceFlag(cmdModelRunCreate, doctl.ArgEvaluationPresetSaveSections, "", []string{}, "Sections to save into a new preset. One of: `candidate`, `metrics`, `judge`, `dataset`, `system_prompt`")
	AddIntFlag(cmdModelRunCreate, doctl.ArgEvaluationEpochs, "", 0, "The number of times to evaluate each dataset row (n-pass).")
	AddStringFlag(cmdModelRunCreate, doctl.ArgEvaluationCandidateInferenceConfig, "", "", "A JSON object of candidate inference settings.")
	AddStringFlag(cmdModelRunCreate, doctl.ArgSimulationStarMetricUUID, "", "", "The UUID of the star metric for the run.")
	AddStringFlag(cmdModelRunCreate, doctl.ArgSimulationStarMetricName, "", "", "The name of the star metric for the run.")
	AddFloatFlag(cmdModelRunCreate, doctl.ArgSimulationStarMetricSuccessThreshold, "", 0, "The success threshold of the star metric.")
	cmdModelRunCreate.Example = "The following example creates a model evaluation run: " +
		"`doctl evaluation model-run create --name nightly-eval --candidate-model-uuid 99a1cbc7-b1b2-4a0d-9c1f-9b9d2b8f9d1e --dataset-uuid f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --judge-model-uuid 6ba7b810-9dad-11d1-80b4-00c04fd430c8 --metric-uuids 11111111-1111-4111-8111-111111111111`"

	cmdModelRunList := CmdBuilder(
		cmd,
		RunModelEvaluationRunList,
		"list",
		"List all model evaluation runs",
		"Retrieves a list of model evaluation runs, where each run contains:"+modelRunDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.ModelEvaluationRun{}),
	)
	AddStringSliceFlag(cmdModelRunList, doctl.ArgGenAIStatuses, "", []string{}, "Filters the results by status. One of: `queued`, `running_dataset`, `evaluating_results`, `cancelling`, `cancelled`, `successful`, `partially_successful`, `failed`")
	AddStringFlag(cmdModelRunList, doctl.ArgEvaluationEvalPresetUUID, "", "", "Filters the results by evaluation preset UUID.")
	AddStringSliceFlag(cmdModelRunList, doctl.ArgEvaluationCandidateTypes, "", []string{}, "Filters the results by candidate model source. One of: `serverless`, `dedicated`, `router`, `agent`")
	addGenAIListFlags(cmdModelRunList, "`created_at`, `status`")
	cmdModelRunList.Example = "The following example lists model evaluation runs that are still queued: " +
		"`doctl evaluation model-run list --statuses queued`"

	cmdModelRunGet := CmdBuilder(
		cmd,
		RunModelEvaluationRunGet,
		"get <run-uuid>",
		"Retrieve a model evaluation run",
		"Retrieves a model evaluation run along with its overall score and progress. Use `-o json` to also see the per-prompt results.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.ModelEvaluationRunDetail{}),
	)
	cmdModelRunGet.Example = "The following example retrieves a model evaluation run: " +
		"`doctl evaluation model-run get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdModelRunUpdate := CmdBuilder(
		cmd,
		RunModelEvaluationRunUpdate,
		"update <run-uuid>",
		"Update a model evaluation run",
		"Renames a model evaluation run.",
		Writer, aliasOpt("u"),
		displayerType(&displayers.ModelEvaluationRun{}),
	)
	AddStringFlag(cmdModelRunUpdate, doctl.ArgGenAIName, "", "", "The new name of the evaluation run.", requiredOpt())
	cmdModelRunUpdate.Example = "The following example renames a model evaluation run: " +
		"`doctl evaluation model-run update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name nightly-eval-v2`"

	cmdModelRunCancel := CmdBuilder(
		cmd,
		RunModelEvaluationRunCancel,
		"cancel <run-uuid>",
		"Cancel a model evaluation run",
		"Cancels a model evaluation run that is still in progress.",
		Writer,
		displayerType(&displayers.ModelEvaluationRun{}),
	)
	AddBoolFlag(cmdModelRunCancel, doctl.ArgForce, doctl.ArgShortForce, false, "Cancels the evaluation run without a confirmation prompt")
	cmdModelRunCancel.Example = "The following example cancels a model evaluation run: " +
		"`doctl evaluation model-run cancel f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdModelRunDelete := CmdBuilder(
		cmd,
		RunModelEvaluationRunDelete,
		"delete <run-uuid>",
		"Delete a model evaluation run",
		"Deletes a model evaluation run by its UUID.",
		Writer, aliasOpt("del", "rm"),
	)
	AddBoolFlag(cmdModelRunDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the evaluation run without a confirmation prompt")
	cmdModelRunDelete.Example = "The following example deletes a model evaluation run: " +
		"`doctl evaluation model-run delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdModelRunResultsDownloadURL := CmdBuilder(
		cmd,
		RunModelEvaluationRunResultsDownloadURL,
		"results-download-url <run-uuid>",
		"Retrieve a download URL for evaluation run results",
		"Retrieves a temporary presigned URL for downloading the results of a model evaluation run.",
		Writer, aliasOpt("results-url"),
		displayerType(&displayers.GenAIDownloadURL{}),
	)
	cmdModelRunResultsDownloadURL.Example = "The following example retrieves a download URL for evaluation run results: " +
		"`doctl evaluation model-run results-download-url f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	return cmd
}

// RunModelEvaluationRunCreate creates a model evaluation run.
func RunModelEvaluationRunCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	candidateModelUUID, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationCandidateModelUUID)
	if err != nil {
		return err
	}

	candidateModelName, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationCandidateModelName)
	if err != nil {
		return err
	}

	rawCandidateModelSource, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationCandidateModelSource)
	if err != nil {
		return err
	}

	datasetUUID, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationDatasetUUID)
	if err != nil {
		return err
	}

	judgeModelUUID, err := c.Doit.GetString(c.NS, doctl.ArgSimulationJudgeModelUUID)
	if err != nil {
		return err
	}

	metricUUIDs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSimulationMetricUUIDs)
	if err != nil {
		return err
	}

	evalPresetUUID, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationEvalPresetUUID)
	if err != nil {
		return err
	}

	presetName, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationPresetName)
	if err != nil {
		return err
	}

	rawPresetSaveSections, err := c.Doit.GetStringSlice(c.NS, doctl.ArgEvaluationPresetSaveSections)
	if err != nil {
		return err
	}

	epochs, err := c.Doit.GetInt(c.NS, doctl.ArgEvaluationEpochs)
	if err != nil {
		return err
	}

	rawInferenceConfig, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationCandidateInferenceConfig)
	if err != nil {
		return err
	}

	req := &godo.CreateModelEvaluationRunRequest{
		Name:                 name,
		CandidateModelUUID:   candidateModelUUID,
		CandidateModelName:   candidateModelName,
		CandidateModelSource: godo.CandidateModelSource(genAIEnumValue(candidateModelSourcePrefix, rawCandidateModelSource)),
		DatasetUUID:          datasetUUID,
		JudgeModelUUID:       judgeModelUUID,
		MetricUUIDs:          metricUUIDs,
		EvalPresetUUID:       evalPresetUUID,
		PresetName:           presetName,
		PresetSaveSections:   genAIEnums[godo.PresetSaveSection](presetSaveSectionPrefix, rawPresetSaveSections),
		Epochs:               uint32(epochs),
	}

	if rawInferenceConfig != "" {
		var inferenceConfig godo.CandidateInferenceConfig
		if err := json.Unmarshal([]byte(rawInferenceConfig), &inferenceConfig); err != nil {
			return fmt.Errorf("unable to parse candidate inference config: %w", err)
		}
		req.CandidateInferenceConfig = &inferenceConfig
	}

	starMetric, err := evaluationStarMetric(c)
	if err != nil {
		return err
	}
	req.StarMetric = starMetric

	create, err := c.AgentPlatform().CreateModelEvaluationRun(req)
	if err != nil {
		return err
	}

	return c.Display(&displayers.ModelEvaluationRunCreate{Create: create})
}

// RunModelEvaluationRunList lists model evaluation runs.
func RunModelEvaluationRunList(c *CmdConfig) error {
	rawStatuses, err := c.Doit.GetStringSlice(c.NS, doctl.ArgGenAIStatuses)
	if err != nil {
		return err
	}

	evalPresetUUID, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationEvalPresetUUID)
	if err != nil {
		return err
	}

	rawCandidateTypes, err := c.Doit.GetStringSlice(c.NS, doctl.ArgEvaluationCandidateTypes)
	if err != nil {
		return err
	}

	search, sortBy, sortDirection, err := genAIListFilters(c, modelEvaluationRunSortFieldPrefix)
	if err != nil {
		return err
	}

	runs, err := c.AgentPlatform().ListModelEvaluationRuns(&godo.ModelEvaluationRunListOptions{
		EvalPresetUUID: evalPresetUUID,
		Statuses:       genAIEnums[godo.ModelEvaluationRunStatus](modelEvaluationRunStatusPrefix, rawStatuses),
		CandidateTypes: genAIEnums[godo.CandidateModelSource](candidateModelSourcePrefix, rawCandidateTypes),
		Search:         search,
		SortBy:         godo.ModelEvaluationRunSortField(sortBy),
		SortDirection:  godo.ModelEvaluationRunSortDirection(sortDirection),
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.ModelEvaluationRun{ModelEvaluationRuns: runs})
}

// RunModelEvaluationRunGet retrieves a model evaluation run by its UUID.
func RunModelEvaluationRunGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	detail, err := c.AgentPlatform().GetModelEvaluationRun(c.Args[0])
	if err != nil {
		return err
	}

	return c.Display(&displayers.ModelEvaluationRunDetail{Detail: detail})
}

// RunModelEvaluationRunUpdate updates a model evaluation run.
func RunModelEvaluationRunUpdate(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	name, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	run, err := c.AgentPlatform().UpdateModelEvaluationRun(c.Args[0], &godo.UpdateModelEvaluationRunRequest{
		Name: name,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.ModelEvaluationRun{ModelEvaluationRuns: do.ModelEvaluationRuns{*run}})
}

// RunModelEvaluationRunCancel cancels an in-progress model evaluation run.
func RunModelEvaluationRunCancel(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirm("cancel this model evaluation run") != nil {
		return errOperationAborted
	}

	run, err := c.AgentPlatform().CancelModelEvaluationRun(c.Args[0])
	if err != nil {
		return err
	}

	return c.Display(&displayers.ModelEvaluationRun{ModelEvaluationRuns: do.ModelEvaluationRuns{*run}})
}

// RunModelEvaluationRunDelete deletes a model evaluation run by its UUID.
func RunModelEvaluationRunDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirmDelete("model evaluation run", 1) != nil {
		return errOperationAborted
	}

	if err := c.AgentPlatform().DeleteModelEvaluationRun(c.Args[0]); err != nil {
		return err
	}

	notice("Model evaluation run deleted successfully")
	return nil
}

// RunModelEvaluationRunResultsDownloadURL retrieves a download URL for evaluation run results.
func RunModelEvaluationRunResultsDownloadURL(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	downloadURL, err := c.AgentPlatform().GetModelEvaluationRunResultsDownloadURL(c.Args[0])
	if err != nil {
		return err
	}

	return c.Display(&displayers.GenAIDownloadURL{DownloadURL: downloadURL})
}

func evaluationStarMetric(c *CmdConfig) (*godo.StarMetric, error) {
	starMetricUUID, err := c.Doit.GetString(c.NS, doctl.ArgSimulationStarMetricUUID)
	if err != nil {
		return nil, err
	}

	starMetricName, err := c.Doit.GetString(c.NS, doctl.ArgSimulationStarMetricName)
	if err != nil {
		return nil, err
	}

	starMetricSuccessThreshold, err := c.Doit.GetFloat64(c.NS, doctl.ArgSimulationStarMetricSuccessThreshold)
	if err != nil {
		return nil, err
	}

	if starMetricUUID == "" && starMetricName == "" && starMetricSuccessThreshold == 0 {
		return nil, nil
	}

	starMetric := &godo.StarMetric{
		MetricUUID: starMetricUUID,
		Name:       starMetricName,
	}
	if starMetricSuccessThreshold != 0 {
		threshold := float32(starMetricSuccessThreshold)
		starMetric.SuccessThreshold = &threshold
	}
	return starMetric, nil
}
