package commands

import "github.com/spf13/cobra"

// Region creates the region commands heirarchy.
func Region() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "region",
		Short: "region commands",
		Long:  "region is used to access region commands",
	}

	CmdBuilder(cmd, RunRegionList, "list", "list regions", Writer, displayerType(&region{}))

	return cmd
}

// RunRegionList all regions.
func RunRegionList(c *CmdConfig) error {
	rs := c.Regions()

	list, err := rs.List()
	if err != nil {
		return err
	}

	image := &region{regions: list}
	return c.Display(image)
}
