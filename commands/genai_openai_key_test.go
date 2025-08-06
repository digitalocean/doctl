package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testOpenAIKey = do.OpenAiApiKey{
		OpenAiApiKey: &godo.OpenAiApiKey{
			Uuid: "d35e5cb7-7957-4643-8e3a-1ab4eb3a494c",
			Name: "Test OpenAI Key",
		},
	}
)

func TestOpenAIKeyCommand(t *testing.T) {
	cmd := OpenAIKeyCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "get-agents", "list", "update")
}

func TestOpenAIKeyGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		openai_key_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, openai_key_id)
		tm.genAI.EXPECT().GetOpenAIAPIKey("00000000-0000-4000-8000-000000000000").Return(&testOpenAIKey, nil)
		err := RunOpenAIKeyGet(config)
		assert.NoError(t, err)
	})
}

func TestOpenAIKeyList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.genAI.EXPECT().ListOpenAIAPIKeys().Return(do.OpenAiApiKeys{testOpenAIKey}, nil)
		err := RunOpenAIKeyList(config)
		assert.NoError(t, err)
	})
}

func TestOpenAIKeyCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {

		config.Doit.Set(config.NS, doctl.ArgOpenAIKeyName, "Test OpenAI Key")
		config.Doit.Set(config.NS, doctl.ArgOpenAIKeyAPIKey, "sk-proddfsefac")

		tm.genAI.EXPECT().CreateOpenAIAPIKey(&godo.OpenAIAPIKeyCreateRequest{
			Name:   "Test OpenAI Key",
			ApiKey: "sk-proddfsefac",
		}).Return(&testOpenAIKey, nil)

		err := RunOpenAIKeyCreate(config)
		assert.NoError(t, err)
	})
}

func TestOpenAIKeyDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		open_ai_api_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, open_ai_api_id)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		tm.genAI.EXPECT().DeleteOpenAIAPIKey("00000000-0000-4000-8000-000000000000").Return(&testOpenAIKey, nil)
		err := RunOpenAIKeyDelete(config)
		assert.NoError(t, err)
	})
}

func TestOpenAIKeyUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		open_ai_api_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, open_ai_api_id)

		config.Doit.Set(config.NS, doctl.ArgOpenAIKeyName, "Updated OpenAI Key")
		config.Doit.Set(config.NS, doctl.ArgOpenAIKeyAPIKey, "updated-api-key")

		tm.genAI.EXPECT().UpdateOpenAIAPIKey("00000000-0000-4000-8000-000000000000", &godo.OpenAIAPIKeyUpdateRequest{
			Name:       "Updated OpenAI Key",
			ApiKey:     "updated-api-key",
			ApiKeyUuid: "00000000-0000-4000-8000-000000000000",
		}).Return(&testOpenAIKey, nil)

		err := RunOpenAIKeyUpdate(config)
		assert.NoError(t, err)
	})
}
