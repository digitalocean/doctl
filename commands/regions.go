package commands

import "github.com/spf13/cobra"

// Region creates the region commands heirarchy.
func Region() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "region",
		Short: "region commands",
		Long:  "region is used to access region commands",
	}

	cmdBuilder(cmd, RunRegionList, "list", "list regions", writer, displayerType(&region{}))

	return cmd
}

// RunRegionList all regions.
func RunRegionList(c *cmdConfig) error {
	rs := c.regionsService()

	list, err := rs.List()
	if err != nil {
		return err
	}

	image := &region{regions: list}
	return c.display(image)
}
