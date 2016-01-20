package commands

import (
	"io"

	"github.com/bryanl/doit"
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

	cmdRegionList := cmdBuilder(RunRegionList, "list", "list regions", writer)
	cmd.AddCommand(cmdRegionList)

	return cmd
}

// RunRegionList all regions.
func RunRegionList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Regions.List(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(f)
	if err != nil {
		return err
	}

	list := make([]godo.Region, len(si))
	for i := range si {
		list[i] = si[i].(godo.Region)
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &region{regions: list},
		out:    out,
	}
	return displayOutput(dc)
}
