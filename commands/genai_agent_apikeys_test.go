package commands

import (
	"errors"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGenAIService struct {
	ListAgentAPIKeysFn      func(string) (do.ApiKeys, error)
	CreateAgentAPIKeyFn     func(string, *godo.AgentAPIKeyCreateRequest) (*do.ApiKey, error)
	UpdateAgentAPIKeyFn     func(string, string, *godo.AgentAPIKeyUpdateRequest) (*do.ApiKey, error)
	DeleteAgentAPIKeyFn     func(string, string) error
	RegenerateAgentAPIKeyFn func(string, string) (*do.ApiKey, error)
}

func (f *fakeGenAIService) ListAgentAPIKeys(agentID string) (do.ApiKeys, error) {
	return f.ListAgentAPIKeysFn(agentID)
}
func (f *fakeGenAIService) CreateAgentAPIKey(agentID string, req *godo.AgentAPIKeyCreateRequest) (*do.ApiKey, error) {
	return f.CreateAgentAPIKeyFn(agentID, req)
}
func (f *fakeGenAIService) UpdateAgentAPIKey(agentID, apikeyID string, req *godo.AgentAPIKeyUpdateRequest) (*do.ApiKey, error) {
	return f.UpdateAgentAPIKeyFn(agentID, apikeyID, req)
}
func (f *fakeGenAIService) DeleteAgentAPIKey(agentID, apikeyID string) error {
	return f.DeleteAgentAPIKeyFn(agentID, apikeyID)
}
func (f *fakeGenAIService) RegenerateAgentAPIKey(agentID, apikeyID string) (*do.ApiKey, error) {
	return f.RegenerateAgentAPIKeyFn(agentID, apikeyID)
}

func TestRunAgentAPIKeyList(t *testing.T) {
	expected := do.ApiKeys{
		{UUID: "key-1", Name: "Key One"},
	}
	fake := &fakeGenAIService{
		ListAgentAPIKeysFn: func(agentID string) (do.ApiKeys, error) {
			assert.Equal(t, "agent-uuid", agentID)
			return expected, nil
		},
	}
	c := &CmdConfig{
		Doit: &doctl.TestConfig{
			StringMap: map[string]string{"agent-uuid": "agent-uuid"},
		},
		GenAI: func() do.GenAIService { return fake },
		Display: func(d displayers.Displayable) error {
			apiKeys, ok := d.(*displayers.ApiKeyInfo)
			require.True(t, ok)
			assert.Equal(t, expected, apiKeys.ApiKeyInfo)
			return nil
		},
		NS: "test",
	}
	err := RunAgentAPIKeyList(c)
	require.NoError(t, err)
}

func TestRunAgentAPIKeyCreate(t *testing.T) {
	expected := do.ApiKeyInfo{UUID: "key-2", Name: "Key Two"}
	fake := &fakeGenAIService{
		CreateAgentAPIKeyFn: func(agentID string, req *godo.AgentAPIKeyCreateRequest) (*do.ApiKey, error) {
			assert.Equal(t, "agent-uuid", agentID)
			assert.Equal(t, "Key Two", req.Name)
			return &expected, nil
		},
	}
	c := &CmdConfig{
		Doit: &doctl.TestConfig{
			StringMap: map[string]string{"name": "Key Two", "agent-uuid": "agent-uuid"},
		},
		GenAI: func() do.GenAIService { return fake },
		Display: func(d displayers.Displayable) error {
			apiKeys, ok := d.(*displayers.ApiKeyInfo)
			require.True(t, ok)
			assert.Equal(t, do.ApiKeys{expected}, apiKeys.ApiKeyInfo)
			return nil
		},
		NS: "test",
	}
	err := RunAgentAPIKeyCreate(c)
	require.NoError(t, err)
}

func TestRunAgentAPIKeyUpdate(t *testing.T) {
	expected := do.ApiKey{UUID: "key-3", Name: "Updated Key"}
	fake := &fakeGenAIService{
		UpdateAgentAPIKeyFn: func(agentID, apikeyID string, req *godo.AgentAPIKeyUpdateRequest) (*do.ApiKey, error) {
			assert.Equal(t, "agent-uuid", agentID)
			assert.Equal(t, "key-3", apikeyID)
			assert.Equal(t, "Updated Key", req.Name)
			return &expected, nil
		},
	}
	c := &CmdConfig{
		Args: []string{"agent-uuid"},
		Doit: &doctl.TestConfig{
			StringMap: map[string]string{
				doctl.ArgAgentName:  "Updated Key",
				doctl.ArgAPIkeyUUID: "key-3",
			},
		},
		GenAI: func() do.GenAIService { return fake },
		Display: func(d displayers.Displayable) error {
			apiKeys, ok := d.(*displayers.ApiKeyInfo)
			require.True(t, ok)
			assert.Equal(t, do.ApiKeys{expected}, apiKeys.ApiKeyInfo)
			return nil
		},
		NS: "test",
	}
	err := RunAgentAPIKeyUpdate(c)
	require.NoError(t, err)
}

func TestRunAgentAPIKeyDelete(t *testing.T) {
	called := false
	fake := &fakeGenAIService{
		DeleteAgentAPIKeyFn: func(agentID, apikeyID string) error {
			called = true
			assert.Equal(t, "agent-uuid", agentID)
			assert.Equal(t, "key-4", apikeyID)
			return nil
		},
	}
	c := &CmdConfig{
		Args: []string{"agent-uuid"},
		Doit: &doctl.TestConfig{
			StringMap: map[string]string{
				doctl.ArgAPIkeyUUID: "key-4",
			},
			BoolMap: map[string]bool{
				doctl.ArgAgentForce: true,
			},
		},
		GenAI: func() do.GenAIService { return fake },
		NS:    "test",
	}
	// Patch notice to avoid printing
	notice = func(string, ...interface{}) {}
	err := RunAgentAPIKeyDelete(c)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRunAgentAPIKeyRegenerate(t *testing.T) {
	expected := do.ApiKey{UUID: "key-5", Name: "Regenerated Key"}
	fake := &fakeGenAIService{
		RegenerateAgentAPIKeyFn: func(agentID, apikeyID string) (*do.ApiKey, error) {
			assert.Equal(t, "agent-uuid", agentID)
			assert.Equal(t, "key-5", apikeyID)
			return &expected, nil
		},
	}
	c := &CmdConfig{
		Args: []string{"agent-uuid"},
		Doit: &doctl.TestConfig{
			StringMap: map[string]string{
				doctl.ArgAPIkeyUUID: "key-5",
			},
		},
		GenAI: func() do.GenAIService { return fake },
		Display: func(d displayers.Displayable) error {
			apiKeys, ok := d.(*displayers.ApiKeyInfo)
			require.True(t, ok)
			assert.Equal(t, do.ApiKeys{expected}, apiKeys.ApiKeyInfo)
			return nil
		},
		NS: "test",
	}
	err := RunAgentAPIKeyRegenerate(c)
	require.NoError(t, err)
}

func TestRunAgentAPIKeyList_Error(t *testing.T) {
	fake := &fakeGenAIService{
		ListAgentAPIKeysFn: func(agentID string) (do.ApiKeys, error) {
			return nil, errors.New("fail")
		},
	}
	c := &CmdConfig{
		Doit: &doctl.TestConfig{
			StringMap: map[string]string{"agent-uuid": "agent-uuid"},
		},
		GenAI: func() do.GenAIService { return fake },
		Display: func(d displayers.Displayable) error {
			return nil
		},
		NS: "test",
	}
	err := RunAgentAPIKeyList(c)
	assert.Error(t, err)
}

func TestRunAgentAPIKeyCreate_Error(t *testing.T) {
	fake := &fakeGenAIService{
		CreateAgentAPIKeyFn: func(agentID string, req *godo.AgentAPIKeyCreateRequest) (*do.ApiKey, error) {
			return nil, errors.New("fail")
		},
	}
	c := &CmdConfig{
		Doit: &doctl.TestConfig{
			StringMap: map[string]string{"name": "Key", "agent-uuid": "agent-uuid"},
		},
		GenAI: func() do.GenAIService { return fake },
		Display: func(d displayers.Displayable) error {
			return nil
		},
		NS: "test",
	}
	err := RunAgentAPIKeyCreate(c)
	assert.Error(t, err)
}
