package commands

import (
	"github.com/digitalocean/doctl/commands/displayers"
)

func ListModelsCmd() *Command {
	// Alias is "lm" only — "models" would collide with doctl inference models.
	cmd := CmdBuilder(nil, RunGradientAIListModels, "list-models", "List available models", `The `+"`doctl inference list-models`"+` command lists all available models from the GenAI platform catalog.

The command returns the following details for each model:
	- The model ID
	- The model name  
	- Agreement name
	- Agreement URL for the model's terms and conditions
	- The model creation date, in ISO8601 combined date and time format
	- The model update date, in ISO8601 combined date and time format
	- Parent ID of the model, this model is based on
	- Model has been fully uploaded
	- Download URL for the model
	- Version information about a model
	- is_foundational: True if it is a foundational model provided by DigitalOcean`, Writer, displayerType(&displayers.Model{}), aliasOpt("lm"))

	cmd.Example = `doctl inference list-models`

	return cmd
}

func RunGradientAIListModels(c *CmdConfig) error {
	models, err := c.GradientAI().ListAvailableModels()
	if err != nil {
		return err
	}

	return c.Display(&displayers.Model{Models: models})
}
