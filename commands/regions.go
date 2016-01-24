package commands

import (
	"io"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/digitalocean/godo"
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

	si, err := rs.List()
	if err != nil {
		return err
	}

	item := &region{regions: []godo.Region{}}
	for _, r := range si {
		item.regions = append(item.regions, *r.Region)
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   item,
		out:    out,
	}
	return displayOutput(dc)
}
