package commands

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Tokens creates the tokens command
func Tokens() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "token",
			Aliases: []string{"tokens"},
			Short:   "[BETA] Display commands for managing DigitalOcean API tokens",
			Long: heredoc.Docf(`
			[BETA] Display commands for managing DigitalOcean API tokens.

			Use the sub-commands of %s to create, manage, or revoke DigitalOcean API tokens.`,
				"`doctl token`",
			),
			Hidden: true,
		},
	}

	tokenDetails := heredoc.Docf(`
	- The ID of the token
	- The name of the token
	- The scopes applied to the token
	- The timestamp for when the token will expire
	- The date that it was last used
	- The timestamp for when it was created
	`)

	CmdBuilder(cmd, RunTokenGet, "get <id|name>",
		"Retrieve information about a DigitalOcean API token",
		heredoc.Docf(`Display the details for a specific DigitalOcean API token on your account. This includes:

		%s`, tokenDetails),
		Writer, aliasOpt("g"), displayerType(&displayers.Tokens{}))

	CmdBuilder(cmd, RunTokenList, "list",
		"List your DigitalOcean API tokens",
		heredoc.Docf(`Lists the following details for DigitalOcean API tokens on your account:

		%s`, tokenDetails),
		Writer, aliasOpt("ls"), displayerType(&displayers.Tokens{}))

	tokensCreateDesc := heredoc.Docf(`Create a new DigitalOcean API token with granular scopes.

		You can find all of the scopes that are available to be applied to a token using the
		%s sub-command. For example, to create a token that can be used to create and manage
		Droplets, use:

		    doctl token create droplet-service-token --scopes droplet:create,droplet:read,droplet:delete,droplet:operate

		By default, tokens will expire in 30 days (2592000 seconds). The maximum
		expiration for a token is 90 days (7776000 seconds). Use the %s flag to customize the
		expiration time.`,
		"`doctl token list-scopes`", "`--expires-in`")
	createTokenCmd := CmdBuilder(cmd, RunTokenCreate, "create <name>",
		"Create a new DigitalOcean API token", tokensCreateDesc,
		Writer, aliasOpt("c", "issue"), displayerType(&displayers.Tokens{}))
	// We hide this by default to prefer --expires-in which accepts a duration,
	// but we still offer --expiry-seconds for compatibility with other commands.
	AddIntFlag(createTokenCmd, doctl.ArgTokenExpirySeconds, "", 0,
		"Number of seconds from created at in which the token will expire", hiddenFlag())
	AddStringFlag(createTokenCmd, doctl.ArgTokenExpiresIn, "", "",
		`Duration (e.g. 24h, 60m, or 3600s) from now until the token will expire. Valid units are: "h", "m", "s".`)
	AddStringSliceFlag(createTokenCmd, doctl.ArgTokenScopes, "", nil,
		`Scopes to assign to the to token (e.g. droplet:read)`, requiredOpt())

	updateTokenCmd := CmdBuilder(cmd, RunTokenUpdate, "update <id|name>",
		"Update an existing DigitalOcean API token",
		heredoc.Docf(`Update the name or scopes for an existing DigitalOcean API token.

		You can find all of the scopes that are available to be applied to a token using the
		%s sub-command.`, "`doctl token list-scopes`"),
		Writer, aliasOpt("u"), displayerType(&displayers.Tokens{}))
	AddStringSliceFlag(updateTokenCmd, doctl.ArgTokenScopes, "", nil,
		`Scopes to assign to the to token (e.g. droplet:read)`)
	AddStringFlag(updateTokenCmd, doctl.ArgTokenUpdatedName, "", "", "The new name for the token")

	revokeTokenCmd := CmdBuilder(cmd, RunTokenRevoke, "revoke <id|name>...",
		"Revoke an existing DigitalOcean API token",
		heredoc.Docf(`Revoke an existing DigitalOcean API token. Once revoked, the token can no longer
		be updated or used to authenticate with the DigitalOcean API.`),
		Writer, aliasOpt("rm", "delete"))
	AddBoolFlag(revokeTokenCmd, doctl.ArgForce, doctl.ArgShortForce, false, "Revoke the token without a confirmation prompt")

	listScopesCmd := CmdBuilder(cmd, RunTokenListScopes, "list-scopes",
		"List available scopes for DigitalOcean API tokens",
		heredoc.Docf(`Use this command to list all of the available scopes for DigitalOcean API tokens.

		A scope consists of a namespace and an action (<namespace>:<action>). The namespace
		describes the resource type the scope authorizes the token to interact with. The action
		describes what the token will be authorized to do with that resource. For example, a
		droplet:read scoped token can list or retrieve information about Droplets but not create
		or destroy one.

		The list of scopes can be filtered by namespace using the %s flag.`, "`--namespace`"),
		Writer, aliasOpt("ls"), displayerType(&displayers.Tokens{}))
	AddStringFlag(listScopesCmd, doctl.ArgTokenScopeNamespace, "", "", "Filter scopes by their namespace")

	return cmd
}

// RunTokenGet returns a single API token.
func RunTokenGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(c.Args[0])
	if err != nil {
		// The argument is not an integer. Look for a name.
		found, err := tokenFromName(c, c.Args[0])
		if err != nil {
			return err
		}

		return c.Display(&displayers.Tokens{Tokens: found})
	}

	token, err := c.Tokens().Get(id)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Tokens{Tokens: []do.Token{*token}})
}

// RunTokenList returns a list of Tokens.
func RunTokenList(c *CmdConfig) error {
	tokens, err := c.Tokens().List()
	if err != nil {
		return err
	}

	return c.Display(&displayers.Tokens{Tokens: tokens})
}

// RunTokenCreate creates a DigitalOcean API token
func RunTokenCreate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	scopes, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTokenScopes)
	if err != nil {
		return err
	}

	expiry, err := c.Doit.GetIntPtr(c.NS, doctl.ArgTokenExpirySeconds)
	if err != nil {
		return err
	}

	expiration, err := c.Doit.GetString(c.NS, doctl.ArgTokenExpiresIn)
	if err != nil {
		return err
	}

	if expiry != nil && expiration != "" {
		return errors.New("the `--expiration` and `--expiry-seconds` flags are mutually exclusive")
	}

	if expiration != "" {
		expirationDur, err := time.ParseDuration(expiration)
		if err != nil {
			return err
		}
		expiry = godo.PtrTo(int(expirationDur.Seconds()))
	}

	createRequest := &godo.TokenCreateRequest{
		Name:          c.Args[0],
		Scopes:        scopes,
		ExpirySeconds: expiry,
	}

	token, err := c.Tokens().Create(createRequest)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Tokens{
		Tokens:          []do.Token{*token},
		WithAccessToken: true,
	})
}

// RunTokenUpdate updates a DigitalOcean API token
func RunTokenUpdate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	var id int
	id, err = strconv.Atoi(c.Args[0])
	if err != nil {
		// The argument is not an integer. Look for a name.
		found, err := tokenFromName(c, c.Args[0])
		if err != nil {
			return err
		}

		id = found[0].ID
	}

	updateRequest := &godo.TokenUpdateRequest{}
	scopes, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTokenScopes)
	if err != nil {
		return err
	}
	if len(scopes) > 0 {
		updateRequest.Scopes = scopes
	}

	updateName, err := c.Doit.GetString(c.NS, doctl.ArgTokenUpdatedName)
	if err != nil {
		return err
	}
	if updateName != "" {
		updateRequest.Name = updateName
	}

	if len(scopes) < 1 && updateName == "" {
		return errors.New("must supply at least one of --scopes or --updated-name")
	}

	token, err := c.Tokens().Update(id, updateRequest)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Tokens{Tokens: []do.Token{*token}})
}

// RunTokenRevoke revokes a DigitalOcean API token
func RunTokenRevoke(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("API token", len(c.Args)) == nil {
		var idList []int
		for _, in := range c.Args {
			var id int
			id, err = strconv.Atoi(in)
			if err != nil {
				// The argument is not an integer. Look for a name.
				found, err := tokenFromName(c, in)
				if err != nil {
					return err
				}

				id = found[0].ID
			}
			idList = append(idList, id)
		}

		for _, id := range idList {
			err = c.Tokens().Revoke(id)
			if err != nil {
				return err
			}
		}
	} else {
		return errOperationAborted
	}

	return nil
}

// RunTokenListScopes returns a list of token scopes.
func RunTokenListScopes(c *CmdConfig) error {
	namespace, err := c.Doit.GetString(c.NS, doctl.ArgTokenScopeNamespace)
	if err != nil {
		return err
	}

	scopes, err := c.Tokens().ListScopes(namespace)
	if err != nil {
		return err
	}

	return c.Display(&displayers.TokenScopes{TokenScopes: scopes})
}

// tokenFromName attempts to find a token by its name. An error is returned
// if more than one token is found with the same name as there is no
// uniqueness constraint on token names. An error is also returned if
// no token is found with the given name.
func tokenFromName(c *CmdConfig, name string) ([]do.Token, error) {
	var found []do.Token
	tokens, err := c.Tokens().List()
	if err != nil {
		return nil, err
	}

	for _, t := range tokens {
		if t.Name == name {
			found = append(found, t)
		}
	}

	if len(found) > 1 {
		return nil, fmt.Errorf("%d tokens named %s found", len(found), name)
	}

	if len(found) < 1 {
		return nil, fmt.Errorf("no tokens named %s found", name)
	}

	return found, nil
}
