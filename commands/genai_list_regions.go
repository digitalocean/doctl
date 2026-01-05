package commands

import (
	"github.com/digitalocean/doctl/commands/displayers"
)

func ListRegionsCmd() *Command {
	cmd := CmdBuilder(nil, RunGenAIListRegions, "list-regions", "List Gradient AI regions", `The `+"`doctl gradient list-regions`"+` command lists all available Gradient AI regions.

The command returns the following details for each region:
	- Inference URL: The URL for the inference server
	- Region: The region code
	- Serves Batch: Whether this datacenter is capable of running batch jobs
	- Serves Inference: Whether this datacenter is capable of serving inference
	- Stream Inference URL: The URL for the inference streaming server`, Writer, displayerType(&displayers.DatacenterRegion{}), aliasOpt("regions", "lr"))

	cmd.Example = `doctl gradient list-regions`

	cmd.Flags().Bool("serves-inference", false, "Filter regions that serve inference")
	cmd.Flags().Bool("serves-batch", false, "Filter regions that serve batch jobs")

	return cmd
}

func RunGenAIListRegions(c *CmdConfig) error {
	var servesInferencePtr, servesBatchPtr *bool

	// Only set pointer if user passed the flag
	if c.Command.Flags().Changed("serves-inference") {
		val, _ := c.Command.Flags().GetBool("serves-inference")
		servesInferencePtr = &val
	}
	if c.Command.Flags().Changed("serves-batch") {
		val, _ := c.Command.Flags().GetBool("serves-batch")
		servesBatchPtr = &val
	}

	DatacenterRegions, err := c.GradientAI().ListDatacenterRegions(servesInferencePtr, servesBatchPtr)
	if err != nil {
		return err
	}

	return c.Display(&displayers.DatacenterRegion{DatacenterRegions: DatacenterRegions})
}
