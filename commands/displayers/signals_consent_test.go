package displayers

import (
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalsConsent_Cols(t *testing.T) {
	d := &SignalsConsent{}
	cols := d.Cols()
	assert.Contains(t, cols, "EnvironmentID")
	assert.NotContains(t, cols, "AgentID")
	cm := d.ColMap()
	for _, c := range cols {
		assert.Contains(t, cm, c)
	}
	assert.Contains(t, cm, "AgentID")
	assert.Equal(t, "Environment ID", cm["EnvironmentID"])
	assert.Equal(t, "Agent ID", cm["AgentID"])
}

func TestSignalsConsent_KV_EnvironmentIDAlias(t *testing.T) {
	d := &SignalsConsent{Consents: do.SignalsConsents{{
		SignalsConsentRecord: &godo.SignalsConsentRecord{
			ID: 1, TeamID: 42, Source: "agent", AgentID: "agent-1", Enabled: true,
		},
	}}}
	kv := d.KV()
	require.Len(t, kv, 1)
	assert.Equal(t, "agent-1", kv[0]["EnvironmentID"])
	assert.Equal(t, kv[0]["EnvironmentID"], kv[0]["AgentID"])
}

func TestSignalsConsent_KV_PrefersEnvironmentID(t *testing.T) {
	d := &SignalsConsent{Consents: do.SignalsConsents{{
		SignalsConsentRecord: &godo.SignalsConsentRecord{
			ID: 1, TeamID: 42, Source: "agent", EnvironmentID: "env-1", Enabled: true,
		},
	}}}
	kv := d.KV()
	require.Len(t, kv, 1)
	assert.Equal(t, "env-1", kv[0]["EnvironmentID"])
	assert.Equal(t, "env-1", kv[0]["AgentID"])
}

func TestSignalsAgentConsent_Cols(t *testing.T) {
	d := &SignalsAgentConsent{}
	cols := d.Cols()
	assert.Contains(t, cols, "EnvironmentID")
	assert.NotContains(t, cols, "AgentID")
	cm := d.ColMap()
	assert.Contains(t, cm, "EnvironmentID")
	assert.Contains(t, cm, "AgentID")
}

func TestSignalsAgentConsent_KV_EnvironmentIDAlias(t *testing.T) {
	d := &SignalsAgentConsent{Consents: []do.SignalsAgentConsent{{
		SignalsAgentConsent: &godo.SignalsAgentConsent{
			TeamID: 42, Source: "agent", AgentID: "agent-1", Enabled: true,
		},
	}}}
	kv := d.KV()
	require.Len(t, kv, 1)
	assert.Equal(t, "agent-1", kv[0]["EnvironmentID"])
	assert.Equal(t, kv[0]["EnvironmentID"], kv[0]["AgentID"])
}
