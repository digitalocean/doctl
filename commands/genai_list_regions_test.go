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

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Test data
var (
	testDatacenterRegion1 = do.DatacenterRegion{
		DatacenterRegions: &godo.DatacenterRegions{
			Region:             "nyc1",
			InferenceUrl:       "https://inference.nyc1.digitalocean.com",
			ServesBatch:        true,
			ServesInference:    true,
			StreamInferenceUrl: "https://stream.nyc1.digitalocean.com",
		},
	}

	testDatacenterRegion2 = do.DatacenterRegion{
		DatacenterRegions: &godo.DatacenterRegions{
			Region:             "sfo3",
			InferenceUrl:       "https://inference.sfo3.digitalocean.com",
			ServesBatch:        false,
			ServesInference:    true,
			StreamInferenceUrl: "https://stream.sfo3.digitalocean.com",
		},
	}

	testDatacenterRegion3 = do.DatacenterRegion{
		DatacenterRegions: &godo.DatacenterRegions{
			Region:             "tor1",
			InferenceUrl:       "https://inference.tor1.digitalocean.com",
			ServesBatch:        true,
			ServesInference:    false,
			StreamInferenceUrl: "https://stream.tor1.digitalocean.com",
		},
	}

	testDatacenterRegions = do.DatacenterRegions{
		testDatacenterRegion1,
		testDatacenterRegion2,
		testDatacenterRegion3,
	}
)

func TestListRegionsCommand(t *testing.T) {
	cmd := ListRegionsCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "list-regions", cmd.Use)
	assert.Equal(t, "List GenAI regions", cmd.Short)
	assert.Contains(t, cmd.Long, "doctl genai list-regions")
}

func TestRunGenAIListRegions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.genAI.EXPECT().ListDatacenterRegions(nil, nil).Return(testDatacenterRegions, nil)

		config.Command = &cobra.Command{}

		err := RunGenAIListRegions(config)
		assert.NoError(t, err)
	})
}

func TestRunGenAIListRegionsError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.genAI.EXPECT().ListDatacenterRegions(nil, nil).Return(nil, assert.AnError)

		config.Command = &cobra.Command{}

		err := RunGenAIListRegions(config)
		assert.Error(t, err)
	})
}
