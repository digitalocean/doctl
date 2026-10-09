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
	testAgentUUID = "a1b2c3d4-e29b-41d4-a716-446655440000"
	testExportID  = "550e8400-e29b-41d4-a716-446655440000"
	testCompleted = int64(1754049690)

	testExportJob = godo.SignalsExportJob{
		ExportID: testExportID,
		AgentID:  testAgentUUID,
		Status:   "complete",
		Filters: godo.SignalsExportFilters{
			SignalType: []string{"MisalignmentCorrection"},
			StartTime:  int64Ptr(1754049600),
			EndTime:    int64Ptr(1754136000),
		},
		CreatedAt:   1754049600,
		CompletedAt: &testCompleted,
	}

	testExport = do.SignalsExport{
		SignalsExportJob: &testExportJob,
	}
)

func int64Ptr(v int64) *int64 { return &v }

func TestSignalsCommand(t *testing.T) {
	cmd := Signals()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "consent", "export", "session")
}

func TestSignalsExportCommand(t *testing.T) {
	cmd := SignalsExport()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "download", "get", "list", "options")
}

func TestSignalsSessionCommand(t *testing.T) {
	cmd := SignalsSession()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "dialogues", "list")
}

func TestSignalsExportList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 20)

		tm.signals.EXPECT().ListExports(&godo.SignalsListExportsOptions{
			SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
				Limit: 20,
			},
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

		tm.signals.EXPECT().ListExports(&godo.SignalsListExportsOptions{
			SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
				Limit: 10,
				After: "cursor-abc",
			},
			AgentID: "agt-1",
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

		tm.signals.EXPECT().CreateExport(&godo.SignalsCreateExportRequest{
			AgentID:    testAgentUUID,
			SignalType: []string{"MisalignmentCorrection"},
			StartTime:  int64Ptr(int64(start)),
			EndTime:    int64Ptr(int64(end)),
		}).Return(&testExport, nil)

		err := RunSignalsExportCreate(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportCreateAgentType(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsType, "agent")
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, testAgentUUID)

		tm.signals.EXPECT().CreateExport(&godo.SignalsCreateExportRequest{
			AgentID: testAgentUUID,
		}).Return(&testExport, nil)

		err := RunSignalsExportCreate(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportCreateAgentMissingAgentID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsType, "agent")

		err := RunSignalsExportCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--agent-id is required")
	})
}

func TestSignalsExportCreateInference(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsType, "inference")

		tm.signals.EXPECT().CreateExport(&godo.SignalsCreateExportRequest{
			AgentID: signalsInferencePlaceholderAgentID,
		}).Return(&testExport, nil)

		err := RunSignalsExportCreate(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportCreateInferenceWithAgentID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsType, "inference")
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, testAgentUUID)

		err := RunSignalsExportCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--agent-id must not be set")
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
			SignalsExportDownload: &godo.SignalsExportDownload{
				DownloadURL: "https://example/export.json.gz",
				ExpiresAt:   1754050500,
			},
		}, nil)
		err := RunSignalsExportDownload(config)
		assert.NoError(t, err)
	})
}

func TestSignalsExportOptions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		opts := &do.SignalsExportOptions{
			SignalsExportOptions: &godo.SignalsExportOptions{
				Filters: godo.SignalsExportFilterOptions{
					SignalType: []string{"MisalignmentCorrection"},
				},
			},
		}
		tm.signals.EXPECT().GetExportOptions().Return(opts, nil)
		err := RunSignalsExportOptions(config)
		assert.NoError(t, err)
	})
}

func TestSignalsSessionList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "agt-1")
		config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 20)

		testSession := do.SignalsSession{
			SignalsSession: &godo.SignalsSession{
				SessionID:       "sess-1",
				TotalTurns:      5,
				StartedAt:       "2026-01-01T00:00:00Z",
				DurationSeconds: 300,
				SignalCount:     2,
			},
		}

		tm.signals.EXPECT().ListAgentSessions("agt-1", &godo.SignalsListAgentSessionsOptions{
			SignalsCursorPageOptions: godo.SignalsCursorPageOptions{Limit: 20},
		}).Return(do.SignalsSessions{testSession}, nil)

		err := RunSignalsSessionList(config)
		assert.NoError(t, err)
	})
}

func TestSignalsSessionDialogueList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSignalsSessionID, "sess-1")
		config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 20)

		testDialogue := do.SignalsSessionDialogue{
			SignalsSessionDialogue: &godo.SignalsSessionDialogue{
				SignalsDialogue: godo.SignalsDialogue{
					ID:          1,
					RunID:       "run-1",
					Sequence:    1,
					UserMessage: "hello",
					RunStatus:   "complete",
				},
				SegmentID:  "seg-1",
				SegmentSeq: 0,
			},
		}

		tm.signals.EXPECT().ListSessionDialogues("sess-1", &godo.SignalsListDialoguesOptions{
			SignalsCursorPageOptions: godo.SignalsCursorPageOptions{Limit: 20},
		}).Return(do.SignalsSessionDialogues{testDialogue}, nil)

		err := RunSignalsSessionDialogueList(config)
		assert.NoError(t, err)
	})
}
