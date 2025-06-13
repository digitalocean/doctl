// /*
// Copyright 2018 The Doctl Authors All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package commands

import (
	"testing"

	"github.com/digitalocean/godo"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
)

// Test data
var (
	testAgent = do.Agent{
		Agent: &godo.Agent{
			Uuid:      "00000000-0000-4000-8000-000000000000",
			Name:      "Agent1",
			Region:    "tor1",
			ProjectId: "00000000-0000-4000-8000-000000000000",
			Model: &godo.Model{
				Uuid: "00000000-0000-4000-8000-000000000000",
			},
			Instruction: "You are an agent who thinks deeply about the world",
		},
	}
)

func TestRunAgentCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentName, "test-agent")
		config.Doit.Set(config.NS, doctl.ArgAgentRegion, "tor1")
		config.Doit.Set(config.NS, doctl.ArgAgentProjectId, "00000000-0000-4000-8000-000000000000")
		config.Doit.Set(config.NS, doctl.ArgModelId, "00000000-0000-4000-8000-000000000000")
		config.Doit.Set(config.NS, doctl.ArgAgentInstruction, "You are an agent who thinks deeply about the world")

		expectedRequest := &godo.AgentCreateRequest{
			Name:        "test-agent",
			Region:      "tor1",
			ProjectId:   "00000000-0000-4000-8000-000000000000",
			ModelUuid:   "00000000-0000-4000-8000-000000000000",
			Instruction: "You are an agent who thinks deeply about the world",
		}

		tm.genAI.EXPECT().CreateAgent(expectedRequest).Return(&testAgent, nil)

		err := RunAgentCreate(config)
		assert.NoError(t, err)
	})
}

func TestRunAgentGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "00000000-0000-4000-8000-000000000000"
		tm.genAI.EXPECT().GetAgent(agentID).Return(&testAgent, nil)

		config.Args = append(config.Args, agentID)

		err := RunAgentGet(config)
		assert.NoError(t, err)
	})
}

func TestRunAgentGetNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{} // No agent ID provided

		err := RunAgentGet(config)
		assert.Error(t, err)
	})
}

func TestRunAgentList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.genAI.EXPECT().ListAgents().Return(do.Agents{testAgent}, nil)

		err := RunAgentList(config)
		assert.NoError(t, err)
	})
}
func TestRunAgentUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, agentID)

		config.Doit.Set(config.NS, doctl.ArgAgentName, "updated-agent")
		config.Doit.Set(config.NS, doctl.ArgAgentDescription, "Updated instruction")

		expectedRequest := &godo.AgentUpdateRequest{
			Name:        "updated-agent",
			Description: "Updated instruction",
		}

		tm.genAI.EXPECT().UpdateAgent(agentID, expectedRequest).Return(&testAgent, nil)

		err := RunAgentUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunAgentDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "00000000-0000-4000-8000-000000000000"
		config.Args = []string{agentID}
		config.Doit.Set(config.NS, doctl.ArgForce, true) // Force delete without confirmation

		tm.genAI.EXPECT().DeleteAgent(agentID).Return(nil)

		err := RunAgentDelete(config)
		assert.NoError(t, err)
	})
}

func TestRunAgentDeleteNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{} // No agent ID provided

		err := RunAgentDelete(config)
		assert.Error(t, err)
	})
}

func TestRunAgentDelete_WithoutForce(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "00000000-0000-4000-8000-000000000000"
		config.Args = []string{agentID}
		config.Doit.Set(config.NS, doctl.ArgForce, false) // Don't force delete

		// Since we can't easily mock user input for confirmation,
		// this test will check that the function attempts to ask for confirmation
		// and aborts when no confirmation is given
		err := RunAgentDelete(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "operation aborted")
	})
}
func TestRunAgentUpdateVisibility(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"00000000-0000-4000-8000-000000000000"}
		config.Doit.Set(config.NS, "visibility", "VISIBILITY_PUBLIC")

		expectedRequest := &godo.AgentVisibilityUpdateRequest{
			Visibility: "VISIBILITY_PUBLIC",
		}

		tm.genAI.EXPECT().UpdateAgentVisibility("00000000-0000-4000-8000-000000000000", expectedRequest).Return(&testAgent, nil)

		err := RunAgentUpdateVisibility(config)
		assert.NoError(t, err)
	})
}

func TestRunAgentUpdateVisibilityNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{} // No agent ID provided

		err := RunAgentUpdateVisibility(config)
		assert.Error(t, err)
	})
}
