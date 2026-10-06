package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testAgentUUID = "a1b2c3d4-e29b-41d4-a716-446655440000"
	testExportID  = "550e8400-e29b-41d4-a716-446655440000"
	testCompleted = int64(1754049690)

	testExport = do.SignalsExport{
		ExportID: testExportID,
		AgentID:  &testAgentUUID,
		Status:   "complete",
		Filters: do.SignalsExportFilters{
			SignalType: []string{"MisalignmentCorrection"},
			StartTime:  int64Ptr(1754049600),
			EndTime:    int64Ptr(1754136000),
		},
		CreatedAt:   1754049600,
		CompletedAt: &testCompleted,
	}
)

func int64Ptr(v int64) *int64 { return &v }

func TestSignalsCommand(t *testing.T) {
	cmd := Signals()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "consent", "export", "export-trigger")
}

func TestSignalsExportCommand(t *testing.T) {
	cmd := SignalsExport()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "download", "get", "list", "options")
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
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, testAgentUUID)
		config.Doit.Set(config.NS, doctl.ArgSignalsExportSignalType, []string{"MisalignmentCorrection"})
		start := 1754049600
		end := 1754136000
		config.Doit.Set(config.NS, doctl.ArgSignalsExportStartTime, start)
		config.Doit.Set(config.NS, doctl.ArgSignalsExportEndTime, end)

		tm.signals.EXPECT().CreateExport(&do.SignalsCreateExportRequest{
			AgentID:    testAgentUUID,
			SignalType: []string{"MisalignmentCorrection"},
			StartTime:  int64Ptr(int64(start)),
			EndTime:    int64Ptr(int64(end)),
		}).Return(&testExport, nil)

		err := RunSignalsExportCreate(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportCreateSessionIDsAndConcerning(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, testAgentUUID)
		config.Doit.Set(config.NS, doctl.ArgSignalsExportSessionIDs, []string{"sess-1"})
		config.Doit.Set(config.NS, doctl.ArgSignalsExportConcerning, true)
		config.Doit.Set(config.NS, doctl.ArgSignalsExportSignalCategory, "misalignment")
		config.Doit.Set(config.NS, doctl.ArgSignalsExportSignalLayer, "interaction")
		concerning := true
		cat := "misalignment"
		layer := "interaction"

		tm.signals.EXPECT().CreateExport(&do.SignalsCreateExportRequest{
			AgentID:        testAgentUUID,
			SessionIDs:     []string{"sess-1"},
			Concerning:     &concerning,
			SignalCategory: &cat,
			SignalLayer:    &layer,
		}).Return(&testExport, nil)

		err := RunSignalsExportCreate(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportCreateInvalidAgentID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "not-a-uuid")
		err := RunSignalsExportCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "valid UUID")
	})
}

func TestSignalsExportGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testExportID)

		tm.signals.EXPECT().GetExport(testExportID).
			Return(&testExport, nil)

		err := RunSignalsExportGet(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportDownload(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testExportID)
		tm.signals.EXPECT().GetExportDownload(testExportID).Return(&do.SignalsExportDownload{
			DownloadURL: "https://example/export.json.gz",
			ExpiresAt:   1754050500,
		}, nil)
		err := RunSignalsExportDownload(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportOptions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		opts := &do.SignalsExportOptions{}
		opts.Filters.SignalType = []string{"MisalignmentCorrection"}
		tm.signals.EXPECT().GetExportOptions().Return(opts, nil)
		err := RunSignalsExportOptions(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportTriggerGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.signals.EXPECT().GetExportTrigger().Return(&do.SignalsExportTrigger{
			Enabled: false,
			Cadence: "weekly",
		}, nil)
		err := RunSignalsExportTriggerGet(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportTriggerSet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsEnabled, true)
		config.Doit.Set(config.NS, doctl.ArgSignalsExportTriggerSignalTypes, []string{"MisalignmentCorrection"})
		config.Doit.Set(config.NS, doctl.ArgSignalsExportCadence, "weekly")

		tm.signals.EXPECT().UpsertExportTrigger(&do.SignalsExportTriggerUpsertRequest{
			Enabled:     true,
			Cadence:     "weekly",
			SignalTypes: []string{"MisalignmentCorrection"},
		}).Return(&do.SignalsExportTrigger{
			Enabled:     true,
			Cadence:     "weekly",
			SignalTypes: []string{"MisalignmentCorrection"},
		}, nil)

		err := RunSignalsExportTriggerSet(config)
		assert.NoError(t, err)
	})
}
