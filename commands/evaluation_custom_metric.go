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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// CustomEvaluationMetricCmd handles operations on custom evaluation metrics.
func CustomEvaluationMetricCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "custom-metric",
			Aliases: []string{"custom-metrics"},
			Short:   "Display commands that manage GenAI custom evaluation metrics.",
			Long:    "The subcommands of `doctl evaluation custom-metric` create, update, and delete custom LLM-as-judge metrics.",
		},
	}

	cmdCustomMetricCreate := CmdBuilder(
		cmd,
		RunCustomEvaluationMetricCreate,
		"create",
		"Create a custom evaluation metric",
		"Creates a custom LLM-as-judge evaluation metric.",
		Writer, aliasOpt("c"),
		displayerType(&displayers.EvaluationMetric{}),
	)
	AddStringFlag(cmdCustomMetricCreate, doctl.ArgGenAIName, "", "", "The name of the custom metric.", requiredOpt())
	AddStringFlag(cmdCustomMetricCreate, doctl.ArgEvaluationScoringPrompt, "", "", "The scoring prompt used by the judge model.", requiredOpt())
	AddStringFlag(cmdCustomMetricCreate, doctl.ArgEvaluationDescription, "", "", "A description of the custom metric.")
	AddBoolFlag(cmdCustomMetricCreate, doctl.ArgEvaluationRequiresGroundTruth, "", false, "Whether the custom metric requires ground-truth values.")
	cmdCustomMetricCreate.Example = "The following example creates a custom evaluation metric: " +
		"`doctl evaluation custom-metric create --name helpfulness --scoring-prompt \"Score helpfulness from 0 to 1\" --description \"Measures response helpfulness\"`"

	cmdCustomMetricUpdate := CmdBuilder(
		cmd,
		RunCustomEvaluationMetricUpdate,
		"update <metric-uuid>",
		"Update a custom evaluation metric",
		"Updates a custom LLM-as-judge evaluation metric.",
		Writer, aliasOpt("u"),
		displayerType(&displayers.EvaluationMetric{}),
	)
	AddStringFlag(cmdCustomMetricUpdate, doctl.ArgGenAIName, "", "", "The new name of the custom metric.")
	AddStringFlag(cmdCustomMetricUpdate, doctl.ArgEvaluationScoringPrompt, "", "", "The scoring prompt used by the judge model.")
	AddStringFlag(cmdCustomMetricUpdate, doctl.ArgEvaluationDescription, "", "", "A description of the custom metric.")
	AddBoolFlag(cmdCustomMetricUpdate, doctl.ArgEvaluationRequiresGroundTruth, "", false, "Whether the custom metric requires ground-truth values.")
	cmdCustomMetricUpdate.Example = "The following example updates a custom evaluation metric: " +
		"`doctl evaluation custom-metric update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name helpfulness-v2`"

	cmdCustomMetricDelete := CmdBuilder(
		cmd,
		RunCustomEvaluationMetricDelete,
		"delete <metric-uuid>",
		"Delete a custom evaluation metric",
		"Deletes a custom evaluation metric by its UUID.",
		Writer, aliasOpt("del", "rm"),
	)
	AddBoolFlag(cmdCustomMetricDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the custom metric without a confirmation prompt")
	cmdCustomMetricDelete.Example = "The following example deletes a custom evaluation metric: " +
		"`doctl evaluation custom-metric delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	return cmd
}

// RunCustomEvaluationMetricCreate creates a custom evaluation metric.
func RunCustomEvaluationMetricCreate(c *CmdConfig) error {
	metricName, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	scoringPrompt, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationScoringPrompt)
	if err != nil {
		return err
	}

	description, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationDescription)
	if err != nil {
		return err
	}

	requiresGroundTruth, err := c.Doit.GetBool(c.NS, doctl.ArgEvaluationRequiresGroundTruth)
	if err != nil {
		return err
	}

	metric, err := c.AgentPlatform().CreateCustomEvaluationMetric(&godo.CreateCustomEvaluationMetricRequest{
		MetricName:  metricName,
		Description: description,
		Config: &godo.CustomEvaluationMetricConfig{
			ScoringPrompt:       scoringPrompt,
			RequiresGroundTruth: requiresGroundTruth,
		},
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.EvaluationMetric{Metrics: do.EvaluationMetrics{*metric}})
}

// RunCustomEvaluationMetricUpdate updates a custom evaluation metric.
func RunCustomEvaluationMetricUpdate(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	metricName, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	scoringPrompt, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationScoringPrompt)
	if err != nil {
		return err
	}

	description, err := c.Doit.GetString(c.NS, doctl.ArgEvaluationDescription)
	if err != nil {
		return err
	}

	req := &godo.UpdateCustomEvaluationMetricRequest{
		MetricUUID:  c.Args[0],
		MetricName:  metricName,
		Description: description,
	}

	// The API requires config on every update. Reuse the existing scoring
	// config when the caller is only changing name/description.
	config, err := customEvaluationMetricConfigForUpdate(c, c.Args[0], scoringPrompt)
	if err != nil {
		return err
	}
	req.Config = config

	metric, err := c.AgentPlatform().UpdateCustomEvaluationMetric(c.Args[0], req)
	if err != nil {
		return err
	}

	return c.Display(&displayers.EvaluationMetric{Metrics: do.EvaluationMetrics{*metric}})
}

// customEvaluationMetricConfigForUpdate builds the config payload required by
// UpdateCustomEvaluationMetric. When --scoring-prompt is set, that value is
// used; otherwise the existing metric's config is loaded so name/description
// updates do not clear the prompt.
func customEvaluationMetricConfigForUpdate(c *CmdConfig, metricUUID, scoringPrompt string) (*godo.CustomEvaluationMetricConfig, error) {
	requiresGroundTruthSet := c.Doit.IsSet(doctl.ArgEvaluationRequiresGroundTruth)
	requiresGroundTruth := false
	if requiresGroundTruthSet {
		var err error
		requiresGroundTruth, err = c.Doit.GetBool(c.NS, doctl.ArgEvaluationRequiresGroundTruth)
		if err != nil {
			return nil, err
		}
	}

	if scoringPrompt != "" && requiresGroundTruthSet {
		return &godo.CustomEvaluationMetricConfig{
			ScoringPrompt:       scoringPrompt,
			RequiresGroundTruth: requiresGroundTruth,
		}, nil
	}

	existing, err := existingCustomEvaluationMetricConfig(c, metricUUID)
	if err != nil {
		return nil, err
	}

	if scoringPrompt != "" {
		existing.ScoringPrompt = scoringPrompt
	}
	if requiresGroundTruthSet {
		existing.RequiresGroundTruth = requiresGroundTruth
	}
	return existing, nil
}

func existingCustomEvaluationMetricConfig(c *CmdConfig, metricUUID string) (*godo.CustomEvaluationMetricConfig, error) {
	metrics, err := c.AgentPlatform().ListModelEvaluationMetrics()
	if err != nil {
		return nil, err
	}

	for _, metric := range metrics {
		if metric.EvaluationMetric == nil || metric.MetricUUID != metricUUID {
			continue
		}
		if metric.CustomEvalConfig == nil {
			return nil, fmt.Errorf("custom metric %s has no scoring config to reuse; pass --%s", metricUUID, doctl.ArgEvaluationScoringPrompt)
		}
		return &godo.CustomEvaluationMetricConfig{
			ScoringPrompt:       metric.CustomEvalConfig.ScoringPrompt,
			RequiresGroundTruth: metric.CustomEvalConfig.RequiresGroundTruth,
		}, nil
	}

	return nil, fmt.Errorf("custom metric %s not found", metricUUID)
}

// RunCustomEvaluationMetricDelete deletes a custom evaluation metric by its UUID.
func RunCustomEvaluationMetricDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirmDelete("custom evaluation metric", 1) != nil {
		return errOperationAborted
	}

	if err := c.AgentPlatform().DeleteCustomEvaluationMetric(c.Args[0]); err != nil {
		return err
	}

	notice("Custom evaluation metric deleted successfully")
	return nil
}
