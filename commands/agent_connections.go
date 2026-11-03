/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// Connection statuses harness-api reports. active is reached once authorization
// completes; expired (the flow lapsed unauthorized) and revoked can never
// become active, so a wait ends on either.
const (
	connectionStatusActive  = "active"
	connectionStatusExpired = "expired"
	connectionStatusRevoked = "revoked"
)

// AgentConnections generates the `doctl harness-runtime connections` subtree,
// which manages external-provider connections (the newer actor-scoped auth
// flow) through harness-api's connection-management proxy. It is separate from
// the legacy team-scoped `doctl harness-runtime auth <provider>` command, which
// is left untouched.
func AgentConnections() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "connections",
			Aliases: []string{"connection", "conn"},
			Short:   "Manage external-provider connections for agents",
			Long: `Manage external-provider connections (e.g. github) that agents act through.

A connection binds an --actor (the actor id a manifest's secret slot references) to a provider, and holds the OAuth grant server-side. Agents never see the token; they reference the connection by actor.

This is the actor-scoped flow. The older ` + agentCLI + ` auth <provider> command connects a single provider for the whole team and is unaffected by these commands.`,
		},
	}

	ns := agentSubNS("agents.connections")

	cmdList := CmdBuilder(cmd, RunAgentsConnectionsList, "list",
		"List external-provider connections",
		`List the team's external-provider connections. Filter by `+"`--actor`"+` or `+"`--status`"+`. Revoked connections are never listed.`,
		Writer, append(ns, aliasOpt("ls"),
			displayerType(&displayers.HostedAgentConnection{}))...)
	AddStringFlag(cmdList, doctl.ArgAgentConnProvider, "", "github", "External provider")
	AddStringFlag(cmdList, doctl.ArgAgentConnActor, "", "", "Filter to a single actor id")
	AddStringFlag(cmdList, doctl.ArgAgentStatus, "", "", "Filter by connection status (active|pending|expired); revoked connections are never listed")
	AddIntFlag(cmdList, doctl.ArgAgentConnPage, "", 0, "Page number (1-based)")
	AddIntFlag(cmdList, doctl.ArgAgentConnPerPage, "", 0, "Connections per page (server defaults to 20, max 100)")
	cmdList.Example = agentCLI + ` connections list --provider github; ` + agentCLI + ` connections list --actor alice`

	cmdCreate := CmdBuilder(cmd, RunAgentsConnectionsCreate, "create",
		"Create (or resume) a connection for an actor",
		`Create a connection binding an actor to a provider. A new connection returns a browser authorization link; this command opens it and waits until authorization completes or the link expires, unless `+"`--no-wait`"+` is set. Omit `+"`--scopes`"+` to request the provider's full configured scope set.`,
		Writer, append(ns, aliasOpt("c"))...)
	AddStringFlag(cmdCreate, doctl.ArgAgentConnProvider, "", "github", "External provider")
	AddStringFlag(cmdCreate, doctl.ArgAgentConnActor, "", "", "Actor id the connection acts for", requiredOpt())
	AddStringSliceFlag(cmdCreate, doctl.ArgAgentConnScopes, "", nil, "OAuth scopes to request (repeatable); omit for the provider default")
	AddBoolFlag(cmdCreate, doctl.ArgAgentAuthNoBrowser, "", false, "Print the authorization URL instead of opening a browser")
	AddBoolFlag(cmdCreate, doctl.ArgAgentAuthNoWait, "", false, "Print the authorization URL and exit without waiting for authorization to complete")
	cmdCreate.Example = agentCLI + ` connections create --actor alice; ` + agentCLI + ` connections create --actor alice --scopes repo,read:org`

	cmdGet := CmdBuilder(cmd, RunAgentsConnectionsGet, "get <connection-id>",
		"Get one connection",
		`Read one connection by id. Useful to check a pending connection's status.`,
		Writer, append(ns, aliasOpt("show"),
			displayerType(&displayers.HostedAgentConnection{}))...)
	AddStringFlag(cmdGet, doctl.ArgAgentConnProvider, "", "github", "External provider")
	cmdGet.Example = agentCLI + ` connections get ` + "`<connection-id>`"

	cmdRevoke := CmdBuilder(cmd, RunAgentsConnectionsRevoke, "revoke <connection-id>",
		"Revoke a connection",
		`Revoke one connection by id. Agents referencing its actor can no longer act through it.`,
		Writer, append(ns, aliasOpt("delete", "rm"))...)
	AddStringFlag(cmdRevoke, doctl.ArgAgentConnProvider, "", "github", "External provider")
	cmdRevoke.Example = agentCLI + ` connections revoke ` + "`<connection-id>`"

	requireAgentSubcommand(cmd)
	return cmd
}

// connectionProviderFlag reads and normalizes the --provider flag.
func connectionProviderFlag(c *CmdConfig) (string, error) {
	provider, err := c.Doit.GetString(c.NS, doctl.ArgAgentConnProvider)
	if err != nil {
		return "", err
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return "", errors.New("a provider is required (e.g. --provider github)")
	}
	return provider, nil
}

// connectionIDArg resolves the positional <connection-id> argument.
func connectionIDArg(c *CmdConfig) (string, error) {
	if len(c.Args) < 1 {
		return "", doctl.NewMissingArgsErr(c.NS)
	}
	id := strings.TrimSpace(c.Args[0])
	if id == "" {
		return "", errors.New("a connection id is required")
	}
	return id, nil
}

// RunAgentsConnectionsList lists connections for a provider.
func RunAgentsConnectionsList(c *CmdConfig) error {
	provider, err := connectionProviderFlag(c)
	if err != nil {
		return err
	}
	opt := &godo.HostedAgentConnectionListOptions{}
	if opt.ActorID, err = c.Doit.GetString(c.NS, doctl.ArgAgentConnActor); err != nil {
		return err
	}
	if opt.Status, err = c.Doit.GetString(c.NS, doctl.ArgAgentStatus); err != nil {
		return err
	}
	if opt.Page, err = c.Doit.GetInt(c.NS, doctl.ArgAgentConnPage); err != nil {
		return err
	}
	if opt.PerPage, err = c.Doit.GetInt(c.NS, doctl.ArgAgentConnPerPage); err != nil {
		return err
	}

	resp, err := c.HostedAgents().ListConnections(provider, opt)
	if err != nil {
		return connectionsNotEnabledHint(err)
	}
	conns := resp.Connections
	if agentStructuredOutput(c) {
		return c.Display(&displayers.HostedAgentConnection{Connections: conns})
	}
	stylingEnabled = detectStyling()
	printConnectionsList(c.Out, conns)
	if resp.Pagination.Total > int32(len(conns)) {
		fmt.Fprintf(c.Out, "\n%s page %d · %d of %d\n", colorize("Showing", colMuted),
			resp.Pagination.Page, len(conns), resp.Pagination.Total)
	}
	return nil
}

// RunAgentsConnectionsCreate creates (or resumes) a connection and drives the
// browser authorization to completion.
func RunAgentsConnectionsCreate(c *CmdConfig) error {
	provider, err := connectionProviderFlag(c)
	if err != nil {
		return err
	}
	actor, err := c.Doit.GetString(c.NS, doctl.ArgAgentConnActor)
	if err != nil {
		return err
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return errors.New("an actor is required (--actor)")
	}
	scopes, err := c.Doit.GetStringSlice(c.NS, doctl.ArgAgentConnScopes)
	if err != nil {
		return err
	}
	noBrowser, err := c.Doit.GetBool(c.NS, doctl.ArgAgentAuthNoBrowser)
	if err != nil {
		return err
	}
	noWait, err := c.Doit.GetBool(c.NS, doctl.ArgAgentAuthNoWait)
	if err != nil {
		return err
	}

	svc := c.HostedAgents()
	env, err := svc.CreateConnection(provider, &godo.HostedAgentConnectionCreateRequest{
		ActorID: actor,
		Scopes:  scopes,
	})
	if err != nil {
		return connectionsNotEnabledHint(err)
	}
	return completeConnectionAuth(c, svc, provider, env, noBrowser, noWait)
}

// RunAgentsConnectionsGet reads one connection.
func RunAgentsConnectionsGet(c *CmdConfig) error {
	provider, err := connectionProviderFlag(c)
	if err != nil {
		return err
	}
	id, err := connectionIDArg(c)
	if err != nil {
		return err
	}
	env, err := c.HostedAgents().GetConnection(provider, id)
	if err != nil {
		return connectionsNotEnabledHint(err)
	}
	if agentStructuredOutput(c) {
		return c.Display(&displayers.HostedAgentConnection{
			Connections: []godo.HostedAgentConnection{env.Connection}, Single: true})
	}
	stylingEnabled = detectStyling()
	printConnectionCard(c.Out, env, false)
	return nil
}

// RunAgentsConnectionsRevoke revokes one connection.
func RunAgentsConnectionsRevoke(c *CmdConfig) error {
	provider, err := connectionProviderFlag(c)
	if err != nil {
		return err
	}
	id, err := connectionIDArg(c)
	if err != nil {
		return err
	}
	if _, err := c.HostedAgents().DeleteConnection(provider, id); err != nil {
		return connectionsNotEnabledHint(err)
	}
	stylingEnabled = detectStyling()
	printAgentSuccess(c.Out, fmt.Sprintf("Revoked connection %s", id))
	return nil
}

// completeConnectionAuth finishes a CreateConnection response: already-active,
// browser connect card, and optional poll until active.
func completeConnectionAuth(c *CmdConfig, svc do.HostedAgentsService, provider string, env *godo.HostedAgentConnectionEnvelope, noBrowser, noWait bool) error {
	if env == nil {
		return fmt.Errorf("empty connection response for %s", provider)
	}
	if strings.EqualFold(env.Connection.Status, connectionStatusActive) {
		stylingEnabled = detectStyling()
		printAgentSuccess(c.Out, fmt.Sprintf("%s connection is already active for %s", provider, env.Connection.ActorID))
		return nil
	}
	auth := env.Authorization
	if auth == nil || auth.ConnectURL == "" {
		return fmt.Errorf("harness-api returned status %q with no authorization URL", env.Connection.Status)
	}

	stylingEnabled = detectStyling()
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Connect "+provider, colHighlight))
	body.WriteString(cardRow("Actor", env.Connection.ActorID))
	body.WriteString(cardRow("URL", auth.ConnectURL))
	if auth.VerificationCode != "" {
		body.WriteString(cardRow("Code", boldColor(auth.VerificationCode, colHighlight)))
	}
	renderAgentCard(c.Out, body.String())
	if !noBrowser {
		if berr := browser.OpenURL(auth.ConnectURL); berr != nil {
			warn("could not open a browser automatically; open the URL above manually: %v", berr)
		}
	}

	if noWait {
		printAgentSuccess(c.Out, fmt.Sprintf("Run `%s connections get %s` after authorizing to confirm", agentCLI, env.Connection.ID))
		return nil
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ctx, cancelDeadline := context.WithDeadline(ctx, authWaitDeadline(auth.ExpiresAt, time.Now()))
	defer cancelDeadline()

	fmt.Fprintln(c.Out, "Waiting for authorization to complete... (Ctrl-C to stop)")
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("authorization was not completed before it expired; run `%s connections get %s` to check it, or `%s connections create --actor %s` for a new link",
					agentCLI, env.Connection.ID, agentCLI, env.Connection.ActorID)
			}
			return fmt.Errorf("stopped waiting; run `%s connections get %s` to check the connection later", agentCLI, env.Connection.ID)
		case <-time.After(agentsAuthPollInterval):
		}

		got, err := svc.GetConnection(provider, env.Connection.ID)
		if err != nil {
			return err
		}
		switch status := strings.ToLower(got.Connection.Status); status {
		case connectionStatusActive:
			printAgentSuccess(c.Out, fmt.Sprintf("%s connection active for %s", provider, got.Connection.ActorID))
			return nil
		case connectionStatusExpired, connectionStatusRevoked:
			return fmt.Errorf("%s connection for %s is %s; authorization was not completed. Run `%s connections create --actor %s` to start a new one",
				provider, got.Connection.ActorID, status, agentCLI, got.Connection.ActorID)
		}
	}
}

// connectionsNotEnabledHint rewrites the 501 harness-api returns when the
// connection surface is disabled into actionable guidance; other errors pass
// through unchanged.
func connectionsNotEnabledHint(err error) error {
	if _, status, ok := agentAPIError(err); ok && status == 501 {
		return errors.New("external-provider connections are not enabled on this environment")
	}
	return err
}

// printConnectionsList renders a compact styled list for `connections list`.
func printConnectionsList(w io.Writer, conns []godo.HostedAgentConnection) {
	if len(conns) == 0 {
		fmt.Fprintln(w, colorize("No connections", colMuted))
		return
	}
	noun := "connections"
	if len(conns) == 1 {
		noun = "connection"
	}
	fmt.Fprintln(w, boldColor(fmt.Sprintf("%d %s", len(conns), noun), colHighlight))
	fmt.Fprintln(w)

	for i, conn := range conns {
		if i > 0 {
			fmt.Fprintln(w)
		}
		actor := strings.TrimSpace(conn.ActorID)
		if actor == "" {
			actor = conn.ID
		}
		fmt.Fprintf(w, "%s %s\n", connectionStatusDot(conn.Status), boldColor(actor, colHighlight))
		meta := []string{colorize(conn.Provider, colMuted), colorize(conn.Status, colMuted)}
		if id := strings.TrimSpace(conn.ID); id != "" {
			meta = append(meta, colorize(id, colMuted))
		}
		if conn.DeletionProtected {
			meta = append(meta, colorize("protected", colWarning))
		}
		fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
		if conn.OAuth != nil && len(conn.OAuth.Scopes) > 0 {
			fmt.Fprintf(w, "  %s %s\n", colorize("scopes", colMuted), strings.Join(conn.OAuth.Scopes, ", "))
		}
	}
}

// connCardLabelWidth fits the detail card's longest label, "Protected"; the
// shared 8-wide default would leave its value one column out of line.
const connCardLabelWidth = 9

// printConnectionCard renders a single connection detail card for get.
func printConnectionCard(w io.Writer, env *godo.HostedAgentConnectionEnvelope, created bool) {
	if env == nil {
		fmt.Fprintln(w, colorize("No connection", colMuted))
		return
	}
	conn := env.Connection
	var body strings.Builder
	if created {
		fmt.Fprintf(&body, "%s\n\n", boldColor("Connection created", colSuccess))
	}
	body.WriteString(cardRowWidth("Actor", conn.ActorID, connCardLabelWidth))
	body.WriteString(cardRowWidth("Provider", conn.Provider, connCardLabelWidth))
	body.WriteString(cardRowWidth("Status", conn.Status, connCardLabelWidth))
	body.WriteString(cardRowWidth("Protected", connectionProtectedLabel(conn.DeletionProtected), connCardLabelWidth))
	if id := strings.TrimSpace(conn.ID); id != "" {
		body.WriteString(cardRowWidth("ID", colorize(id, colMuted), connCardLabelWidth))
	}
	if conn.OAuth != nil && len(conn.OAuth.Scopes) > 0 {
		body.WriteString(cardRowWidth("Scopes", strings.Join(conn.OAuth.Scopes, ", "), connCardLabelWidth))
	}
	if conn.CreatedAt != nil && !conn.CreatedAt.Time.IsZero() {
		body.WriteString(cardRowWidth("Created", colorize(formatCreatedAt(conn.CreatedAt.Time), colMuted), connCardLabelWidth))
	}
	if env.Authorization != nil && env.Authorization.ConnectURL != "" {
		fmt.Fprintln(&body)
		fmt.Fprintln(&body, colorize("Pending authorization", colMuted))
		body.WriteString(cardRowWidth("URL", env.Authorization.ConnectURL, connCardLabelWidth))
		if env.Authorization.VerificationCode != "" {
			body.WriteString(cardRowWidth("Code", env.Authorization.VerificationCode, connCardLabelWidth))
		}
	}
	renderAgentCard(w, body.String())
}

// connectionProtectedLabel renders deletion_protected for the detail card. A
// protected connection is one an agent environment still depends on, so an
// ordinary revoke is refused; "no" is printed rather than omitted so the card
// answers the question either way.
func connectionProtectedLabel(protected bool) string {
	if protected {
		return colorize("yes", colWarning)
	}
	return colorize("no", colMuted)
}

// connectionStatusDot colors the list bullet by connection status.
func connectionStatusDot(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case connectionStatusActive:
		return colorize("●", colSuccess)
	case "pending":
		return colorize("●", colWarning)
	default:
		return colorize("●", colMuted)
	}
}
