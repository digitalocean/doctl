package commands

import (
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
)

var (
	testConsentTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	testConsent = do.SignalsConsent{
		ID:        1,
		TeamID:    42,
		AgentID:   "agt-1",
		Enabled:   true,
		UpdatedAt: testConsentTime,
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
			Return(&testConsent, nil)

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

		disabledConsent := testConsent
		disabledConsent.Enabled = false

		tm.signals.EXPECT().SetConsent("agt-1", false).
			Return(&disabledConsent, nil)

		err := RunSignalsConsentSet(config)
		assert.NoError(t, err)
	})
}
