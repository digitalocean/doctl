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
			ID:        "tok-1",
			Name:      "default",
			Value:     "secret-token-value",
			IsManaged: false,
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
	assert.True(t, subcommands["list"], "Expected list subcommand")
	assert.True(t, subcommands["delete"], "Expected delete subcommand")
	assert.True(t, subcommands["list-accelerators"], "Expected list-accelerators subcommand")
	assert.True(t, subcommands["create-token"], "Expected create-token subcommand")
	assert.True(t, subcommands["list-tokens"], "Expected list-tokens subcommand")
	assert.True(t, subcommands["revoke-token"], "Expected revoke-token subcommand")
	assert.True(t, subcommands["get-sizes"], "Expected get-sizes subcommand")
	assert.True(t, subcommands["get-gpu-model-config"], "Expected get-gpu-model-config subcommand")
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

		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceSpec, tmpFile)
		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		expectedReq := &godo.DedicatedInferenceUpdateRequest{
			Spec: testDedicatedInferenceSpecRequest,
		}

		tm.dedicatedInferences.EXPECT().Update("00000000-0000-4000-8000-000000000000", expectedReq).Return(&testDedicatedInference, nil)

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

		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceSpec, tmpFile)
		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceHuggingFaceToken, "hf_test_token")
		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		expectedReq := &godo.DedicatedInferenceUpdateRequest{
			Spec: testDedicatedInferenceSpecRequest,
			Secrets: &godo.DedicatedInferenceSecrets{
				HuggingFaceToken: "hf_test_token",
			},
		}

		tm.dedicatedInferences.EXPECT().Update("00000000-0000-4000-8000-000000000000", expectedReq).Return(&testDedicatedInference, nil)

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

func TestRunDedicatedInferenceDelete_MissingID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDedicatedInferenceDelete(config)
		assert.Error(t, err)
	})
}

func TestRunDedicatedInferenceList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		testListItems := do.DedicatedInferenceListItems{
			{
				DedicatedInferenceListItem: &godo.DedicatedInferenceListItem{
					ID:     "00000000-0000-4000-8000-000000000000",
					Name:   "test-dedicated-inference",
					Region: "nyc2",
					Status: "ACTIVE",
				},
			},
			{
				DedicatedInferenceListItem: &godo.DedicatedInferenceListItem{
					ID:     "11111111-1111-4111-8111-111111111111",
					Name:   "another-endpoint",
					Region: "sfo3",
					Status: "PROVISIONING",
				},
			},
		}

		tm.dedicatedInferences.EXPECT().List("", "").Return(testListItems, nil)

		err := RunDedicatedInferenceList(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceList_WithRegion(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		testListItems := do.DedicatedInferenceListItems{
			{
				DedicatedInferenceListItem: &godo.DedicatedInferenceListItem{
					ID:     "00000000-0000-4000-8000-000000000000",
					Name:   "test-dedicated-inference",
					Region: "nyc2",
					Status: "ACTIVE",
				},
			},
		}

		tm.dedicatedInferences.EXPECT().List("nyc2", "").Return(testListItems, nil)

		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceRegion, "nyc2")

		err := RunDedicatedInferenceList(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceList_WithName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		testListItems := do.DedicatedInferenceListItems{
			{
				DedicatedInferenceListItem: &godo.DedicatedInferenceListItem{
					ID:     "00000000-0000-4000-8000-000000000000",
					Name:   "test-dedicated-inference",
					Region: "nyc2",
					Status: "ACTIVE",
				},
			},
		}

		tm.dedicatedInferences.EXPECT().List("", "test-dedicated-inference").Return(testListItems, nil)

		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceName, "test-dedicated-inference")

		err := RunDedicatedInferenceList(config)
		assert.NoError(t, err)
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

func TestRunDedicatedInferenceCreateToken(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		testToken := &do.DedicatedInferenceToken{
			DedicatedInferenceToken: &godo.DedicatedInferenceToken{
				ID:        "tok-123",
				Name:      "my-token",
				Value:     "secret-value-abc",
				IsManaged: false,
			},
		}

		expectedReq := &godo.DedicatedInferenceTokenCreateRequest{
			Name: "my-token",
		}

		tm.dedicatedInferences.EXPECT().CreateToken("00000000-0000-4000-8000-000000000000", expectedReq).Return(testToken, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")
		config.Doit.Set(config.NS, doctl.ArgDedicatedInferenceTokenName, "my-token")

		err := RunDedicatedInferenceCreateToken(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceCreateToken_MissingID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDedicatedInferenceCreateToken(config)
		assert.Error(t, err)
	})
}

func TestRunDedicatedInferenceListTokens(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		testTokens := do.DedicatedInferenceTokens{
			{
				DedicatedInferenceToken: &godo.DedicatedInferenceToken{
					ID:        "tok-1",
					Name:      "default",
					IsManaged: true,
				},
			},
			{
				DedicatedInferenceToken: &godo.DedicatedInferenceToken{
					ID:        "tok-2",
					Name:      "my-token",
					IsManaged: false,
				},
			},
		}

		tm.dedicatedInferences.EXPECT().ListTokens("00000000-0000-4000-8000-000000000000").Return(testTokens, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		err := RunDedicatedInferenceListTokens(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceListTokens_MissingID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDedicatedInferenceListTokens(config)
		assert.Error(t, err)
	})
}

func TestRunDedicatedInferenceRevokeToken(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.dedicatedInferences.EXPECT().RevokeToken("00000000-0000-4000-8000-000000000000", "tok-123").Return(nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000", "tok-123")
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunDedicatedInferenceRevokeToken(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceRevokeToken_MissingArgs(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunDedicatedInferenceRevokeToken(config)
		assert.Error(t, err)
	})
}

func TestRunDedicatedInferenceRevokeToken_MissingTokenID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		err := RunDedicatedInferenceRevokeToken(config)
		assert.Error(t, err)
	})
}

func TestRunDedicatedInferenceGetSizes(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		testSizes := do.DedicatedInferenceSizes{
			{
				DedicatedInferenceSize: &godo.DedicatedInferenceSize{
					GPUSlug:      "gpu-mi300x1-192gb",
					PricePerHour: "3.59",
					Regions:      []string{"nyc2", "sfo3"},
					Currency:     "USD",
					CPU:          24,
					Memory:       98304,
					GPU: &godo.DedicatedInferenceSizeGPU{
						Count:  1,
						VramGb: 192,
						Slug:   "mi300x",
					},
				},
			},
		}
		testRegions := []string{"nyc2", "sfo3"}

		tm.dedicatedInferences.EXPECT().GetSizes().Return(testRegions, testSizes, nil)

		err := RunDedicatedInferenceGetSizes(config)
		assert.NoError(t, err)
	})
}

func TestRunDedicatedInferenceGetGPUModelConfig(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		testConfigs := do.DedicatedInferenceGPUModelConfigs{
			{
				DedicatedInferenceGPUModelConfig: &godo.DedicatedInferenceGPUModelConfig{
					ModelSlug:    "mistral/mistral-7b-instruct-v3",
					ModelName:    "Mistral 7B Instruct v3",
					IsModelGated: false,
					GPUSlugs:     []string{"gpu-mi300x1-192gb", "gpu-h100x1-80gb"},
				},
			},
			{
				DedicatedInferenceGPUModelConfig: &godo.DedicatedInferenceGPUModelConfig{
					ModelSlug:    "meta-llama/llama-3-70b",
					ModelName:    "Llama 3 70B",
					IsModelGated: true,
					GPUSlugs:     []string{"gpu-mi300x1-192gb"},
				},
			},
		}

		tm.dedicatedInferences.EXPECT().GetGPUModelConfig().Return(testConfigs, nil)

		err := RunDedicatedInferenceGetGPUModelConfig(config)
		assert.NoError(t, err)
	})
}
