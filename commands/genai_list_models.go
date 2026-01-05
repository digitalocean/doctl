package commands

import (
	"github.com/digitalocean/doctl/commands/displayers"
)

func ListModelsCmd() *Command {
	cmd := CmdBuilder(nil, RunGenAIListModels, "list-models", "List Gradient AI models", `The `+"`doctl gradient list-models`"+` command lists all available Gradient AI models.

The command returns the following details for each model:
	- The model ID
	- The model name  
	- Agreement name
	- The model creation date, in ISO8601 combined date and time format
	- The model update date, in ISO8601 combined date and time format
	- Parent ID of the model, this model is based on
	- Model has been fully uploaded
	- Download URL for the model
	- Version information about a model
	- is_foundational: True if it is a foundational model provided by DigitalOcean`, Writer, displayerType(&displayers.Model{}), aliasOpt("models", "lm"))

	cmd.Example = `doctl gradient list-models`

	return cmd
}

func RunGenAIListModels(c *CmdConfig) error {
	models, err := c.GradientAI().ListAvailableModels()
	if err != nil {
		return err
	}

	return c.Display(&displayers.Model{Models: models})
}
