package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestCustomEvaluationMetricCommand(t *testing.T) {
	cmd := CustomEvaluationMetricCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "update", "delete")
}

func TestCustomEvaluationMetricCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "helpfulness")
		config.Doit.Set(config.NS, doctl.ArgEvaluationScoringPrompt, "Score helpfulness from 0 to 1")
		config.Doit.Set(config.NS, doctl.ArgEvaluationDescription, "Measures response helpfulness")
		config.Doit.Set(config.NS, doctl.ArgEvaluationRequiresGroundTruth, true)

		tm.agentPlatform.EXPECT().CreateCustomEvaluationMetric(&godo.CreateCustomEvaluationMetricRequest{
			MetricName:  "helpfulness",
			Description: "Measures response helpfulness",
			Config: &godo.CustomEvaluationMetricConfig{
				ScoringPrompt:       "Score helpfulness from 0 to 1",
				RequiresGroundTruth: true,
			},
		}).Return(&testEvaluationMetric, nil)

		err := RunCustomEvaluationMetricCreate(config)
		assert.NoError(t, err)
	})
}

func TestCustomEvaluationMetricUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testEvaluationMetricUUID)
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "helpfulness-v2")
		config.Doit.Set(config.NS, doctl.ArgEvaluationDescription, "updated by doctl")

		existing := do.EvaluationMetric{
			EvaluationMetric: &godo.EvaluationMetric{
				MetricUUID: testEvaluationMetricUUID,
				MetricName: "helpfulness",
				Source:     godo.EvaluationMetricSourceCustom,
				CustomEvalConfig: &godo.CustomEvaluationMetricConfig{
					ScoringPrompt:       "Score helpfulness from 0 to 1",
					RequiresGroundTruth: false,
				},
			},
		}
		tm.agentPlatform.EXPECT().ListModelEvaluationMetrics().
			Return(do.EvaluationMetrics{existing}, nil)
		tm.agentPlatform.EXPECT().UpdateCustomEvaluationMetric(testEvaluationMetricUUID, &godo.UpdateCustomEvaluationMetricRequest{
			MetricUUID:  testEvaluationMetricUUID,
			MetricName:  "helpfulness-v2",
			Description: "updated by doctl",
			Config: &godo.CustomEvaluationMetricConfig{
				ScoringPrompt:       "Score helpfulness from 0 to 1",
				RequiresGroundTruth: false,
			},
		}).Return(&existing, nil)

		err := RunCustomEvaluationMetricUpdate(config)
		assert.NoError(t, err)
	})
}

func TestCustomEvaluationMetricDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testEvaluationMetricUUID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.agentPlatform.EXPECT().DeleteCustomEvaluationMetric(testEvaluationMetricUUID).Return(nil)

		err := RunCustomEvaluationMetricDelete(config)
		assert.NoError(t, err)
	})
}
