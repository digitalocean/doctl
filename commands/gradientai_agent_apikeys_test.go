package commands

import (
	"errors"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

// Test data
var (
	testAPIKey = do.ApiKeyInfo{
		ApiKeyInfo: &godo.ApiKeyInfo{
			Uuid: "00000000-0000-4000-8000-000000000000",
			Name: "Key-1",
		},
	}
	testAPIKeys = do.ApiKeys{testAPIKey}
)

func TestRunAgentAPIKeyList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "00000000-0000-4000-8000-000000000000"
		config.Doit.Set(config.NS, doctl.ArgAgentId, agentID)
		tm.gradientAI.EXPECT().ListAgentAPIKeys(agentID).Return(testAPIKeys, nil)

		err := RunAgentAPIKeyList(config)
		assert.NoError(t, err)
	})
}

func TestRunAgentAPIKeyCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "00000000-0000-4000-8000-000000000000"
		name := "Key Two"
		config.Doit.Set(config.NS, doctl.ArgAgentAPIKeyName, name)
		config.Doit.Set(config.NS, doctl.ArgAgentId, agentID)

		expectedReq := &godo.AgentAPIKeyCreateRequest{
			Name:      name,
			AgentUuid: agentID,
		}
		tm.gradientAI.EXPECT().CreateAgentAPIKey(agentID, expectedReq).Return(&testAPIKey, nil)

		err := RunAgentAPIKeyCreate(config)
		assert.NoError(t, err)
	})
}

func TestRunAgentAPIKeyUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "agent-uuid"
		apikeyID := "key-3"
		name := "Updated Key"
		config.Args = []string{apikeyID}
		config.Doit.Set(config.NS, doctl.ArgAgentName, name)
		config.Doit.Set(config.NS, doctl.ArgAgentId, agentID)

		expectedReq := &godo.AgentAPIKeyUpdateRequest{
			Name:       name,
			AgentUuid:  agentID,
			APIKeyUuid: apikeyID,
		}
		tm.gradientAI.EXPECT().UpdateAgentAPIKey(agentID, apikeyID, expectedReq).Return(&testAPIKey, nil)

		err := RunAgentAPIKeyUpdate(config)
		assert.NoError(t, err)
	})
}

func TestRunAgentAPIKeyDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "agent-uuid"
		apikeyID := "key-4"
		config.Args = []string{apikeyID}
		config.Doit.Set(config.NS, doctl.ArgAgentId, agentID)
		config.Doit.Set(config.NS, doctl.ArgAgentForce, true)

		tm.gradientAI.EXPECT().DeleteAgentAPIKey(agentID, apikeyID).Return(nil)

		err := RunAgentAPIKeyDelete(config)
		assert.NoError(t, err)
	})
}

func TestRunAgentAPIKeyRegenerate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "agent-uuid"
		apikeyID := "key-5"
		config.Args = []string{apikeyID}
		config.Doit.Set(config.NS, doctl.ArgAgentId, agentID)

		tm.gradientAI.EXPECT().RegenerateAgentAPIKey(agentID, apikeyID).Return(&testAPIKey, nil)
		err := RunAgentAPIKeyRegenerate(config)
		assert.NoError(t, err)
	})
}

func TestRunAgentAPIKeyList_Error(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "agent-uuid"
		config.Doit.Set(config.NS, doctl.ArgAgentId, agentID)

		tm.gradientAI.EXPECT().ListAgentAPIKeys(agentID).Return(nil, errors.New("fail"))

		err := RunAgentAPIKeyList(config)
		assert.Error(t, err)
	})
}

func TestRunAgentAPIKeyCreate_Error(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agentID := "agent-uuid"
		name := "Key"
		config.Doit.Set(config.NS, doctl.ArgAgentAPIKeyName, name)
		config.Doit.Set(config.NS, doctl.ArgAgentId, agentID)

		expectedReq := &godo.AgentAPIKeyCreateRequest{
			Name:      name,
			AgentUuid: agentID,
		}
		tm.gradientAI.EXPECT().CreateAgentAPIKey(agentID, expectedReq).Return(nil, errors.New("fail"))

		err := RunAgentAPIKeyCreate(config)
		assert.Error(t, err)
	})
}
