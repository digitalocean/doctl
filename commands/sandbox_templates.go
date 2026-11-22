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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const sandboxTemplateWaitInterval = 5 * time.Second

// addSandboxTemplateCmds adds the `template` group to `doctl sandbox`.
func addSandboxTemplateCmds(parent *Command) {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "template",
			Aliases: []string{"templates"},
			Short:   "Manage sandbox templates",
			Long: "The subcommands of `doctl sandbox template` manage your team's custom sandbox templates. " +
				"A template is built by rebasing an OCI image onto a base template, such as `" + defaultSandboxTemplate + "`. " +
				"Once it is ready, pass its name to `doctl sandbox create --template`. Global base templates are not listed here.",
		},
	}
	parent.AddCommand(cmd)

	cmdCreate := CmdBuilder(cmd, RunSandboxTemplateCreate, "create <name>",
		"Create a template",
		"Builds a custom template from an OCI image and returns as soon as the build starts. "+
			"Add `--wait` to wait until the template is ready.",
		Writer, aliasOpt("c"), displayerType(&displayers.SandboxTemplate{}))
	AddStringFlag(cmdCreate, doctl.ArgSandboxImage, "", "",
		"The OCI image to build from, as `registry/repository:tag` or `registry/repository@digest`", requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgSandboxBaseTemplate, "", defaultSandboxTemplate,
		"The base template to rebase the image onto")
	AddWaitFlagsWithTimeout(cmdCreate, false, "Wait for the template build to finish before returning", 15*time.Minute)
	cmdCreate.Example = "The following example builds a template from an image and waits for it: " +
		"doctl sandbox template create my-template --image registry.digitalocean.com/my-team/my-image:v1 --wait"

	cmdList := CmdBuilder(cmd, RunSandboxTemplateList, "list",
		"List your custom templates",
		"Lists your team's custom templates. Global base templates are not included.",
		Writer, aliasOpt("ls"), displayerType(&displayers.SandboxTemplate{}))
	cmdList.Example = "The following example lists your templates: doctl sandbox template list"

	cmdGet := CmdBuilder(cmd, RunSandboxTemplateGet, "get <template-id>",
		"Retrieve a template",
		"Retrieves a custom template: its name, status, base template and image.",
		Writer, aliasOpt("g"), displayerType(&displayers.SandboxTemplate{}))
	cmdGet.Example = "The following example shows only the status of a template: " +
		"doctl sandbox template get 11111111-2222-4333-8444-555555555555 --format Name,Status"

	cmdUpdate := CmdBuilder(cmd, RunSandboxTemplateUpdate, "update <template-id>",
		"Update a template",
		"Changes a template's base template and/or image and starts a new build. Fields you leave out keep their values.",
		Writer, aliasOpt("u"), displayerType(&displayers.SandboxTemplate{}))
	AddStringFlag(cmdUpdate, doctl.ArgSandboxImage, "", "", "The new OCI image to build from")
	AddStringFlag(cmdUpdate, doctl.ArgSandboxBaseTemplate, "", "", "The new base template")
	AddWaitFlagsWithTimeout(cmdUpdate, false, "Wait for the template build to finish before returning", 15*time.Minute)
	cmdUpdate.Example = "The following example moves a template to a new image tag: " +
		"doctl sandbox template update 11111111-2222-4333-8444-555555555555 --image registry.digitalocean.com/my-team/my-image:v2"

	cmdDelete := CmdBuilder(cmd, RunSandboxTemplateDelete, "delete <template-id>...",
		"Permanently delete one or more templates",
		"Permanently deletes the templates you name. This is irreversible.",
		Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the template(s) without a confirmation prompt")
	cmdDelete.Example = "The following example deletes a template without asking: " +
		"doctl sandbox template delete 11111111-2222-4333-8444-555555555555 --force"

	build := &Command{
		Command: &cobra.Command{
			Use:     "build",
			Aliases: []string{"builds"},
			Short:   "Inspect template builds",
			Long:    "The subcommands of `doctl sandbox template build` list and show the builds of a template.",
		},
	}
	cmd.AddCommand(build)

	cmdBuildList := CmdBuilder(build, RunSandboxTemplateBuildList, "list <template-id>",
		"List the builds of a template",
		"Lists the builds of a template with their status and, for a failed build, the error.",
		Writer, aliasOpt("ls"), displayerType(&displayers.SandboxTemplateBuild{}))
	cmdBuildList.Example = "The following example lists the builds of a template: doctl sandbox template build list 11111111-2222-4333-8444-555555555555"

	cmdBuildGet := CmdBuilder(build, RunSandboxTemplateBuildGet, "get <template-id> <build-id>",
		"Retrieve a template build",
		"Retrieves one build of a template.",
		Writer, aliasOpt("g"), displayerType(&displayers.SandboxTemplateBuild{}))
	cmdBuildGet.Example = "The following example shows one build: doctl sandbox template build get 11111111-2222-4333-8444-555555555555 22222222-3333-4444-8555-666666666666"
}

// RunSandboxTemplateCreate creates a template.
func RunSandboxTemplateCreate(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	name := strings.TrimSpace(c.Args[0])
	if name == "" {
		return doctl.NewMissingArgsErr("the template name must not be empty")
	}
	image, err := c.Doit.GetString(c.NS, doctl.ArgSandboxImage)
	if err != nil {
		return err
	}
	if strings.TrimSpace(image) == "" {
		return doctl.NewMissingArgsErr("--image is required")
	}
	base, err := c.Doit.GetString(c.NS, doctl.ArgSandboxBaseTemplate)
	if err != nil {
		return err
	}
	if base == "" {
		base = defaultSandboxTemplate
	}
	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	svc := c.Sandboxes()
	tpl, err := svc.CreateTemplate(&godo.SandboxTemplateCreateRequest{Name: name, BaseTemplate: base, SourceOCIRef: image})
	if err != nil {
		return err
	}
	if wait {
		if tpl, err = waitForSandboxTemplate(c, svc, tpl); err != nil {
			return err
		}
	}
	return c.Display(&displayers.SandboxTemplate{Templates: do.SandboxTemplates{*tpl}})
}

// RunSandboxTemplateList lists templates.
func RunSandboxTemplateList(c *CmdConfig) error {
	if len(c.Args) != 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	list, err := c.Sandboxes().ListTemplates()
	if err != nil {
		return err
	}
	return c.Display(&displayers.SandboxTemplate{Templates: list})
}

// RunSandboxTemplateGet retrieves a template.
func RunSandboxTemplateGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	tpl, err := c.Sandboxes().GetTemplate(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.SandboxTemplate{Templates: do.SandboxTemplates{*tpl}})
}

// RunSandboxTemplateUpdate updates a template and starts a new build.
func RunSandboxTemplateUpdate(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	image, err := c.Doit.GetString(c.NS, doctl.ArgSandboxImage)
	if err != nil {
		return err
	}
	base, err := c.Doit.GetString(c.NS, doctl.ArgSandboxBaseTemplate)
	if err != nil {
		return err
	}
	if strings.TrimSpace(image) == "" && strings.TrimSpace(base) == "" {
		return doctl.NewMissingArgsErr("pass --image and/or --base-template")
	}
	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	svc := c.Sandboxes()
	tpl, err := svc.UpdateTemplate(c.Args[0], &godo.SandboxTemplateUpdateRequest{BaseTemplate: base, SourceOCIRef: image})
	if err != nil {
		return err
	}
	if wait {
		if tpl, err = waitForSandboxTemplate(c, svc, tpl); err != nil {
			return err
		}
	}
	return c.Display(&displayers.SandboxTemplate{Templates: do.SandboxTemplates{*tpl}})
}

// RunSandboxTemplateDelete deletes one or more templates.
func RunSandboxTemplateDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if err := confirmDelete(force, "sandbox template", len(c.Args)); err != nil {
		return err
	}
	svc := c.Sandboxes()
	for _, id := range c.Args {
		if err := svc.DeleteTemplate(id); err != nil {
			return fmt.Errorf("deleting template %s: %w", id, err)
		}
	}
	return nil
}

// RunSandboxTemplateBuildList lists the builds of a template.
func RunSandboxTemplateBuildList(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	list, err := c.Sandboxes().ListTemplateBuilds(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.SandboxTemplateBuild{Builds: list})
}

// RunSandboxTemplateBuildGet retrieves one build of a template.
func RunSandboxTemplateBuildGet(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if len(c.Args) > 2 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	b, err := c.Sandboxes().GetTemplateBuild(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}
	return c.Display(&displayers.SandboxTemplateBuild{Builds: do.SandboxTemplateBuilds{*b}})
}

func waitForSandboxTemplate(c *CmdConfig, svc do.SandboxesService, tpl *do.SandboxTemplate) (*do.SandboxTemplate, error) {
	w, err := newWaiter(c)
	if err != nil {
		return nil, err
	}
	return waitForTemplate(w, svc, tpl, sandboxTemplateWaitInterval)
}

// waitForTemplate polls a template until it is ready. A template whose build
// failed is reported as a failure, with the newest failed build's error.
func waitForTemplate(w waiter, svc do.SandboxesService, tpl *do.SandboxTemplate, interval time.Duration) (*do.SandboxTemplate, error) {
	current := tpl
	err := w.wait(waitOp{
		Activity: fmt.Sprintf("Building template (%s)", tpl.Name),
		Subject:  fmt.Sprintf("template (%s) to be ready", tpl.Name),
		Success:  fmt.Sprintf("Template (%s) is ready", tpl.Name),
		Interval: interval,
	}, func() (bool, string, error) {
		t, err := svc.GetTemplate(tpl.ID)
		if err != nil {
			return false, "", err
		}
		current = t
		switch t.Status {
		case godo.SandboxTemplateStatusReady:
			return true, "", nil
		case godo.SandboxTemplateStatusFailed:
			return false, "", sandboxTemplateFailure(svc, t)
		}
		return false, displayers.FormatSandboxTemplateStatus(t.Status), nil
	})
	if err != nil {
		return nil, err
	}
	return current, nil
}

func sandboxTemplateFailure(svc do.SandboxesService, t *do.SandboxTemplate) error {
	msg := fmt.Sprintf("template %s (%s) failed to build", t.Name, t.ID)
	builds, err := svc.ListTemplateBuilds(t.ID)
	if err != nil {
		return errors.New(msg)
	}
	var newest *godo.SandboxTemplateBuild
	for _, b := range builds {
		if b.Status == godo.SandboxTemplateBuildStatusFailed && b.Error != "" &&
			(newest == nil || b.UpdatedAt.Time.After(newest.UpdatedAt.Time)) {
			newest = b.SandboxTemplateBuild
		}
	}
	if newest != nil {
		return fmt.Errorf("%s: %s", msg, newest.Error)
	}
	return errors.New(msg)
}
