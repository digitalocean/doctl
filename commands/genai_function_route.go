package commands

import (
	"encoding/json"
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func FunctionRoute() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "functionroute",
			Aliases: []string{"functionroute", "fr"},
			Short:   "Display commands that manages function routes for GenAI Agents.",
			Long:    "The subcommands of `doctl genai agent functionroute` allows you to access and manage function routes for GenAI Agents.",
		},
	}

	// Create command
	cmdFunctionRouteCreate := CmdBuilder(
		cmd,
		RunFunctionRouteCreate,
		"create",
		"Creates a function route",
		"Create a function route for your GenAI agent.The command requires values for the "+"`"+"--agent-id"+"`"+"`"+"--description"+"`"+"`"+"--faas_name"+"`"+"`"+"--faas_namespace"+"`"+" "+"`"+"--function_name"+"`"+"`"+"--input_schema "+"`, and "+"`"+"--output_schema "+"`"+" flags.",
		Writer, aliasOpt("c"),
		displayerType(&displayers.FunctionRoute{}))
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgAgentId, "", "", "GenAI Agent UUID", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionName, "", "", "Name of the function.", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionRouteDescription, "", "", "Description of the function.", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionRouteFaasName, "", "", "Name of the function route in the DigitalOcean functions platform", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionRouteFaasNamespace, "", "", "Namespace of the function route in the DigitalOcean functions platform", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionRouteInputSchema, "", "", "Input schema of the function route", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionRouteOutputSchema, "", "", "Output schema of the function route", requiredOpt())
	cmdFunctionRouteCreate.Example = `doctl genai agent functionroute create --agent-id "0f0e928f-4649-11f0-bf8f-4e013e2ddde4" --name "get-weather" --description "Creates a weather-lookup route" --faas-name "default/testing" --faas-namespace "fn-b90faf52-2b42-49c2-9792-75edfbb6f397" --input-schema '{"parameters":[{"name":"zipCode","in":"query","schema":{"type":"string"},"required":false,"description":"Zip description in input"},{"name":"measurement","in":"query","schema":{"type":"string","enum":["F","C"]},"required":false,"description":"Temperature unit (F or C)"}]}' --output-schema '{"properties":{"temperature":{"type":"number","description":"Temperature for the specified location"},"measurement":{"type":"string","description":"Unit used (F or C)"},"conditions":{"type":"string","description":"Weather conditions (Sunny, Cloudy, etc.)"}}}'`

	// Delete command
	cmdFunctionRouteDelete := CmdBuilder(
		cmd,
		RunFunctionRouteDelete,
		"delete",
		"Delete a function route",
		"Use this command to delete a function route of an agent. The command requires values for the "+"`"+"--agent-id"+"` and "+"`"+"--function-id"+"`"+" flags.",
		Writer, aliasOpt("d", "del", "rm"),
		displayerType(&displayers.FunctionRoute{}))
	AddStringFlag(cmdFunctionRouteDelete, doctl.ArgAgentId, "", "", "GenAI Agent UUID", requiredOpt())
	AddStringFlag(cmdFunctionRouteDelete, doctl.ArgFunctionID, "", "", "Function route ID to delete", requiredOpt())
	cmdFunctionRouteDelete.Example = `doctl genai agent functionroute delete  --agent-id "0f0e928f-4649-11f0-bf8f-4e013e2ddde4" --function-id "e40dc785-5e69-11f0-bf8f-4e013e2ddde4"`

	// Update command
	cmdFunctionRouteUpdate := CmdBuilder(cmd,
		RunFunctionRouteUpdate,
		"update",
		"Updates a function route",
		"Use this command to update function route of an agent.The command requires values for the "+"`"+"--agent-id"+"` and "+"`"+"--function-id"+"`"+" flags.",
		Writer, aliasOpt("u"),
		displayerType(&displayers.FunctionRoute{}))
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgAgentId, "", "", "GenAI Agent UUID", requiredOpt())
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionID, "", "", "Function route ID to update", requiredOpt())
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionRouteDescription, "", "", "Updated description of the function route")
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionName, "", "", "Name of the function.")
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionRouteFaasName, "", "", "Name of the function route in the DigitalOcean functions platform")
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionRouteFaasNamespace, "", "", "Namespace of the function route in the DigitalOcean functions platform")
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionRouteInputSchema, "", "", "Input schema of the function route")
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionRouteOutputSchema, "", "", "Output schema of the function route")
	cmdFunctionRouteUpdate.Example = `doctl genai agent functionroute update --agent-id "0f0e928f-4649-11f0-bf8f-4e013e2ddde4" --function-id "e40dc785-5e69-11f0-bf8f-4e013e2ddde4"  --name "doctl-updated23"  --description "Creating via doctl again"  --faas-name "default/testing"  --faas-namespace "fn-b90faf52-2b42-49c2-9792-75edfbb6f397"  --input-schema '{"parameters": [{"name": "zipCode", "in": "query", "schema": { "type": "string" },"required": false, "description": "Zip description in input"},{"name": "measurement","in": "query", "schema": { "type": "string", "enum": ["F","C"]},"required": false, "description": "Temperature unit (F or C)"}]}'   --output-schema '{"properties": {"temperature": {"type": "number", "description": "Temperature for the specified location"}, "measurement": { "type": "string", "description": "Unit used (F or C)"},"conditions": { "type": "string","description": "Weather conditions (Sunny, Cloudy, etc.)"}}}'`

	return cmd
}

// RunFunctionRouteCreate creates a new function route for a GenAI agent.
func RunFunctionRouteCreate(c *CmdConfig) error {
	agentUUID, err := c.Doit.GetString(c.NS, doctl.ArgAgentId)
	if err != nil {
		return err
	}
	functionName, err := c.Doit.GetString(c.NS, doctl.ArgFunctionName)
	if err != nil {
		return err
	}
	description, err := c.Doit.GetString(c.NS, doctl.ArgFunctionRouteDescription)
	if err != nil {
		return err
	}
	faasName, err := c.Doit.GetString(c.NS, doctl.ArgFunctionRouteFaasName)
	if err != nil {
		return err
	}
	faasNamespace, err := c.Doit.GetString(c.NS, doctl.ArgFunctionRouteFaasNamespace)
	if err != nil {
		return err
	}
	inputSchemaStr, err := c.Doit.GetString(c.NS, doctl.ArgFunctionRouteInputSchema)
	if err != nil {
		return err
	}
	outputSchemaStr, err := c.Doit.GetString(c.NS, doctl.ArgFunctionRouteOutputSchema)
	if err != nil {
		return err
	}

	if agentUUID == "" || functionName == "" || description == "" || faasName == "" || faasNamespace == "" || inputSchemaStr == "" || outputSchemaStr == "" {
		return doctl.NewMissingArgsErr("agent-id, function name, description, faas_name, faas_namespace, input_schema and output_schema are required")
	}

	var inputSchema godo.FunctionInputSchema
	if err := json.Unmarshal([]byte(inputSchemaStr), &inputSchema); err != nil {
		return fmt.Errorf("input_schema must be valid JSON matching FunctionInputSchema: %w", err)
	}
	var rawOutput json.RawMessage
	if err := json.Unmarshal([]byte(outputSchemaStr), &rawOutput); err != nil {
		return fmt.Errorf("output_schema must be valid JSON: %w", err)
	}

	req := &godo.FunctionRouteCreateRequest{
		AgentUuid:     agentUUID,
		FunctionName:  functionName,
		Description:   description,
		FaasName:      faasName,
		FaasNamespace: faasNamespace,
		InputSchema:   inputSchema,
		OutputSchema:  json.RawMessage(outputSchemaStr),
	}

	gs := c.GenAI()

	functionRoute, err := gs.CreateFunctionRoute(agentUUID, req)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Agent{Agents: do.Agents{*functionRoute}})
}

// RunFunctionRouteUpdate updates an existing function route for a GenAI agent.
func RunFunctionRouteUpdate(c *CmdConfig) error {
	agentUUID, err := c.Doit.GetString(c.NS, doctl.ArgAgentId)
	if err != nil {
		return err
	}
	functionID, err := c.Doit.GetString(c.NS, doctl.ArgFunctionID)
	if err != nil {
		return err
	}
	if agentUUID == "" || functionID == "" {
		return doctl.NewMissingArgsErr("agent-id and function-id are required")
	}

	name, _ := c.Doit.GetString(c.NS, doctl.ArgFunctionName)
	desc, _ := c.Doit.GetString(c.NS, doctl.ArgFunctionRouteDescription)
	faasName, _ := c.Doit.GetString(c.NS, doctl.ArgFunctionRouteFaasName)
	faasNS, _ := c.Doit.GetString(c.NS, doctl.ArgFunctionRouteFaasNamespace)
	inSchemaStr, _ := c.Doit.GetString(c.NS, doctl.ArgFunctionRouteInputSchema)
	outSchemaStr, _ := c.Doit.GetString(c.NS, doctl.ArgFunctionRouteOutputSchema)

	if name == "" && desc == "" && faasName == "" && faasNS == "" &&
		inSchemaStr == "" && outSchemaStr == "" {
		return doctl.NewMissingArgsErr("at least one field to update must be supplied")
	}

	var inSchema godo.FunctionInputSchema
	if inSchemaStr != "" {
		if err := json.Unmarshal([]byte(inSchemaStr), &inSchema); err != nil {
			return fmt.Errorf("input_schema must be valid JSON matching FunctionInputSchema: %w", err)
		}
	}

	var outSchema json.RawMessage
	if outSchemaStr != "" {
		if err := json.Unmarshal([]byte(outSchemaStr), &outSchema); err != nil {
			return fmt.Errorf("output_schema must be valid JSON: %w", err)
		}
	}

	req := &godo.FunctionRouteUpdateRequest{
		AgentUuid:    agentUUID,
		FunctionUuid: functionID,
	}
	if name != "" {
		req.FunctionName = name
	}
	if desc != "" {
		req.Description = desc
	}
	if faasName != "" {
		req.FaasName = faasName
	}
	if faasNS != "" {
		req.FaasNamespace = faasNS
	}
	if inSchemaStr != "" {
		req.InputSchema = inSchema
	}
	if outSchemaStr != "" {
		req.OutputSchema = outSchema
	}

	gs := c.GenAI()
	updated, err := gs.UpdateFunctionRoute(agentUUID, functionID, req)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Agent{Agents: do.Agents{*updated}})
}

// RunFunctionRouteDelete deletes a function route from a GenAI agent.
func RunFunctionRouteDelete(c *CmdConfig) error {
	agentUUID, err := c.Doit.GetString(c.NS, doctl.ArgAgentId)
	if err != nil {
		return err
	}
	functionID, err := c.Doit.GetString(c.NS, doctl.ArgFunctionID)
	if err != nil {
		return err
	}

	if agentUUID == "" || functionID == "" {
		return doctl.NewMissingArgsErr("agent-id and function-id are required")
	}

	gs := c.GenAI()
	functionRoute, err := gs.DeleteFunctionRoute(agentUUID, functionID)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Agent{Agents: do.Agents{*functionRoute}})
}
