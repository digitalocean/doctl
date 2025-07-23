package commands

import (
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func OpenAIKeyCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "openai-key",
			Aliases: []string{"ok"},
			Short:   "Display commands that manage DigitalOcean OpenAI API Keys.",
			Long:    "The subcommands of `doctl genai knowledge-base` allow you to access and manage knowledge bases of agents.",
		},
	}

	cmdOpenAIKeyList := CmdBuilder(
		cmd,
		RunOpenAIKeyList,
		"list",
		"Retrieves an OpenAI API Key",
		"Retrieves information about an OpenAI API Key",
		Writer, aliasOpt("ls"),
		displayerType(&displayers.OpenAiApiKey{}),
	)
	cmdOpenAIKeyList.Example = `The following example lists information about all OpenAI API Keys  ` + "\n" +
		` doctl genai openai-key list `

	cmdOpenAIKeyGet := CmdBuilder(
		cmd,
		RunOpenAIKeyGet,
		"get <openai-api-key-uuid>",
		"Retrieves an OpenAI API Key by its UUID",
		"Retrieves information about an OpenAI API Key",
		Writer, aliasOpt("g"),
		displayerType(&displayers.OpenAiApiKey{}),
	)
	cmdOpenAIKeyGet.Example = `The following example retrieves information about an OpenAI API Key with ID - f81d4fae-0000-11d0-a765-000000000000` + "\n" +
		` doctl genai openai-key get f81d4fae-0000-11d0-a765-000000000000`

	cmdOpenAIKeyGetAgents := CmdBuilder(
		cmd,
		RunOpenAIKeyGetAgents,
		"get-agents <openai-api-key-uuid>",
		"Retrieves an OpenAI API Key by its UUID",
		"Retrieves information about an OpenAI API Key",
		Writer, aliasOpt("ga"),
		displayerType(&displayers.Agent{}),
	)
	cmdOpenAIKeyGetAgents.Example = `The following example retrieves information about an OpenAI API Key with ID - f81d4fae-0000-11d0-a765-000000000000` + "\n" +
		` doctl genai openai-key get-agents f81d4fae-0000-11d0-a765-000000000000 `

	cmdOpenAIKeyCreate := CmdBuilder(
		cmd,
		RunOpenAIKeyCreate,
		"create",
		"Creates an OpenAI API Key",
		"Creates a new OpenAI API Key with the specified name and API key.",
		Writer, aliasOpt("c"),
		displayerType(&displayers.OpenAiApiKey{}),
	)
	cmdOpenAIKeyCreate.Example = `The following example creates an OpenAI API Key  ` + "\n" +
		` doctl genai openai-key create --name my-key --api-key sk-1234567890abcdef1234567890abcdef `
	AddStringFlag(cmdOpenAIKeyCreate, "name", "", "", "The name of the OpenAI API Key.", requiredOpt())
	AddStringFlag(cmdOpenAIKeyCreate, "api-key", "", "", "The API key for the OpenAI API Key.", requiredOpt())

	cmdOpenAIKeyUpdate := CmdBuilder(
		cmd,
		RunOpenAIKeyUpdate,
		"update <openai-api-key-uuid>",
		"Updates an OpenAI API Key by its UUID",
		"Updates an existing OpenAI API Key with the specified name and API key.",
		Writer, aliasOpt("u"),
		displayerType(&displayers.OpenAiApiKey{}),
	)
	cmdOpenAIKeyUpdate.Example = `The following example updates an OpenAI API Key with ID - f81d4fae-0000-11d0-a765-000000000000 ` + "\n" +
		` doctl genai openai-key update f81d4fae-0000-11d0-a765-000000000000 --name my-key --api-key sk-1234567890abcdef1234567890abcdef `
	AddStringFlag(cmdOpenAIKeyUpdate, "name", "", "", "The name of the OpenAI API Key.")
	AddStringFlag(cmdOpenAIKeyUpdate, "api-key", "", "", "The API key for the OpenAI API Key.")

	cmdOpenAIKeyDelete := CmdBuilder(
		cmd,
		RunOpenAIKeyDelete,
		"delete <openai-api-key-uuid>",
		"Deletes an OpenAI API Key by its UUID",
		"Deletes an OpenAI API Key by its UUID.",
		Writer, aliasOpt("rm"),
	)
	cmdOpenAIKeyDelete.Example = `The following example deletes an OpenAI API Key with ID - f81d4fae-0000-11d0-a765-000000000000 ` + "\n" +
		` doctl genai openai-key delete f81d4fae-0000-11d0-a765-000000000000 ` + "\n" +
		`Note - OpenAI Keys linked to DO Agents cannot be deleted unless you change it from agent`
	AddBoolFlag(cmdOpenAIKeyDelete, doctl.ArgForce, "f", false, "Forces deletion without confirmation.")

	return cmd
}

func RunOpenAIKeyList(c *CmdConfig) error {
	openAIApiKeys, err := c.GenAI().ListOpenAIAPIKeys()
	if err != nil {
		return err
	}
	return c.Display(&displayers.OpenAiApiKey{OpenAiApiKeys: openAIApiKeys})
}

func RunOpenAIKeyGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	openAIApiKey, err := c.GenAI().GetOpenAIAPIKey(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.OpenAiApiKey{OpenAiApiKeys: do.OpenAiApiKeys{*openAIApiKey}})
}

func RunOpenAIKeyGetAgents(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	openAIApiKeyID := c.Args[0]
	agents, err := c.GenAI().ListAgentsByOpenAIAPIKey(openAIApiKeyID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: agents})
}

func RunOpenAIKeyCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgOpenAIKeyName)
	if err != nil {
		return err
	}

	apiKey, err := c.Doit.GetString(c.NS, doctl.ArgOpenAIKeyAPIKey)
	if err != nil {
		return err
	}

	openAIApiKeyCreate := &godo.OpenAIAPIKeyCreateRequest{
		Name:   name,
		ApiKey: apiKey,
	}

	openAIApiKey, err := c.GenAI().CreateOpenAIAPIKey(openAIApiKeyCreate)
	if err != nil {
		return err
	}

	return c.Display(&displayers.OpenAiApiKey{OpenAiApiKeys: do.OpenAiApiKeys{*openAIApiKey}})
}

func RunOpenAIKeyUpdate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	openAIApiKeyID := c.Args[0]

	name, err := c.Doit.GetString(c.NS, doctl.ArgOpenAIKeyName)
	if err != nil {
		return err
	}

	apiKey, err := c.Doit.GetString(c.NS, doctl.ArgOpenAIKeyAPIKey)
	if err != nil {
		return err
	}

	openAIApiKeyUpdate := &godo.OpenAIAPIKeyUpdateRequest{
		Name:       name,
		ApiKey:     apiKey,
		ApiKeyUuid: openAIApiKeyID,
	}

	openAIApiKey, err := c.GenAI().UpdateOpenAIAPIKey(openAIApiKeyID, openAIApiKeyUpdate)
	if err != nil {
		return err
	}

	return c.Display(&displayers.OpenAiApiKey{OpenAiApiKeys: do.OpenAiApiKeys{*openAIApiKey}})
}

func RunOpenAIKeyDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	openAIApiKeyID := c.Args[0]
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if force || AskForConfirmDelete("OpenAI API Key", 1) == nil {
		_, err := c.GenAI().DeleteOpenAIAPIKey(openAIApiKeyID)
		if err != nil {
			return err
		}
		notice("OpenAI API Key deleted successfully")
	} else {
		return fmt.Errorf("operation aborted")
	}
	return nil
}
