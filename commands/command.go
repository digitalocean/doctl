package commands

import "github.com/spf13/cobra"

// Command is a wrapper around cobra.Command that adds doctl specific
// functionality.
type Command struct {
	*cobra.Command

	// DocCategories are the documentation categories this command belongs to.
	DocCategories []string

	fmtCols []string
}
