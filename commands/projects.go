/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"fmt"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Projects creates the projects commands hierarchy.
func Projects() *Command {
	projectsDesc := `

Projects allow you to organize your DigitalOcean resources (like Droplets, Spaces, load balancers, domains, and floating IPs) into groups that fit the way you work. You can create projects that align with the applications, environments, and clients that you host on DigitalOcean.
`

	projectDetails := `

  - The project's ID, in UUID format
  - The project owner's account UUID
  - The name of the project
  - The project's description
  - The project's specified purpose
  - The project's environment (Development, Staging, or Production)
  - A boolean indicating whether it is you default project
  - The date and time when the project was created
  - The date and time when the project was last updated
`

	cmd := &Command{
		Command: &cobra.Command{
			Use:   "projects",
			Short: "Manage projects and assign resources to them",
			Long:  "The subcommands of `doctl projects` allow you to create, manage, and assign resources to your projects." + projectsDesc,
		},
	}

	CmdBuilder(cmd, RunProjectsList, "list", "List existing projects",
		"List details for for your DigitalOcean projects, including:"+projectDetails,
		Writer, aliasOpt("ls"), displayerType(&displayers.Project{}))
	CmdBuilder(cmd, RunProjectsGet, "get <id>", "Retrieve details for a specific project",
		"Display the following details for an existing project specified by its ID (use `default` for <id> to retieve your default project):"+projectDetails,
		Writer, aliasOpt("g"), displayerType(&displayers.Project{}))

	cmdProjectsCreate := CmdBuilder(cmd, RunProjectsCreate, "create",
		"Create a new project", "Create a new project to organize your resources specifying its name and purpose."+projectsDesc,
		Writer, aliasOpt("c"), displayerType(&displayers.Project{}))
	AddStringFlag(cmdProjectsCreate, doctl.ArgProjectName, "", "",
		"A name for the project", requiredOpt())
	AddStringFlag(cmdProjectsCreate, doctl.ArgProjectPurpose, "", "",
		"The project's purpose", requiredOpt())
	AddStringFlag(cmdProjectsCreate, doctl.ArgProjectDescription, "", "",
		"A description of the project")
	AddStringFlag(cmdProjectsCreate, doctl.ArgProjectEnvironment, "", "",
		"The environment in which your project resides. Possible values: `Development`, `Staging`, or `Production`")

	cmdProjectsUpdate := CmdBuilder(cmd, RunProjectsUpdate, "update <id>",
		"Update an existing project",
		"Update information about an existing project specified by its ID (use `default` for <id> to update your default project).",
		Writer, aliasOpt("u"), displayerType(&displayers.Project{}))
	AddStringFlag(cmdProjectsUpdate, doctl.ArgProjectName, "", "", "A name for the project")
	AddStringFlag(cmdProjectsUpdate, doctl.ArgProjectPurpose, "", "", "The project's purpose")
	AddStringFlag(cmdProjectsUpdate, doctl.ArgProjectDescription, "", "",
		"A description of the project")
	AddStringFlag(cmdProjectsUpdate, doctl.ArgProjectEnvironment, "", "",
		"The environment in which your project resides. Possible values: `Development`, `Staging`, or `Production`")
	AddBoolFlag(cmdProjectsUpdate, doctl.ArgProjectIsDefault, "", false,
		"Set the specified project as your default project")

	cmdProjectsDelete := CmdBuilder(cmd, RunProjectsDelete, "delete <id> [<id> ...]",
		"Delete the specified project", "Delete a project by specifying its ID. To be deleted, a project must not have any resources assigned to it.",
		Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdProjectsDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the project without confirmation")

	cmd.AddCommand(ProjectResourcesCmd())

	return cmd
}

// ProjectResourcesCmd creates the project resources commands hierarchy.
func ProjectResourcesCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "resources",
			Short: "Manage resources assigned to a project",
			Long:  "The subcommands of `doctl projects resources` allow you to list and assign resources to your projects.",
		},
	}

	urnDesc := `

A valid URN has the format: ` + "`" + `do:resource_type:resource_id` + "`" + `. For example:

  - ` + "`" + `do:droplet:4126873` + "`" + `
  - ` + "`" + `do:volume:6fc4c277-ea5c-448a-93cd-dd496cfef71f` + "`" + `
`

	CmdBuilder(cmd, RunProjectResourcesList, "list <project-id>", "List resources assigned to a project",
		"List all of the resources assigned to the specified project displaying their uniform resource names (\"URNs\").",
		Writer, aliasOpt("ls"), displayerType(&displayers.ProjectResource{}))
	CmdBuilder(cmd, RunProjectResourcesGet, "get <urn>", "Retrieve a resource by its URN",
		"Retrieve information about a resource by specifying its uniform resource name (\"URN\"). Currently, ony Droplets, floating IPs, load balancers, domains, and volumes are supported."+urnDesc,
		Writer, aliasOpt("g"), displayerType(&displayers.ProjectResource{}))

	cmdProjectResourcesAssign := CmdBuilder(cmd, RunProjectResourcesAssign,
		"assign <project-id> --resource=<urn> [--resource=<urn> ...]",
		"Assign one or more resources to a project",
		"Assign one or more resources to a project by specifying the resource's uniform resource name (\"URN\")."+urnDesc,
		Writer, aliasOpt("a"))
	AddStringSliceFlag(cmdProjectResourcesAssign, doctl.ArgProjectResource, "",
		[]string{}, "URNs specifying resources to assign to the project")

	return cmd
}

// RunProjectsList lists Projects.
func RunProjectsList(c *CmdConfig) error {
	ps := c.Projects()
	list, err := ps.List()
	if err != nil {
		return err
	}

	return c.Display(&displayers.Project{Projects: list})
}

// RunProjectsGet retrieves an existing Project by its identifier. Use "default"
// as an identifier to retrieve your default project.
func RunProjectsGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]

	ps := c.Projects()
	p, err := ps.Get(id)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Project{Projects: do.Projects{*p}})
}

// RunProjectsCreate creates a new Project with a given configuration.
func RunProjectsCreate(c *CmdConfig) error {
	r := new(godo.CreateProjectRequest)
	if err := buildProjectsCreateRequestFromArgs(c, r); err != nil {
		return err
	}

	ps := c.Projects()
	p, err := ps.Create(r)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Project{Projects: do.Projects{*p}})
}

// RunProjectsUpdate updates an existing Project with a given configuration.
func RunProjectsUpdate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]

	r := new(godo.UpdateProjectRequest)
	if err := buildProjectsUpdateRequestFromArgs(c, r); err != nil {
		return err
	}

	ps := c.Projects()
	p, err := ps.Update(id, r)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Project{Projects: do.Projects{*p}})
}

// RunProjectsDelete deletes a Project with a given configuration.
func RunProjectsDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	ps := c.Projects()
	if force || AskForConfirmDelete("project", len(c.Args)) == nil {
		for _, id := range c.Args {
			if err := ps.Delete(id); err != nil {
				return err
			}
		}

		return nil
	}

	return fmt.Errorf("operation aborted")
}

// RunProjectResourcesList lists the Projects.
func RunProjectResourcesList(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]

	ps := c.Projects()
	list, err := ps.ListResources(id)
	if err != nil {
		return err
	}

	return c.Display(&displayers.ProjectResource{ProjectResources: list})
}

// RunProjectResourcesGet retrieves a Project Resource.
func RunProjectResourcesGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	urn := c.Args[0]

	parts, isValid := validateURN(urn)
	if !isValid {
		return fmt.Errorf(`URN must be in the format "do:<resource_type>:<resource_id>" but was %q`, urn)
	}

	c.Args = []string{parts[2]}
	switch parts[1] {
	case "droplet":
		return RunDropletGet(c)
	case "floatingip":
		return RunFloatingIPGet(c)
	case "loadbalancer":
		return RunLoadBalancerGet(c)
	case "domain":
		return RunDomainGet(c)
	case "volume":
		return RunVolumeGet(c)
	default:
		return fmt.Errorf("%q is an invalid resource type, consult the documentation", parts[1])
	}
}

// RunProjectResourcesAssign assigns a Project Resource.
func RunProjectResourcesAssign(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	projectUUID := c.Args[0]

	urns, err := c.Doit.GetStringSlice(c.NS, doctl.ArgProjectResource)
	if err != nil {
		return err
	}

	ps := c.Projects()
	list, err := ps.AssignResources(projectUUID, urns)
	if err != nil {
		return err
	}

	return c.Display(&displayers.ProjectResource{ProjectResources: list})
}

func validateURN(urn string) ([]string, bool) {
	parts := strings.Split(urn, ":")
	if len(parts) != 3 {
		return nil, false
	}

	if parts[0] != "do" {
		return nil, false
	}

	if strings.TrimSpace(parts[1]) == "" {
		return nil, false
	}

	if strings.TrimSpace(parts[2]) == "" {
		return nil, false
	}

	return parts, true
}

func buildProjectsCreateRequestFromArgs(c *CmdConfig, r *godo.CreateProjectRequest) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgProjectName)
	if err != nil {
		return err
	}
	r.Name = name

	purpose, err := c.Doit.GetString(c.NS, doctl.ArgProjectPurpose)
	if err != nil {
		return err
	}
	r.Purpose = purpose

	description, err := c.Doit.GetString(c.NS, doctl.ArgProjectDescription)
	if err != nil {
		return err
	}
	r.Description = description

	environment, err := c.Doit.GetString(c.NS, doctl.ArgProjectEnvironment)
	if err != nil {
		return err
	}
	r.Environment = environment

	return nil
}

func buildProjectsUpdateRequestFromArgs(c *CmdConfig, r *godo.UpdateProjectRequest) error {
	if c.Doit.IsSet(doctl.ArgProjectName) {
		name, err := c.Doit.GetString(c.NS, doctl.ArgProjectName)
		if err != nil {
			return err
		}
		r.Name = name
	}

	if c.Doit.IsSet(doctl.ArgProjectPurpose) {
		purpose, err := c.Doit.GetString(c.NS, doctl.ArgProjectPurpose)
		if err != nil {
			return err
		}
		r.Purpose = purpose
	}

	if c.Doit.IsSet(doctl.ArgProjectDescription) {
		description, err := c.Doit.GetString(c.NS, doctl.ArgProjectDescription)
		if err != nil {
			return err
		}
		r.Description = description
	}

	if c.Doit.IsSet(doctl.ArgProjectEnvironment) {
		environment, err := c.Doit.GetString(c.NS, doctl.ArgProjectEnvironment)
		if err != nil {
			return err
		}
		r.Environment = environment
	}

	if c.Doit.IsSet(doctl.ArgProjectIsDefault) {
		isDefault, err := c.Doit.GetBool(c.NS, doctl.ArgProjectIsDefault)
		if err != nil {
			return err
		}
		r.IsDefault = isDefault
	}

	return nil
}
