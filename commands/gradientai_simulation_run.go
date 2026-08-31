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

// SimulationRunCmd handles operations on simulation runs and their journeys.
func SimulationRunCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "simulation-run",
			Aliases: []string{"sim", "simulation-runs"},
			Short:   "Display commands that manage Gradient AI simulation runs.",
			Long:    "The subcommands of `doctl gradient simulation-run` run scenario sets against an agent and inspect the resulting journeys.",
		},
	}

	simulationRunDetails := `
		- The simulation run UUID
		- The simulation run name
		- The simulation run status
		- The UUID of the scenario set being run
		- The UUID of the candidate agent under test
		- The total number of journeys and how many have finished
		- The simulation run creation timestamp
	`

	cmdSimulationRunCreate := CmdBuilder(
		cmd,
		RunSimulationRunCreate,
		"create",
		"Create a simulation run",
		"Starts a simulation of a scenario set against an agent and returns:"+simulationRunDetails,
		Writer, aliasOpt("c"),
		displayerType(&displayers.SimulationRun{}),
	)
	AddStringFlag(cmdSimulationRunCreate, doctl.ArgSimulationScenarioSetUUID, "", "", "The UUID of the scenario set to run.", requiredOpt())
	AddStringFlag(cmdSimulationRunCreate, doctl.ArgSimulationAgentUUID, "", "", "The UUID of the agent under test.", requiredOpt())
	AddStringFlag(cmdSimulationRunCreate, doctl.ArgGenAIName, "", "", "The name of the simulation run.")
	AddStringFlag(cmdSimulationRunCreate, doctl.ArgSimulationAgentDeploymentUUID, "", "", "The deployment UUID of the agent under test.")
	AddStringFlag(cmdSimulationRunCreate, doctl.ArgSimulationAgentName, "", "", "The display name of the agent under test.")
	AddStringFlag(cmdSimulationRunCreate, doctl.ArgSimulationUserSimulatorModelUUID, "", "", "The UUID of the model that simulates the user.")
	AddStringFlag(cmdSimulationRunCreate, doctl.ArgSimulationJudgeModelUUID, "", "", "The UUID of the model that judges each journey.")
	AddStringFlag(cmdSimulationRunCreate, doctl.ArgSimulationUserSimulatorConfig, "", "", "A JSON object of user simulator settings.")
	AddIntFlag(cmdSimulationRunCreate, doctl.ArgSimulationExplorationBudget, "", 0, "The number of journeys to run per scenario.")
	AddIntFlag(cmdSimulationRunCreate, doctl.ArgSimulationMaxTurns, "", 0, "The maximum number of conversation turns per journey.")
	AddStringSliceFlag(cmdSimulationRunCreate, doctl.ArgSimulationMetricUUIDs, "", []string{}, "The UUIDs of evaluation metrics to score the run with.")
	AddStringFlag(cmdSimulationRunCreate, doctl.ArgSimulationStarMetricUUID, "", "", "The UUID of the star metric for the run.")
	AddStringFlag(cmdSimulationRunCreate, doctl.ArgSimulationStarMetricName, "", "", "The name of the star metric for the run.")
	AddFloatFlag(cmdSimulationRunCreate, doctl.ArgSimulationStarMetricSuccessThreshold, "", 0, "The success threshold of the star metric.")
	cmdSimulationRunCreate.Example = "The following example runs a scenario set against an agent: " +
		"`doctl gradient simulation-run create --scenario-set-uuid f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --agent-uuid 99a1cbc7-b1b2-4a0d-9c1f-9b9d2b8f9d1e --name nightly-regression`"

	cmdSimulationRunList := CmdBuilder(
		cmd,
		RunSimulationRunList,
		"list",
		"List all simulation runs",
		"Retrieves a list of simulation runs, where each run contains:"+simulationRunDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.SimulationRun{}),
	)
	AddStringFlag(cmdSimulationRunList, doctl.ArgSimulationScenarioSetUUID, "", "", "Filters the results by scenario set UUID.")
	AddStringSliceFlag(cmdSimulationRunList, doctl.ArgGenAIStatuses, "", []string{}, "Filters the results by status. One of: `pending`, `running`, `evaluating`, `succeeded`, `partially_successful`, `failed`, `cancelled`")
	addGenAIListFlags(cmdSimulationRunList, "`created_at`, `name`, `status`, `updated_at`")
	cmdSimulationRunList.Example = "The following example lists the simulation runs that are still running: " +
		"`doctl gradient simulation-run list --statuses running`"

	cmdSimulationRunGet := CmdBuilder(
		cmd,
		RunSimulationRunGet,
		"get <run-uuid>",
		"Retrieve a simulation run",
		"Retrieves a simulation run along with its verdict counts. Use `-o json` to also see the per-scenario results.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.SimulationRunDetail{}),
	)
	cmdSimulationRunGet.Example = "The following example retrieves a simulation run: " +
		"`doctl gradient simulation-run get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdSimulationRunUpdate := CmdBuilder(
		cmd,
		RunSimulationRunUpdate,
		"update <run-uuid>",
		"Update a simulation run",
		"Renames a simulation run.",
		Writer, aliasOpt("u"),
		displayerType(&displayers.SimulationRun{}),
	)
	AddStringFlag(cmdSimulationRunUpdate, doctl.ArgGenAIName, "", "", "The new name of the simulation run.", requiredOpt())
	cmdSimulationRunUpdate.Example = "The following example renames a simulation run: " +
		"`doctl gradient simulation-run update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name nightly-regression-v2`"

	cmdSimulationRunCancel := CmdBuilder(
		cmd,
		RunSimulationRunCancel,
		"cancel <run-uuid>",
		"Cancel a simulation run",
		"Cancels a simulation run that is still in progress.",
		Writer,
		displayerType(&displayers.SimulationRun{}),
	)
	AddBoolFlag(cmdSimulationRunCancel, doctl.ArgForce, doctl.ArgShortForce, false, "Cancels the simulation run without a confirmation prompt")
	cmdSimulationRunCancel.Example = "The following example cancels a simulation run: " +
		"`doctl gradient simulation-run cancel f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdSimulationRunDelete := CmdBuilder(
		cmd,
		RunSimulationRunDelete,
		"delete <run-uuid>",
		"Delete a simulation run",
		"Deletes a simulation run by its UUID.",
		Writer, aliasOpt("del", "rm"),
	)
	AddBoolFlag(cmdSimulationRunDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the simulation run without a confirmation prompt")
	cmdSimulationRunDelete.Example = "The following example deletes a simulation run: " +
		"`doctl gradient simulation-run delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	journeyDetails := `
		- The journey UUID
		- The UUID of the scenario the journey exercises
		- The journey index within the scenario
		- The journey status and judge verdict
		- The journey duration in seconds
		- The journey creation timestamp
	`

	cmdSimulationRunListJourneys := CmdBuilder(
		cmd,
		RunSimulationRunListJourneys,
		"list-journeys <run-uuid>",
		"List the journeys in a simulation run",
		"Retrieves the journeys of a simulation run, where each journey contains:"+journeyDetails,
		Writer, aliasOpt("ls-j"),
		displayerType(&displayers.SimulationJourney{}),
	)
	AddStringFlag(cmdSimulationRunListJourneys, doctl.ArgSimulationScenarioUUID, "", "", "Filters the results by scenario UUID.")
	AddStringSliceFlag(cmdSimulationRunListJourneys, doctl.ArgGenAIStatuses, "", []string{}, "Filters the results by status. One of: `preparing`, `running`, `finished`, `failed`")
	AddStringSliceFlag(cmdSimulationRunListJourneys, doctl.ArgSimulationJourneyVerdicts, "", []string{}, "Filters the results by verdict. One of: `success`, `failure`, `inconclusive`")
	addGenAIListFlags(cmdSimulationRunListJourneys, "`scenario`, `created_at`, `status`, `verdict`")
	cmdSimulationRunListJourneys.Example = "The following example lists the failed journeys of a simulation run: " +
		"`doctl gradient simulation-run list-journeys f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --verdicts failure`"

	cmdSimulationRunGetJourney := CmdBuilder(
		cmd,
		RunSimulationRunGetJourney,
		"get-journey <run-uuid> <journey-uuid>",
		"Retrieve a journey in a simulation run",
		"Retrieves a single journey of a simulation run, including:"+journeyDetails,
		Writer, aliasOpt("g-j"),
		displayerType(&displayers.SimulationJourney{}),
	)
	cmdSimulationRunGetJourney.Example = "The following example retrieves a journey: " +
		"`doctl gradient simulation-run get-journey f81d4fae-7dec-11d0-a765-00a0c91e6bf6 6ba7b810-9dad-11d1-80b4-00c04fd430c8`"

	cmdSimulationRunGetTrajectory := CmdBuilder(
		cmd,
		RunSimulationRunGetJourneyTrajectory,
		"get-trajectory <run-uuid> <journey-uuid>",
		"Retrieve the trajectory of a journey",
		"Retrieves the trajectory of a journey, which records the full conversation, the tool calls the agent made, and the judge's reasoning. Use `-o json` to see the messages and evaluation metrics.",
		Writer, aliasOpt("g-t"),
		displayerType(&displayers.SimulationTrajectory{}),
	)
	cmdSimulationRunGetTrajectory.Example = "The following example retrieves the trajectory of a journey: " +
		"`doctl gradient simulation-run get-trajectory f81d4fae-7dec-11d0-a765-00a0c91e6bf6 6ba7b810-9dad-11d1-80b4-00c04fd430c8 -o json`"

	cmdSimulationRunGetTrajectoryURL := CmdBuilder(
		cmd,
		RunSimulationRunGetJourneyTrajectoryURL,
		"get-trajectory-url <run-uuid> <journey-uuid>",
		"Retrieve a download URL for a journey trajectory",
		"Retrieves a temporary presigned URL for downloading the trajectory file of a journey.",
		Writer, aliasOpt("g-t-url"),
		displayerType(&displayers.GenAIDownloadURL{}),
	)
	cmdSimulationRunGetTrajectoryURL.Example = "The following example retrieves a download URL for a journey trajectory: " +
		"`doctl gradient simulation-run get-trajectory-url f81d4fae-7dec-11d0-a765-00a0c91e6bf6 6ba7b810-9dad-11d1-80b4-00c04fd430c8`"

	return cmd
}

// RunSimulationRunCreate creates a simulation run.
func RunSimulationRunCreate(c *CmdConfig) error {
	scenarioSetUUID, err := c.Doit.GetString(c.NS, doctl.ArgSimulationScenarioSetUUID)
	if err != nil {
		return err
	}

	agentUUID, err := c.Doit.GetString(c.NS, doctl.ArgSimulationAgentUUID)
	if err != nil {
		return err
	}

	agentDeploymentUUID, err := c.Doit.GetString(c.NS, doctl.ArgSimulationAgentDeploymentUUID)
	if err != nil {
		return err
	}

	agentName, err := c.Doit.GetString(c.NS, doctl.ArgSimulationAgentName)
	if err != nil {
		return err
	}

	name, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	userSimulatorModelUUID, err := c.Doit.GetString(c.NS, doctl.ArgSimulationUserSimulatorModelUUID)
	if err != nil {
		return err
	}

	judgeModelUUID, err := c.Doit.GetString(c.NS, doctl.ArgSimulationJudgeModelUUID)
	if err != nil {
		return err
	}

	rawUserSimulatorConfig, err := c.Doit.GetString(c.NS, doctl.ArgSimulationUserSimulatorConfig)
	if err != nil {
		return err
	}

	explorationBudget, err := c.Doit.GetInt(c.NS, doctl.ArgSimulationExplorationBudget)
	if err != nil {
		return err
	}

	maxTurns, err := c.Doit.GetInt(c.NS, doctl.ArgSimulationMaxTurns)
	if err != nil {
		return err
	}

	req := &godo.CreateSimulationRunRequest{
		ScenarioSetUUID: scenarioSetUUID,
		Name:            name,
		AgentConfig: &godo.CandidateAgentConfig{
			AgentUUID:           agentUUID,
			AgentDeploymentUUID: agentDeploymentUUID,
			Name:                agentName,
		},
		UserSimulatorModelUUID: userSimulatorModelUUID,
		JudgeModelUUID:         judgeModelUUID,
		ExplorationBudget:      uint32(explorationBudget),
		MaxTurns:               uint32(maxTurns),
	}

	if rawUserSimulatorConfig != "" {
		var userSimulatorConfig map[string]any
		if err := json.Unmarshal([]byte(rawUserSimulatorConfig), &userSimulatorConfig); err != nil {
			return fmt.Errorf("unable to parse user simulator config: %w", err)
		}
		req.UserSimulatorConfig = userSimulatorConfig
	}

	evaluationConfig, err := simulationEvaluationConfig(c)
	if err != nil {
		return err
	}
	req.EvaluationConfig = evaluationConfig

	run, err := c.GradientAI().CreateSimulationRun(req)
	if err != nil {
		return err
	}

	return c.Display(&displayers.SimulationRun{SimulationRuns: do.SimulationRuns{*run}})
}

// RunSimulationRunList lists simulation runs.
func RunSimulationRunList(c *CmdConfig) error {
	scenarioSetUUID, err := c.Doit.GetString(c.NS, doctl.ArgSimulationScenarioSetUUID)
	if err != nil {
		return err
	}

	rawStatuses, err := c.Doit.GetStringSlice(c.NS, doctl.ArgGenAIStatuses)
	if err != nil {
		return err
	}

	search, sortBy, sortDirection, err := genAIListFilters(c, simulationRunSortFieldPrefix)
	if err != nil {
		return err
	}

	runs, err := c.GradientAI().ListSimulationRuns(&godo.SimulationRunListOptions{
		ScenarioSetUUID: scenarioSetUUID,
		Statuses:        genAIEnums[godo.SimulationRunStatus](simulationRunStatusPrefix, rawStatuses),
		Search:          search,
		SortBy:          godo.SimulationRunSortField(sortBy),
		SortDirection:   godo.GenAISortDirection(sortDirection),
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.SimulationRun{SimulationRuns: runs})
}

// RunSimulationRunGet retrieves a simulation run by its UUID.
func RunSimulationRunGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	detail, err := c.GradientAI().GetSimulationRun(c.Args[0])
	if err != nil {
		return err
	}

	return c.Display(&displayers.SimulationRunDetail{Detail: detail})
}

// RunSimulationRunUpdate updates a simulation run.
func RunSimulationRunUpdate(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	name, err := c.Doit.GetString(c.NS, doctl.ArgGenAIName)
	if err != nil {
		return err
	}

	runUUID := c.Args[0]
	run, err := c.GradientAI().UpdateSimulationRun(runUUID, &godo.UpdateSimulationRunRequest{
		RunUUID: runUUID,
		Name:    name,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.SimulationRun{SimulationRuns: do.SimulationRuns{*run}})
}

// RunSimulationRunCancel cancels an in-progress simulation run.
func RunSimulationRunCancel(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirm("cancel this simulation run") != nil {
		return errOperationAborted
	}

	run, err := c.GradientAI().CancelSimulationRun(c.Args[0])
	if err != nil {
		return err
	}

	return c.Display(&displayers.SimulationRun{SimulationRuns: do.SimulationRuns{*run}})
}

// RunSimulationRunDelete deletes a simulation run by its UUID.
func RunSimulationRunDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if !force && AskForConfirmDelete("simulation run", 1) != nil {
		return errOperationAborted
	}

	if err := c.GradientAI().DeleteSimulationRun(c.Args[0]); err != nil {
		return err
	}

	notice("Simulation run deleted successfully")
	return nil
}

// RunSimulationRunListJourneys lists the journeys of a simulation run.
func RunSimulationRunListJourneys(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}

	scenarioUUID, err := c.Doit.GetString(c.NS, doctl.ArgSimulationScenarioUUID)
	if err != nil {
		return err
	}

	rawStatuses, err := c.Doit.GetStringSlice(c.NS, doctl.ArgGenAIStatuses)
	if err != nil {
		return err
	}

	rawVerdicts, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSimulationJourneyVerdicts)
	if err != nil {
		return err
	}

	search, sortBy, sortDirection, err := genAIListFilters(c, journeySortFieldPrefix)
	if err != nil {
		return err
	}

	journeys, err := c.GradientAI().ListSimulationJourneys(c.Args[0], &godo.SimulationJourneyListOptions{
		ScenarioUUID:  scenarioUUID,
		Statuses:      genAIEnums[godo.SimulationJourneyStatus](journeyStatusPrefix, rawStatuses),
		Verdicts:      genAIEnums[godo.SimulationJourneyVerdict](journeyVerdictPrefix, rawVerdicts),
		Search:        search,
		SortBy:        godo.SimulationJourneySortField(sortBy),
		SortDirection: godo.GenAISortDirection(sortDirection),
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.SimulationJourney{Journeys: journeys})
}

// RunSimulationRunGetJourney retrieves a single journey of a simulation run.
func RunSimulationRunGetJourney(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	journey, err := c.GradientAI().GetSimulationJourney(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}

	return c.Display(&displayers.SimulationJourney{Journeys: do.SimulationJourneys{*journey}})
}

// RunSimulationRunGetJourneyTrajectory retrieves the trajectory of a journey.
func RunSimulationRunGetJourneyTrajectory(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	trajectory, err := c.GradientAI().GetSimulationJourneyTrajectory(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}

	return c.Display(&displayers.SimulationTrajectory{Trajectory: trajectory})
}

// RunSimulationRunGetJourneyTrajectoryURL retrieves a download URL for a journey trajectory.
func RunSimulationRunGetJourneyTrajectoryURL(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	trajectoryURL, err := c.GradientAI().GetSimulationJourneyTrajectoryURL(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}

	return c.Display(&displayers.GenAIDownloadURL{DownloadURL: trajectoryURL})
}

// simulationEvaluationConfig builds the optional evaluation config for a run.
func simulationEvaluationConfig(c *CmdConfig) (*godo.SimulationEvaluationConfig, error) {
	metricUUIDs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSimulationMetricUUIDs)
	if err != nil {
		return nil, err
	}

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

	var starMetric *godo.StarMetric
	if starMetricUUID != "" || starMetricName != "" || starMetricSuccessThreshold != 0 {
		starMetric = &godo.StarMetric{
			MetricUUID: starMetricUUID,
			Name:       starMetricName,
		}
		if starMetricSuccessThreshold != 0 {
			threshold := float32(starMetricSuccessThreshold)
			starMetric.SuccessThreshold = &threshold
		}
	}

	if len(metricUUIDs) == 0 && starMetric == nil {
		return nil, nil
	}

	return &godo.SimulationEvaluationConfig{
		MetricUUIDs: metricUUIDs,
		StarMetric:  starMetric,
	}, nil
}
