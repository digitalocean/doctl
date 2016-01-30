package commands

import (
	"io"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/spf13/cobra"
)

// Account creates the account commands heirarchy.
func Account() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "account commands",
		Long:  "account is used to access account commands",
	}

	cmdBuilder(cmd, RunAccountGet, "get", "get account", writer,
		aliasOpt("g"), displayerType(&account{}))

	cmdBuilder(cmd, RunAccountRateLimit, "ratelimit", "get API rate limits", writer,
		aliasOpt("rl"), displayerType(&rateLimit{}))

	return cmd
}

// RunAccountGet runs account get.
func RunAccountGet(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	accountSvc := do.NewAccountService(client)
	a, err := accountSvc.Get()
	if err != nil {
		return err
	}

	dc := &displayer{
		ns:     ns,
		config: config,
		item:   &account{Account: a},
		out:    out,
	}

	return dc.Display()
}

// RunAccountRateLimit retrieves API rate limits for the account.
func RunAccountRateLimit(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	accountSvc := do.NewAccountService(client)
	rl, err := accountSvc.RateLimit()
	if err != nil {
		return err
	}

	dc := &displayer{
		ns:     ns,
		config: config,
		item:   &rateLimit{RateLimit: rl},
		out:    out,
	}

	return dc.Display()
}
