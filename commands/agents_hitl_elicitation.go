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
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// hitlElicitationShape classifies a HITLRequested payload into the interaction
// it needs. command_execution-style approvals (bash, GitHub writes) carry no
// mode and stay a plain three-way approve/reject/defer verdict; MCP
// elicitations add a `mode` field and, for forms, a `requestedSchema` that
// may or may not ask for actual values.
type hitlElicitationShape int

const (
	// hitlShapeCommand is the existing approve/reject/defer verdict — bash,
	// GitHub commit/PR/branch actions. No `mode`, no `content` to collect.
	hitlShapeCommand hitlElicitationShape = iota
	// hitlShapeURL is an MCP OAuth/connect elicitation: a link to open and a
	// verification code to confirm, no fields to fill in.
	hitlShapeURL
	// hitlShapeApproval is an MCP elicitation with mode "form" and an empty
	// requestedSchema — a yes/no tool-call approval with nothing to fill in.
	hitlShapeApproval
	// hitlShapeForm is an MCP elicitation with mode "form" and a non-empty
	// requestedSchema — the caller must supply values matching it.
	hitlShapeForm
)

const mcpElicitationKind = "mcp_elicitation"

// isMCPElicitation reports whether this HITL request is an MCP elicitation
// (kind == "mcp_elicitation") rather than the older command_execution-style
// approval. harness-api sends `kind` alongside `mode` on every elicitation;
// checking it (rather than mode's mere presence) keeps this from
// misclassifying a future non-elicitation payload that happens to reuse the
// word "mode" for something else.
func (p hitlRequestedPayload) isMCPElicitation() bool {
	kind, _ := p.fields()["kind"].(string)
	return kind == mcpElicitationKind
}

// mode returns the MCP elicitation mode ("url" or "form"), or "" for a plain
// command_execution-style approval.
func (p hitlRequestedPayload) mode() string {
	if !p.isMCPElicitation() {
		return ""
	}
	mode, _ := p.fields()["mode"].(string)
	return mode
}

// elicitationMessage is the human-readable prompt an MCP elicitation carries
// (e.g. "Allow the do_actions MCP server to run tool \"action_invoke\"?").
func (p hitlRequestedPayload) elicitationMessage() string {
	msg, _ := p.fields()["message"].(string)
	return msg
}

// elicitationServerName is the MCP server asking (e.g. "do_actions").
func (p hitlRequestedPayload) elicitationServerName() string {
	name, _ := p.fields()["serverName"].(string)
	return name
}

// elicitationURL is the OAuth/connect link for a mode-"url" elicitation.
func (p hitlRequestedPayload) elicitationURL() string {
	url, _ := p.fields()["url"].(string)
	return url
}

// requestedSchema is the raw JSON Schema object a mode-"form" elicitation
// expects the answer's `content` to satisfy, or nil if absent.
func (p hitlRequestedPayload) requestedSchema() map[string]any {
	schema, _ := p.fields()["requestedSchema"].(map[string]any)
	return schema
}

// toolMeta is the optional `_meta` block a tool-call approval carries —
// tool_title, tool_description, tool_params_display.
func (p hitlRequestedPayload) toolMeta() map[string]any {
	meta, _ := p.fields()["_meta"].(map[string]any)
	return meta
}

// schemaFields parses requestedSchema.properties/required into an ordered
// (sorted by name, for determinism — JSON object key order isn't preserved
// through the generic map[string]any this payload is decoded into) slice of
// fields. Per the MCP spec, requestedSchema is restricted to flat objects
// with primitive properties (string/number/integer/boolean/string-enum); a
// property outside that set is returned with unsupported=true rather than
// dropped, so a caller can still see it exists and refuse to auto-submit.
func (p hitlRequestedPayload) schemaFields() []schemaField {
	schema := p.requestedSchema()
	if schema == nil {
		return nil
	}
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		return nil
	}
	required := map[string]bool{}
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if name, ok := r.(string); ok {
				required[name] = true
			}
		}
	}
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	sort.Strings(names)

	fields := make([]schemaField, 0, len(names))
	for _, name := range names {
		prop, _ := props[name].(map[string]any)
		fields = append(fields, parseSchemaField(name, prop, required[name]))
	}
	return fields
}

// schemaField is one property of an MCP form elicitation's requestedSchema,
// reduced to what a terminal prompt needs to render and validate it.
type schemaField struct {
	name           string
	title          string
	description    string
	jsonType       string // "string", "number", "integer", "boolean"
	format         string // e.g. "email", "uri", "date", "date-time"
	pattern        string
	maxLength      int // 0 means unset
	required       bool
	enum           []string
	unsupported    bool
	unsupportedWhy string
}

func parseSchemaField(name string, prop map[string]any, required bool) schemaField {
	f := schemaField{name: name, title: name, required: required}
	if prop == nil {
		f.unsupported = true
		f.unsupportedWhy = "no schema for this property"
		return f
	}
	if title, ok := prop["title"].(string); ok && title != "" {
		f.title = title
	}
	if desc, ok := prop["description"].(string); ok {
		f.description = desc
	}
	if format, ok := prop["format"].(string); ok {
		f.format = format
	}
	if pattern, ok := prop["pattern"].(string); ok {
		f.pattern = pattern
	}
	if maxLen, ok := prop["maxLength"].(float64); ok {
		f.maxLength = int(maxLen)
	}

	jsonType, _ := prop["type"].(string)
	f.jsonType = jsonType

	if jsonType == "string" {
		if rawEnum, ok := prop["enum"].([]any); ok && len(rawEnum) > 0 {
			for _, v := range rawEnum {
				if s, ok := v.(string); ok {
					f.enum = append(f.enum, s)
				}
			}
			if len(f.enum) == 0 {
				f.unsupported = true
				f.unsupportedWhy = "enum values must be strings"
			}
			return f
		}
		return f
	}
	if jsonType == "number" || jsonType == "integer" || jsonType == "boolean" {
		return f
	}

	f.unsupported = true
	if jsonType == "" {
		f.unsupportedWhy = "missing or unrecognized schema type"
	} else {
		f.unsupportedWhy = fmt.Sprintf("unsupported schema type %q", jsonType)
	}
	return f
}

// isApproveBooleanOnly reports the ticket's special case: a schema of exactly
// one required boolean field named "approve". The console renders this as a
// plain Approve/Deny control instead of a one-field form.
func isApproveBooleanOnly(fields []schemaField) bool {
	if len(fields) != 1 {
		return false
	}
	f := fields[0]
	return f.name == "approve" && f.jsonType == "boolean" && f.required
}

// shape classifies the payload per hitlElicitationShape's doc comment.
func (p hitlRequestedPayload) shape() hitlElicitationShape {
	if !p.isMCPElicitation() {
		return hitlShapeCommand
	}
	switch p.mode() {
	case "url":
		return hitlShapeURL
	case "form":
		if len(p.schemaFields()) == 0 {
			return hitlShapeApproval
		}
		return hitlShapeForm
	default:
		// mode is required on every elicitation payload; an empty/unknown
		// value is malformed input, not a reason to crash. Treat it as a
		// bare approval — accept/decline/cancel with no content is always a
		// safe answer to any elicitation shape.
		return hitlShapeApproval
	}
}

// elicitationSummary is the one-line label shown for an MCP elicitation
// wherever hitlCommandSummary would otherwise show a bash command — the
// approval line, the headless event log, and `logs` replay.
func (p hitlRequestedPayload) elicitationSummary() string {
	switch p.shape() {
	case hitlShapeURL:
		if name := p.elicitationServerName(); name != "" {
			return "authorize " + name + " (OAuth)"
		}
		return "authorize via OAuth"
	case hitlShapeApproval:
		if title, _ := p.toolMeta()["tool_title"].(string); title != "" {
			return title
		}
		if name := p.elicitationServerName(); name != "" {
			return "approve " + name + " tool call"
		}
		return "approve tool call"
	case hitlShapeForm:
		names := make([]string, 0, len(p.schemaFields()))
		for _, f := range p.schemaFields() {
			names = append(names, f.name)
		}
		if name := p.elicitationServerName(); name != "" {
			return fmt.Sprintf("provide %s for %s", strings.Join(names, ", "), name)
		}
		return "provide " + strings.Join(names, ", ")
	default:
		return ""
	}
}

// renderElicitationCard writes the read-only detail block for an MCP
// elicitation — message, URL/verification code, tool context, or the list of
// fields being asked for — printed once before the accept/decline/cancel
// prompt takes over. Callers that already print `message` via
// elicitationSummary/renderApprovalLine skip duplicating it here.
func renderElicitationCard(w io.Writer, p hitlRequestedPayload) {
	write := func(format string, args ...any) { fmt.Fprintf(w, format, args...) }

	if msg := p.elicitationMessage(); msg != "" {
		write("  %s\n", msg)
	}
	switch p.shape() {
	case hitlShapeURL:
		if url := p.elicitationURL(); url != "" {
			write("  %s %s\n", colorize("open:", colMuted), url)
		}
	case hitlShapeApproval:
		meta := p.toolMeta()
		if desc, _ := meta["tool_description"].(string); desc != "" {
			write("  %s\n", colorize(desc, colMuted))
		}
		if display, ok := meta["tool_params_display"].([]any); ok {
			for _, entry := range display {
				e, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				name, _ := e["name"].(string)
				if name == "" {
					continue
				}
				write("    %s %v\n", colorize(name+":", colMuted), e["value"])
			}
		}
	case hitlShapeForm:
		for _, f := range p.schemaFields() {
			marker := ""
			if f.required {
				marker = " " + colorize("(required)", colMuted)
			}
			write("    %s%s\n", boldColor(f.title, colHighlight), marker)
			if f.description != "" {
				write("      %s\n", colorize(f.description, colMuted))
			}
			if f.unsupported {
				write("      %s\n", colorize("not supported here: "+f.unsupportedWhy, colWarning))
			}
		}
	}
}

// elicitationForm drives the attach TTY's field-by-field prompt for a
// schema-bearing MCP elicitation (hitlShapeForm, excluding the
// isApproveBooleanOnly special case, which gets its own two-key prompt
// instead). Lives on attachState; handleAttachByte routes bytes here while
// it's set instead of through the plain approve/reject/defer menu.
type elicitationForm struct {
	hitlID string
	fields []schemaField
	idx    int
	values map[string]any
}

func (f *elicitationForm) current() schemaField { return f.fields[f.idx] }

// promptString renders the current field as "[i/n] Title (required): ".
func (f *elicitationForm) promptString() string {
	field := f.current()
	marker := ""
	if field.required {
		marker = " " + colorize("(required)", colMuted)
	}
	return fmt.Sprintf("[%d/%d] %s%s: ", f.idx+1, len(f.fields), field.title, marker)
}

// validateSchemaFieldInput converts and validates one line of raw input
// against a schema field's constraints. skip is true when an optional field
// was left blank — the caller should omit it from content entirely rather
// than send an empty string.
func validateSchemaFieldInput(f schemaField, raw string) (value any, skip bool, err error) {
	if f.unsupported {
		return nil, false, fmt.Errorf("not supported here (%s) — use `--content` instead", f.unsupportedWhy)
	}
	if raw == "" {
		if f.required {
			return nil, false, fmt.Errorf("required")
		}
		return nil, true, nil
	}
	switch f.jsonType {
	case "boolean":
		switch strings.ToLower(raw) {
		case "true", "y", "yes":
			return true, false, nil
		case "false", "n", "no":
			return false, false, nil
		}
		return nil, false, fmt.Errorf("enter true or false")
	case "number", "integer":
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, false, fmt.Errorf("enter a number")
		}
		if f.jsonType == "integer" {
			return int64(n), false, nil
		}
		return n, false, nil
	default: // "string"
		if len(f.enum) > 0 {
			ok := false
			for _, e := range f.enum {
				if e == raw {
					ok = true
					break
				}
			}
			if !ok {
				return nil, false, fmt.Errorf("must be one of: %s", strings.Join(f.enum, ", "))
			}
		}
		if f.maxLength > 0 && len(raw) > f.maxLength {
			return nil, false, fmt.Errorf("must be at most %d characters", f.maxLength)
		}
		if f.pattern != "" {
			if re, reErr := regexp.Compile(f.pattern); reErr == nil && !re.MatchString(raw) {
				return nil, false, fmt.Errorf("must match pattern %s", f.pattern)
			}
		}
		if f.format == "email" && !strings.Contains(raw, "@") {
			return nil, false, fmt.Errorf("enter a valid email address")
		}
		return raw, false, nil
	}
}

// handleElicitationFormByte routes one input byte while state.form is active.
// Character editing reuses handleAttachEditingKey and the same printable-byte
// insertion the plain input prompt uses; Enter validates and consumes the
// current field instead of sending a chat message.
func handleElicitationFormByte(c *CmdConfig, svc do.HostedAgentsService, sessionID string, b byte, state *attachState) (stop bool, err error) {
	switch b {
	case 0x03, 0x04: // Ctrl-C / Ctrl-D: detach, same as a plain pending verdict.
		state.display.echo([]byte("\r\n"))
		state.clearElicitationForm()
		printDetachNotice(c.Out, state.sessionRef)
		return true, nil
	case 0x0d, 0x0a: // Enter: validate and consume the current field.
		state.mu.Lock()
		raw := submittedInput(state.lineBuf)
		state.lineBuf = state.lineBuf[:0]
		state.cursor = 0
		state.mu.Unlock()
		state.display.echo([]byte("\r\n"))

		form := state.elicitationForm()
		if form == nil {
			return false, nil
		}
		field := form.current()
		value, skip, verr := validateSchemaFieldInput(field, raw)
		if verr != nil {
			fmt.Fprintf(c.Out, "  %s\n", colorize(verr.Error(), colError))
			state.display.redraw()
			return false, nil
		}
		if !skip {
			form.values[field.name] = value
		}
		form.idx++
		if form.idx >= len(form.fields) {
			id, content := form.hitlID, form.values
			state.clearElicitationForm()
			state.call(func() bool {
				if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
					Outcome: godo.HostedAgentHITLOutcomeApprove,
					Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
					Content: content,
				}); err != nil {
					fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
				} else {
					state.pending.clearIf(id)
				}
				return false
			})
		}
		state.display.redraw()
		return false, nil
	}
	if handleAttachEditingKey(b, state) {
		return false, nil
	}
	if b >= 0x20 && b < 0x7f {
		state.mu.Lock()
		atEnd := state.cursor == len(state.lineBuf)
		if atEnd {
			state.lineBuf = append(state.lineBuf, b)
		} else {
			state.lineBuf = append(state.lineBuf[:state.cursor], append([]byte{b}, state.lineBuf[state.cursor:]...)...)
		}
		state.cursor++
		state.mu.Unlock()
		state.display.redraw()
	}
	return false, nil
}

// handleApproveBooleanByte is the two-key Approve/Deny prompt for the ticket's
// special case: a requestedSchema of exactly one required boolean field named
// "approve". Both keys resolve with Outcome: Approve — they differ only in
// Content, matching the console's McpElicitationCard (acceptApproveBoolean).
// Decline/Cancel keep the plain Reject/Defer, no-content behavior.
func handleApproveBooleanByte(c *CmdConfig, svc do.HostedAgentsService, sessionID, hitlID string, b byte, state *attachState) (stop bool, err error) {
	var outcome godo.HostedAgentHITLOutcome
	var content map[string]any
	matched := true
	switch b {
	case 'y', 'Y':
		outcome, content = godo.HostedAgentHITLOutcomeApprove, map[string]any{"approve": true}
	case 'n', 'N':
		outcome, content = godo.HostedAgentHITLOutcomeApprove, map[string]any{"approve": false}
	case 'r', 'R':
		outcome = godo.HostedAgentHITLOutcomeReject
	case 'd', 'D':
		outcome = godo.HostedAgentHITLOutcomeDefer
	case 0x03, 0x04: // Ctrl-C / Ctrl-D
		state.display.echo([]byte("\r\n"))
		printDetachNotice(c.Out, state.sessionRef)
		return true, nil
	default:
		matched = false
	}
	if !matched {
		return false, nil
	}
	state.display.echo([]byte{b, '\r', '\n'})
	state.call(func() bool {
		if err := svc.ResolveHITL(sessionID, hitlID, &godo.HostedAgentResolveHITLRequest{
			Outcome: outcome,
			Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
			Content: content,
		}); err != nil {
			fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
		} else {
			state.pending.clearIf(hitlID)
		}
		return false
	})
	state.display.redraw()
	return false, nil
}
