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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// AgentTriggers generates the `doctl harness-runtime triggers` subtree.
func AgentTriggers() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "triggers",
			Aliases: []string{"trigger"},
			Short:   "Manage webhook and cron triggers for hosted agent runs",
			Long:    agentsTriggersRootHelpMD,
		},
	}

	cmdList := CmdBuilder(cmd, RunAgentTriggersList, "list",
		"List triggers",
		agentsTriggersListHelpMD,
		Writer, agentPrettyErrors(), aliasOpt("ls"),
		displayerType(&displayers.HostedAgentTrigger{}))
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of triggers to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	AddStringFlag(cmdList, doctl.ArgAgentTriggerKind, "", "", "Filter by kind (webhook|cron)")
	AddStringFlag(cmdList, doctl.ArgAgentStatus, "", "", "Filter by status (active|paused)")
	cmdList.Example = `doctl harness-runtime triggers list --kind webhook --status active`

	cmdCreate := CmdBuilder(cmd, RunAgentTriggersCreate, "create",
		"Create a webhook or cron trigger",
		agentsTriggersCreateHelpMD,
		Writer, agentPrettyErrors(),
		displayerType(&displayers.HostedAgentTrigger{}))
	AddStringFlag(cmdCreate, doctl.ArgAgentTriggerKind, "", "", "Trigger kind (webhook|cron)", requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgAgentName, "", "", "Unique trigger name for the team (1–64 letters/digits/`-`/`.`/`_`, start and end alphanumeric, not a UUID)", requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgAgentTriggerSessionMode, "", "", "Session mode (fresh|reuse)", requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgAgentTriggerPrompt, "", "", "Prompt template sent on each fire", requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgAgentTriggerOutputMode, "", "none", "Output delivery mode (none|email|slack)")
	AddStringFlag(cmdCreate, doctl.ArgAgentTriggerOutputEmail, "", "", "Destination email when output-mode=email")
	AddStringFlag(cmdCreate, doctl.ArgAgentTriggerOutputSlackWebhook, "", "", "Slack incoming webhook URL when output-mode=slack")
	AddStringFlag(cmdCreate, doctl.ArgAgentSpec, "", "", "Path to agents.yaml manifest for session-mode=fresh (\"-\" reads stdin). ${VAR} references are resolved from the local environment at create time and stored expanded.")
	AddStringFlag(cmdCreate, doctl.ArgAgentTriggerBoundSessionID, "", "", "Paused session ID for session-mode=reuse")
	AddStringFlag(cmdCreate, doctl.ArgAgentTriggerProvider, "", "", "Webhook provider (github|gitlab|custom); default custom")
	AddStringFlag(cmdCreate, doctl.ArgAgentTriggerCronExpr, "", "", "Cron expression when kind=cron")
	AddStringFlag(cmdCreate, doctl.ArgAgentTriggerTimezone, "", "", "IANA timezone when kind=cron (e.g. America/New_York)")
	cmdCreate.Example = `doctl harness-runtime triggers create --kind webhook --name gh-ci --session-mode fresh --prompt 'Review this PR: {{payload}}' --spec ./agent.yaml --provider github`

	CmdBuilder(cmd, RunAgentTriggersGet, "get <trigger-id>",
		"Get a trigger",
		"Shows details for one trigger by ID.",
		Writer, agentPrettyErrors(), aliasOpt("show"),
		displayerType(&displayers.HostedAgentTrigger{}))

	cmdUpdate := CmdBuilder(cmd, RunAgentTriggersUpdate, "update <trigger-id>",
		"Update a trigger",
		agentsTriggersUpdateHelpMD,
		Writer, agentPrettyErrors(),
		displayerType(&displayers.HostedAgentTrigger{}))
	AddStringFlag(cmdUpdate, doctl.ArgAgentName, "", "", "New unique trigger name (same identifier rules as create)")
	AddStringFlag(cmdUpdate, doctl.ArgAgentStatus, "", "", "Lifecycle status (active|paused)")
	AddStringFlag(cmdUpdate, doctl.ArgAgentTriggerPrompt, "", "", "Updated prompt template")
	AddStringFlag(cmdUpdate, doctl.ArgAgentTriggerOutputMode, "", "", "Output delivery mode (none|email|slack)")
	AddStringFlag(cmdUpdate, doctl.ArgAgentTriggerOutputEmail, "", "", "Destination email when output-mode=email")
	AddStringFlag(cmdUpdate, doctl.ArgAgentTriggerOutputSlackWebhook, "", "", "Slack incoming webhook URL when output-mode=slack")
	AddStringFlag(cmdUpdate, doctl.ArgAgentSpec, "", "", "Updated agents.yaml manifest for fresh triggers (\"-\" reads stdin). ${VAR} references are resolved from the local environment at update time and stored expanded.")
	AddStringFlag(cmdUpdate, doctl.ArgAgentTriggerBoundSessionID, "", "", "Updated bound session ID for reuse triggers")
	AddStringFlag(cmdUpdate, doctl.ArgAgentTriggerCronExpr, "", "", "Updated cron expression")
	AddStringFlag(cmdUpdate, doctl.ArgAgentTriggerTimezone, "", "", "Updated IANA timezone")
	cmdUpdate.Example = `doctl harness-runtime triggers update TRIGGER_ID --status paused; doctl harness-runtime triggers update TRIGGER_ID --prompt 'New prompt'`

	cmdDelete := CmdBuilder(cmd, RunAgentTriggersDelete, "delete <trigger-id>",
		"Delete a trigger",
		agentsTriggersDeleteHelpMD,
		Writer, agentPrettyErrors(), aliasOpt("rm"))
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete without confirmation")
	cmdDelete.Example = `doctl harness-runtime triggers delete TRIGGER_ID --force`

	CmdBuilder(cmd, RunAgentTriggersPause, "pause <trigger-id>",
		"Pause a trigger",
		agentsTriggersPauseHelpMD,
		Writer, agentPrettyErrors(),
		displayerType(&displayers.HostedAgentTrigger{}))

	CmdBuilder(cmd, RunAgentTriggersResume, "resume <trigger-id>",
		"Resume a paused trigger",
		agentsTriggersResumeHelpMD,
		Writer, agentPrettyErrors(), aliasOpt("activate", "enable"),
		displayerType(&displayers.HostedAgentTrigger{}))

	CmdBuilder(cmd, RunAgentTriggersRotateSecret, "rotate-secret <trigger-id>",
		"Rotate a webhook trigger's secret",
		agentsTriggersRotateSecretHelpMD,
		Writer, agentPrettyErrors())

	cmdListExec := CmdBuilder(cmd, RunAgentTriggersListExecutions, "list-executions <trigger-id>",
		"List a trigger's execution history",
		agentsTriggersListExecutionsHelpMD,
		Writer, agentPrettyErrors(), aliasOpt("executions"),
		displayerType(&displayers.HostedAgentTriggerExecution{}))
	AddIntFlag(cmdListExec, doctl.ArgAgentPageSize, "", 0, "Maximum number of executions to return per page")
	AddStringFlag(cmdListExec, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	AddStringFlag(cmdListExec, doctl.ArgAgentStatus, "", "", "Filter by execution status (pending|running|succeeded|failed)")
	cmdListExec.Example = `doctl harness-runtime triggers list-executions TRIGGER_ID --status failed`

	CmdBuilder(cmd, RunAgentTriggersGetExecution, "get-execution <trigger-id> <execution-id>",
		"Get a single trigger execution",
		agentsTriggersGetExecutionHelpMD,
		Writer, agentPrettyErrors(),
		displayerType(&displayers.HostedAgentTriggerExecution{}))

	CmdBuilder(cmd, RunAgentTriggersGetBySession, "get-by-session <session-id>",
		"Look up the trigger for a session",
		agentsTriggersGetBySessionHelpMD,
		Writer, agentPrettyErrors(),
		displayerType(&displayers.HostedAgentTrigger{}))

	cmdReusable := CmdBuilder(cmd, RunAgentTriggersListReusableSessions, "list-reusable-sessions",
		"List paused sessions for reuse binding",
		agentsTriggersListReusableHelpMD,
		Writer, agentPrettyErrors(), aliasOpt("reusable-sessions"),
		displayerType(&displayers.HostedAgentReusableSession{}))
	AddIntFlag(cmdReusable, doctl.ArgAgentPageSize, "", 0, "Maximum number of sessions to return per page")
	AddStringFlag(cmdReusable, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")

	CmdBuilder(cmd, RunAgentTriggersListProviders, "list-providers",
		"List supported webhook providers",
		agentsTriggersListProvidersHelpMD,
		Writer, agentPrettyErrors(), aliasOpt("providers"),
		displayerType(&displayers.HostedAgentWebhookProvider{}))

	requireAgentSubcommand(cmd)
	return cmd
}

// --- runners ----------------------------------------------------------------

// RunAgentTriggersList lists team triggers.
func RunAgentTriggersList(c *CmdConfig) error {
	opt, err := agentTriggersListOptions(c)
	if err != nil {
		return err
	}
	triggers, next, err := c.HostedAgentTriggers().List(opt)
	if err != nil {
		return err
	}
	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentTrigger{Triggers: triggers}); err != nil {
			return err
		}
		return printNextPageToken(c, next)
	}
	stylingEnabled = detectStyling()
	printTriggersList(c.Out, triggers)
	printAgentNextPage(c.Out, next)
	return nil
}

// RunAgentTriggersCreate creates a webhook or cron trigger.
func RunAgentTriggersCreate(c *CmdConfig) error {
	req, err := agentTriggerCreateRequest(c)
	if err != nil {
		return err
	}
	result, err := c.HostedAgentTriggers().Create(req)
	if err != nil {
		return err
	}
	if Output == "json" {
		// JSON mode emits nothing but the document: the one-time secret travels
		// in the payload, so re-printing it as a human banner would only
		// duplicate a secret onto a second stream that callers may capture.
		type jsonCreateResult struct {
			*godo.HostedAgentTrigger
			WebhookSecret string `json:"webhook_secret,omitempty"`
		}
		var trigger *godo.HostedAgentTrigger
		if result.Trigger != nil {
			trigger = result.Trigger.HostedAgentTrigger
		}
		return json.NewEncoder(c.Out).Encode(jsonCreateResult{
			HostedAgentTrigger: trigger,
			WebhookSecret:      result.WebhookSecret,
		})
	}
	stylingEnabled = detectStyling()
	webhookURL := ""
	if result.Trigger != nil && result.Trigger.Webhook != nil {
		webhookURL = result.Trigger.Webhook.WebhookURL
	}
	printWebhookSecretCard(c.Out, result.WebhookSecret, webhookURL)
	if result.Trigger == nil {
		return nil
	}
	printTriggerCard(c.Out, result.Trigger, true)
	return nil
}

// RunAgentTriggersGet fetches one trigger.
func RunAgentTriggersGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	t, err := c.HostedAgentTriggers().Get(c.Args[0])
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentTrigger{Triggers: []do.HostedAgentTrigger{*t}, Single: true})
	}
	stylingEnabled = detectStyling()
	printTriggerCard(c.Out, t, false)
	return nil
}

// RunAgentTriggersUpdate partially updates a trigger.
func RunAgentTriggersUpdate(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	update, err := agentTriggerUpdateRequest(c)
	if err != nil {
		return err
	}
	t, err := c.HostedAgentTriggers().Update(c.Args[0], update)
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentTrigger{Triggers: []do.HostedAgentTrigger{*t}, Single: true})
	}
	stylingEnabled = detectStyling()
	printTriggerCard(c.Out, t, false)
	return nil
}

// RunAgentTriggersDelete soft-deletes a trigger.
func RunAgentTriggersDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if !(force || AskForConfirmDelete("trigger", 1) == nil) {
		return fmt.Errorf("operation aborted")
	}
	if err := c.HostedAgentTriggers().Delete(c.Args[0]); err != nil {
		return err
	}
	stylingEnabled = detectStyling()
	printAgentSuccess(c.Out, fmt.Sprintf("Deleted trigger %s", c.Args[0]))
	return nil
}

// RunAgentTriggersPause sets status=paused.
func RunAgentTriggersPause(c *CmdConfig) error {
	return agentTriggersSetStatus(c, godo.HostedAgentTriggerStatusPaused)
}

// RunAgentTriggersResume sets status=active.
func RunAgentTriggersResume(c *CmdConfig) error {
	return agentTriggersSetStatus(c, godo.HostedAgentTriggerStatusActive)
}

func agentTriggersSetStatus(c *CmdConfig, status godo.HostedAgentTriggerStatus) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	t, err := c.HostedAgentTriggers().Update(c.Args[0], &godo.HostedAgentTriggerUpdateRequest{Status: status})
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentTrigger{Triggers: []do.HostedAgentTrigger{*t}, Single: true})
	}
	stylingEnabled = detectStyling()
	printTriggerCard(c.Out, t, false)
	return nil
}

// RunAgentTriggersRotateSecret rotates a webhook secret.
func RunAgentTriggersRotateSecret(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	secret, err := c.HostedAgentTriggers().RotateSecret(c.Args[0])
	if err != nil {
		return err
	}
	if Output == "json" {
		return json.NewEncoder(c.Out).Encode(map[string]string{"webhook_secret": secret})
	}
	stylingEnabled = detectStyling()
	printWebhookSecretCard(c.Out, secret, "")
	return nil
}

// RunAgentTriggersListExecutions lists execution history for a trigger.
func RunAgentTriggersListExecutions(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	opt, err := agentTriggerExecutionListOptions(c)
	if err != nil {
		return err
	}
	execs, next, err := c.HostedAgentTriggers().ListExecutions(c.Args[0], opt)
	if err != nil {
		return err
	}
	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentTriggerExecution{Executions: execs}); err != nil {
			return err
		}
		return printNextPageToken(c, next)
	}
	stylingEnabled = detectStyling()
	printTriggerExecutionsList(c.Out, execs)
	printAgentNextPage(c.Out, next)
	return nil
}

// RunAgentTriggersGetExecution fetches one execution including payload/output.
func RunAgentTriggersGetExecution(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if len(c.Args) > 2 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	e, err := c.HostedAgentTriggers().GetExecution(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentTriggerExecution{Executions: []do.HostedAgentTriggerExecution{*e}, Single: true})
	}
	stylingEnabled = detectStyling()
	printTriggerExecutionCard(c.Out, e)
	return nil
}

// RunAgentTriggersGetBySession reverse-looks-up a trigger by session ID.
func RunAgentTriggersGetBySession(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	t, err := c.HostedAgentTriggers().GetBySession(c.Args[0])
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentTrigger{Triggers: []do.HostedAgentTrigger{*t}, Single: true})
	}
	stylingEnabled = detectStyling()
	printTriggerCard(c.Out, t, false)
	return nil
}

// RunAgentTriggersListReusableSessions lists PAUSED sessions for reuse binding.
func RunAgentTriggersListReusableSessions(c *CmdConfig) error {
	opt, err := agentReusableSessionListOptions(c)
	if err != nil {
		return err
	}
	sessions, next, err := c.HostedAgentTriggers().ListReusableSessions(opt)
	if err != nil {
		return err
	}
	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentReusableSession{Sessions: sessions}); err != nil {
			return err
		}
		return printNextPageToken(c, next)
	}
	stylingEnabled = detectStyling()
	printReusableSessionsList(c.Out, sessions)
	printAgentNextPage(c.Out, next)
	return nil
}

// RunAgentTriggersListProviders lists the webhook provider registry.
func RunAgentTriggersListProviders(c *CmdConfig) error {
	providers, err := c.HostedAgentTriggers().ListWebhookProviders()
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentWebhookProvider{Providers: providers})
	}
	stylingEnabled = detectStyling()
	printWebhookProvidersList(c.Out, providers)
	return nil
}

// --- helpers ----------------------------------------------------------------

func printNextPageToken(c *CmdConfig, next string) error {
	if next == "" {
		return nil
	}
	if Output == "json" {
		fmt.Fprintf(os.Stderr, "Next page token: %s\n", next)
	} else {
		fmt.Fprintf(c.Out, "\n%s %s\n", colorize("Next page token:", colMuted), next)
	}
	return nil
}

func agentTriggersListOptions(c *CmdConfig) (*godo.HostedAgentTriggerListOptions, error) {
	pageSize, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPageSize)
	if err != nil {
		return nil, err
	}
	pageToken, err := c.Doit.GetString(c.NS, doctl.ArgAgentPageToken)
	if err != nil {
		return nil, err
	}
	kind, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerKind)
	if err != nil {
		return nil, err
	}
	status, err := c.Doit.GetString(c.NS, doctl.ArgAgentStatus)
	if err != nil {
		return nil, err
	}
	if pageSize == 0 && pageToken == "" && kind == "" && status == "" {
		return nil, nil
	}
	opt := &godo.HostedAgentTriggerListOptions{}
	if pageSize > 0 {
		opt.PageSize = pageSize
	}
	if pageToken != "" {
		opt.PageToken = pageToken
	}
	if kind != "" {
		opt.Kind = godo.HostedAgentTriggerKind(strings.ToLower(kind))
	}
	if status != "" {
		opt.Status = godo.HostedAgentTriggerStatus(strings.ToLower(status))
	}
	return opt, nil
}

func agentTriggerExecutionListOptions(c *CmdConfig) (*godo.HostedAgentTriggerExecutionListOptions, error) {
	pageSize, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPageSize)
	if err != nil {
		return nil, err
	}
	pageToken, err := c.Doit.GetString(c.NS, doctl.ArgAgentPageToken)
	if err != nil {
		return nil, err
	}
	status, err := c.Doit.GetString(c.NS, doctl.ArgAgentStatus)
	if err != nil {
		return nil, err
	}
	if pageSize == 0 && pageToken == "" && status == "" {
		return nil, nil
	}
	opt := &godo.HostedAgentTriggerExecutionListOptions{}
	if pageSize > 0 {
		opt.PageSize = pageSize
	}
	if pageToken != "" {
		opt.PageToken = pageToken
	}
	if status != "" {
		opt.Status = godo.HostedAgentTriggerExecutionStatus(strings.ToLower(status))
	}
	return opt, nil
}

func agentReusableSessionListOptions(c *CmdConfig) (*godo.HostedAgentReusableSessionListOptions, error) {
	pageSize, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPageSize)
	if err != nil {
		return nil, err
	}
	pageToken, err := c.Doit.GetString(c.NS, doctl.ArgAgentPageToken)
	if err != nil {
		return nil, err
	}
	if pageSize == 0 && pageToken == "" {
		return nil, nil
	}
	opt := &godo.HostedAgentReusableSessionListOptions{}
	if pageSize > 0 {
		opt.PageSize = pageSize
	}
	if pageToken != "" {
		opt.PageToken = pageToken
	}
	return opt, nil
}

func agentTriggerCreateRequest(c *CmdConfig) (*godo.HostedAgentTriggerCreateRequest, error) {
	kind, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerKind)
	if err != nil {
		return nil, err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	if err != nil {
		return nil, err
	}
	if err := validateHostedAgentIdentifier(name); err != nil {
		return nil, err
	}
	sessionMode, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerSessionMode)
	if err != nil {
		return nil, err
	}
	prompt, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerPrompt)
	if err != nil {
		return nil, err
	}
	output, err := agentTriggerOutputWrite(c, true)
	if err != nil {
		return nil, err
	}

	req := &godo.HostedAgentTriggerCreateRequest{
		Kind:           godo.HostedAgentTriggerKind(strings.ToLower(kind)),
		Name:           name,
		SessionMode:    godo.HostedAgentTriggerSessionMode(strings.ToLower(sessionMode)),
		PromptTemplate: prompt,
		Output:         *output,
	}

	switch req.SessionMode {
	case godo.HostedAgentTriggerSessionModeFresh:
		specPath, err := c.Doit.GetString(c.NS, doctl.ArgAgentSpec)
		if err != nil {
			return nil, err
		}
		if specPath == "" {
			return nil, fmt.Errorf("--%s is required when --%s=fresh", doctl.ArgAgentSpec, doctl.ArgAgentTriggerSessionMode)
		}
		manifest, err := readManifest(os.Stdin, specPath)
		if err != nil {
			return nil, err
		}
		if err := reportAgentManifestValidation(validateAgentManifest(manifest)); err != nil {
			return nil, err
		}
		req.SessionTemplate = string(manifest)
	case godo.HostedAgentTriggerSessionModeReuse:
		bound, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerBoundSessionID)
		if err != nil {
			return nil, err
		}
		if bound == "" {
			return nil, fmt.Errorf("--%s is required when --%s=reuse", doctl.ArgAgentTriggerBoundSessionID, doctl.ArgAgentTriggerSessionMode)
		}
		req.BoundSessionID = bound
	default:
		return nil, fmt.Errorf("invalid --%s %q; want fresh or reuse", doctl.ArgAgentTriggerSessionMode, sessionMode)
	}

	switch req.Kind {
	case godo.HostedAgentTriggerKindWebhook:
		provider, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerProvider)
		if err != nil {
			return nil, err
		}
		if provider != "" {
			req.Webhook = &godo.HostedAgentCreateWebhookConfig{
				Provider: godo.HostedAgentWebhookProviderKey(strings.ToLower(provider)),
			}
		} else {
			req.Webhook = &godo.HostedAgentCreateWebhookConfig{}
		}
	case godo.HostedAgentTriggerKindCron:
		expr, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerCronExpr)
		if err != nil {
			return nil, err
		}
		tz, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerTimezone)
		if err != nil {
			return nil, err
		}
		if expr == "" || tz == "" {
			return nil, fmt.Errorf("--%s and --%s are required when --%s=cron", doctl.ArgAgentTriggerCronExpr, doctl.ArgAgentTriggerTimezone, doctl.ArgAgentTriggerKind)
		}
		req.Cron = &godo.HostedAgentCreateCronConfig{CronExpr: expr, Timezone: tz}
	default:
		return nil, fmt.Errorf("invalid --%s %q; want webhook or cron", doctl.ArgAgentTriggerKind, kind)
	}

	return req, nil
}

func agentTriggerUpdateRequest(c *CmdConfig) (*godo.HostedAgentTriggerUpdateRequest, error) {
	update := &godo.HostedAgentTriggerUpdateRequest{}
	changed := false

	if name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName); err != nil {
		return nil, err
	} else if name != "" {
		if err := validateHostedAgentIdentifier(name); err != nil {
			return nil, err
		}
		update.Name = name
		changed = true
	}
	if status, err := c.Doit.GetString(c.NS, doctl.ArgAgentStatus); err != nil {
		return nil, err
	} else if status != "" {
		update.Status = godo.HostedAgentTriggerStatus(strings.ToLower(status))
		changed = true
	}
	if prompt, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerPrompt); err != nil {
		return nil, err
	} else if prompt != "" {
		update.PromptTemplate = prompt
		changed = true
	}

	outputMode, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerOutputMode)
	if err != nil {
		return nil, err
	}
	if outputMode != "" {
		output, err := agentTriggerOutputWrite(c, true)
		if err != nil {
			return nil, err
		}
		update.Output = output
		changed = true
	}

	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAgentSpec)
	if err != nil {
		return nil, err
	}
	if specPath != "" {
		manifest, err := readManifest(os.Stdin, specPath)
		if err != nil {
			return nil, err
		}
		if err := reportAgentManifestValidation(validateAgentManifest(manifest)); err != nil {
			return nil, err
		}
		update.SessionTemplate = string(manifest)
		changed = true
	}

	bound, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerBoundSessionID)
	if err != nil {
		return nil, err
	}
	if bound != "" {
		update.BoundSessionID = bound
		changed = true
	}

	expr, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerCronExpr)
	if err != nil {
		return nil, err
	}
	tz, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerTimezone)
	if err != nil {
		return nil, err
	}
	if expr != "" || tz != "" {
		if expr == "" || tz == "" {
			return nil, fmt.Errorf("both --%s and --%s are required to update cron schedule", doctl.ArgAgentTriggerCronExpr, doctl.ArgAgentTriggerTimezone)
		}
		update.Cron = &godo.HostedAgentCreateCronConfig{CronExpr: expr, Timezone: tz}
		changed = true
	}

	if !changed {
		return nil, fmt.Errorf("provide at least one field to update")
	}
	return update, nil
}

func agentTriggerOutputWrite(c *CmdConfig, required bool) (*godo.HostedAgentTriggerOutputWrite, error) {
	mode, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerOutputMode)
	if err != nil {
		return nil, err
	}
	if mode == "" {
		if required {
			return nil, fmt.Errorf("--%s is required", doctl.ArgAgentTriggerOutputMode)
		}
		return nil, nil
	}
	out := &godo.HostedAgentTriggerOutputWrite{
		Mode: godo.HostedAgentTriggerOutputMode(strings.ToLower(mode)),
	}
	switch out.Mode {
	case godo.HostedAgentTriggerOutputModeNone:
		return out, nil
	case godo.HostedAgentTriggerOutputModeEmail:
		email, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerOutputEmail)
		if err != nil {
			return nil, err
		}
		if email == "" {
			return nil, fmt.Errorf("--%s is required when --%s=email", doctl.ArgAgentTriggerOutputEmail, doctl.ArgAgentTriggerOutputMode)
		}
		out.Email = email
		return out, nil
	case godo.HostedAgentTriggerOutputModeSlack:
		url, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerOutputSlackWebhook)
		if err != nil {
			return nil, err
		}
		if url == "" {
			return nil, fmt.Errorf("--%s is required when --%s=slack", doctl.ArgAgentTriggerOutputSlackWebhook, doctl.ArgAgentTriggerOutputMode)
		}
		out.Slack = &godo.HostedAgentTriggerSlackOutputWrite{WebhookURL: url}
		return out, nil
	default:
		return nil, fmt.Errorf("invalid --%s %q; want none, email, or slack", doctl.ArgAgentTriggerOutputMode, mode)
	}
}
