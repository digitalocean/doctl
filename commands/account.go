package commands

import "github.com/spf13/cobra"

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
func RunAccountGet(c *cmdConfig) error {
	a, err := c.accountService().Get()
	if err != nil {
		return err
	}

	return c.display(&account{Account: a})
}

// RunAccountRateLimit retrieves API rate limits for the account.
func RunAccountRateLimit(c *cmdConfig) error {
	rl, err := c.accountService().RateLimit()
	if err != nil {
		return err
	}

	return c.display(&rateLimit{RateLimit: rl})
}
