package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testModelEvaluationPresetUUID = "77777777-7777-4777-8777-777777777777"

	testModelEvaluationPreset = do.ModelEvaluationPreset{
		ModelEvaluationPreset: &godo.ModelEvaluationPreset{
			EvalPresetUuid:     testModelEvaluationPresetUUID,
			Name:               "default-preset",
			CandidateModelName: "llama-3",
			DatasetName:        "support-prompts",
			JudgeModelName:     "judge-model",
		},
	}
)

func TestModelEvaluationPresetCommand(t *testing.T) {
	cmd := ModelEvaluationPresetCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list", "get", "delete")
}

func TestModelEvaluationPresetList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.gradientAI.EXPECT().ListModelEvaluationPresets().
			Return(do.ModelEvaluationPresets{testModelEvaluationPreset}, nil)

		err := RunModelEvaluationPresetList(config)
		assert.NoError(t, err)
	})
}

func TestModelEvaluationPresetGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testModelEvaluationPresetUUID)

		tm.gradientAI.EXPECT().GetModelEvaluationPreset(testModelEvaluationPresetUUID).
			Return(&testModelEvaluationPreset, nil)

		err := RunModelEvaluationPresetGet(config)
		assert.NoError(t, err)
	})
}

func TestModelEvaluationPresetDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testModelEvaluationPresetUUID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.gradientAI.EXPECT().DeleteModelEvaluationPreset(testModelEvaluationPresetUUID).Return(nil)

		err := RunModelEvaluationPresetDelete(config)
		assert.NoError(t, err)
	})
}
