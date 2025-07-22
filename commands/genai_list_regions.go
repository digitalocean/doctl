package commands

import (
	"github.com/digitalocean/doctl/commands/displayers"
)

func ListRegionsCmd() *Command {
	cmd := CmdBuilder(nil, RunGenAIListRegions, "list-regions", "List GenAI regions", `The `+"`doctl genai list-regions`"+` command lists all available GenAI regions.

The command returns the following details for each region:
	- Inference URL: The URL for the inference server
	- Region: The region code
	- Serves Batch: Whether this datacenter is capable of running batch jobs
	- Serves Inference: Whether this datacenter is capable of serving inference
	- Stream Inference URL: The URL for the inference streaming server`, Writer, displayerType(&displayers.DatacenterRegion{}), aliasOpt("regions", "lr"))

	cmd.Example = `doctl genai list-regions`

	return cmd
}

func RunGenAIListRegions(c *CmdConfig) error {
	DatacenterRegions, err := c.GenAI().ListDatacenterRegions()
	if err != nil {
		return err
	}

	return c.Display(&displayers.DatacenterRegion{DatacenterRegions: DatacenterRegions})
}
