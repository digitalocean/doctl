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
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

// Test data
var (
	testModel1 = do.Model{
		Model: &godo.Model{
			Uuid:             "model-1",
			Name:             "GPT-4 Turbo",
			IsFoundational:   true,
			Provider:         "OpenAI",
			InferenceName:    "gpt-4-turbo",
			InferenceVersion: "2024-04-09",
			UploadComplete:   true,
			Url:              "https://api.openai.com/v1/models/gpt-4-turbo",
			CreatedAt:        &godo.Timestamp{Time: time.Now().AddDate(0, -2, 0)},
			UpdatedAt:        &godo.Timestamp{Time: time.Now().AddDate(0, -1, 0)},
			Usecases:         []string{"text-generation", "chat"},
			Agreement: &godo.Agreement{
				Name:        "OpenAI Terms of Service",
				Description: "Standard OpenAI API terms and conditions",
			},
			Version: &godo.ModelVersion{
				Major: 4,
				Minor: 0,
				Patch: 0,
			},
		},
	}

	testModel2 = do.Model{
		Model: &godo.Model{
			Uuid:             "model-2",
			Name:             "Claude 3.5 Sonnet",
			IsFoundational:   true,
			Provider:         "Anthropic",
			InferenceName:    "claude-3-5-sonnet",
			InferenceVersion: "20240620",
			UploadComplete:   true,
			Url:              "https://api.anthropic.com/v1/models/claude-3-5-sonnet",
			CreatedAt:        &godo.Timestamp{Time: time.Now().AddDate(0, -1, -15)},
			UpdatedAt:        &godo.Timestamp{Time: time.Now().AddDate(0, 0, -10)},
			Usecases:         []string{"text-generation", "analysis", "coding"},
			Agreement: &godo.Agreement{
				Name:        "Anthropic Service Terms",
				Description: "Anthropic API service agreement",
			},
			Version: &godo.ModelVersion{
				Major: 3,
				Minor: 5,
				Patch: 0,
			},
		},
	}

	testModels = do.Models{testModel1, testModel2}
)

func TestListModelsCommand(t *testing.T) {
	cmd := ListModelsCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "list-models", cmd.Use)
	assert.Contains(t, cmd.Aliases, "models")
	assert.Contains(t, cmd.Aliases, "lm")
	assert.Equal(t, "List GenAI models", cmd.Short)
	assert.Contains(t, cmd.Long, "doctl genai list-models")
}

func TestRunGenAIListModels(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.gradientAI.EXPECT().ListAvailableModels().Return(testModels, nil)

		err := RunGenAIListModels(config)
		assert.NoError(t, err)
	})
}

func TestRunGenAIListModelsError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.gradientAI.EXPECT().ListAvailableModels().Return(nil, assert.AnError)

		err := RunGenAIListModels(config)
		assert.Error(t, err)
	})
}
