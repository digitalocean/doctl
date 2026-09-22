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
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// typeInto sends each byte of s through handleAttachByte, as if typed.
func typeInto(t *testing.T, config *CmdConfig, svc do.HostedAgentsService, sessionID string, state *attachState, s string) {
	t.Helper()
	for i := 0; i < len(s); i++ {
		stop, err := handleAttachByte(config, svc, sessionID, s[i], state, nil, nil)
		require.NoError(t, err)
		require.False(t, stop)
	}
}

func mustHITLPayload(t *testing.T, raw string) hitlRequestedPayload {
	t.Helper()
	var p hitlRequestedPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return p
}

func TestHITLElicitationShape(t *testing.T) {
	t.Run("command_execution stays hitlShapeCommand", func(t *testing.T) {
		p := mustHITLPayload(t, `{"hitl_id":"h1","action":"HITL_ACTION_BASH","details":{"command":"ls"}}`)
		assert.Equal(t, hitlShapeCommand, p.shape())
		assert.False(t, p.isMCPElicitation())
	})

	t.Run("mode url", func(t *testing.T) {
		p := mustHITLPayload(t, `{"hitl_id":"h1","details":{
			"kind":"mcp_elicitation","mode":"url","serverName":"do_actions",
			"message":"Open https://example.com to authorize. Code abc123.",
			"url":"https://example.com"
		}}`)
		assert.Equal(t, hitlShapeURL, p.shape())
		assert.Equal(t, "https://example.com", p.elicitationURL())
		assert.Equal(t, "do_actions", p.elicitationServerName())
		assert.Contains(t, p.elicitationSummary(), "do_actions")
	})

	t.Run("mode form, empty schema, is approval", func(t *testing.T) {
		p := mustHITLPayload(t, `{"hitl_id":"h1","details":{
			"kind":"mcp_elicitation","mode":"form","serverName":"do_actions",
			"message":"Allow the do_actions MCP server to run tool \"action_invoke\"?",
			"requestedSchema":{"type":"object","properties":{}}
		}}`)
		assert.Equal(t, hitlShapeApproval, p.shape())
		assert.Empty(t, p.schemaFields())
	})

	t.Run("mode form, non-empty schema, is form", func(t *testing.T) {
		p := mustHITLPayload(t, `{"hitl_id":"h1","details":{
			"kind":"mcp_elicitation","mode":"form","serverName":"jira",
			"message":"Configure the jira connection.",
			"requestedSchema":{
				"type":"object",
				"properties":{"site_url":{"type":"string","title":"Jira site URL","format":"uri","maxLength":255}},
				"required":["site_url"]
			}
		}}`)
		assert.Equal(t, hitlShapeForm, p.shape())
		fields := p.schemaFields()
		assert.Len(t, fields, 1)
		assert.Equal(t, "site_url", fields[0].name)
		assert.Equal(t, "Jira site URL", fields[0].title)
		assert.True(t, fields[0].required)
		assert.Equal(t, "uri", fields[0].format)
		assert.False(t, isApproveBooleanOnly(fields))
	})

	t.Run("approve boolean special case", func(t *testing.T) {
		p := mustHITLPayload(t, `{"hitl_id":"h1","details":{
			"kind":"mcp_elicitation","mode":"form",
			"requestedSchema":{
				"type":"object",
				"properties":{"approve":{"type":"boolean"}},
				"required":["approve"]
			}
		}}`)
		assert.Equal(t, hitlShapeForm, p.shape())
		fields := p.schemaFields()
		assert.True(t, isApproveBooleanOnly(fields))
	})

	t.Run("unsupported property type is flagged, not dropped", func(t *testing.T) {
		p := mustHITLPayload(t, `{"hitl_id":"h1","details":{
			"kind":"mcp_elicitation","mode":"form",
			"requestedSchema":{
				"type":"object",
				"properties":{"tags":{"type":"array"}},
				"required":["tags"]
			}
		}}`)
		fields := p.schemaFields()
		assert.Len(t, fields, 1)
		assert.True(t, fields[0].unsupported)
	})
}

func TestValidateSchemaFieldInput(t *testing.T) {
	t.Run("required empty errors", func(t *testing.T) {
		_, _, err := validateSchemaFieldInput(schemaField{name: "x", required: true}, "")
		assert.Error(t, err)
	})

	t.Run("optional empty skips", func(t *testing.T) {
		_, skip, err := validateSchemaFieldInput(schemaField{name: "x"}, "")
		assert.NoError(t, err)
		assert.True(t, skip)
	})

	t.Run("string maxLength enforced", func(t *testing.T) {
		_, _, err := validateSchemaFieldInput(schemaField{name: "x", jsonType: "string", maxLength: 3}, "abcd")
		assert.Error(t, err)
	})

	t.Run("email format enforced", func(t *testing.T) {
		_, _, err := validateSchemaFieldInput(schemaField{name: "x", jsonType: "string", format: "email"}, "not-an-email")
		assert.Error(t, err)
		val, _, err := validateSchemaFieldInput(schemaField{name: "x", jsonType: "string", format: "email"}, "a@b.com")
		assert.NoError(t, err)
		assert.Equal(t, "a@b.com", val)
	})

	t.Run("enum enforced", func(t *testing.T) {
		f := schemaField{name: "x", jsonType: "string", enum: []string{"a", "b"}}
		_, _, err := validateSchemaFieldInput(f, "c")
		assert.Error(t, err)
		val, _, err := validateSchemaFieldInput(f, "a")
		assert.NoError(t, err)
		assert.Equal(t, "a", val)
	})

	t.Run("boolean parsed", func(t *testing.T) {
		val, _, err := validateSchemaFieldInput(schemaField{name: "x", jsonType: "boolean"}, "yes")
		assert.NoError(t, err)
		assert.Equal(t, true, val)
		_, _, err = validateSchemaFieldInput(schemaField{name: "x", jsonType: "boolean"}, "maybe")
		assert.Error(t, err)
	})

	t.Run("integer parsed", func(t *testing.T) {
		val, _, err := validateSchemaFieldInput(schemaField{name: "x", jsonType: "integer"}, "42")
		assert.NoError(t, err)
		assert.Equal(t, int64(42), val)
	})

	t.Run("unsupported field always errors", func(t *testing.T) {
		_, _, err := validateSchemaFieldInput(schemaField{name: "x", unsupported: true, unsupportedWhy: "array"}, "anything")
		assert.Error(t, err)
	})
}

// TestAttachFormElicitation drives the raw-TTY field-by-field prompt end to
// end: the head of pending is a schema-bearing elicitation, so typing a value
// and pressing Enter must fill the one field and submit with Content, not
// send the text as a chat message.
func TestAttachFormElicitation(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf

		pending := &pendingHITL{}
		payload := mustHITLPayload(t, `{"hitl_id":"hitl_1","details":{"kind":"mcp_elicitation","mode":"form","serverName":"jira","requestedSchema":{"type":"object","properties":{"site_url":{"type":"string","title":"Jira site URL"}},"required":["site_url"]}}}`)
		pending.setElicitation("hitl_1", payload)

		tm.hostedAgents.EXPECT().
			ResolveHITL("sess", "hitl_1", &godo.HostedAgentResolveHITLRequest{
				Outcome: godo.HostedAgentHITLOutcomeApprove,
				Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
				Content: map[string]any{"site_url": "https://acme.atlassian.net"},
			}).
			Return(nil)

		state := newAttachState(io.Discard, pending)
		state.display.setRaw(true)

		// First byte on a fresh form-shaped pending entry lazily enters form
		// mode and must still be consumed as the first character typed.
		stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 'h', state, nil, nil)
		require.NoError(t, err)
		require.False(t, stop)
		require.NotNil(t, state.elicitationForm())

		typeInto(t, config, tm.hostedAgents, "sess", state, "ttps://acme.atlassian.net")
		stop, err = handleAttachByte(config, tm.hostedAgents, "sess", 0x0d, state, nil, nil)
		require.NoError(t, err)
		require.False(t, stop)

		assert.Nil(t, state.elicitationForm(), "form clears once every field is submitted")
		assert.Equal(t, "", pending.get(), "resolved entry must leave the queue")
	})
}

// TestAttachFormElicitationRequiredFieldRejectsEmpty: pressing Enter on an
// empty required field must not submit, and must not clear the pending entry.
func TestAttachFormElicitationRequiredFieldRejectsEmpty(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf

		pending := &pendingHITL{}
		payload := mustHITLPayload(t, `{"hitl_id":"hitl_1","details":{"kind":"mcp_elicitation","mode":"form","requestedSchema":{"type":"object","properties":{"site_url":{"type":"string"}},"required":["site_url"]}}}`)
		pending.setElicitation("hitl_1", payload)

		state := newAttachState(io.Discard, pending)
		state.display.setRaw(true)

		stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 0x0d, state, nil, nil)
		require.NoError(t, err)
		require.False(t, stop)

		assert.NotNil(t, state.elicitationForm(), "form stays open on a validation error")
		assert.Equal(t, "hitl_1", pending.get())
		assert.Contains(t, buf.String(), "required")
	})
}

// TestAttachApproveBooleanElicitation covers the ticket's special case: a
// schema of exactly {approve: boolean} renders as a two-key Approve/Deny
// prompt, and both keys resolve with Outcome: Approve, differing only in
// Content.
func TestAttachApproveBooleanElicitation(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		pending := &pendingHITL{}
		payload := mustHITLPayload(t, `{"hitl_id":"hitl_1","details":{"kind":"mcp_elicitation","mode":"form","requestedSchema":{"type":"object","properties":{"approve":{"type":"boolean"}},"required":["approve"]}}}`)
		pending.setElicitation("hitl_1", payload)

		tm.hostedAgents.EXPECT().
			ResolveHITL("sess", "hitl_1", &godo.HostedAgentResolveHITLRequest{
				Outcome: godo.HostedAgentHITLOutcomeApprove,
				Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
				Content: map[string]any{"approve": false},
			}).
			Return(nil)

		state := newAttachState(io.Discard, pending)
		state.display.setRaw(true)

		stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 'n', state, nil, nil)
		require.NoError(t, err)
		require.False(t, stop)
		assert.Nil(t, state.elicitationForm(), "approve-boolean never opens the multi-field form")
	})
}
