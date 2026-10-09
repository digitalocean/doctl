package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testSession = do.SignalsSession{
		SignalsSession: &godo.SignalsSession{
			SessionID:       "sess-1",
			TotalTurns:      3,
			StartedAt:       "2026-01-01T00:00:00Z",
			DurationSeconds: 120,
			SignalCount:     1,
		},
	}
)

func TestSignalsSessionListAgent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsType, "agent")
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "agt-1")
		config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 20)

		tm.signals.EXPECT().ListAgentSessions("agt-1", &godo.SignalsListAgentSessionsOptions{
			SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
				Limit: 20,
			},
		}).Return(do.SignalsSessions{testSession}, nil)

		err := RunSignalsSessionList(config)
		assert.NoError(t, err)
	})
}

func TestSignalsSessionListAgentDefaultType(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "agt-1")
		config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 20)

		tm.signals.EXPECT().ListAgentSessions("agt-1", &godo.SignalsListAgentSessionsOptions{
			SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
				Limit: 20,
			},
		}).Return(do.SignalsSessions{testSession}, nil)

		err := RunSignalsSessionList(config)
		assert.NoError(t, err)
	})
}

func TestSignalsSessionListAgentMissingAgentID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsType, "agent")

		err := RunSignalsSessionList(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--agent-id is required")
	})
}

func TestSignalsSessionListInference(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsType, "inference")
		config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 20)

		tm.signals.EXPECT().ListAgentSessions(signalsInferencePlaceholderAgentID, &godo.SignalsListAgentSessionsOptions{
			SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
				Limit: 20,
			},
		}).Return(do.SignalsSessions{testSession}, nil)

		err := RunSignalsSessionList(config)
		assert.NoError(t, err)
	})
}

func TestSignalsSessionListInferenceWithAgentID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsType, "inference")
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "agt-1")

		err := RunSignalsSessionList(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--agent-id must not be set")
	})
}

func TestSignalsSessionListInvalidType(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsType, "bogus")

		err := RunSignalsSessionList(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `invalid --type "bogus"`)
	})
}
