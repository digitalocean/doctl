package commands

// cmdOption allow configuration of a command.
type cmdOption func(*command)

// aliasOpt adds aliases for a command.
func aliasOpt(aliases ...string) cmdOption {
	return func(c *command) {
		if c.Aliases == nil {
			c.Aliases = []string{}
		}

		for _, a := range aliases {
			c.Aliases = append(c.Aliases, a)
		}
	}
}

// displayerType sets the columns for display for a command.
func displayerType(d displayable) cmdOption {
	return func(c *command) {
		c.fmtCols = d.Cols()
	}
}

// hiddenCmd make a command hidden.
func hiddenCmd() cmdOption {
	return func(c *command) {
		c.Hidden = true
	}
}
