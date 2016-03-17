package commands

import "github.com/spf13/cobra"

// Account creates the account commands heirarchy.
func Account() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "account commands",
		Long:  "account is used to access account commands",
	}

	CmdBuilder(cmd, RunAccountGet, "get", "get account", Writer,
		aliasOpt("g"), displayerType(&account{}), docCategories("account"))

	CmdBuilder(cmd, RunAccountRateLimit, "ratelimit", "get API rate limits", Writer,
		aliasOpt("rl"), displayerType(&rateLimit{}), docCategories("account"))

	return cmd
}

// RunAccountGet runs account get.
func RunAccountGet(c *CmdConfig) error {
	a, err := c.Account().Get()
	if err != nil {
		return err
	}

	return c.Display(&account{Account: a})
}

// RunAccountRateLimit retrieves API rate limits for the account.
func RunAccountRateLimit(c *CmdConfig) error {
	rl, err := c.Account().RateLimit()
	if err != nil {
		return err
	}

	return c.Display(&rateLimit{RateLimit: rl})
}
