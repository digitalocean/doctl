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
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// GenAI creates the agenßt command.
func GenAI() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "agent",
			Aliases: []string{"a"},
			Short:   "Display commands for working with GenAI agents",
			Long:    "The subcommands of `doctl agent` manage your GenAI agents.",
		},
	}

	cmdAgentCreate := CmdBuilder(
		cmd,
		RunAgentCreate,
		"create <agent-name>...",
		"Create a GenAI agent",
		"Create a GenAI agent on your account. The command requires values for the "+"`"+"--name"+"`"+"`"+"--project-id"+"`"+"`"+"--model-id"+"`"+"`"+"--region"+"`"+", and "+"`"+"--instruction"+"`"+" flags.",
		Writer,
		aliasOpt("c"),
		displayerType(&displayers.Agent{}),
	)
	AddStringFlag(cmdAgentCreate, doctl.ArgAgentName, "", "", "Agent name", requiredOpt())
	AddStringFlag(cmdAgentCreate, doctl.ArgAgentInstruction, "", "", "Agent instruction. Instructions help your agent to perform its job effectively.", requiredOpt())
	AddStringFlag(cmdAgentCreate, doctl.ArgAgentDescription, "", "", "Description of agent")
	AddStringFlag(cmdAgentCreate, doctl.ArgModelId, "", "", "Model ID", requiredOpt())
	AddStringFlag(cmdAgentCreate, doctl.ArgProjectID, "", "", "The DigitalOcean project ID associated with the agent", requiredOpt())
	AddStringFlag(cmdAgentCreate, doctl.ArgAgentRegion, "", "", "Region", requiredOpt())
	AddStringFlag(cmdAgentCreate, doctl.ArgAnthropicKeyId, "", "", "Anthropic Key ID")
	AddStringFlag(cmdAgentCreate, doctl.ArgKnowledgeBaseId, "", "", "Knowledge Base ID")
	AddStringFlag(cmdAgentCreate, doctl.ArgOpenAIKeyId, "", "", "OpenAI Key ID")
	AddStringFlag(cmdAgentCreate, doctl.ArgTags, "", "", "Tags")
	cmdAgentCreate.Example = `The following example creates an agent: doctl compute agent create --name "My Agent" --project-id "12345678-1234-1234-1234-123456789012" --model-id "12345678-1234-1234-1234-123456789013" --region "nyc" --instruction "You are an agent who thinks deeply about the world"`

	AgentDetails := `
	- The Agent ID
	- The Agent name
	- The Agent's description
	- The Agent's instructions
	- The Agent's model ID
	- The Agent's project ID
	- The Agent's region
	- The Agent creatio date, in ISO8601 combined date and time format
	- The Agent's tags
	- The Agent Update date, in ISO8601 combined date and time format
	- The ID of the user who created the agent`

	cmdAgentList := CmdBuilder(
		cmd,
		RunAgentList,
		"list",
		"List GenAI agents",
		"Retrieves a list of all the agents on your account, including the following information for each:"+AgentDetails,
		Writer,
		aliasOpt("ls"),
		displayerType(&displayers.Agent{}),
	)
	cmdAgentList.Example = `The following example creates an agent: doctl compute agent create --name "My Agent" --project-id "12345678-1234-1234-1234-123456789012" --model-id "12345678-1234-1234-1234-123456789013" --region "nyc" --instruction "You are an agent who thinks deeply about the world"`
	AddStringFlag(cmdAgentList, doctl.ArgAgentRegion, "", "", "Retrieves a list of Agents in a specified region")
	AddStringFlag(cmdAgentList, doctl.ArgTag, "", "", "Retrieves a list of Agents with a specified tag")
	cmdAgentList.Example = `The following example retrieves a list of all Agent in the ` + "`" + `nyc1` + "`" + ` region: doctl compute agent list --region nyc1`

	cmdAgentGet := CmdBuilder(
		cmd,
		RunAgentGet,
		"get <agent-id>",
		"Retrieve information about an agent",
		"Retrieves information about an agent, including:"+AgentDetails,
		Writer,
		aliasOpt("g"),
		displayerType(&displayers.Agent{}),
	)
	cmdAgentGet.Example = `The following example retrieves information about an agent: doctl compute agent get 12345678-1234-1234-1234-123456789012`

	update := CmdBuilder(
		cmd,
		RunAgentUpdate,
		"update <agent-id>",
		"Update a GenAI agent",
		"Update a GenAI agent's name and description.",
		Writer,
		aliasOpt("u"),
		displayerType(&displayers.Agent{}),
	)
	AddStringFlag(update, "name", "", "", "Agent name")
	AddStringFlag(update, "description", "", "", "Agent description")

	CmdBuilder(
		cmd,
		RunAgentDelete,
		"delete <agent-id>",
		"Delete a GenAI agent",
		"Delete a GenAI agent by ID.",
		Writer,
		aliasOpt("d", "rm"),
	)

	updateVisibility := CmdBuilder(
		cmd,
		RunAgentUpdateVisibility,
		"update-visibility <agent-id>",
		"Update visibility of a GenAI agent",
		"Update the visibility of a GenAI agent.",
		Writer,
	)
	AddStringFlag(updateVisibility, "visibility", "", "", "Agent visibility (e.g. public, private)", requiredOpt())

	return cmd
}

// RunAgentList lists all agents.
func RunAgentList(c *CmdConfig) error {
	region, _ := c.Doit.GetString(c.NS, "region")
	projectId, _ := c.Doit.GetString(c.NS, "project-id")
	//modelId, _ := c.Doit.GetString(c.NS, "model-id")
	tag, _ := c.Doit.GetString(c.NS, "tag")

	agents, err := c.GenAI().List()
	if err != nil {
		return err
	}

	filtered := make(do.Agents, 0, len(agents))
	for _, agent := range agents {
		if region != "" && agent.Agent.Region != region {
			continue
		}
		if projectId != "" && agent.Agent.ProjectId != projectId {
			continue
		}
		// if modelId != "" && agent.Agent.Model!= modelId {
		// 	continue
		// }
		if tag != "" {
			found := false
			for _, t := range agent.Agent.Tags {
				if t == tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		filtered = append(filtered, agent)
	}
	return c.Display(&displayers.Agent{Agents: agents})
}

// RunAgentCreate creates a new agent.
func RunAgentCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, "name")
	if err != nil {
		return err
	}
	instruction, err := c.Doit.GetString(c.NS, "instruction")
	if err != nil {
		return err
	}
	region, err := c.Doit.GetString(c.NS, "region")
	if err != nil {
		return err
	}
	projectId, err := c.Doit.GetString(c.NS, "project-id")
	if err != nil {
		return err
	}
	modelId, err := c.Doit.GetString(c.NS, "model-id")
	if err != nil {
		return err
	}
	req := &godo.AgentCreateRequest{
		Name:        name,
		Instruction: instruction,
		Region:      region,
		ProjectId:   projectId,
		ModelUuid:   modelId,
	}
	agent, err := c.GenAI().Create(req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: do.Agents{*agent}})
}

// RunAgentGet gets an agent by ID.
func RunAgentGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	agentID := c.Args[0]
	agent, err := c.GenAI().Get(agentID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: do.Agents{*agent}})
}

// RunAgentUpdate updates an agent by ID.
func RunAgentUpdate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	agentID := c.Args[0]
	name, _ := c.Doit.GetString(c.NS, "name")
	description, _ := c.Doit.GetString(c.NS, "description")
	req := &godo.AgentUpdateRequest{
		Name:        name,
		Description: description,
	}
	agent, err := c.GenAI().Update(agentID, req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: do.Agents{*agent}})
}

// RunAgentDelete deletes an agent by ID.
func RunAgentDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	agentID := c.Args[0]
	err := c.GenAI().Delete(agentID)
	if err != nil {
		return err
	}
	notice("Agent deleted")
	return nil
}

// RunAgentUpdateVisibility updates the visibility of an agent by ID.
func RunAgentUpdateVisibility(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	agentID := c.Args[0]
	visibility, err := c.Doit.GetString(c.NS, "visibility")
	if err != nil {
		return err
	}
	req := &godo.AgentVisibilityUpdateRequest{
		Visibility: visibility,
	}
	agent, err := c.GenAI().UpdateVisibility(agentID, req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: do.Agents{*agent}})
}
