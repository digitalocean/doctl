package commands

import (
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
)

var (
	testExportTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	testExport = do.SignalsExport{
		ID:          "exp-1",
		TeamID:      42,
		AgentID:     "agt-1",
		Status:      "completed",
		SignalTypes:  []string{"hallucination", "toxicity"},
		CreatedAt:   testExportTime,
		UpdatedAt:   testExportTime,
	}
)

func TestSignalsExportCommand(t *testing.T) {
	cmd := SignalsExport()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list", "create", "get")
}

func TestSignalsExportList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.signals.EXPECT().ListExports(&do.SignalsExportListOptions{
			Limit: 0,
		}).Return(do.SignalsExports{testExport}, nil)

		err := RunSignalsExportList(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportListWithFilters(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "agt-1")
		config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 10)
		config.Doit.Set(config.NS, doctl.ArgSignalsExportAfter, "cursor-abc")

		tm.signals.EXPECT().ListExports(&do.SignalsExportListOptions{
			AgentID: "agt-1",
			Limit:   10,
			After:   "cursor-abc",
		}).Return(do.SignalsExports{testExport}, nil)

		err := RunSignalsExportList(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "agt-1")
		config.Doit.Set(config.NS, doctl.ArgSignalsExportSignalTypes, []string{"hallucination"})

		tm.signals.EXPECT().CreateExport(&do.SignalsCreateExportRequest{
			AgentID:     "agt-1",
			SignalTypes: []string{"hallucination"},
		}).Return(&testExport, nil)

		err := RunSignalsExportCreate(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "exp-1")

		tm.signals.EXPECT().GetExport("exp-1").
			Return(&testExport, nil)

		err := RunSignalsExportGet(config)
		assert.NoError(t, err)
	})
}
