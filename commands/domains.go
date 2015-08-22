package commands

import (
	"io"
	"os"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Domain creates the domain commands heirarchy.
func Domain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "domain commands",
		Long:  "domain is used to access domain commands",
	}

	cmdDomainCreate := NewCmdDomainCreate(os.Stdout)
	addStringFlag(cmdDomainCreate, doit.ArgDomainName, "", "Domain name")
	addStringFlag(cmdDomainCreate, doit.ArgIPAddress, "", "IP Address")

	cmd.AddCommand(cmdDomainCreate)

	return cmd
}

// NewCmdDomainCreate creates a domain create command.
func NewCmdDomainCreate(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "create domain",
		Long:  "create a domain",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDomainCreate(out))
		},
	}
}

// RunDomainCreate runs domain create.
func RunDomainCreate(out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	req := &godo.DomainCreateRequest{
		Name:      doit.VConfig.GetString("domain-name"),
		IPAddress: doit.VConfig.GetString("ip-address"),
	}

	d, _, err := client.Domains.Create(req)
	if err != nil {
		return err
	}
	return doit.WriteJSON(d, out)
}
