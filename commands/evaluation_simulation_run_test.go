package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testSimulationRunUUID     = "33333333-3333-4333-8333-333333333333"
	testSimulationJourneyUUID = "44444444-4444-4444-8444-444444444444"

	testSimulationRun = do.SimulationRun{
		SimulationRun: &godo.SimulationRun{
			RunUUID:         testSimulationRunUUID,
			Name:            "Nightly Regression",
			ScenarioSetUUID: testScenarioSetUUID,
			Status:          godo.SimulationRunStatusRunning,
			AgentConfig: &godo.CandidateAgentConfig{
				AgentUUID: "agent-uuid",
			},
			TotalJourneys:    6,
			JourneysFinished: 2,
		},
	}

	testSimulationJourney = do.SimulationJourney{
		SimulationJourney: &godo.SimulationJourney{
			JourneyUUID:  testSimulationJourneyUUID,
			RunUUID:      testSimulationRunUUID,
			ScenarioUUID: testScenario.ScenarioUUID,
			JourneyIndex: 1,
			Status:       godo.SimulationJourneyStatusFinished,
			Verdict:      godo.SimulationJourneyVerdictSuccess,
			DurationSec:  "42",
		},
	}
)

func TestSimulationRunCommand(t *testing.T) {
	cmd := SimulationRunCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "get", "update", "cancel", "delete", "list-journeys", "get-journey", "get-trajectory", "get-trajectory-url")
}

func TestSimulationRunCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSimulationScenarioSetUUID, testScenarioSetUUID)
		config.Doit.Set(config.NS, doctl.ArgSimulationAgentUUID, "agent-uuid")
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Nightly Regression")
		config.Doit.Set(config.NS, doctl.ArgSimulationUserSimulatorModelUUID, "user-simulator-model-uuid")
		config.Doit.Set(config.NS, doctl.ArgSimulationJudgeModelUUID, "judge-model-uuid")
		config.Doit.Set(config.NS, doctl.ArgSimulationExplorationBudget, 3)
		config.Doit.Set(config.NS, doctl.ArgSimulationMaxTurns, 10)

		tm.gradientAI.EXPECT().CreateSimulationRun(&godo.CreateSimulationRunRequest{
			ScenarioSetUUID: testScenarioSetUUID,
			Name:            "Nightly Regression",
			AgentConfig: &godo.CandidateAgentConfig{
				AgentUUID: "agent-uuid",
			},
			UserSimulatorModelUUID: "user-simulator-model-uuid",
			JudgeModelUUID:         "judge-model-uuid",
			ExplorationBudget:      3,
			MaxTurns:               10,
		}).Return(&testSimulationRun, nil)

		err := RunSimulationRunCreate(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunCreateWithEvaluationConfig(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSimulationScenarioSetUUID, testScenarioSetUUID)
		config.Doit.Set(config.NS, doctl.ArgSimulationAgentUUID, "agent-uuid")
		config.Doit.Set(config.NS, doctl.ArgSimulationUserSimulatorConfig, `{"tone":"terse"}`)
		config.Doit.Set(config.NS, doctl.ArgSimulationMetricUUIDs, []string{"metric-uuid-1", "metric-uuid-2"})
		config.Doit.Set(config.NS, doctl.ArgSimulationStarMetricUUID, "metric-uuid-1")
		config.Doit.Set(config.NS, doctl.ArgSimulationStarMetricName, "Task Success")
		config.Doit.Set(config.NS, doctl.ArgSimulationStarMetricSuccessThreshold, 0.8)

		successThreshold := float32(0.8)
		tm.gradientAI.EXPECT().CreateSimulationRun(&godo.CreateSimulationRunRequest{
			ScenarioSetUUID: testScenarioSetUUID,
			AgentConfig: &godo.CandidateAgentConfig{
				AgentUUID: "agent-uuid",
			},
			UserSimulatorConfig: map[string]any{"tone": "terse"},
			EvaluationConfig: &godo.SimulationEvaluationConfig{
				MetricUUIDs: []string{"metric-uuid-1", "metric-uuid-2"},
				StarMetric: &godo.StarMetric{
					MetricUUID:       "metric-uuid-1",
					Name:             "Task Success",
					SuccessThreshold: &successThreshold,
				},
			},
		}).Return(&testSimulationRun, nil)

		err := RunSimulationRunCreate(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.gradientAI.EXPECT().ListSimulationRuns(&godo.SimulationRunListOptions{}).
			Return(do.SimulationRuns{testSimulationRun}, nil)

		err := RunSimulationRunList(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunListWithFilters(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSimulationScenarioSetUUID, testScenarioSetUUID)
		config.Doit.Set(config.NS, doctl.ArgGenAIStatuses, []string{"running", "partially_successful"})
		config.Doit.Set(config.NS, doctl.ArgGenAISortBy, "status")
		config.Doit.Set(config.NS, doctl.ArgGenAISortDirection, "asc")

		tm.gradientAI.EXPECT().ListSimulationRuns(&godo.SimulationRunListOptions{
			ScenarioSetUUID: testScenarioSetUUID,
			Statuses: []godo.SimulationRunStatus{
				godo.SimulationRunStatusRunning,
				godo.SimulationRunStatusPartiallySuccessful,
			},
			SortBy:        godo.SimulationRunSortFieldStatus,
			SortDirection: godo.GenAISortDirectionAsc,
		}).Return(do.SimulationRuns{testSimulationRun}, nil)

		err := RunSimulationRunList(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testSimulationRunUUID)

		tm.gradientAI.EXPECT().GetSimulationRun(testSimulationRunUUID).Return(&do.SimulationRunDetail{
			SimulationRunGetResponse: &godo.SimulationRunGetResponse{
				SimulationRun: testSimulationRun.SimulationRun,
				ScenarioResults: []*godo.SimulationScenarioResult{{
					ScenarioUUID:  testScenario.ScenarioUUID,
					TotalJourneys: 3,
				}},
			},
		}, nil)

		err := RunSimulationRunGet(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testSimulationRunUUID)
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Nightly Regression v2")

		tm.gradientAI.EXPECT().UpdateSimulationRun(testSimulationRunUUID, &godo.UpdateSimulationRunRequest{
			RunUUID: testSimulationRunUUID,
			Name:    "Nightly Regression v2",
		}).Return(&testSimulationRun, nil)

		err := RunSimulationRunUpdate(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunCancel(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testSimulationRunUUID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.gradientAI.EXPECT().CancelSimulationRun(testSimulationRunUUID).Return(&testSimulationRun, nil)

		err := RunSimulationRunCancel(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testSimulationRunUUID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.gradientAI.EXPECT().DeleteSimulationRun(testSimulationRunUUID).Return(nil)

		err := RunSimulationRunDelete(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunListJourneys(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testSimulationRunUUID)
		config.Doit.Set(config.NS, doctl.ArgSimulationScenarioUUID, testScenario.ScenarioUUID)
		config.Doit.Set(config.NS, doctl.ArgGenAIStatuses, []string{"finished"})
		config.Doit.Set(config.NS, doctl.ArgSimulationJourneyVerdicts, []string{"failure"})

		tm.gradientAI.EXPECT().ListSimulationJourneys(testSimulationRunUUID, &godo.SimulationJourneyListOptions{
			ScenarioUUID: testScenario.ScenarioUUID,
			Statuses:     []godo.SimulationJourneyStatus{godo.SimulationJourneyStatusFinished},
			Verdicts:     []godo.SimulationJourneyVerdict{godo.SimulationJourneyVerdictFailure},
		}).Return(do.SimulationJourneys{testSimulationJourney}, nil)

		err := RunSimulationRunListJourneys(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunGetJourney(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testSimulationRunUUID, testSimulationJourneyUUID)

		tm.gradientAI.EXPECT().GetSimulationJourney(testSimulationRunUUID, testSimulationJourneyUUID).
			Return(&testSimulationJourney, nil)

		err := RunSimulationRunGetJourney(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunGetJourneyRequiresTwoArgs(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testSimulationRunUUID)

		err := RunSimulationRunGetJourney(config)
		assert.Error(t, err)
	})
}

func TestSimulationRunGetJourneyTrajectory(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testSimulationRunUUID, testSimulationJourneyUUID)

		tm.gradientAI.EXPECT().GetSimulationJourneyTrajectory(testSimulationRunUUID, testSimulationJourneyUUID).
			Return(&do.SimulationTrajectory{
				SimulationTrajectory: &godo.SimulationTrajectory{
					JourneyUUID: testSimulationJourneyUUID,
					RunUUID:     testSimulationRunUUID,
					Status:      godo.SimulationTrajectoryStatusCompleted,
					Verdict:     godo.SimulationJourneyVerdictSuccess,
					TurnCount:   4,
				},
			}, nil)

		err := RunSimulationRunGetJourneyTrajectory(config)
		assert.NoError(t, err)
	})
}

func TestSimulationRunGetJourneyTrajectoryURL(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testSimulationRunUUID, testSimulationJourneyUUID)

		tm.gradientAI.EXPECT().GetSimulationJourneyTrajectoryURL(testSimulationRunUUID, testSimulationJourneyUUID).
			Return(&do.GenAIDownloadURL{DownloadURL: "https://example.com/trajectory"}, nil)

		err := RunSimulationRunGetJourneyTrajectoryURL(config)
		assert.NoError(t, err)
	})
}
