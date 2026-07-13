package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testAnthropicKey = do.AnthropicApiKey{
		AnthropicApiKeyInfo: &godo.AnthropicApiKeyInfo{
			Uuid: "d35e5cb7-7957-4643-8e3a-1ab4eb3a494c",
			Name: "Test Anthropic Key",
		},
	}
)

func TestAnthropicKeyCommand(t *testing.T) {
	cmd := AnthropicKeyCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "get-agents", "list", "update")
}

func TestAnthropicKeyGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		anthropic_key_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, anthropic_key_id)
		tm.gradientAI.EXPECT().GetAnthropicAPIKey("00000000-0000-4000-8000-000000000000").Return(&testAnthropicKey, nil)
		err := RunAnthropicKeyGet(config)
		assert.NoError(t, err)
	})
}

func TestAnthropicKeyList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.gradientAI.EXPECT().ListAnthropicAPIKeys().Return(do.AnthropicApiKeys{testAnthropicKey}, nil)
		err := RunAnthropicKeyList(config)
		assert.NoError(t, err)
	})
}

func TestAnthropicKeyCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {

		config.Doit.Set(config.NS, doctl.ArgAnthropicKeyName, "Test Anthropic Key")
		config.Doit.Set(config.NS, doctl.ArgAnthropicKeyAPIKey, "sk-ant-proddfsefac")

		tm.gradientAI.EXPECT().CreateAnthropicAPIKey(&godo.AnthropicAPIKeyCreateRequest{
			Name:   "Test Anthropic Key",
			ApiKey: "sk-ant-proddfsefac",
		}).Return(&testAnthropicKey, nil)

		err := RunAnthropicKeyCreate(config)
		assert.NoError(t, err)
	})
}

func TestAnthropicKeyDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		anthropic_api_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, anthropic_api_id)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		tm.gradientAI.EXPECT().DeleteAnthropicAPIKey("00000000-0000-4000-8000-000000000000").Return(&testAnthropicKey, nil)
		err := RunAnthropicKeyDelete(config)
		assert.NoError(t, err)
	})
}

func TestAnthropicKeyUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		anthropic_api_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, anthropic_api_id)

		config.Doit.Set(config.NS, doctl.ArgAnthropicKeyName, "Updated Anthropic Key")
		config.Doit.Set(config.NS, doctl.ArgAnthropicKeyAPIKey, "updated-api-key")

		tm.gradientAI.EXPECT().UpdateAnthropicAPIKey("00000000-0000-4000-8000-000000000000", &godo.AnthropicAPIKeyUpdateRequest{
			Name:       "Updated Anthropic Key",
			ApiKey:     "updated-api-key",
			ApiKeyUuid: "00000000-0000-4000-8000-000000000000",
		}).Return(&testAnthropicKey, nil)

		err := RunAnthropicKeyUpdate(config)
		assert.NoError(t, err)
	})
}

func TestAnthropicKeyGetAgents(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		anthropic_key_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, anthropic_key_id)
		tm.gradientAI.EXPECT().ListAgentsByAnthropicAPIKey("00000000-0000-4000-8000-000000000000").Return(do.Agents{}, nil)
		err := RunAnthropicKeyGetAgents(config)
		assert.NoError(t, err)
	})
}
