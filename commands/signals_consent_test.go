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
		Source:    godo.SignalsConsentSourceAgent,
		AgentID:   "agt-1",
		Enabled:   true,
		UpdatedAt: "2026-01-01T00:00:00Z",
	}

	testConsent = do.SignalsConsent{
		SignalsConsentRecord: &testConsentRecord,
	}

	testInferenceConsentRecord = godo.SignalsConsentRecord{
		ID:        7,
		TeamID:    42,
		Source:    godo.SignalsConsentSourceInference,
		AgentID:   "",
		Enabled:   true,
		UpdatedAt: "2026-10-09T13:28:22Z",
	}

	testInferenceConsent = do.SignalsConsent{
		SignalsConsentRecord: &testInferenceConsentRecord,
	}

	testAgentConsentRecord = godo.SignalsAgentConsent{
		ID:        1,
		TeamID:    42,
		Source:    godo.SignalsConsentSourceAgent,
		AgentID:   "agt-1",
		Enabled:   true,
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
			Return(do.SignalsConsents{testConsent, testInferenceConsent}, nil)

		err := RunSignalsConsentList(config)
		assert.NoError(t, err)
	})
}

func TestSignalsConsentListBySource(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsConsentSource, "inference")

		tm.signals.EXPECT().ListConsentsBySource("inference").
			Return(do.SignalsConsents{testInferenceConsent}, nil)

		err := RunSignalsConsentList(config)
		assert.NoError(t, err)
	})
}

func TestSignalsConsentListInvalidSource(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsConsentSource, "bogus")

		err := RunSignalsConsentList(config)
		assert.ErrorContains(t, err, "invalid --source")
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

func TestSignalsConsentGetInference(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsConsentSource, "inference")

		tm.signals.EXPECT().GetInferenceConsent().
			Return(&testInferenceConsent, nil)

		err := RunSignalsConsentGet(config)
		assert.NoError(t, err)
	})
}

func TestSignalsConsentGetAgentRequiresAgentID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunSignalsConsentGet(config)
		assert.ErrorContains(t, err, "--agent-id is required")
	})
}

func TestSignalsConsentGetInferenceRejectsAgentID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsConsentSource, "inference")
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "agt-1")

		err := RunSignalsConsentGet(config)
		assert.ErrorContains(t, err, "must not be set")
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

func TestSignalsConsentSetInference(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsConsentSource, "inference")
		config.Doit.Set(config.NS, doctl.ArgSignalsEnabled, true)

		tm.signals.EXPECT().SetInferenceConsent(true).
			Return(&testInferenceConsent, nil)

		err := RunSignalsConsentSet(config)
		assert.NoError(t, err)
	})
}

func TestSignalsConsentSetInferenceDisable(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsConsentSource, "inference")
		config.Doit.Set(config.NS, doctl.ArgSignalsEnabled, false)

		disabledRecord := testInferenceConsentRecord
		disabledRecord.Enabled = false
		disabledConsent := do.SignalsConsent{SignalsConsentRecord: &disabledRecord}

		tm.signals.EXPECT().SetInferenceConsent(false).
			Return(&disabledConsent, nil)

		err := RunSignalsConsentSet(config)
		assert.NoError(t, err)
	})
}

func TestSignalsConsentSetInvalidSource(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsConsentSource, "bogus")
		config.Doit.Set(config.NS, doctl.ArgSignalsEnabled, true)

		err := RunSignalsConsentSet(config)
		assert.ErrorContains(t, err, "invalid --source")
	})
}
