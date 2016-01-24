package commands

import (
	"io"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/digitalocean/godo"
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

	si, err := rs.List()
	if err != nil {
		return err
	}

	item := &size{sizes: []godo.Size{}}
	for _, r := range si {
		item.sizes = append(item.sizes, *r.Size)
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   item,
		out:    out,
	}
	return displayOutput(dc)
}
