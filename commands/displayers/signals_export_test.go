package displayers

import (
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalsExport_Cols(t *testing.T) {
	d := &SignalsExport{}
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

func TestSignalsExport_KV_EnvironmentIDAlias(t *testing.T) {
	d := &SignalsExport{Exports: do.SignalsExports{{
		SignalsExportJob: &godo.SignalsExportJob{
			ExportID: "exp-1", AgentID: "agent-1", Status: "completed",
			Filters:   godo.SignalsExportFilters{SignalType: []string{"guardrail"}},
			CreatedAt: 100,
		},
	}}}
	kv := d.KV()
	require.Len(t, kv, 1)
	assert.Equal(t, "agent-1", kv[0]["EnvironmentID"])
	assert.Equal(t, kv[0]["EnvironmentID"], kv[0]["AgentID"])
}

func TestSignalsExport_KV_PrefersEnvironmentID(t *testing.T) {
	d := &SignalsExport{Exports: do.SignalsExports{{
		SignalsExportJob: &godo.SignalsExportJob{
			ExportID: "exp-1", EnvironmentID: "env-1", Status: "completed",
			Filters:   godo.SignalsExportFilters{SignalType: []string{"guardrail"}},
			CreatedAt: 100,
		},
	}}}
	kv := d.KV()
	require.Len(t, kv, 1)
	assert.Equal(t, "env-1", kv[0]["EnvironmentID"])
	assert.Equal(t, "env-1", kv[0]["AgentID"])
}
