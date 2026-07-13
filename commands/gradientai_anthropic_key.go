package commands

import (
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func AnthropicKeyCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "anthropic-key",
			Aliases: []string{"ak"},
			Short:   "Display commands that manage DigitalOcean Anthropic API Keys.",
			Long:    "The subcommands of `doctl gradient anthropic-key` allow you to access and manage Anthropic API keys.",
		},
	}

	cmdAnthropicKeyList := CmdBuilder(
		cmd,
		RunAnthropicKeyList,
		"list",
		"Lists all Anthropic API Keys",
		"Lists all Anthropic API Keys available in your account",
		Writer, aliasOpt("ls"),
		displayerType(&displayers.AnthropicApiKey{}),
	)
	cmdAnthropicKeyList.Example = `The following example lists information about all Anthropic API Keys  ` + "\n" +
		` doctl gradient anthropic-key list `

	cmdAnthropicKeyGet := CmdBuilder(
		cmd,
		RunAnthropicKeyGet,
		"get <anthropic-api-key-uuid>",
		"Retrieves an Anthropic API Key by its UUID",
		"Retrieves information about an Anthropic API Key",
		Writer, aliasOpt("g"),
		displayerType(&displayers.AnthropicApiKey{}),
	)
	cmdAnthropicKeyGet.Example = `The following example retrieves information about an Anthropic API Key with ID - f81d4fae-0000-11d0-a765-000000000000` + "\n" +
		` doctl gradient anthropic-key get f81d4fae-0000-11d0-a765-000000000000`

	cmdAnthropicKeyGetAgents := CmdBuilder(
		cmd,
		RunAnthropicKeyGetAgents,
		"get-agents <anthropic-api-key-uuid>",
		"Lists agents using an Anthropic API Key",
		"Lists all agents that are using the specified Anthropic API Key",
		Writer, aliasOpt("ga"),
		displayerType(&displayers.Agent{}),
	)
	cmdAnthropicKeyGetAgents.Example = `The following example retrieves information about an Anthropic API Key with ID - f81d4fae-0000-11d0-a765-000000000000` + "\n" +
		` doctl gradient anthropic-key get-agents f81d4fae-0000-11d0-a765-000000000000 `

	cmdAnthropicKeyCreate := CmdBuilder(
		cmd,
		RunAnthropicKeyCreate,
		"create",
		"Creates an Anthropic API Key",
		"Creates a new Anthropic API Key with the specified name and API key.",
		Writer, aliasOpt("c"),
		displayerType(&displayers.AnthropicApiKey{}),
	)
	cmdAnthropicKeyCreate.Example = `The following example creates an Anthropic API Key  ` + "\n" +
		` doctl gradient anthropic-key create --name my-key --api-key sk-ant-1234567890abcdef1234567890abcdef `
	AddStringFlag(cmdAnthropicKeyCreate, "name", "", "", "The name of the Anthropic API Key.", requiredOpt())
	AddStringFlag(cmdAnthropicKeyCreate, "api-key", "", "", "The API key for the Anthropic API Key.", requiredOpt())

	cmdAnthropicKeyUpdate := CmdBuilder(
		cmd,
		RunAnthropicKeyUpdate,
		"update <anthropic-api-key-uuid>",
		"Updates an Anthropic API Key by its UUID",
		"Updates an existing Anthropic API Key with the specified name and API key.",
		Writer, aliasOpt("u"),
		displayerType(&displayers.AnthropicApiKey{}),
	)
	cmdAnthropicKeyUpdate.Example = `The following example updates an Anthropic API Key with ID - f81d4fae-0000-11d0-a765-000000000000 ` + "\n" +
		` doctl gradient anthropic-key update f81d4fae-0000-11d0-a765-000000000000 --name my-key --api-key sk-ant-1234567890abcdef1234567890abcdef `
	AddStringFlag(cmdAnthropicKeyUpdate, "name", "", "", "The name of the Anthropic API Key.")
	AddStringFlag(cmdAnthropicKeyUpdate, "api-key", "", "", "The API key for the Anthropic API Key.")

	cmdAnthropicKeyDelete := CmdBuilder(
		cmd,
		RunAnthropicKeyDelete,
		"delete <anthropic-api-key-uuid>",
		"Deletes an Anthropic API Key by its UUID",
		"Deletes an Anthropic API Key by its UUID.",
		Writer, aliasOpt("rm"),
	)
	cmdAnthropicKeyDelete.Example = `The following example deletes an Anthropic API Key with ID - f81d4fae-0000-11d0-a765-000000000000 ` + "\n" +
		` doctl gradient anthropic-key delete f81d4fae-0000-11d0-a765-000000000000 ` + "\n" +
		`Note - Anthropic Keys linked to DO Agents cannot be deleted unless you change it from agent`
	AddBoolFlag(cmdAnthropicKeyDelete, doctl.ArgForce, "f", false, "Forces deletion without confirmation.")

	return cmd
}

func RunAnthropicKeyList(c *CmdConfig) error {
	anthropicApiKeys, err := c.GradientAI().ListAnthropicAPIKeys()
	if err != nil {
		return err
	}
	return c.Display(&displayers.AnthropicApiKey{AnthropicApiKeys: anthropicApiKeys})
}

func RunAnthropicKeyGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	anthropicApiKey, err := c.GradientAI().GetAnthropicAPIKey(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.AnthropicApiKey{AnthropicApiKeys: do.AnthropicApiKeys{*anthropicApiKey}})
}

func RunAnthropicKeyGetAgents(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	anthropicApiKeyID := c.Args[0]
	agents, err := c.GradientAI().ListAgentsByAnthropicAPIKey(anthropicApiKeyID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: agents})
}

func RunAnthropicKeyCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgAnthropicKeyName)
	if err != nil {
		return err
	}

	apiKey, err := c.Doit.GetString(c.NS, doctl.ArgAnthropicKeyAPIKey)
	if err != nil {
		return err
	}

	anthropicApiKeyCreate := &godo.AnthropicAPIKeyCreateRequest{
		Name:   name,
		ApiKey: apiKey,
	}

	anthropicApiKey, err := c.GradientAI().CreateAnthropicAPIKey(anthropicApiKeyCreate)
	if err != nil {
		return err
	}

	return c.Display(&displayers.AnthropicApiKey{AnthropicApiKeys: do.AnthropicApiKeys{*anthropicApiKey}})
}

func RunAnthropicKeyUpdate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	anthropicApiKeyID := c.Args[0]

	name, err := c.Doit.GetString(c.NS, doctl.ArgAnthropicKeyName)
	if err != nil {
		return err
	}

	apiKey, err := c.Doit.GetString(c.NS, doctl.ArgAnthropicKeyAPIKey)
	if err != nil {
		return err
	}

	anthropicApiKeyUpdate := &godo.AnthropicAPIKeyUpdateRequest{
		Name:       name,
		ApiKey:     apiKey,
		ApiKeyUuid: anthropicApiKeyID,
	}

	anthropicApiKey, err := c.GradientAI().UpdateAnthropicAPIKey(anthropicApiKeyID, anthropicApiKeyUpdate)
	if err != nil {
		return err
	}

	return c.Display(&displayers.AnthropicApiKey{AnthropicApiKeys: do.AnthropicApiKeys{*anthropicApiKey}})
}

func RunAnthropicKeyDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	anthropicApiKeyID := c.Args[0]
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if force || AskForConfirmDelete("Anthropic API Key", 1) == nil {
		_, err := c.GradientAI().DeleteAnthropicAPIKey(anthropicApiKeyID)
		if err != nil {
			return err
		}
		notice("Anthropic API Key deleted successfully")
	} else {
		return fmt.Errorf("operation aborted")
	}
	return nil
}
