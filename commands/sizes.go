package commands

import "github.com/spf13/cobra"

// Size creates the size commands heirarchy.
func Size() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "size",
		Short: "size commands",
		Long:  "size is used to access size commands",
	}

	cmdBuilder2(cmd, RunSizeList, "list", "list sizes", writer, displayerType(&size{}))

	return cmd
}

// RunSizeList all sizes.
func RunSizeList(c *cmdConfig) error {
	rs := c.sizesService()

	list, err := rs.List()
	if err != nil {
		return err
	}

	item := &size{sizes: list}
	return c.display(item)
}
