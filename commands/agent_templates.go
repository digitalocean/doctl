/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    15|See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	templateRefPageSize = 200

	baseTemplateCodingBase     = "coding-base"
	baseTemplateCodingCodex    = "coding-codex"
	baseTemplateCodingOpenCode = "coding-opencode"
)

// AgentTemplates generates the `doctl harness-runtime template` subtree, which
// wraps the godo team custom template API (/v2/agents/templates).
func AgentTemplates() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "template",
			Aliases: []string{"templates", "tpl"},
			Short:   "Manage team custom sandbox templates",
			Long:    agentsTemplatesRootHelpMD,
		},
	}

	ns := agentSubNS("agents.template")

	cmdCreate := CmdBuilder(cmd, RunAgentsTemplateCreate, "create",
		"Create a team custom template",
		agentsTemplatesCreateHelpMD,
		Writer, append(ns, aliasOpt("c"),
			displayerType(&displayers.HostedAgentTemplate{}))...)
	AddStringFlag(cmdCreate, doctl.ArgAgentName, "", "", "Team-unique name for the template", requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgAgentBaseTemplate, "", "", "Platform base to rebase onto (coding-base, coding-codex, coding-opencode)", requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgAgentSourceOCIRef, "", "", "Customer OCI image (registry/repo:tag or digest)", requiredOpt())
	cmdCreate.Example = `doctl harness-runtime template create --name my-image --base-template coding-opencode --source-oci-ref registry.digitalocean.com/myreg/agent:latest`

	cmdList := CmdBuilder(cmd, RunAgentsTemplateList, "list",
		"List team custom templates",
		agentsTemplatesListHelpMD,
		Writer, append(ns, aliasOpt("ls"),
			displayerType(&displayers.HostedAgentTemplate{}))...)
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of templates to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	cmdList.Example = `doctl harness-runtime template list --page-size 10`

	CmdBuilder(cmd, RunAgentsTemplateGet, "get <template>",
		"Get a team custom template",
		agentsTemplatesGetHelpMD,
		Writer, append(ns, aliasOpt("show"),
			displayerType(&displayers.HostedAgentTemplate{}))...)

	cmdUpdate := CmdBuilder(cmd, RunAgentsTemplateUpdate, "update <template>",
		"Update a template and kick a new build",
		agentsTemplatesUpdateHelpMD,
		Writer, append(ns, aliasOpt("u"),
			displayerType(&displayers.HostedAgentTemplate{}))...)
	AddStringFlag(cmdUpdate, doctl.ArgAgentSourceOCIRef, "", "", "New customer OCI image (registry/repo:tag or digest)")
	AddStringFlag(cmdUpdate, doctl.ArgAgentBaseTemplate, "", "", "New platform base (coding-base, coding-codex, coding-opencode)")
	cmdUpdate.Example = `doctl harness-runtime template update my-image --source-oci-ref registry.digitalocean.com/myreg/agent:v2`

	CmdBuilder(cmd, RunAgentsTemplateDelete, "delete <template>",
		"Delete a team custom template",
		agentsTemplatesDeleteHelpMD,
		Writer, append(ns, aliasOpt("rm"))...)

	cmdBuilds := CmdBuilder(cmd, RunAgentsTemplateListBuilds, "list-builds <template>",
		"List builds for a template",
		agentsTemplatesListBuildsHelpMD,
		Writer, append(ns, aliasOpt("builds"),
			displayerType(&displayers.HostedAgentTemplateBuild{}))...)
	AddIntFlag(cmdBuilds, doctl.ArgAgentPageSize, "", 0, "Maximum number of builds to return per page")
	AddStringFlag(cmdBuilds, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	cmdBuilds.Example = `doctl harness-runtime template list-builds my-image`

	CmdBuilder(cmd, RunAgentsTemplateGetBuild, "get-build <template> <build-id>",
		"Get a template build",
		agentsTemplatesGetBuildHelpMD,
		Writer, append(ns, aliasOpt("show-build"),
			displayerType(&displayers.HostedAgentTemplateBuild{}))...)

	requireAgentSubcommand(cmd)
	return cmd
}

// RunAgentsTemplateCreate creates a team custom template and kicks a build.
func RunAgentsTemplateCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	if err != nil {
		return err
	}
	base, err := c.Doit.GetString(c.NS, doctl.ArgAgentBaseTemplate)
	if err != nil {
		return err
	}
	if err := validateBaseTemplate(base); err != nil {
		return err
	}
	src, err := c.Doit.GetString(c.NS, doctl.ArgAgentSourceOCIRef)
	if err != nil {
		return err
	}
	tpl, err := c.HostedAgents().CreateTemplate(&godo.HostedAgentTemplateCreateRequest{
		Name:         name,
		BaseTemplate: base,
		SourceOCIRef: src,
	})
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentTemplate{Templates: []godo.HostedAgentTemplate{*tpl}, Single: true})
	}
	stylingEnabled = detectStyling()
	printTemplateCard(c.Out, tpl, true)
	return nil
}

// RunAgentsTemplateList lists team custom templates.
func RunAgentsTemplateList(c *CmdConfig) error {
	opt := &godo.HostedAgentTemplateListOptions{}
	pageSize, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPageSize)
	if err != nil {
		return err
	}
	opt.PageSize = pageSize
	pageToken, err := c.Doit.GetString(c.NS, doctl.ArgAgentPageToken)
	if err != nil {
		return err
	}
	opt.PageToken = pageToken

	templates, next, err := c.HostedAgents().ListTemplates(opt)
	if err != nil {
		return err
	}
	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentTemplate{Templates: templates}); err != nil {
			return err
		}
		if next != "" {
			fmt.Fprintf(os.Stderr, "Next page token: %s\n", next)
		}
		return nil
	}
	stylingEnabled = detectStyling()
	printTemplatesList(c.Out, templates)
	printAgentNextPage(c.Out, next)
	return nil
}

// RunAgentsTemplateGet fetches one team custom template.
func RunAgentsTemplateGet(c *CmdConfig) error {
	templateID, err := templateIDArg(c)
	if err != nil {
		return err
	}
	tpl, err := c.HostedAgents().GetTemplate(templateID)
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentTemplate{Templates: []godo.HostedAgentTemplate{*tpl}, Single: true})
	}
	stylingEnabled = detectStyling()
	printTemplateCard(c.Out, tpl, false)
	return nil
}

// RunAgentsTemplateUpdate updates a template source and kicks a new build.
func RunAgentsTemplateUpdate(c *CmdConfig) error {
	templateID, err := templateIDArg(c)
	if err != nil {
		return err
	}
	src, err := c.Doit.GetString(c.NS, doctl.ArgAgentSourceOCIRef)
	if err != nil {
		return err
	}
	base, err := c.Doit.GetString(c.NS, doctl.ArgAgentBaseTemplate)
	if err != nil {
		return err
	}
	if src == "" && base == "" {
		return fmt.Errorf("pass --%s and/or --%s", doctl.ArgAgentSourceOCIRef, doctl.ArgAgentBaseTemplate)
	}
	if base != "" {
		if err := validateBaseTemplate(base); err != nil {
			return err
		}
	}
	tpl, err := c.HostedAgents().UpdateTemplate(templateID, &godo.HostedAgentTemplateUpdateRequest{
		SourceOCIRef: src,
		BaseTemplate: base,
	})
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentTemplate{Templates: []godo.HostedAgentTemplate{*tpl}, Single: true})
	}
	stylingEnabled = detectStyling()
	printTemplateCard(c.Out, tpl, false)
	return nil
}

// RunAgentsTemplateDelete deletes a team custom template.
func RunAgentsTemplateDelete(c *CmdConfig) error {
	templateID, err := templateIDArg(c)
	if err != nil {
		return err
	}
	resp, err := c.HostedAgents().DeleteTemplate(templateID)
	if err != nil {
		return err
	}
	stylingEnabled = detectStyling()
	id := templateID
	if resp != nil && resp.TemplateID != "" {
		id = resp.TemplateID
	}
	printAgentSuccess(c.Out, fmt.Sprintf("Deleted template %s", id))
	return nil
}

// RunAgentsTemplateListBuilds lists build history for a template.
func RunAgentsTemplateListBuilds(c *CmdConfig) error {
	templateID, err := templateIDArg(c)
	if err != nil {
		return err
	}
	opt := &godo.HostedAgentTemplateBuildListOptions{}
	pageSize, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPageSize)
	if err != nil {
		return err
	}
	opt.PageSize = pageSize
	pageToken, err := c.Doit.GetString(c.NS, doctl.ArgAgentPageToken)
	if err != nil {
		return err
	}
	opt.PageToken = pageToken

	builds, next, err := c.HostedAgents().ListTemplateBuilds(templateID, opt)
	if err != nil {
		return err
	}
	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentTemplateBuild{Builds: builds}); err != nil {
			return err
		}
		if next != "" {
			fmt.Fprintf(os.Stderr, "Next page token: %s\n", next)
		}
		return nil
	}
	stylingEnabled = detectStyling()
	printTemplateBuildsList(c.Out, builds)
	printAgentNextPage(c.Out, next)
	return nil
}

// RunAgentsTemplateGetBuild fetches one template build.
func RunAgentsTemplateGetBuild(c *CmdConfig) error {
	templateID, buildID, err := templateBuildArgs(c)
	if err != nil {
		return err
	}
	build, err := c.HostedAgents().GetTemplateBuild(templateID, buildID)
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentTemplateBuild{Builds: []godo.HostedAgentTemplateBuild{*build}, Single: true})
	}
	stylingEnabled = detectStyling()
	printTemplateBuildCard(c.Out, build)
	return nil
}

func validateBaseTemplate(base string) error {
	switch base {
	case baseTemplateCodingBase, baseTemplateCodingCodex, baseTemplateCodingOpenCode:
		return nil
	default:
		return fmt.Errorf("base-template must be one of %s, %s, %s",
			baseTemplateCodingBase, baseTemplateCodingCodex, baseTemplateCodingOpenCode)
	}
}

func looksLikeTemplateID(ref string) bool {
	return sessionUUIDRe.MatchString(ref)
}

func resolveTemplateRef(svc do.HostedAgentsService, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("a template ID or name is required")
	}
	if looksLikeTemplateID(ref) {
		return ref, nil
	}

	var matches []godo.HostedAgentTemplate
	pageToken := ""
	for {
		templates, next, err := svc.ListTemplates(&godo.HostedAgentTemplateListOptions{
			PageSize:  templateRefPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			return "", fmt.Errorf("resolving template name %q: %w", ref, err)
		}
		for _, tpl := range templates {
			if strings.EqualFold(strings.TrimSpace(tpl.Name), ref) {
				matches = append(matches, tpl)
			}
		}
		if next == "" || len(templates) == 0 {
			break
		}
		pageToken = next
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no template goes by the name %q; pass a template ID or run `%s template list` to see available templates", ref, agentCLI)
	case 1:
		return matches[0].TemplateID, nil
	default:
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, m.TemplateID)
		}
		return "", fmt.Errorf("many templates go by the name %q, they have the following IDs: %s", ref, strings.Join(ids, ", "))
	}
}

func templateIDArg(c *CmdConfig) (string, error) {
	if len(c.Args) < 1 {
		return "", doctl.NewMissingArgsErr(c.NS)
	}
	return resolveTemplateRef(c.HostedAgents(), c.Args[0])
}

func templateBuildArgs(c *CmdConfig) (templateID, buildID string, err error) {
	if len(c.Args) < 2 {
		return "", "", doctl.NewMissingArgsErr(c.NS)
	}
	templateID, err = resolveTemplateRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return "", "", err
	}
	buildID = strings.TrimSpace(c.Args[1])
	if buildID == "" {
		return "", "", doctl.NewMissingArgsErr(c.NS)
	}
	return templateID, buildID, nil
}

func printTemplatesList(w io.Writer, templates []godo.HostedAgentTemplate) {
	if len(templates) == 0 {
		fmt.Fprintln(w, colorize("No templates", colMuted))
		return
	}
	noun := "templates"
	if len(templates) == 1 {
		noun = "template"
	}
	fmt.Fprintln(w, boldColor(fmt.Sprintf("%d %s", len(templates), noun), colHighlight))
	fmt.Fprintln(w)

	for i, tpl := range templates {
		if i > 0 {
			fmt.Fprintln(w)
		}
		printTemplateListItem(w, &tpl)
	}
}

func printTemplateListItem(w io.Writer, tpl *godo.HostedAgentTemplate) {
	if tpl == nil {
		return
	}
	name := strings.TrimSpace(tpl.Name)
	if name == "" {
		name = tpl.TemplateID
	}
	fmt.Fprintf(w, "%s %s\n", templateStatusGlyph(tpl.Status), boldColor(name, colHighlight))
	meta := []string{colorizeTemplateStatus(tpl.Status)}
	if base := templateBase(tpl); base != "" {
		meta = append(meta, colorize(base, colMuted))
	}
	if !tpl.CreatedAt.Time.IsZero() {
		meta = append(meta, colorize(createdAgo(tpl.CreatedAt.Time), colMuted))
	}
	fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
	if id := strings.TrimSpace(tpl.TemplateID); id != "" && id != name {
		fmt.Fprintf(w, "  %s\n", colorize(id, colMuted))
	}
}

func printTemplateCard(w io.Writer, tpl *godo.HostedAgentTemplate, created bool) {
	if tpl == nil {
		fmt.Fprintln(w, colorize("No template", colMuted))
		return
	}
	name := strings.TrimSpace(tpl.Name)
	if name == "" {
		name = tpl.TemplateID
	}

	var body strings.Builder
	if created {
		fmt.Fprintf(&body, "%s\n\n", boldColor("Template created", colSuccess))
	}
	body.WriteString(cardRow("Name", name))
	if id := strings.TrimSpace(tpl.TemplateID); id != "" && id != name {
		body.WriteString(cardRow("ID", colorize(id, colMuted)))
	}
	body.WriteString(cardRow("Status", templateStatusGlyph(tpl.Status)+" "+colorizeTemplateStatus(tpl.Status)))
	if base := templateBase(tpl); base != "" {
		body.WriteString(cardRow("Base", colorize(base, colMuted)))
	}
	if img := templateImageRef(tpl); img != "" {
		body.WriteString(cardRow("Image", colorize(img, colMuted)))
	}
	if !tpl.CreatedAt.Time.IsZero() {
		body.WriteString(cardRow("Created", colorize(formatCreatedAt(tpl.CreatedAt.Time), colMuted)))
	}
	if !tpl.UpdatedAt.Time.IsZero() {
		body.WriteString(cardRow("Updated", colorize(formatCreatedAt(tpl.UpdatedAt.Time), colMuted)))
	}
	if ref := name; strings.TrimSpace(ref) != "" {
		fmt.Fprintln(&body)
		fmt.Fprintln(&body, colorize("Next step", colMuted))
		body.WriteString(cardRow("builds", agentCLI+" template list-builds "+ref))
	}
	renderAgentCard(w, body.String())
}

func printTemplateBuildsList(w io.Writer, builds []godo.HostedAgentTemplateBuild) {
	if len(builds) == 0 {
		fmt.Fprintln(w, colorize("No builds", colMuted))
		return
	}
	noun := "builds"
	if len(builds) == 1 {
		noun = "build"
	}
	fmt.Fprintln(w, boldColor(fmt.Sprintf("%d %s", len(builds), noun), colHighlight))
	fmt.Fprintln(w)

	for i, b := range builds {
		if i > 0 {
			fmt.Fprintln(w)
		}
		printTemplateBuildListItem(w, &b)
	}
}

func printTemplateBuildListItem(w io.Writer, b *godo.HostedAgentTemplateBuild) {
	if b == nil {
		return
	}
	title := strings.TrimSpace(b.Name)
	if title == "" {
		title = b.BuildID
	}
	fmt.Fprintf(w, "%s %s\n", templateBuildStatusGlyph(b.Status), boldColor(title, colHighlight))
	meta := []string{colorizeTemplateBuildStatus(b.Status)}
	if !b.CreatedAt.Time.IsZero() {
		meta = append(meta, colorize(createdAgo(b.CreatedAt.Time), colMuted))
	}
	fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
	if id := strings.TrimSpace(b.BuildID); id != "" && id != title {
		fmt.Fprintf(w, "  %s\n", colorize(id, colMuted))
	}
	if errMsg := strings.TrimSpace(b.Error); errMsg != "" {
		fmt.Fprintf(w, "  %s %s\n", colorize("fail", colError), colorize(errMsg, colMuted))
	}
}

func printTemplateBuildCard(w io.Writer, b *godo.HostedAgentTemplateBuild) {
	if b == nil {
		fmt.Fprintln(w, colorize("No build", colMuted))
		return
	}
	title := strings.TrimSpace(b.Name)
	if title == "" {
		title = b.BuildID
	}
	var body strings.Builder
	body.WriteString(cardRow("Name", title))
	if id := strings.TrimSpace(b.BuildID); id != "" && id != title {
		body.WriteString(cardRow("ID", colorize(id, colMuted)))
	}
	if tid := strings.TrimSpace(b.TemplateID); tid != "" {
		body.WriteString(cardRow("Template", colorize(tid, colMuted)))
	}
	body.WriteString(cardRow("Status", templateBuildStatusGlyph(b.Status)+" "+colorizeTemplateBuildStatus(b.Status)))
	if b.Spec != nil && strings.TrimSpace(b.Spec.BaseTemplate) != "" {
		body.WriteString(cardRow("Base", colorize(b.Spec.BaseTemplate, colMuted)))
	}
	if errMsg := strings.TrimSpace(b.Error); errMsg != "" {
		body.WriteString(cardRow("Error", colorize(errMsg, colError)))
	}
	if !b.CreatedAt.Time.IsZero() {
		body.WriteString(cardRow("Created", colorize(formatCreatedAt(b.CreatedAt.Time), colMuted)))
	}
	renderAgentCard(w, body.String())
}

func templateBase(tpl *godo.HostedAgentTemplate) string {
	if tpl == nil || tpl.Spec == nil {
		return ""
	}
	return strings.TrimSpace(tpl.Spec.BaseTemplate)
}

func templateImageRef(tpl *godo.HostedAgentTemplate) string {
	if tpl == nil || tpl.Spec == nil || tpl.Spec.Image == nil {
		return ""
	}
	img := tpl.Spec.Image
	ref := strings.TrimSpace(img.Registry)
	if repo := strings.TrimSpace(img.Repository); repo != "" {
		if ref != "" {
			ref += "/" + repo
		} else {
			ref = repo
		}
	}
	if digest := strings.TrimSpace(img.Digest); digest != "" {
		if ref != "" {
			return ref + "@" + digest
		}
		return digest
	}
	if tag := strings.TrimSpace(img.Tag); tag != "" {
		if ref != "" {
			return ref + ":" + tag
		}
		return tag
	}
	return ref
}

func prettyTemplateStatus(status godo.HostedAgentTemplateStatus) string {
	s := strings.TrimPrefix(string(status), "TEMPLATE_STATUS_")
	return strings.ToLower(s)
}

func prettyTemplateBuildStatus(status godo.HostedAgentTemplateBuildStatus) string {
	return strings.ToLower(string(status))
}

func templateStatusGlyph(status godo.HostedAgentTemplateStatus) string {
	switch status {
	case godo.HostedAgentTemplateStatusReady:
		return colorize("●", colSuccess)
	case godo.HostedAgentTemplateStatusPending, godo.HostedAgentTemplateStatusBuilding:
		return colorize("…", colWarning)
	case godo.HostedAgentTemplateStatusFailed:
		return colorize("✗", colError)
	default:
		return colorize("·", colMuted)
	}
}

func colorizeTemplateStatus(status godo.HostedAgentTemplateStatus) string {
	label := prettyTemplateStatus(status)
	switch status {
	case godo.HostedAgentTemplateStatusReady:
		return colorize(label, colSuccess)
	case godo.HostedAgentTemplateStatusPending, godo.HostedAgentTemplateStatusBuilding:
		return colorize(label, colWarning)
	case godo.HostedAgentTemplateStatusFailed:
		return colorize(label, colError)
	default:
		return colorize(label, colMuted)
	}
}

func templateBuildStatusGlyph(status godo.HostedAgentTemplateBuildStatus) string {
	switch status {
	case godo.HostedAgentTemplateBuildStatusSucceeded:
		return colorize("●", colSuccess)
	case godo.HostedAgentTemplateBuildStatusPending, godo.HostedAgentTemplateBuildStatusBuilding:
		return colorize("…", colWarning)
	case godo.HostedAgentTemplateBuildStatusFailed:
		return colorize("✗", colError)
	default:
		return colorize("·", colMuted)
	}
}

func colorizeTemplateBuildStatus(status godo.HostedAgentTemplateBuildStatus) string {
	label := prettyTemplateBuildStatus(status)
	switch status {
	case godo.HostedAgentTemplateBuildStatusSucceeded:
		return colorize(label, colSuccess)
	case godo.HostedAgentTemplateBuildStatusPending, godo.HostedAgentTemplateBuildStatusBuilding:
		return colorize(label, colWarning)
	case godo.HostedAgentTemplateBuildStatusFailed:
		return colorize(label, colError)
	default:
		return colorize(label, colMuted)
	}
}
