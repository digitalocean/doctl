package commands

import (
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// AgentRouteCmd creates the agent route command and its subcommands.
func AgentRouteCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "route",
			Aliases: []string{"routes", "r"},
			Short:   "Display commands for working with Gradient AI agent routes",
			Long:    "The subcommands of `doctl gradient agent route` manage your Gradient AI agent routes.",
		},
	}

	cmdAddAgentRoute := CmdBuilder(
		cmd,
		RunAgentRouteAdd,
		"add",
		"Adds an agent route to an agent",
		"Use this command to add an agent route to an agent. The command requires values for the "+"`"+"--parent-agent-id"+"` and "+"`"+"--child-agent-id"+"`"+" flags.",
		Writer,
		aliasOpt("c"),
		displayerType(&displayers.AgentRoute{}),
	)
	AddStringFlag(cmdAddAgentRoute, doctl.ArgParentAgentId, "", "", "Parent agent ID (required)", requiredOpt())
	AddStringFlag(cmdAddAgentRoute, doctl.ArgChildAgentId, "", "", "Child agent ID (required)", requiredOpt())
	AddStringFlag(cmdAddAgentRoute, doctl.ArgAgentRouteId, "", "", "Unique id of linkage")
	AddStringFlag(cmdAddAgentRoute, doctl.ArgAgentRouteName, "", "", "Route name")
	AddStringFlag(cmdAddAgentRoute, doctl.ArgAgentRouteIfCase, "", "", "Describes the case in which the child agent should be used")
	cmdAddAgentRoute.Example = `doctl gradient agent route add --parent-agent-id "12345678-1234-1234-1234-123456789012" --child-agent-id "12345678-1234-1234-1234-123456789013"`

	cmdUpdateAgentRoute := CmdBuilder(
		cmd,
		RunAgentRouteUpdate,
		"update",
		"Updates an agent route to an agent",
		"Use this command to updates an agent route to an agent.The command requires values for the "+"`"+"--parent-agent-id"+"` and "+"`"+"--child-agent-id"+"`"+" flags.",
		Writer,
		aliasOpt("u"),
		displayerType(&displayers.AgentRoute{}),
	)
	AddStringFlag(cmdUpdateAgentRoute, doctl.ArgParentAgentId, "", "", "Parent agent ID (required)", requiredOpt())
	AddStringFlag(cmdUpdateAgentRoute, doctl.ArgChildAgentId, "", "", "Child agent ID (required)", requiredOpt())
	AddStringFlag(cmdUpdateAgentRoute, doctl.ArgAgentRouteId, "", "", "Unique id of linkage")
	AddStringFlag(cmdUpdateAgentRoute, doctl.ArgAgentRouteName, "", "", "Route name")
	AddStringFlag(cmdUpdateAgentRoute, doctl.ArgAgentRouteIfCase, "", "", "Describes the case in which the child agent should be used")
	cmdUpdateAgentRoute.Example = `doctl gradient agent route update --parent-agent-id "12345678-1234-1234-1234-123456789012" --child-agent-id "12345678-1234-1234-1234-123456789013" --route-name "test_route" --if-case "use this to get test information"`

	cmdDeleteAgentRoute := CmdBuilder(
		cmd,
		RunAgentRouteDelete,
		"delete",
		"Deletes an agent route to an agent",
		"Use this command to delete an agent route to an agent. The command requires values for the "+"`"+"--parent-agent-id"+"` and "+"`"+"--child-agent-id"+"`"+" flags.",
		Writer,
		aliasOpt("d", "del", "rm"),
	)
	AddStringFlag(cmdDeleteAgentRoute, doctl.ArgParentAgentId, "", "", "Parent agent ID (required)", requiredOpt())
	AddStringFlag(cmdDeleteAgentRoute, doctl.ArgChildAgentId, "", "", "Child agent ID (required)", requiredOpt())
	AddBoolFlag(cmdDeleteAgentRoute, doctl.ArgForce, doctl.ArgShortForce, false, "Force route deletion without confirmation")
	cmdDeleteAgentRoute.Example = `doctl gradient agent route delete --parent-agent-id "12345678-1234-1234-1234-123456789012" --child-agent-id "12345678-1234-1234-1234-123456789013"`

	return cmd
}

// RunAgentRouteAdd adds a route to an agent.
func RunAgentRouteAdd(c *CmdConfig) error {
	parentAgentID, err := c.Doit.GetString(c.NS, doctl.ArgParentAgentId)
	if err != nil {
		return err
	}

	childAgentID, err := c.Doit.GetString(c.NS, doctl.ArgChildAgentId)
	if err != nil {
		return err
	}

	routeResponse, err := c.GradientAI().AddAgentRoute(parentAgentID, childAgentID)
	if err != nil {
		return err
	}

	return c.Display(&displayers.AgentRoute{AgentRouteResponses: []do.AgentRouteResponse{*routeResponse}})
}

// RunAgentRouteUpdate updates a route to an agent.
func RunAgentRouteUpdate(c *CmdConfig) error {
	parentAgentID, err := c.Doit.GetString(c.NS, doctl.ArgParentAgentId)
	if err != nil {
		return err
	}

	childAgentID, err := c.Doit.GetString(c.NS, doctl.ArgChildAgentId)
	if err != nil {
		return err
	}

	routeUUID, _ := c.Doit.GetString(c.NS, doctl.ArgAgentRouteId)
	routeName, _ := c.Doit.GetString(c.NS, doctl.ArgAgentRouteName)
	ifCase, _ := c.Doit.GetString(c.NS, doctl.ArgAgentRouteIfCase)

	req := &godo.AgentRouteUpdateRequest{
		ChildAgentUuid:  childAgentID,
		IfCase:          ifCase,
		ParentAgentUuid: parentAgentID,
		RouteName:       routeName,
		UUID:            routeUUID,
	}

	routeResponse, err := c.GradientAI().UpdateAgentRoute(parentAgentID, childAgentID, req)
	if err != nil {
		return err
	}

	return c.Display(&displayers.AgentRoute{AgentRouteResponses: []do.AgentRouteResponse{*routeResponse}})
}

// RunAgentRouteDelete deletes a route to an agent.
func RunAgentRouteDelete(c *CmdConfig) error {
	parentAgentID, err := c.Doit.GetString(c.NS, doctl.ArgParentAgentId)
	if err != nil {
		return err
	}

	childAgentID, err := c.Doit.GetString(c.NS, doctl.ArgChildAgentId)
	if err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("Agent Route", 1) == nil {
		err := c.GradientAI().DeleteAgentRoute(parentAgentID, childAgentID)
		if err != nil {
			return err
		}
		notice("Agent route deleted successfully")
		return nil
	}

	return c.GradientAI().DeleteAgentRoute(parentAgentID, childAgentID)
}
