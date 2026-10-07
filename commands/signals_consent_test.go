package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testConsentRecord = godo.SignalsConsentRecord{
		ID:        1,
		TeamID:    42,
		AgentID:   "agt-1",
		Enabled:   true,
		UpdatedAt: "2026-01-01T00:00:00Z",
	}

	testConsent = do.SignalsConsent{
		SignalsConsentRecord: &testConsentRecord,
	}

	testAgentConsentRecord = godo.SignalsAgentConsent{
		ID:        1,
		TeamID:    42,
		AgentID:   "agt-1",
		Enabled:   true,
		Allowed:   true,
		UpdatedAt: "2026-01-01T00:00:00Z",
	}

	testAgentConsent = do.SignalsAgentConsent{
		SignalsAgentConsent: &testAgentConsentRecord,
	}
)

func TestSignalsConsentCommand(t *testing.T) {
	cmd := SignalsConsent()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list", "get", "set")
}

func TestSignalsConsentList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.signals.EXPECT().ListConsents().
			Return(do.SignalsConsents{testConsent}, nil)

		err := RunSignalsConsentList(config)
		assert.NoError(t, err)
	})
}

func TestSignalsConsentGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "agt-1")

		tm.signals.EXPECT().GetConsent("agt-1").
			Return(&testAgentConsent, nil)

		err := RunSignalsConsentGet(config)
		assert.NoError(t, err)
	})
}

func TestSignalsConsentSet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "agt-1")
		config.Doit.Set(config.NS, doctl.ArgSignalsEnabled, true)

		tm.signals.EXPECT().SetConsent("agt-1", true).
			Return(&testConsent, nil)

		err := RunSignalsConsentSet(config)
		assert.NoError(t, err)
	})
}

func TestSignalsConsentSetDisable(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "agt-1")
		config.Doit.Set(config.NS, doctl.ArgSignalsEnabled, false)

		disabledRecord := testConsentRecord
		disabledRecord.Enabled = false
		disabledConsent := do.SignalsConsent{SignalsConsentRecord: &disabledRecord}

		tm.signals.EXPECT().SetConsent("agt-1", false).
			Return(&disabledConsent, nil)

		err := RunSignalsConsentSet(config)
		assert.NoError(t, err)
	})
}
