package commands

import (
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// AgentCmd creates the agent command and its subcommands.
func AgentCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "agent",
			Aliases: []string{"agents", "a"},
			Short:   "Display commands for working with GenAI agents",
			Long:    "The subcommands of `doctl agent` manage your GenAI agents.",
		},
	}

	cmdAgentCreate := CmdBuilder(
		cmd,
		RunAgentCreate,
		"create <agent-name>...",
		"Creates a GenAI agent",
		"Creates a GenAI agent on your account. The command requires values for the "+"`"+"--name"+"`"+"`"+"--project-id"+"`"+"`"+"--model-id"+"`"+"`"+"--region"+"`"+", and "+"`"+"--instruction"+"`"+" flags.",
		Writer,
		aliasOpt("c"),
		displayerType(&displayers.Agent{}),
	)
	AddStringFlag(cmdAgentCreate, doctl.ArgAgentId, "", "", "The ID of the agent to create")
	AddStringFlag(cmdAgentCreate, doctl.ArgAgentName, "", "", "Agent name", requiredOpt())
	AddStringFlag(cmdAgentCreate, doctl.ArgAgentInstruction, "", "", "Agent instruction. Instructions help your agent to perform its job effectively.", requiredOpt())
	AddStringFlag(cmdAgentCreate, doctl.ArgAgentDescription, "", "", "Description of an agent")
	AddStringFlag(cmdAgentCreate, doctl.ArgModelId, "", "", "Model ID. Identifier for the foundation model.", requiredOpt())
	AddStringFlag(cmdAgentCreate, doctl.ArgProjectID, "", "", "UUID of the project to assign the Agent to", requiredOpt())
	AddStringFlag(cmdAgentCreate, doctl.ArgAgentRegion, "", "", `specifies the region to create an Agent in, such as tor1. Use the "doctl compute region list" command for a list of valid regions.`, requiredOpt())
	AddStringFlag(cmdAgentCreate, doctl.ArgAnthropicKeyId, "", "", "Anthropic Key ID")
	AddStringFlag(cmdAgentCreate, doctl.ArgKnowledgeBaseId, "", "", "Ids of the knowledge base(s) to attach to the agent")
	AddStringFlag(cmdAgentCreate, doctl.ArgOpenAIKeyId, "", "", "OpenAI API key ID to use with OpenAI models")
	AddStringFlag(cmdAgentCreate, doctl.ArgTags, "", "", "Applies a tag to the agent. ")
	cmdAgentCreate.Example = `The following example creates an agent: doctl compute agent create --name "My Agent" --project-id "12345678-1234-1234-1234-123456789012" --model-id "12345678-1234-1234-1234-123456789013" --region "tor1" --instruction "You are an agent who thinks deeply about the world"`

	AgentDetails := `
	- The Agent ID
	- The Agent name
	- The Agent's region
	- The Agent's model ID
	- The Agent's project ID
	- The Agent creation date, in ISO8601 combined date and time format
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
	AddStringFlag(cmdAgentList, doctl.ArgAgentRegion, "", "", "Retrieves a list of Agents in a specified region")
	AddStringFlag(cmdAgentList, doctl.ArgTag, "", "", "Retrieves a list of Agents with a specified tag")
	cmdAgentList.Example = `The following example retrieves a list of all Agent in the ` + "`" + `tor1` + "`" + ` region: doctl compute agent list --region tor1`

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

	cmdAgentUpdate := CmdBuilder(
		cmd,
		RunAgentUpdate,
		"update <agent-id>",
		"Updates a GenAI agent name and configuration",
		"Use this command to update the name and configuration of an agent.",
		Writer,
		aliasOpt("u"),
		displayerType(&displayers.Agent{}),
	)
	AddStringFlag(cmdAgentUpdate, doctl.ArgAgentName, "", "", "Agent name")
	AddStringFlag(cmdAgentUpdate, doctl.ArgAgentDescription, "", "", "Description of an agent")
	AddStringFlag(cmdAgentUpdate, doctl.ArgModelId, "", "", "Model ID. Identifier for the foundation model.")
	AddStringFlag(cmdAgentUpdate, doctl.ArgAgentInstruction, "", "", "Agent instruction. Instructions help your agent to perform its job effectively.")
	AddStringFlag(cmdAgentUpdate, doctl.ArgProjectID, "", "", "UUID of the project to assign the Agent to")
	AddStringFlag(cmdAgentUpdate, doctl.ArgAgentRegion, "", "", `specifies the region to create an Agent in, such as tor1. Use the "doctl compute region list" command for a list of valid regions.`)
	AddStringFlag(cmdAgentUpdate, doctl.ArgAnthropicKeyId, "", "", "Anthropic Key ID")
	AddStringFlag(cmdAgentUpdate, doctl.ArgOpenAIKeyId, "", "", "OpenAI API key ID to use with OpenAI models")
	AddIntFlag(cmdAgentUpdate, doctl.ArgK, "", 0, "specifies how many results should be considered from an attached knowledge base")
	AddIntFlag(cmdAgentUpdate, doctl.ArgMaxTokens, "", 0, "Specifies the maximum number of tokens the model can process in a single input or output, set as a number between 1 and 512. This determines the length of each response.")
	AddStringFlag(cmdAgentUpdate, doctl.ArgRetrievalMethod, "", "", "Specifies the method used to retrieve information. The options are 'RETRIEVAL_METHOD_UNKNOWN', 'RETRIEVAL_METHOD_REWRITE','RETRIEVAL_METHOD_STEP_BACK','RETRIEVAL_METHOD_SUB_QUERIES' and 'RETRIEVAL_METHOD_NONE'. The default is 'RETRIEVAL_METHOD_UNKNOWN'.")
	AddFloatFlag(cmdAgentUpdate, doctl.ArgTemperature, "", 0, "Specifies the temperature of the model. The temperature is a number between 0 and 1 that determines how creative or random the model's responses are. A lower temperature results in more predictable responses, while a higher temperature results in more creative responses.")
	AddFloatFlag(cmdAgentUpdate, doctl.ArgTopProbability, "", 0, "the cumulative probability threshold for word selection, specified as a number between 0 and 1. Higher values allow for more diverse outputs, while lower values ensure focused and coherent responses.")
	AddStringFlag(cmdAgentUpdate, doctl.ArgAgentId, "", "", "The ID of the agent to update")
	cmdAgentUpdate.Example = `The following example updates the name of an Agent with the ID ` +
		"`" + `12345678-1234-1234-1234-123456789012` + "`" + ` to ` + "`" + `new-name` + "`" +
		`: doctl compute agent update 12345678-1234-1234-1234-123456789012 --name "new-name"`

	cmdAgentDelete := CmdBuilder(
		cmd,
		RunAgentDelete,
		"delete <agent-id>",
		"Deletes a GenAI agent",
		"Deletes a GenAI agent by ID.",
		Writer,
		aliasOpt("d", "del", "rm"),
	)
	AddBoolFlag(cmdAgentDelete, doctl.ArgAgentForce, doctl.ArgShortForce, false, "Deletes the Agent without a confirmation prompt")
	cmdAgentDelete.Example = `The following example deletes an agent with the ID ` + "`" + `12345678-1234-1234-1234-123456789012` + "`" + `: doctl compute agent delete 12345678-1234-1234-1234-123456789012`

	cmdAgentUpdateVisibility := CmdBuilder(
		cmd,
		RunAgentUpdateVisibility,
		"update-visibility <agent-id>",
		"Update visibility of a GenAI agent",
		"Update the visibility of a GenAI agent.",
		Writer,
		aliasOpt("uv", "update-visibility", "update-vis"),
	)
	AddStringFlag(cmdAgentUpdateVisibility, "visibility", "", "", "Agent deployment visibility. Possible Options: `VISIBILITY_PLAYGROUND`, `VISIBILITY_PUBLIC`, `VISIBILITY_PRIVATE`. Default: `VISIBILITY_UNKNOWN`", requiredOpt())
	cmdAgentUpdateVisibility.Example = `The following example updates the visibility of an agent with the ID ` + "`" + `12345678-1234-1234-1234-123456789012` + "`" + ` to ` + "`" + `VISIBILITY_PUBLIC` + "`" + `: doctl compute agent update-visibility 12345678-1234-1234-1234-123456789012 --visibility 'VISIBILITY_PUBLIC'`

	return cmd
}

// RunAgentList lists all agents.
func RunAgentList(c *CmdConfig) error {
	region, _ := c.Doit.GetString(c.NS, "region")
	projectId, _ := c.Doit.GetString(c.NS, "project-id")
	tag, _ := c.Doit.GetString(c.NS, "tag")

	agents, err := c.GenAI().ListAgents()
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
	return c.Display(&displayers.Agent{Agents: filtered})
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
	agent, err := c.GenAI().CreateAgent(req)
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
	agent, err := c.GenAI().GetAgent(agentID)
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
	name, _ := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	description, _ := c.Doit.GetString(c.NS, doctl.ArgAgentDescription)
	instruction, _ := c.Doit.GetString(c.NS, doctl.ArgAgentInstruction)
	k, _ := c.Doit.GetInt(c.NS, doctl.ArgK)
	maxTokens, _ := c.Doit.GetInt(c.NS, doctl.ArgMaxTokens)
	retrievalMethod, _ := c.Doit.GetString(c.NS, doctl.ArgRetrievalMethod)
	temperature, _ := c.Doit.GetFloat64(c.NS, doctl.ArgTemperature)
	top_p, _ := c.Doit.GetFloat64(c.NS, doctl.ArgTopProbability)
	anthropicKeyId, _ := c.Doit.GetString(c.NS, doctl.ArgAnthropicKeyId)
	openAIKeyId, _ := c.Doit.GetString(c.NS, doctl.ArgOpenAIKeyId)
	modelId, _ := c.Doit.GetString(c.NS, doctl.ArgModelId)
	projectId, _ := c.Doit.GetString(c.NS, doctl.ArgProjectID)
	tags, _ := c.Doit.GetStringSlice(c.NS, doctl.ArgTags)

	req := &godo.AgentUpdateRequest{
		Name:             name,
		Description:      description,
		Instruction:      instruction,
		K:                k,
		MaxTokens:        maxTokens,
		RetrievalMethod:  retrievalMethod,
		Temperature:      float64(temperature),
		TopP:             float64(top_p),
		AnthropicKeyUuid: anthropicKeyId,
		OpenAiKeyUuid:    openAIKeyId,
		ModelUuid:        modelId,
		ProjectId:        projectId,
		Tags:             tags,
	}
	agent, err := c.GenAI().UpdateAgent(agentID, req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: do.Agents{*agent}})
}

// RunAgentDelete deletes an agent by ID.
func RunAgentDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	agentID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgAgentForce)
	if err != nil {
		return err
	}

	// Ask for confirmation unless --force is set
	if force || AskForConfirmDelete("Agent", 1) == nil {
		agents := c.GenAI()
		err := agents.DeleteAgent(agentID)
		if err != nil {
			return err
		}
		notice("Agent deleted successfully")
	} else {
		return fmt.Errorf("operation aborted")
	}

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
	agent, err := c.GenAI().UpdateAgentVisibility(agentID, req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: do.Agents{*agent}})
}
