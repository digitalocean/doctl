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
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/spf13/cobra"
)

// ModelEvaluationMetricCmd handles operations on model evaluation metrics.
func ModelEvaluationMetricCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "metric",
			Aliases: []string{"metrics"},
			Short:   "Display commands that list GenAI model evaluation metrics.",
			Long:    "The subcommands of `doctl evaluation metric` list the metrics available for model evaluation runs.",
		},
	}

	cmdMetricList := CmdBuilder(
		cmd,
		RunModelEvaluationMetricList,
		"list",
		"List all model evaluation metrics",
		"Retrieves a list of model evaluation metrics that can be selected when creating a model evaluation run.",
		Writer, aliasOpt("ls"),
		displayerType(&displayers.EvaluationMetric{}),
	)
	cmdMetricList.Example = "The following example lists model evaluation metrics: " +
		"`doctl evaluation metric list`"

	return cmd
}

// RunModelEvaluationMetricList lists model evaluation metrics.
func RunModelEvaluationMetricList(c *CmdConfig) error {
	metrics, err := c.GradientAI().ListModelEvaluationMetrics()
	if err != nil {
		return err
	}

	return c.Display(&displayers.EvaluationMetric{Metrics: metrics})
}
