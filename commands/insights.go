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

import "github.com/spf13/cobra"

// Insights creates the top-level command group for Insights.
func Insights() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "insights",
			Short:   "Display commands for managing Insights",
			Long:    "The subcommands of `doctl insights` manage alerting and resource signal enablement, and query observability data.",
			GroupID: manageResourcesGroup,
		},
	}

	cmd.AddCommand(InsightsNotificationChannel())
	cmd.AddCommand(InsightsAlertRule())
	cmd.AddCommand(insightsAlertInstances())
	cmd.AddCommand(insightsMetrics())
	cmd.AddCommand(insightsLogs())
	cmd.AddCommand(insightsSpans())
	cmd.AddCommand(insightsTraces())
	cmd.AddCommand(insightsResources())

	return cmd
}
