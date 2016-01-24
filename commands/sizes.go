package commands

import (
	"io"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/spf13/cobra"
)

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
func RunSizeList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	rs := do.NewSizesService(client)

	list, err := rs.List()
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &size{sizes: list},
		out:    out,
	}
	return displayOutput(dc)
}
