package commands

import (
	"os"
	"testing"

	"github.com/digitalocean/godo"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
)

// Test data
var (
	testDedicatedInferenceSpecRequest = &godo.DedicatedInferenceSpecRequest{
		Version: 0,
		Name:    "test-dedicated-inference",
		Region:  "nyc2",
		VPC: &godo.DedicatedInferenceVPCRequest{
			UUID: "00000000-0000-4000-8000-000000000001",
		},
		EnablePublicEndpoint: true,
		ModelDeployments: []*godo.DedicatedInferenceModelRequest{
			{
				ModelSlug:     "mistral/mistral-7b-instruct-v3",
				ModelProvider: "hugging_face",
				Accelerators: []*godo.DedicatedInferenceAcceleratorRequest{
					{
						Scale:           2,
						Type:            "prefill",
						AcceleratorSlug: "gpu-mi300x1-192gb",
					},
					{
						Scale:           4,
						Type:            "decode",
						AcceleratorSlug: "gpu-mi300x1-192gb",
					},
				},
			},
		},
	}

	testDedicatedInference = do.DedicatedInference{
		DedicatedInference: &godo.DedicatedInference{
			ID:      "00000000-0000-4000-8000-000000000000",
			Name:    "test-dedicated-inference",
			Status:  "PROVISIONING",
			Region:  "nyc2",
			VPCUUID: "00000000-0000-4000-8000-000000000001",
		},
	}

	testDedicatedInferenceToken = &do.DedicatedInferenceToken{
		DedicatedInferenceToken: &godo.DedicatedInferenceToken{
			ID:    "tok-1",
			Name:  "default",
			Value: "secret-token-value",
		},
	}
)

func TestDedicatedInferenceCommand(t *testing.T) {
	cmd := DedicatedInferenceCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "dedicated-inference", cmd.Name())

	// Verify subcommands
	subcommands := make(map[string]bool)
	for _, c := range cmd.Commands() {
		subcommands[c.Name()] = true
	}
	assert.True(t, subcommands["create"], "Expected create subcommand")
	assert.True(t, subcommands["get"], "Expected get subcommand")
	assert.True(t, subcommands["update"], "Expected update subcommand")
	assert.True(t, subcommands["delete"], "Expected delete subcommand")
	assert.True(t, subcommands["list-accelerators"], "Expected list-accelerators subcommand")
}

func TestRunDedicatedInferenceCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// Write a temp spec file
		specJSON := `{
			"version": 0,
			"name": "test-dedicated-inference",
			"region": "nyc2",
			"vpc": {"uuid": "00000000-0000-4000-8000-000000000001"},
			"enable_public_endpoint": true,
			"model_deployments": [
				{
					"model_slug": "mistral/mistral-7b-instruct-v3",
					"model_provider": "hugging_face",
					"accelerators": [
						{"scale": 2, "type": "prefill", "accelerator_slug": "gpu-mi300x1-192gb"},
						{"scale": 4, "type": "decode", "accelerator_slug": "gpu-mi300x1-192gb"}
					]
				}
			]
		}`
		tmpFile := t.TempDir() + "/spec.json"
		err := os.WriteFile(tmpFile, []byte(specJSON), 0644)
		assert.NoError(t, err)

		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceSpec, tmpFile)

		expectedReq := &godo.DedicatedInferenceCreateRequest{
			Spec: testDedicatedInferenceSpecRequest,
		}

		tm.dedicatedInferences.EXPECT().Create(expectedReq).Return(&testDedicatedInference, testDedicatedInferenceToken, nil)

		err = RunDedicatedInferenceCreate(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceCreate_WithHuggingFaceToken(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		specJSON := `{
			"version": 0,
			"name": "test-dedicated-inference",
			"region": "nyc2",
			"vpc": {"uuid": "00000000-0000-4000-8000-000000000001"},
			"enable_public_endpoint": true,
			"model_deployments": [
				{
					"model_slug": "mistral/mistral-7b-instruct-v3",
					"model_provider": "hugging_face",
					"accelerators": [
						{"scale": 2, "type": "prefill", "accelerator_slug": "gpu-mi300x1-192gb"},
						{"scale": 4, "type": "decode", "accelerator_slug": "gpu-mi300x1-192gb"}
					]
				}
			]
		}`
		tmpFile := t.TempDir() + "/spec.json"
		err := os.WriteFile(tmpFile, []byte(specJSON), 0644)
		assert.NoError(t, err)

		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceSpec, tmpFile)
		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceHuggingFaceToken, "hf_test_token")

		expectedReq := &godo.DedicatedInferenceCreateRequest{
			Spec: testDedicatedInferenceSpecRequest,
			Secrets: &godo.DedicatedInferenceSecrets{
				HuggingFaceToken: "hf_test_token",
			},
		}

		tm.dedicatedInferences.EXPECT().Create(expectedReq).Return(&testDedicatedInference, testDedicatedInferenceToken, nil)

		err = RunDedicatedInferenceCreate(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dedicatedInferences.EXPECT().Get("00000000-0000-4000-8000-000000000000").Return(&testDedicatedInference, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		err := RunDedicatedInferenceGet(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceGet_MissingID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDedicatedInferenceGet(config)
		assert.Error(t, err)
	})
}

func TestRunDedicatedInferenceDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dedicatedInferences.EXPECT().Delete("00000000-0000-4000-8000-000000000000").Return(nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunDedicatedInferenceDelete(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceDelete_MissingID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDedicatedInferenceDelete(config)
		assert.Error(t, err)
	})
}

func TestRunDedicatedInferenceUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		specJSON := `{
			"version": 0,
			"name": "test-dedicated-inference",
			"region": "nyc2",
			"vpc": {"uuid": "00000000-0000-4000-8000-000000000001"},
			"enable_public_endpoint": true,
			"model_deployments": [
				{
					"model_slug": "mistral/mistral-7b-instruct-v3",
					"model_provider": "hugging_face",
					"accelerators": [
						{"scale": 2, "type": "prefill", "accelerator_slug": "gpu-mi300x1-192gb"},
						{"scale": 4, "type": "decode", "accelerator_slug": "gpu-mi300x1-192gb"}
					]
				}
			]
		}`
		tmpFile := t.TempDir() + "/spec.json"
		err := os.WriteFile(tmpFile, []byte(specJSON), 0644)
		assert.NoError(t, err)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")
		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceSpec, tmpFile)

		expectedReq := &godo.DedicatedInferenceUpdateRequest{
			Spec: testDedicatedInferenceSpecRequest,
		}

		updatedDI := testDedicatedInference
		tm.dedicatedInferences.EXPECT().Update("00000000-0000-4000-8000-000000000000", expectedReq).Return(&updatedDI, nil)

		err = RunDedicatedInferenceUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceUpdate_WithHuggingFaceToken(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		specJSON := `{
			"version": 0,
			"name": "test-dedicated-inference",
			"region": "nyc2",
			"vpc": {"uuid": "00000000-0000-4000-8000-000000000001"},
			"enable_public_endpoint": true,
			"model_deployments": [
				{
					"model_slug": "mistral/mistral-7b-instruct-v3",
					"model_provider": "hugging_face",
					"accelerators": [
						{"scale": 2, "type": "prefill", "accelerator_slug": "gpu-mi300x1-192gb"},
						{"scale": 4, "type": "decode", "accelerator_slug": "gpu-mi300x1-192gb"}
					]
				}
			]
		}`
		tmpFile := t.TempDir() + "/spec.json"
		err := os.WriteFile(tmpFile, []byte(specJSON), 0644)
		assert.NoError(t, err)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")
		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceSpec, tmpFile)
		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceHuggingFaceToken, "hf_test_token")

		expectedReq := &godo.DedicatedInferenceUpdateRequest{
			Spec: testDedicatedInferenceSpecRequest,
			Secrets: &godo.DedicatedInferenceSecrets{
				HuggingFaceToken: "hf_test_token",
			},
		}

		updatedDI := testDedicatedInference
		tm.dedicatedInferences.EXPECT().Update("00000000-0000-4000-8000-000000000000", expectedReq).Return(&updatedDI, nil)

		err = RunDedicatedInferenceUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceUpdate_MissingID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDedicatedInferenceUpdate(config)
		assert.Error(t, err)
	})
}

func TestRunDedicatedInferenceListAccelerators(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		testAccelerators := do.DedicatedInferenceAcceleratorInfos{
			{
				DedicatedInferenceAcceleratorInfo: &godo.DedicatedInferenceAcceleratorInfo{
					ID:     "accel-1",
					Name:   "gpu-mi300x1-192gb",
					Slug:   "gpu-mi300x1-192gb",
					Status: "active",
				},
			},
			{
				DedicatedInferenceAcceleratorInfo: &godo.DedicatedInferenceAcceleratorInfo{
					ID:     "accel-2",
					Name:   "gpu-mi300x1-192gb",
					Slug:   "gpu-mi300x1-192gb",
					Status: "active",
				},
			},
		}

		tm.dedicatedInferences.EXPECT().ListAccelerators("00000000-0000-4000-8000-000000000000", "").Return(testAccelerators, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		err := RunDedicatedInferenceListAccelerators(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceListAccelerators_WithSlug(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		testAccelerators := do.DedicatedInferenceAcceleratorInfos{
			{
				DedicatedInferenceAcceleratorInfo: &godo.DedicatedInferenceAcceleratorInfo{
					ID:     "accel-1",
					Name:   "mi300x1-ghfpsf",
					Slug:   "gpu-mi300x1-192gb",
					Status: "ACTIVE",
				},
			},
		}

		tm.dedicatedInferences.EXPECT().ListAccelerators("00000000-0000-4000-8000-000000000000", "gpu-mi300x1-192gb").Return(testAccelerators, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")
		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceAcceleratorSlug, "gpu-mi300x1-192gb")

		err := RunDedicatedInferenceListAccelerators(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceListAccelerators_MissingID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDedicatedInferenceListAccelerators(config)
		assert.Error(t, err)
	})
}
