package commands

import (
	"io"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/spf13/cobra"
)

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
func RunRegionList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	rs := do.NewRegionsService(client)

	list, err := rs.List()
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &region{regions: list},
		out:    out,
	}
	return displayOutput(dc)
}
