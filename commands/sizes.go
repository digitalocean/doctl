package commands

import "github.com/spf13/cobra"

// Size creates the size commands heirarchy.
func Size() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "size",
		Short: "size commands",
		Long:  "size is used to access size commands",
	}

	cmdBuilder(cmd, RunSizeList, "list", "list sizes", writer, displayerType(&size{}))

	return cmd
}

// RunSizeList all sizes.
func RunSizeList(c *cmdConfig) error {
	sizes := c.sizes()

	list, err := sizes.List()
	if err != nil {
		return err
	}

	item := &size{sizes: list}
	return c.display(item)
}
