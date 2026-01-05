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
		"Creates an API key for your GenAI agent on your account. The command requires values for the "+"`"+"--name"+"`"+"`"+"--agent-id"+"`"+" The API key is created in the specified agent.",
		Writer,
		aliasOpt("c"),
		displayerType(&displayers.ApiKeyInfo{}),
	)
	AddStringFlag(cmdAgentAPIKeyCreate, doctl.ArgAgentId, "", "", "The ID of the agent to create API Keys into", requiredOpt())
	AddStringFlag(cmdAgentAPIKeyCreate, doctl.ArgAgentAPIKeyName, "", "", "API Key name", requiredOpt())
	cmdAgentAPIKeyCreate.Example = `The following example creates an apikey for an agent with ID ` + `12345678-1234-1234-1234-123456789013` + `: doctl genai agent apikeys create --name "My test API Keys" --agent-id "12345678-1234-1234-1234-123456789012"`

	ApiKeyDetails := `
	- The ApiKey ID
	- The ApiKey name
	- The ID of the user who created the Apikey
	- The ApiKey's Secret Key
	- The ApiKey deletion date, in ISO8601 combined date and time format
	- The ApiKey creation date, in ISO8601 combined date and time format
	`

	cmdAgentAPIKeyList := CmdBuilder(
		cmd,
		RunAgentAPIKeyList,
		"list",
		"List API Keys for an agent",
		"Retrieves a list of all the api keys for your agent, including the following details:\n"+ApiKeyDetails,
		Writer,
		aliasOpt("ls"),
		displayerType(&displayers.ApiKeyInfo{}),
	)
	AddStringFlag(cmdAgentAPIKeyList, doctl.ArgAgentId, "", "", "The ID of the agent for which to list API Keys")
	cmdAgentAPIKeyList.Example = `The following example lists the apikeys for an agent with ID ` + `12345678-1234-1234-1234-123456789013` +
		`: doctl genai agent apikeys list --agent-id "12345678-1234-1234-1234-123456789013" `
	cmdAgentAPIKeyUpdate := CmdBuilder(
		cmd,
		RunAgentAPIKeyUpdate,
		"update <apikey-id>",
		"Updates the name of an apikey of a GenAI agent ",
		"Use this command to update the name of an API key.",
		Writer,
		aliasOpt("u"),
		displayerType(&displayers.ApiKeyInfo{}),
	)
	AddStringFlag(cmdAgentAPIKeyUpdate, doctl.ArgAgentAPIKeyName, "", "", "API Key name")
	AddStringFlag(cmdAgentAPIKeyUpdate, doctl.ArgAgentId, "", "", "The ID of the agent for which to update the API Key")
	cmdAgentAPIKeyUpdate.Example = `The following example updates the name of an api-key with the ID ` +
		"`" + `12345678-1234-1234-1234-123456789012` + `for an agent with the ID ` + "`" + `12345678-1234-1234-1234-123456789013` + "`" + ` to ` + "`" + `new-name` + "`" +
		`: doctl genai agent apikeys update 12345678-1234-1234-1234-123456789012 --agent-id "12345678-1234-1234-1234-123456789013" --name "new-name"`

	cmdAgentAPIKeyDelete := CmdBuilder(
		cmd,
		RunAgentAPIKeyDelete,
		"delete <apikey-id>",
		"Deletes an api-key for an agent",
		"Deletes an api-key for an agent by ID.",
		Writer,
		aliasOpt("d", "del", "rm"),
	)
	AddBoolFlag(cmdAgentAPIKeyDelete, doctl.ArgAPIKeyForce, doctl.ArgShortForce, false, "Deletes the API Key without a confirmation prompt")

	AddStringFlag(cmdAgentAPIKeyDelete, doctl.ArgAgentId, "", "", "The ID of the agent for which to update the API Key")
	cmdAgentAPIKeyDelete.Example = `The following example deletes an apikey with ID ` + "`" + `12345678-1234-1234-1234-123456789012` + `for an agent with the ID ` + "`" + `12345678-1234-1234-1234-123456789013` + "`" + `: doctl genai agent apikeys delete 12345678-1234-1234-1234-123456789012 --agent-id "12345678-1234-1234-1234-123456789013"`

	cmdAgentAPIKeyRegenerate := CmdBuilder(
		cmd,
		RunAgentAPIKeyRegenerate,
		"regenerate <apikey-id>",
		"Regenerates an api-key for an agent",
		"Regenerates an api-key for an agent by ID.",
		Writer,
		aliasOpt("regen-api-key"),
	)
	AddStringFlag(cmdAgentAPIKeyRegenerate, doctl.ArgAgentId, "", "", "The ID of the agent for which to update the API Key")
	cmdAgentAPIKeyRegenerate.Example = `The following example regenerates apikey with the ID ` + "`" + `12345678-1234-1234-1234-123456789012` + "`" + `for an agent with the ID ` + "`" + `12345678-1234-1234-1234-123456789013` + "`" +
		`: doctl genai agent apikeys regen-api-key 12345678-1234-1234-1234-123456789012 --agent-id "12345678-1234-1234-1234-123456789013"`
	return cmd
}

// RunAgentAPIKeyList lists all API Keys linked with an agent.
func RunAgentAPIKeyList(c *CmdConfig) error {
	agentID, _ := c.Doit.GetString(c.NS, doctl.ArgAgentId)

	apikeysInfo, err := c.GradientAI().ListAgentAPIKeys(agentID)
	if err != nil {
		return err
	}
	if len(apikeysInfo) == 0 {
		return c.Display(&displayers.ApiKeyInfo{ApiKeyInfo: do.ApiKeys{}})
	}

	filtered := make(do.ApiKeys, 0, len(apikeysInfo))
	filtered = append(filtered, apikeysInfo...)
	return c.Display(&displayers.ApiKeyInfo{ApiKeyInfo: filtered})
}

// RunAgentAPIKeyCreate creates a new api key.
func RunAgentAPIKeyCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentAPIKeyName)
	if err != nil {
		return err
	}
	agentID, err := c.Doit.GetString(c.NS, doctl.ArgAgentId)
	if err != nil {
		return err
	}

	req := &godo.AgentAPIKeyCreateRequest{
		Name:      name,
		AgentUuid: agentID,
	}
	apikeyInfo, err := c.GradientAI().CreateAgentAPIKey(agentID, req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.ApiKeyInfo{ApiKeyInfo: do.ApiKeys{*apikeyInfo}})
}

// RunAgentAPIKeyUpdate updates an api key by ID.
func RunAgentAPIKeyUpdate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	apikeyID := c.Args[0]
	name, _ := c.Doit.GetString(c.NS, doctl.ArgAgentAPIKeyName)
	agentID, _ := c.Doit.GetString(c.NS, doctl.ArgAgentId)

	req := &godo.AgentAPIKeyUpdateRequest{
		Name:       name,
		AgentUuid:  agentID,
		APIKeyUuid: apikeyID,
	}
	apikeyInfo, err := c.GradientAI().UpdateAgentAPIKey(agentID, apikeyID, req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.ApiKeyInfo{ApiKeyInfo: do.ApiKeys{*apikeyInfo}})
}

// RunAgentAPIKeyDelete deletes an API Key by ID.
func RunAgentAPIKeyDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	apikeyID := c.Args[0]
	agentID, _ := c.Doit.GetString(c.NS, doctl.ArgAgentId)

	force, err := c.Doit.GetBool(c.NS, doctl.ArgAPIKeyForce)
	if err != nil {
		return err
	}

	// Ask for confirmation unless --force is set
	if force || AskForConfirmDelete("ApiKey", 1) == nil {
		apikeys := c.GradientAI()
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

// RunAgentAPIKeyRegenerate regenrates an API Key by ID.
func RunAgentAPIKeyRegenerate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	apikeyID := c.Args[0]
	agentID, _ := c.Doit.GetString(c.NS, doctl.ArgAgentId)

	apikeyInfo, err := c.GradientAI().RegenerateAgentAPIKey(agentID, apikeyID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.ApiKeyInfo{ApiKeyInfo: do.ApiKeys{*apikeyInfo}})
}
