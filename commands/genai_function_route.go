package commands

import (
	"encoding/json"
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// GenAI creates a new command for GenAI operations.
func FunctionRouteCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "genai",
			Aliases: []string{"genai"},
			Short:   "Display commands that manage DigitalOcean GenAI Agents.",
			Long:    "The subcommands of `doctl agents` allow you to access and manage GenAI Agents.",
		},
	}
	cmd.AddCommand(FunctionRoute())

	return cmd
}

func FunctionRoute() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "functionroute",
			Aliases: []string{"functionroute", "fr"},
			Short:   "Display commands that manage function routes for GenAI Agents.",
			Long:    "The subcommands of `doctl genai functionroute` allow you to access and manage function routes for GenAI Agents.",
		},
	}

	// Create command
	cmdFunctionRouteCreate := CmdBuilder(cmd, RunFunctionRouteCreate, "create", "Create a function route", "Create a function route for your GenAI agent", Writer, aliasOpt("c"), displayerType(&displayers.Agent{}))
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgAgentUUID, "", "", "GenAI Agent UUID", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionName, "", "", "Name of the function.", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionRouteDescription, "", "", "Description of the function.", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionRouteFaasName, "", "", "Name of the function route in the DigitalOcean functions platform", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionRouteFaasNamespace, "", "", "Namespace of the function route in the DigitalOcean functions platform", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionRouteInputSchema, "", "", "Input schema of the function route", requiredOpt())
	AddStringFlag(cmdFunctionRouteCreate, doctl.ArgFunctionRouteOutputSchema, "", "", "Output schema of the function route", requiredOpt())
	cmdFunctionRouteCreate.Example = `The following example creates a function route named ` + "`" + `funzip` + "`" + `. The Function route is created for the Agent with UUID  ` + "`" + `1b418231-b7d6-11ef-bf8f-4e013e2ddde4` + "`" + `  with the description ` + "`" + `Use when you need the zipcode for a city` + "`" + ` and faas_name is ` + "`" + `default/get-zipcode` + "`" + `.`

	// Delete command
	cmdFunctionRouteDelete := CmdBuilder(cmd, RunFunctionRouteDelete, "delete", "Delete a function route", "Delete a function route from your GenAI agent", Writer, aliasOpt("d"))
	AddStringFlag(cmdFunctionRouteDelete, doctl.ArgAgentUUID, "", "", "GenAI Agent UUID", requiredOpt())
	AddStringFlag(cmdFunctionRouteDelete, doctl.ArgFunctionID, "", "", "Function route ID to delete", requiredOpt())

	// Update command
	cmdFunctionRouteUpdate := CmdBuilder(cmd, RunFunctionRouteUpdate, "update", "Update a function route", "Update a function route for your GenAI agent", Writer, aliasOpt("u"), displayerType(&displayers.Agent{}))
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgAgentUUID, "", "", "GenAI Agent UUID", requiredOpt())
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionID, "", "", "Function route ID to update", requiredOpt())
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionRouteDescription, "", "", "Updated description of the function route")
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionName, "", "", "Name of the function.")
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionRouteFaasName, "", "", "Name of the function route in the DigitalOcean functions platform")
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionRouteFaasNamespace, "", "", "Namespace of the function route in the DigitalOcean functions platform")
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionRouteInputSchema, "", "", "Input schema of the function route")
	AddStringFlag(cmdFunctionRouteUpdate, doctl.ArgFunctionRouteOutputSchema, "", "", "Output schema of the function route")

	return cmd
}

// RunFunctionRouteCreate creates a new function route for a GenAI agent.
func RunFunctionRouteCreate(c *CmdConfig) error {
	agentUUID, err := c.Doit.GetString(c.NS, doctl.ArgAgentUUID)
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
		return doctl.NewMissingArgsErr("agentid, function name, description, faas_name, faas_namespace, input_schema and output_schema are required")
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

	return c.Display(&displayers.Agent{Agents: *functionRoute})
}

// RunFunctionRouteUpdate updates an existing function route for a GenAI agent.
func RunFunctionRouteUpdate(c *CmdConfig) error {
	agentUUID, err := c.Doit.GetString(c.NS, doctl.ArgAgentUUID)
	if err != nil {
		return err
	}
	functionID, err := c.Doit.GetString(c.NS, doctl.ArgFunctionID)
	if err != nil {
		return err
	}
	if agentUUID == "" || functionID == "" {
		return doctl.NewMissingArgsErr("agentid and functionid are required")
	}

	// ── gather user-supplied fields ───────────────────────────────────────────────
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

	// ── (de)serialize schemas only when present ──────────────────────────────────
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

	// ── build the request – only set what the user wants ─────────────────────────
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
	return c.Display(&displayers.Agent{Agents: *updated})
}

// RunFunctionRouteDelete deletes a function route from a GenAI agent.
func RunFunctionRouteDelete(c *CmdConfig) error {
	agentUUID, err := c.Doit.GetString(c.NS, doctl.ArgAgentUUID)
	if err != nil {
		return err
	}
	functionID, err := c.Doit.GetString(c.NS, doctl.ArgFunctionID)
	if err != nil {
		return err
	}

	if agentUUID == "" || functionID == "" {
		return doctl.NewMissingArgsErr("agentid and function-id are required")
	}

	gs := c.GenAI()
	functionRoute, err := gs.DeleteFunctionRoute(agentUUID, functionID)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Agent{Agents: *functionRoute})
}
