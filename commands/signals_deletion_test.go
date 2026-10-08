package commands

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testDeletionID  = "01J9ZX0N5Q7M2K3R4S5T6V7W8X"
	testDeletionJob = do.SignalsDeletion{
		SignalsDeletionJob: &godo.SignalsDeletionJob{
			TeamID:     42,
			DeletionID: testDeletionID,
			Type:       godo.SignalsDeletionTypeManagedAgent,
			AgentID:    testAgentUUID,
			Status:     godo.SignalsDeletionStatusQueued,
			CreatedAt:  1754049600,
		},
	}
)

func TestSignalsDeletionCommand(t *testing.T) {
	cmd := SignalsDeletion()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "get", "list")
	assert.Equal(t, []string{"deletions"}, cmd.Aliases)

	aliases := map[string]string{}
	for _, sub := range cmd.Commands() {
		if len(sub.Aliases) == 1 {
			aliases[sub.Name()] = sub.Aliases[0]
		}
	}
	assert.Equal(t, map[string]string{"create": "c", "get": "g", "list": "ls"}, aliases)
}

func TestSignalsDeletionConfirmMessage(t *testing.T) {
	for _, msg := range []string{
		signalsDeletionConfirmMessage("managed_agent", testAgentUUID, 42),
		signalsDeletionConfirmMessage("inference", "", 42),
	} {
		assert.True(t, strings.HasPrefix(msg, "permanently delete"), msg)
		assert.False(t, strings.HasSuffix(msg, "?"), msg)
		assert.Contains(t, msg, "42")
		assert.Contains(t, msg, "cannot be undone")
	}
	assert.Contains(t, signalsDeletionConfirmMessage("managed_agent", testAgentUUID, 42), testAgentUUID)
	assert.Contains(t, signalsDeletionConfirmMessage("inference", "", 42), "whole team")
}

func TestSignalsDeletionCreate_Success(t *testing.T) {
	tests := []struct {
		name  string
		setup func(config *CmdConfig, tm *tcMocks)
	}{
		{
			name: "managed agent with explicit team id does not look up the team",
			setup: func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, "managed_agent")
				config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, testAgentUUID)
				config.Doit.Set(config.NS, doctl.ArgSignalsTeamID, 42)
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				tm.signals.EXPECT().CreateDeletion(&godo.SignalsCreateDeletionRequest{
					Type: "managed_agent", TeamID: 42, AgentID: testAgentUUID,
				}).Return(&testDeletionJob, false, nil)
			},
		},
		{
			name: "inference sends no agent id",
			setup: func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, "inference")
				config.Doit.Set(config.NS, doctl.ArgSignalsTeamID, 42)
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				tm.signals.EXPECT().CreateDeletion(&godo.SignalsCreateDeletionRequest{
					Type: "inference", TeamID: 42,
				}).Return(&testDeletionJob, false, nil)
			},
		},
		{
			name: "team id is auto-detected when not given",
			setup: func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, "managed_agent")
				config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, testAgentUUID)
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				tm.signals.EXPECT().GetTeamID().Return(int64(42), nil)
				tm.signals.EXPECT().CreateDeletion(&godo.SignalsCreateDeletionRequest{
					Type: "managed_agent", TeamID: 42, AgentID: testAgentUUID,
				}).Return(&testDeletionJob, false, nil)
			},
		},
		{
			name: "existing active job is still a success",
			setup: func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, "inference")
				config.Doit.Set(config.NS, doctl.ArgSignalsTeamID, 42)
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				tm.signals.EXPECT().CreateDeletion(gomock.Any()).Return(&testDeletionJob, true, nil)
			},
		},
		{
			name: "agent id with surrounding spaces is trimmed",
			setup: func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, " managed_agent ")
				config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "  "+testAgentUUID+" ")
				config.Doit.Set(config.NS, doctl.ArgSignalsTeamID, 42)
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				tm.signals.EXPECT().CreateDeletion(&godo.SignalsCreateDeletionRequest{
					Type: "managed_agent", TeamID: 42, AgentID: testAgentUUID,
				}).Return(&testDeletionJob, false, nil)
			},
		},
		{
			name: "agent id is sent in canonical lowercase form",
			setup: func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, "managed_agent")
				config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, strings.ToUpper(testAgentUUID))
				config.Doit.Set(config.NS, doctl.ArgSignalsTeamID, 42)
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				tm.signals.EXPECT().CreateDeletion(&godo.SignalsCreateDeletionRequest{
					Type: "managed_agent", TeamID: 42, AgentID: testAgentUUID,
				}).Return(&testDeletionJob, false, nil)
			},
		},
		{
			name: "inference with a spaces-only agent id counts as not set",
			setup: func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, "inference")
				config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, "   ")
				config.Doit.Set(config.NS, doctl.ArgSignalsTeamID, 42)
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				tm.signals.EXPECT().CreateDeletion(&godo.SignalsCreateDeletionRequest{
					Type: "inference", TeamID: 42,
				}).Return(&testDeletionJob, false, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tt.setup(config, tm)
				assert.NoError(t, RunSignalsDeletionCreate(config))
			})
		})
	}
}

// Every case here must fail without any mock call (gomock fails the test on an
// unexpected call).
func TestSignalsDeletionCreate_InputErrors(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		agentID string
		teamID  int
		wantErr string
	}{
		{name: "missing type", typ: "", agentID: testAgentUUID, teamID: 42, wantErr: `--type must be "managed_agent" or "inference"`},
		{name: "unknown type", typ: "agent", agentID: testAgentUUID, teamID: 42, wantErr: `--type must be "managed_agent" or "inference"`},
		{name: "wrong case type", typ: "Managed_Agent", agentID: testAgentUUID, teamID: 42, wantErr: `--type must be "managed_agent" or "inference"`},
		{name: "managed agent without agent id", typ: "managed_agent", teamID: 42, wantErr: "--agent-id is required"},
		{name: "managed agent with spaces-only agent id", typ: "managed_agent", agentID: "  ", teamID: 42, wantErr: "--agent-id is required"},
		{name: "managed agent with non-uuid agent id", typ: "managed_agent", agentID: "not-a-uuid", teamID: 42, wantErr: "--agent-id must be a valid UUID"},
		{name: "inference with agent id", typ: "inference", agentID: testAgentUUID, teamID: 42, wantErr: "--agent-id must not be set when --type is inference"},
		{name: "negative team id", typ: "inference", teamID: -1, wantErr: "--team-id must not be negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, tt.typ)
				config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, tt.agentID)
				config.Doit.Set(config.NS, doctl.ArgSignalsTeamID, tt.teamID)
				config.Doit.Set(config.NS, doctl.ArgForce, true)

				err := RunSignalsDeletionCreate(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			})
		})
	}
}

func TestSignalsDeletionCreate_TeamLookupFails(t *testing.T) {
	for _, cause := range []error{
		errors.New("GET /v1/consent: 403 forbidden"),
		errors.New("could not determine your team id; pass --team-id"),
	} {
		t.Run(cause.Error(), func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, "inference")
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				tm.signals.EXPECT().GetTeamID().Return(int64(0), cause)

				err := RunSignalsDeletionCreate(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "--team-id")
				assert.ErrorIs(t, err, cause)
			})
		})
	}
}

func TestSignalsDeletionCreate_ServiceErrorPassedThrough(t *testing.T) {
	for _, msg := range []string{"403 forbidden", "404 agent not found", "429 too many active deletion jobs", "500 internal"} {
		t.Run(msg, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, "managed_agent")
				config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, testAgentUUID)
				config.Doit.Set(config.NS, doctl.ArgSignalsTeamID, 42)
				config.Doit.Set(config.NS, doctl.ArgForce, true)
				want := errors.New(msg)
				tm.signals.EXPECT().CreateDeletion(gomock.Any()).Return(nil, false, want)

				err := RunSignalsDeletionCreate(config)
				assert.Equal(t, want, err)
			})
		})
	}
}

func TestSignalsDeletionCreate_NoForceNonInteractiveRefuses(t *testing.T) {
	prev := Interactive
	Interactive = false
	t.Cleanup(func() { Interactive = prev })

	for _, typ := range []string{"managed_agent", "inference"} {
		t.Run(typ, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsDeletionType, typ)
				if typ == "managed_agent" {
					config.Doit.Set(config.NS, doctl.ArgSignalsAgentID, testAgentUUID)
				}
				config.Doit.Set(config.NS, doctl.ArgSignalsTeamID, 42)
				// No CreateDeletion expectation: calling it fails the test.

				err := RunSignalsDeletionCreate(config)
				assert.ErrorIs(t, err, ErrExitSilently)
			})
		})
	}
}

func TestSignalsDeletionGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Args = append(config.Args, testDeletionID)
			tm.signals.EXPECT().GetDeletion(testDeletionID).Return(&testDeletionJob, nil)
			assert.NoError(t, RunSignalsDeletionGet(config))
		})
	})

	t.Run("id is trimmed", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Args = append(config.Args, "  "+testDeletionID+" ")
			tm.signals.EXPECT().GetDeletion(testDeletionID).Return(&testDeletionJob, nil)
			assert.NoError(t, RunSignalsDeletionGet(config))
		})
	})

	t.Run("no args", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			assert.Error(t, RunSignalsDeletionGet(config))
		})
	})

	t.Run("two args", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Args = append(config.Args, "a", "b")
			assert.Error(t, RunSignalsDeletionGet(config))
		})
	})

	for _, id := range []string{"", "   "} {
		t.Run("empty id "+id, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Args = append(config.Args, id)
				err := RunSignalsDeletionGet(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "must not be empty")
			})
		})
	}

	t.Run("service error", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Args = append(config.Args, testDeletionID)
			want := errors.New("404 deletion not found")
			tm.signals.EXPECT().GetDeletion(testDeletionID).Return(nil, want)
			assert.Equal(t, want, RunSignalsDeletionGet(config))
		})
	})
}

func TestSignalsDeletionList(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 20)
			tm.signals.EXPECT().ListDeletions(&godo.SignalsListDeletionsOptions{
				SignalsCursorPageOptions: godo.SignalsCursorPageOptions{Limit: 20},
			}).Return(do.SignalsDeletions{testDeletionJob}, "", nil)
			assert.NoError(t, RunSignalsDeletionList(config))
		})
	})

	t.Run("after cursor", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 5)
			config.Doit.Set(config.NS, doctl.ArgSignalsExportAfter, "cursor-abc")
			tm.signals.EXPECT().ListDeletions(&godo.SignalsListDeletionsOptions{
				SignalsCursorPageOptions: godo.SignalsCursorPageOptions{Limit: 5, After: "cursor-abc"},
			}).Return(do.SignalsDeletions{testDeletionJob}, "", nil)
			assert.NoError(t, RunSignalsDeletionList(config))
		})
	})

	t.Run("next cursor still succeeds", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 20)
			tm.signals.EXPECT().ListDeletions(gomock.Any()).Return(do.SignalsDeletions{testDeletionJob}, "next-1", nil)
			assert.NoError(t, RunSignalsDeletionList(config))
		})
	})

	t.Run("empty result", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 20)
			tm.signals.EXPECT().ListDeletions(gomock.Any()).Return(do.SignalsDeletions{}, "", nil)
			assert.NoError(t, RunSignalsDeletionList(config))
		})
	})

	for _, limit := range []int{-1, 0, 101} {
		limit := limit
		t.Run(fmt.Sprintf("invalid limit %d", limit), func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, limit)
				err := RunSignalsDeletionList(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "--limit must be between 1 and 100")
			})
		})
	}

	for _, limit := range []int{1, 100} {
		limit := limit
		t.Run(fmt.Sprintf("boundary limit %d accepted", limit), func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, limit)
				tm.signals.EXPECT().ListDeletions(gomock.Any()).Return(do.SignalsDeletions{}, "", nil)
				assert.NoError(t, RunSignalsDeletionList(config))
			})
		})
	}

	t.Run("service error", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgSignalsExportLimit, 20)
			want := errors.New("400 invalid pagination cursor")
			tm.signals.EXPECT().ListDeletions(gomock.Any()).Return(nil, "", want)
			assert.Equal(t, want, RunSignalsDeletionList(config))
		})
	})
}
