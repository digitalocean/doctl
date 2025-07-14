package commands

import (
	"encoding/json"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

const (
	testAgentUUID    = "11111111-2222-3333-4444-555555555555"
	testFunctionUUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
)

var (
	inputSchemaJSON = `{
		"parameters": [{
			"name": "zipCode",
			"in": "query",
			"schema": { "type": "string" },
			"required": false,
			"description": "Zip description in input"
		}]
	}`

	outputSchemaJSON = `{
		"properties": {
			"temperature": { "type": "number" }
		}
	}`

	testAgentResponse = &do.Agent{
		Agent: &godo.Agent{
			Uuid: testAgentUUID,
			Name: "test-agent",
		},
	}
)

func TestFunctionRouteCreate(t *testing.T) {
	withTestClient(t, func(cfg *CmdConfig, tm *tcMocks) {
		var inSchema godo.FunctionInputSchema
		_ = json.Unmarshal([]byte(inputSchemaJSON), &inSchema)

		expReq := &godo.FunctionRouteCreateRequest{
			AgentUuid:     testAgentUUID,
			FunctionName:  "my-fn",
			Description:   "unit-test create",
			FaasName:      "default/testing",
			FaasNamespace: "ns",
			InputSchema:   inSchema,
			OutputSchema:  json.RawMessage(outputSchemaJSON),
		}

		tm.genAI.EXPECT().
			CreateFunctionRoute(testAgentUUID, expReq).
			Return(testAgentResponse, nil)

		cfg.Doit.Set(cfg.NS, doctl.ArgAgentUUID, testAgentUUID)
		cfg.Doit.Set(cfg.NS, doctl.ArgFunctionName, "my-fn")
		cfg.Doit.Set(cfg.NS, doctl.ArgFunctionRouteDescription, "unit-test create")
		cfg.Doit.Set(cfg.NS, doctl.ArgFunctionRouteFaasName, "default/testing")
		cfg.Doit.Set(cfg.NS, doctl.ArgFunctionRouteFaasNamespace, "ns")
		cfg.Doit.Set(cfg.NS, doctl.ArgFunctionRouteInputSchema, inputSchemaJSON)
		cfg.Doit.Set(cfg.NS, doctl.ArgFunctionRouteOutputSchema, outputSchemaJSON)

		assert.NoError(t, RunFunctionRouteCreate(cfg))
	})
}

func TestFunctionRouteUpdate(t *testing.T) {
	withTestClient(t, func(cfg *CmdConfig, tm *tcMocks) {
		expReq := &godo.FunctionRouteUpdateRequest{
			AgentUuid:    testAgentUUID,
			FunctionUuid: testFunctionUUID,
			Description:  "updated-desc",
			FaasName:     "default/updated",
		}

		tm.genAI.EXPECT().
			UpdateFunctionRoute(testAgentUUID, testFunctionUUID, expReq).
			Return(testAgentResponse, nil)

		cfg.Doit.Set(cfg.NS, doctl.ArgAgentUUID, testAgentUUID)
		cfg.Doit.Set(cfg.NS, doctl.ArgFunctionID, testFunctionUUID)
		cfg.Doit.Set(cfg.NS, doctl.ArgFunctionRouteDescription, "updated-desc")
		cfg.Doit.Set(cfg.NS, doctl.ArgFunctionRouteFaasName, "default/updated")

		assert.NoError(t, RunFunctionRouteUpdate(cfg))
	})
}

func TestFunctionRouteDelete(t *testing.T) {
	withTestClient(t, func(cfg *CmdConfig, tm *tcMocks) {
		tm.genAI.EXPECT().
			DeleteFunctionRoute(testAgentUUID, testFunctionUUID).
			Return(testAgentResponse, nil)

		cfg.Doit.Set(cfg.NS, doctl.ArgAgentUUID, testAgentUUID)
		cfg.Doit.Set(cfg.NS, doctl.ArgFunctionID, testFunctionUUID)

		assert.NoError(t, RunFunctionRouteDelete(cfg))
	})
}
