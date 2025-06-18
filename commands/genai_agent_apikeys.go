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
func AgentAPIKeyCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "apikeys",
			Aliases: []string{"apikeys", "apk"},
			Short:   "Display commands for working with API keys of GenAI agents",
			Long:    "The subcommands of `doctl genai agent apikeys` manage your API Keys linked with agents.",
		},
	}

	cmdAgentAPIKeyCreate := CmdBuilder(
		cmd,
		RunAgentAPIKeyCreate,
		"create <apikey-name>...",
		"Creates an api-key for your GenAI agent",
		"Creates an API key for your GenAI agent on your account. The command requires values for the "+"`"+"--name"+"`"+"`"+"--agent-uuid"+"`"+" The API key is created in the specified agent.",
		Writer,
		aliasOpt("c"),
		displayerType(&displayers.Agent{}),
	)
	AddStringFlag(cmdAgentAPIKeyCreate, doctl.ArgAgentUUID, "", "", "The UUID of the agent to create API Keys into", requiredOpt())
	AddStringFlag(cmdAgentAPIKeyCreate, doctl.ArgAgentAPIKeyName, "", "", "API Key name", requiredOpt())
	cmdAgentAPIKeyCreate.Example = `The following example creates an agent: doctl genai agent apikeys create --name "My test API Keys" --agent-uuid "12345678-1234-1234-1234-123456789012"`

	cmdAgentAPIKeyList := CmdBuilder(
		cmd,
		RunAgentAPIKeyList,
		"list",
		"List API Keys for an agent",
		"Retrieves a list of all the api keys for your agent",
		Writer,
		aliasOpt("ls"),
		displayerType(&displayers.Agent{}),
	)
	AddStringFlag(cmdAgentAPIKeyList, doctl.ArgAgentUUID, "", "", "The UUID of the agent for which to list API Keys")
	cmdAgentAPIKeyList.Example = `The following example retrieves the apikeys for Agent with uuid: doctl genai agent apikeys list --agent-uuid "12345678-1234-1234-1234-123456789012`

	cmdAgentAPIKeyUpdate := CmdBuilder(
		cmd,
		RunAgentAPIKeyUpdate,
		"update <apikey-id>",
		"Updates the name of an apikey of a GenAI agent ",
		"Use this command to update the name of an API key.",
		Writer,
		aliasOpt("u"),
		displayerType(&displayers.Agent{}),
	)
	AddStringFlag(cmdAgentAPIKeyUpdate, doctl.ArgAgentName, "", "", "Agent name")
	AddStringFlag(cmdAgentAPIKeyUpdate, doctl.ArgAgentUUID, "", "", "The UUID of the agent for which to update the API Key")
	AddStringFlag(cmdAgentAPIKeyUpdate, doctl.ArgAPIkeyUUID, "", "", "The UUID of the api key to be updated")
	cmdAgentAPIKeyUpdate.Example = `The following example updates the name of an api-key with the ID ` +
		"`" + `12345678-1234-1234-1234-123456789012` + "`" + ` to ` + "`" + `new-name` + "`" +
		`: doctl genai agent apikeys update api-key-uuid 12345678-1234-1234-1234-123456789012 --agent-uuid "12345678-1234-1234-1234-123456789013" --name "new-name"`

	cmdAgentAPIKeyDelete := CmdBuilder(
		cmd,
		RunAgentAPIKeyDelete,
		"delete <apikey-id>",
		"Deletes an api-key for an agent",
		"Deletes an api-key for an agent by ID.",
		Writer,
		aliasOpt("d", "del", "rm"),
	)
	AddStringFlag(cmdAgentAPIKeyDelete, doctl.ArgAgentUUID, "", "", "The UUID of the agent for which to update the API Key")
	AddStringFlag(cmdAgentAPIKeyDelete, doctl.ArgAPIkeyUUID, "", "", "The UUID of the api key to be updated")
	cmdAgentAPIKeyDelete.Example = `The following example deletes an apikey the ID ` + "`" + `12345678-1234-1234-1234-123456789012` + "`" + `: doctl genai agent apikeys delete api-key-uuid 12345678-1234-1234-1234-123456789012 --agent-uuid "12345678-1234-1234-1234-123456789013"`

	cmdAgentAPIKeyRegenerate := CmdBuilder(
		cmd,
		RunAgentAPIKeyRegenerate,
		"regenerate <apikey-id>",
		"Regenerates an api-key for an agent",
		"Regenerates an api-key for an agent by ID.",
		Writer,
		aliasOpt("regen-api-key"),
	)
	AddStringFlag(cmdAgentAPIKeyRegenerate, doctl.ArgAgentUUID, "", "", "The UUID of the agent for which to update the API Key")
	AddStringFlag(cmdAgentAPIKeyRegenerate, doctl.ArgAPIkeyUUID, "", "", "The UUID of the api key to be updated")
	cmdAgentAPIKeyRegenerate.Example = `The following example renegrates apikey with the ID ` + "`" + `12345678-1234-1234-1234-123456789012` + "`" + `for an agent with the ID ` + "`" + `12345678-1234-1234-1234-123456789013` + "`" +
		`: doctl genai agent apikeys regen-api-key api-key-uuid 12345678-1234-1234-1234-123456789012 --agent-uuid "12345678-1234-1234-1234-123456789013"`
	return cmd
}

// RunAgentList lists all agents.
func RunAgentAPIKeyList(c *CmdConfig) error {
	agentID, _ := c.Doit.GetString(c.NS, "agent-uuid")

	apikeysInfo, err := c.GenAI().ListAgentAPIKeys(agentID)
	if err != nil {
		return err
	}

	filtered := make(do.ApiKeys, 0, len(apikeysInfo))
	for _, apikeyInfo := range apikeysInfo {

		filtered = append(filtered, apikeyInfo)
	}
	return c.Display(&displayers.ApiKeyInfo{ApiKeyInfo: filtered})
}

// RunAgentCreate creates a new agent.
func RunAgentAPIKeyCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, "name")
	if err != nil {
		return err
	}
	agentID, err := c.Doit.GetString(c.NS, "agent-uuid")
	if err != nil {
		return err
	}

	req := &godo.AgentAPIKeyCreateRequest{
		Name:      name,
		AgentUuid: agentID,
	}
	apikeyInfo, err := c.GenAI().CreateAgentAPIKey(agentID, req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.ApiKeyInfo{ApiKeyInfo: do.ApiKeys{*apikeyInfo}})
}

// RunAgentUpdate updates an agent by ID.
func RunAgentAPIKeyUpdate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	agentID := c.Args[0]
	name, _ := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	apikeyID, _ := c.Doit.GetString(c.NS, doctl.ArgAPIkeyUUID)

	req := &godo.AgentAPIKeyUpdateRequest{
		Name:       name,
		AgentUuid:  agentID,
		APIKeyUuid: apikeyID,
	}
	apikeyInfo, err := c.GenAI().UpdateAgentAPIKey(agentID, apikeyID, req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.ApiKeyInfo{ApiKeyInfo: do.ApiKeys{*apikeyInfo}})
}

// RunAgentDelete deletes an agent by ID.
func RunAgentAPIKeyDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	agentID := c.Args[0]
	apikeyID, _ := c.Doit.GetString(c.NS, doctl.ArgAPIkeyUUID)

	force, err := c.Doit.GetBool(c.NS, doctl.ArgAgentForce)
	if err != nil {
		return err
	}

	// Ask for confirmation unless --force is set
	if force || AskForConfirmDelete("ApiKey", 1) == nil {
		apikeys := c.GenAI()
		err := apikeys.DeleteAgentAPIKey(agentID, apikeyID)
		if err != nil {
			return err
		}
		notice("API Key deleted successfully")
	} else {
		return fmt.Errorf("operation aborted")
	}

	return nil
}

// RunAgentUpdateVisibility updates the visibility of an agent by ID.
func RunAgentAPIKeyRegenerate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	agentID := c.Args[0]
	apikeyID, _ := c.Doit.GetString(c.NS, doctl.ArgAPIkeyUUID)

	apikeyInfo, err := c.GenAI().RegenerateAgentAPIKey(agentID, apikeyID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.ApiKeyInfo{ApiKeyInfo: do.ApiKeys{*apikeyInfo}})
}
