package commands

import "github.com/spf13/cobra"

// Size creates the size commands heirarchy.
func Size() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "size",
		Short: "size commands",
		Long:  "size is used to access size commands",
	}

	CmdBuilder(cmd, RunSizeList, "list", "list sizes", Writer, displayerType(&size{}),
		docCategories("compute"))

	return cmd
}

// RunSizeList all sizes.
func RunSizeList(c *CmdConfig) error {
	sizes := c.Sizes()

	list, err := sizes.List()
	if err != nil {
		return err
	}

	item := &size{sizes: list}
	return c.Display(item)
}
