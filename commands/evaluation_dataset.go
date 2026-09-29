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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	evaluationDatasetTypePrefix     = "EVALUATION_DATASET_TYPE_"
	evaluationDatasetParadigmPrefix = "EVALUATION_DATASET_PARADIGM_"
)

// EvaluationDatasetCmd handles operations on evaluation datasets.
func EvaluationDatasetCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "dataset",
			Aliases: []string{"datasets", "ds"},
			Short:   "Display commands that manage GenAI evaluation datasets.",
			Long:    "The subcommands of `doctl evaluation dataset` upload and manage datasets used by model evaluation runs.",
		},
	}

	datasetDetails := `
		- The dataset UUID
		- The dataset name
		- The dataset type
		- The dataset paradigm
		- The number of rows
		- Whether the dataset includes ground truth
		- The dataset creation timestamp
	`

	cmdDatasetCreate := CmdBuilder(
		cmd,
		RunEvaluationDatasetCreate,
		"create",
		"Create an evaluation dataset",
		"Uploads a local file and registers it as an evaluation dataset, returning the dataset UUID.",
		Writer, aliasOpt("c"),
		displayerType(&displayers.EvaluationDatasetCreate{}),
	)
	AddStringFlag(cmdDatasetCreate, doctl.ArgGenAIName, "", "", "The name of the evaluation dataset.", requiredOpt())
	AddStringFlag(cmdDatasetCreate, doctl.ArgScenarioSetFile, "", "", "The path to a local CSV or JSONL dataset file to upload.", requiredOpt())
	AddStringFlag(cmdDatasetCreate, doctl.ArgEvaluationDatasetParadigm, "", "single_turn", "The paradigm of the dataset. One of: `single_turn`, `multi_turn`, `coding`, `n_plus_1`")
	cmdDatasetCreate.Example = "The following example creates an evaluation dataset from a local file: " +
		"`doctl evaluation dataset create --name support-prompts --file ./prompts.csv`"

	cmdDatasetList := CmdBuilder(
		cmd,
		RunEvaluationDatasetList,
		"list",
		"List all evaluation datasets",
		"Retrieves a list of evaluation datasets, where each dataset contains:"+datasetDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.EvaluationDataset{}),
	)
	AddStringFlag(cmdDatasetList, doctl.ArgEvaluationDatasetType, "", "", "Filters the results by dataset type. One of: `adk`, `non_adk`, `model`")
	AddStringFlag(cmdDatasetList, doctl.ArgEvaluationDatasetParadigm, "", "", "Filters the results by dataset paradigm. One of: `single_turn`, `multi_turn`, `coding`, `n_plus_1`")
	AddBoolFlag(cmdDatasetList, doctl.ArgEvaluationHasGroundTruth, "", false, "Filters the results by whether the dataset includes ground truth.")
	cmdDatasetList.Example = "The following example lists model evaluation datasets: " +
		"`doctl evaluation dataset list --dataset-type model`"

	cmdDatasetDelete := CmdBuilder(
		cmd,
		RunEvaluationDatasetDelete,
		"delete <dataset-uuid>",
		"Delete an evaluation dataset",
		"Deletes an evaluation dataset by its UUID.",
		Writer, aliasOpt("del", "rm"),
	)
	AddBoolFlag(cmdDatasetDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the evaluation dataset without a confirmation prompt")
	cmdDatasetDelete.Example = "The following example deletes an evaluation dataset: " +
		"`doctl evaluation dataset delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	return cmd
}

// RunEvaluationDatasetCreate creates an evaluation dataset from a local file.
func RunEvaluationDatasetCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	path, err := c.Doit.GetString(c.NS, doctl.ArgScenarioSetFile)
	if err != nil {
		return err
	}

	rawParadigm, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationDatasetParadigm)
	if err != nil {
		return err
	}
	if rawParadigm == "" {
		rawParadigm = "single_turn"
	}

	dataSource, err := uploadEvaluationDatasetFile(c, path)
	if err != nil {
		return err
	}

	create, err := c.GradientAI().CreateEvaluationDataset(&godo.CreateEvaluationDatasetRequest{
		Name:              name,
		DatasetType:       godo.EvaluationDatasetTypeModel,
		DatasetParadigm:   godo.EvaluationDatasetParadigm(genAIEnumValue(evaluationDatasetParadigmPrefix, rawParadigm)),
		FileUploadDataset: dataSource,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.EvaluationDatasetCreate{Create: create})
}

// RunEvaluationDatasetList lists evaluation datasets.
func RunEvaluationDatasetList(c *CmdConfig) error {
	rawType, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationDatasetType)
	if err != nil {
		return err
	}

	rawParadigm, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationDatasetParadigm)
	if err != nil {
		return err
	}

	opt := &godo.EvaluationDatasetListOptions{
		DatasetType:     godo.EvaluationDatasetType(genAIEnumValue(evaluationDatasetTypePrefix, rawType)),
		DatasetParadigm: godo.EvaluationDatasetParadigm(genAIEnumValue(evaluationDatasetParadigmPrefix, rawParadigm)),
	}

	if c.Doit.IsSet(doctl.ArgEvaluationHasGroundTruth) {
		hasGroundTruth, err := c.Doit.GetBoolPtr(c.NS, doctl.ArgEvaluationHasGroundTruth)
		if err != nil {
			return err
		}
		opt.HasGroundTruth = hasGroundTruth
	}

	datasets, err := c.GradientAI().ListEvaluationDatasets(opt)
	if err != nil {
		return err
	}

	return c.Display(&displayers.EvaluationDataset{EvaluationDatasets: datasets})
}

// RunEvaluationDatasetDelete deletes an evaluation dataset by its UUID.
func RunEvaluationDatasetDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirmDelete("evaluation dataset", 1) != nil {
		return errOperationAborted
	}

	if err := c.GradientAI().DeleteEvaluationDataset(c.Args[0]); err != nil {
		return err
	}

	notice("Evaluation dataset deleted successfully")
	return nil
}

func uploadEvaluationDatasetFile(c *CmdConfig, path string) (*godo.FileUploadDataSource, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory, not a dataset file", path)
	}

	fileName := filepath.Base(path)
	size := strconv.FormatInt(info.Size(), 10)

	uploads, err := c.GradientAI().CreateModelEvalDatasetUploadPresignedURLs(&godo.CreateModelEvalDatasetUploadPresignedURLsRequest{
		Files: []*godo.PresignedUrlFile{{
			FileName: fileName,
			FileSize: size,
		}},
	})
	if err != nil {
		return nil, err
	}
	if uploads == nil || len(uploads.Uploads) == 0 {
		return nil, fmt.Errorf("no presigned upload URL was returned for %q", fileName)
	}

	upload := uploads.Uploads[0]
	if upload == nil || upload.PresignedURL == "" || upload.ObjectKey == "" {
		return nil, fmt.Errorf("the presigned upload response for %q was incomplete", fileName)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if err := putPresignedFile(upload.PresignedURL, file, info.Size(), evaluationDatasetContentType(fileName)); err != nil {
		return nil, err
	}

	return &godo.FileUploadDataSource{
		OriginalFileName: fileName,
		Size:             size,
		StoredObjectKey:  upload.ObjectKey,
	}, nil
}

func evaluationDatasetContentType(fileName string) string {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".csv":
		return "text/csv"
	case ".jsonl":
		return "application/jsonl"
	default:
		return ""
	}
}
