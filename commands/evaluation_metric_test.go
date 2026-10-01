package commands

import (
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testEvaluationMetricUUID = "88888888-8888-4888-8888-888888888888"

	testEvaluationMetric = do.EvaluationMetric{
		EvaluationMetric: &godo.EvaluationMetric{
			MetricUUID: testEvaluationMetricUUID,
			MetricName: "Task Success",
			Source:     godo.EvaluationMetricSourceBuiltin,
			Category:   godo.MetricCategoryCorrectness,
		},
	}
)

func TestModelEvaluationMetricCommand(t *testing.T) {
	cmd := ModelEvaluationMetricCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list")
}

func TestModelEvaluationMetricList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.agentPlatform.EXPECT().ListModelEvaluationMetrics().
			Return(do.EvaluationMetrics{testEvaluationMetric}, nil)

		err := RunModelEvaluationMetricList(config)
		assert.NoError(t, err)
	})
}
