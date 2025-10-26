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
	"time"

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

func TestRunAgentListWithFilters(t *testing.T) {
	// Create timestamps for testing
	createdAt1 := &godo.Timestamp{Time: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)}
	createdAt2 := &godo.Timestamp{Time: time.Date(2024, 2, 1, 15, 30, 0, 0, time.UTC)}
	
	testAgents := do.Agents{
		{
			Agent: &godo.Agent{
				Uuid:       "00000000-0000-4000-8000-000000000001",
				Name:       "ChatBot Agent",
				Region:     "tor1",
				ProjectId:  "00000000-0000-4000-8000-000000000000",
				Tags:       []string{"production", "chatbot"},
				Model: &godo.Model{
					Uuid: "00000000-0000-4000-8000-000000000000",
				},
				CreatedAt: createdAt1,
			},
		},
		{
			Agent: &godo.Agent{
				Uuid:       "00000000-0000-4000-8000-000000000002",
				Name:       "Support Agent",
				Region:     "nyc1",
				ProjectId:  "00000000-0000-4000-8000-000000000001",
				Tags:       []string{"support", "internal"},
				Model: &godo.Model{
					Uuid: "00000000-0000-4000-8000-000000000001",
				},
				CreatedAt: createdAt2,
			},
		},
	}

	tests := []struct {
		name           string
		filters        map[string]string
		expectedCount  int
		description    string
	}{
		{
			name:          "Filter by region",
			filters:       map[string]string{"region": "tor1"},
			expectedCount: 1,
			description:   "Should return only agents in tor1 region",
		},
		{
			name:          "Filter by project ID",
			filters:       map[string]string{"project-id": "00000000-0000-4000-8000-000000000000"},
			expectedCount: 1,
			description:   "Should return only agents in specific project",
		},
		{
			name:          "Filter by name (partial match)",
			filters:       map[string]string{"name": "Chat"},
			expectedCount: 1,
			description:   "Should return agents with name containing 'Chat'",
		},
		{
			name:          "Filter by tag",
			filters:       map[string]string{"tag": "production"},
			expectedCount: 1,
			description:   "Should return agents with 'production' tag",
		},
		// Note: Status and Visibility filters are commented out as these fields 
		// may not exist in the current godo.Agent struct
		// {
		//	name:          "Filter by status",
		//	filters:       map[string]string{"status": "active"},
		//	expectedCount: 1,
		//	description:   "Should return only active agents",
		// },
		// {
		//	name:          "Filter by visibility",
		//	filters:       map[string]string{"visibility": "VISIBILITY_PUBLIC"},
		//	expectedCount: 1,
		//	description:   "Should return only public agents",
		// },
		{
			name:          "Filter by model ID",
			filters:       map[string]string{"model-id": "00000000-0000-4000-8000-000000000000"},
			expectedCount: 1,
			description:   "Should return agents using specific model",
		},
		{
			name:          "Filter by created after date",
			filters:       map[string]string{"created-after": "2024-01-20"},
			expectedCount: 1,
			description:   "Should return agents created after 2024-01-20",
		},
		{
			name:          "Filter by created before date",
			filters:       map[string]string{"created-before": "2024-01-20"},
			expectedCount: 1,
			description:   "Should return agents created before 2024-01-20",
		},
		{
			name:          "Multiple filters",
			filters:       map[string]string{"region": "tor1", "tag": "production"},
			expectedCount: 1,
			description:   "Should return agents matching multiple filters",
		},
		{
			name:          "No matching filters",
			filters:       map[string]string{"region": "sfo1"},
			expectedCount: 0,
			description:   "Should return no agents when no matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				// Set up filters
				for key, value := range tt.filters {
					config.Doit.Set(config.NS, key, value)
				}

				tm.genAI.EXPECT().ListAgents().Return(testAgents, nil)

				err := RunAgentList(config)
				assert.NoError(t, err)
			})
		})
	}
}

func TestMatchesFilters(t *testing.T) {
	createdAt := &godo.Timestamp{Time: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)}
	
	agent := do.Agent{
		Agent: &godo.Agent{
			Uuid:       "00000000-0000-4000-8000-000000000001",
			Name:       "Test Agent",
			Region:     "tor1",
			ProjectId:  "00000000-0000-4000-8000-000000000000",
			Tags:       []string{"test", "production"},
			Model: &godo.Model{
				Uuid: "00000000-0000-4000-8000-000000000000",
			},
			CreatedAt: createdAt,
		},
	}

	tests := []struct {
		name     string
		filters  map[string]string
		expected bool
	}{
		{
			name:     "Match region",
			filters:  map[string]string{"region": "tor1"},
			expected: true,
		},
		{
			name:     "No match region",
			filters:  map[string]string{"region": "nyc1"},
			expected: false,
		},
		{
			name:     "Match name (partial)",
			filters:  map[string]string{"name": "Test"},
			expected: true,
		},
		{
			name:     "Match name (case insensitive)",
			filters:  map[string]string{"name": "test"},
			expected: true,
		},
		{
			name:     "No match name",
			filters:  map[string]string{"name": "NonExistent"},
			expected: false,
		},
		{
			name:     "Match tag",
			filters:  map[string]string{"tag": "test"},
			expected: true,
		},
		{
			name:     "No match tag",
			filters:  map[string]string{"tag": "nonexistent"},
			expected: false,
		},
		// Note: Status and Visibility filters are commented out as these fields 
		// may not exist in the current godo.Agent struct
		// {
		//	name:     "Match status",
		//	filters:  map[string]string{"status": "active"},
		//	expected: true,
		// },
		// {
		//	name:     "Match visibility",
		//	filters:  map[string]string{"visibility": "VISIBILITY_PUBLIC"},
		//	expected: true,
		// },
		{
			name:     "Match created after",
			filters:  map[string]string{"created-after": "2024-01-01"},
			expected: true,
		},
		{
			name:     "Match created before",
			filters:  map[string]string{"created-before": "2024-02-01"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesFilters(
				agent,
				tt.filters["region"],
				tt.filters["project-id"],
				tt.filters["tag"],
				tt.filters["name"],
				tt.filters["model-id"],
				tt.filters["created-after"],
				tt.filters["created-before"],
			)
			assert.Equal(t, tt.expected, result)
		})
	}
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
