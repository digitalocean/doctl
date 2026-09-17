/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Top-off consent (--resume-on-topoff) reaches the server as a query parameter
// on the manifest create path, as a body field on the config create path, and as
// a body field on the session PATCH, because the manifest itself is immutable
// and shared through Agent Configs while the consent is per-session and
// revocable. These tests pin all three wire forms.

// TestResumeOnTopoffWireNames pins the spellings the server parses. Renaming
// either one silently stops the consent from arriving: the query parameter is
// dropped as unknown, and the body field is dropped by the strict decoder.
func TestResumeOnTopoffWireNames(t *testing.T) {
	t.Run("query parameter on the manifest create path", func(t *testing.T) {
		f, ok := reflect.TypeOf(godo.HostedAgentManifestCreateOptions{}).FieldByName("ResumeOnTopoff")
		require.True(t, ok)
		// omitempty keeps the withheld case off the wire entirely rather than
		// sending resume_on_topoff=false, which the server treats identically.
		assert.Equal(t, "resume_on_topoff,omitempty", f.Tag.Get("url"))
	})

	t.Run("body field on the config create path", func(t *testing.T) {
		f, ok := reflect.TypeOf(godo.HostedAgentSessionFromConfigRequest{}).FieldByName("ResumeOnTopoff")
		require.True(t, ok)
		assert.Equal(t, "resume_on_topoff,omitempty", f.Tag.Get("json"))
	})

	t.Run("read back off the session", func(t *testing.T) {
		f, ok := reflect.TypeOf(godo.HostedAgentSession{}).FieldByName("ResumeOnTopoff")
		require.True(t, ok)
		assert.Equal(t, "resume_on_topoff,omitempty", f.Tag.Get("json"))
	})

	t.Run("body field on the session patch path", func(t *testing.T) {
		f, ok := reflect.TypeOf(godo.HostedAgentSessionUpdateRequest{}).FieldByName("ResumeOnTopoff")
		require.True(t, ok)
		assert.Equal(t, "resume_on_topoff,omitempty", f.Tag.Get("json"))
		// A pointer, not a bool: PATCH leaves omitted properties alone, so an
		// unmentioned field has to stay off the wire instead of arriving as
		// false and revoking a consent the user never named.
		assert.Equal(t, reflect.Ptr, f.Type.Kind(), "the patch field must be a pointer to distinguish unset from false")
	})
}

func TestRunAgentsCreate_ResumeOnTopoff(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(
				[]byte(sampleManifest),
				&godo.HostedAgentManifestCreateOptions{ResumeOnTopoff: true},
			).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID:      "sess_topoff",
					AgentKind:      godo.HostedAgentKindOpenCode,
					Status:         godo.HostedAgentSessionStatusProvisioning,
					ResumeOnTopoff: true,
				},
			}, nil)
		tm.hostedAgents.EXPECT().
			GetSession("sess_topoff").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID:      "sess_topoff",
					AgentKind:      godo.HostedAgentKindOpenCode,
					Status:         godo.HostedAgentSessionStatusReady,
					ResumeOnTopoff: true,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, true)
		require.NoError(t, RunAgentsCreate(config))
	})
}

// TestRunAgentsCreate_ResumeOnTopoffNotWrittenIntoManifest guards the reason
// the option is a query parameter: a manifest that acquired a resume_on_topoff
// key would be rejected by the server's strict decoder, and would carry
// per-session spending consent into every Agent Config reuse.
func TestRunAgentsCreate_ResumeOnTopoffNotWrittenIntoManifest(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			DoAndReturn(func(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				assert.NotContains(t, string(manifest), "resume_on_topoff")
				assert.NotContains(t, string(manifest), "resumeOnTopoff")
				require.NotNil(t, opt)
				assert.True(t, opt.ResumeOnTopoff)
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_topoff",
						Status:    godo.HostedAgentSessionStatusReady,
					},
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetSession("sess_topoff").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_topoff",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, true)
		require.NoError(t, RunAgentsCreate(config))
	})
}

func TestRunAgentsCreate_FromConfig_ResumeOnTopoff(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
				Name:           "demo",
				ConfigID:       "cfg_abc123",
				ResumeOnTopoff: true,
			}).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID:      "sess_cfg_topoff",
					Name:           "demo",
					ConfigID:       "cfg_abc123",
					Status:         godo.HostedAgentSessionStatusReady,
					AgentKind:      godo.HostedAgentKindOpenCode,
					ResumeOnTopoff: true,
				},
			}, nil)
		tm.hostedAgents.EXPECT().
			GetSession("sess_cfg_topoff").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID:      "sess_cfg_topoff",
					Name:           "demo",
					ConfigID:       "cfg_abc123",
					Status:         godo.HostedAgentSessionStatusReady,
					AgentKind:      godo.HostedAgentKindOpenCode,
					ResumeOnTopoff: true,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, true)
		require.NoError(t, RunAgentsCreate(config))
	})
}

func TestAgentConfigStartSession_ResumeOnTopoff(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
				Name:           "my-session",
				ConfigID:       "cfg_1",
				ResumeOnTopoff: true,
			}).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID:      "sess_1",
					Name:           "my-session",
					ResumeOnTopoff: true,
				},
			}, nil)

		config.Args = append(config.Args, "cfg_1")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-session")
		config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, true)
		require.NoError(t, RunAgentsConfigStartSession(config))
	})
}

// TestRunAgentsLaunch_ResumeOnTopoffPatchesExistingSession: `launch <session>`
// now applies the consent on the way into the chat, so enrolling a session and
// opening it stay one command. UpdateSession returns a sentinel to prove the
// patch is reached without dragging the attach machinery into the test.
func TestRunAgentsLaunch_ResumeOnTopoffPatchesExistingSession(t *testing.T) {
	stubInteractiveTerminal(t, true)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListSessions(&godo.HostedAgentSessionListOptions{Name: "my-session"}).
			Return([]do.HostedAgentSession{{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_live",
					Name:      "my-session",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}}, "", nil)
		tm.hostedAgents.EXPECT().
			GetSession("sess_live").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_live",
					Name:      "my-session",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		granted := true
		tm.hostedAgents.EXPECT().
			UpdateSession("sess_live", &godo.HostedAgentSessionUpdateRequest{ResumeOnTopoff: &granted}).
			Return(nil, assertCalledErr)

		config.Args = []string{"my-session"}
		config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, true)

		err := RunAgentsLaunch(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), assertCalledErr.Error())
	})
}

// A terminal session cannot be patched (the API answers 409), and attach has a
// far better explanation for it, so launch must not send the request at all.
// gomock fails the test if UpdateSession is called.
func TestRunAgentsLaunch_ResumeOnTopoffSkippedForTerminalSession(t *testing.T) {
	stubInteractiveTerminal(t, true)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListSessions(&godo.HostedAgentSessionListOptions{Name: "gone"}).
			Return([]do.HostedAgentSession{{
				HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_gone", Name: "gone"},
			}}, "", nil)
		tm.hostedAgents.EXPECT().
			GetSession("sess_gone").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_gone",
					Name:      "gone",
					Status:    godo.HostedAgentSessionStatusDestroyed,
				},
			}, nil)

		config.Args = []string{"gone"}
		config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, true)

		require.Error(t, RunAgentsLaunch(config))
	})
}

func TestRunAgentsUpdate_ResumeOnTopoff(t *testing.T) {
	t.Run("granted", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			granted := true
			tm.hostedAgents.EXPECT().
				UpdateSession("11111111-1111-1111-1111-111111111111",
					&godo.HostedAgentSessionUpdateRequest{ResumeOnTopoff: &granted}).
				Return(&do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID:      "11111111-1111-1111-1111-111111111111",
						Name:           "demo",
						Status:         godo.HostedAgentSessionStatusReady,
						ResumeOnTopoff: true,
					},
				}, nil)

			var buf bytes.Buffer
			config.Out = &buf
			config.Args = []string{"11111111-1111-1111-1111-111111111111"}
			config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, true)

			require.NoError(t, RunAgentsUpdate(config))
			assert.Contains(t, buf.String(), "will resume automatically")
		})
	})

	// The revoke case is why the command prints its own line: the show card
	// only renders the top-off row when the consent is granted, so the card
	// alone would confirm nothing here.
	t.Run("revoked", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			revoked := false
			tm.hostedAgents.EXPECT().
				UpdateSession("sess_abc123",
					&godo.HostedAgentSessionUpdateRequest{ResumeOnTopoff: &revoked}).
				Return(&do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_abc123",
						Name:      "demo",
						Status:    godo.HostedAgentSessionStatusReady,
					},
				}, nil)

			var buf bytes.Buffer
			config.Out = &buf
			config.Args = []string{"sess_abc123"}
			config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, false)

			require.NoError(t, RunAgentsUpdate(config))
			assert.Contains(t, buf.String(), "will no longer resume automatically")
		})
	})
}

// An empty PATCH body is a 400, so the command has to fail locally and name the
// flag to pass. gomock fails the test if UpdateSession is called anyway.
func TestRunAgentsUpdate_NothingToUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"sess_abc123"}

		err := RunAgentsUpdate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nothing to update")
		assert.Contains(t, err.Error(), doctl.ArgAgentResumeOnTopoff)
	})
}

// `--resume-on-topoff false` is not a value, it is the flag set to true plus a
// stray argument. Left unexplained that reads as "too many arguments" on update
// and as a missing session named "false" on launch.
func TestRejectBoolFlagValueAsArg(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, true)

		for _, value := range []string{"true", "false", "1", "0"} {
			t.Run(value, func(t *testing.T) {
				withStubbedOSArgs(t, []string{"doctl", "harness-runtime", "update", "demo", "--resume-on-topoff", value})

				err := rejectBoolFlagValueAsArg(config, doctl.ArgAgentResumeOnTopoff)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "attached with an equals sign")
				assert.Contains(t, err.Error(), "--resume-on-topoff="+value)
			})
		}

		t.Run("equals form is fine", func(t *testing.T) {
			withStubbedOSArgs(t, []string{"doctl", "harness-runtime", "update", "demo", "--resume-on-topoff=false"})
			assert.NoError(t, rejectBoolFlagValueAsArg(config, doctl.ArgAgentResumeOnTopoff))
		})

		// A session legitimately named "1" is why the check reads os.Args for a
		// literal directly after the flag rather than scanning the positionals.
		t.Run("bool-looking session name is not a misparse", func(t *testing.T) {
			withStubbedOSArgs(t, []string{"doctl", "harness-runtime", "update", "1", "--resume-on-topoff"})
			assert.NoError(t, rejectBoolFlagValueAsArg(config, doctl.ArgAgentResumeOnTopoff))
		})
	})

	t.Run("silent when the flag was never passed", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			withStubbedOSArgs(t, []string{"doctl", "harness-runtime", "update", "demo", "--resume-on-topoff", "false"})
			assert.NoError(t, rejectBoolFlagValueAsArg(config, doctl.ArgAgentResumeOnTopoff))
		})
	})
}

func withStubbedOSArgs(t *testing.T, args []string) {
	t.Helper()
	prev := os.Args
	os.Args = args
	t.Cleanup(func() { os.Args = prev })
}

func TestPrintSessionShowCard_ResumeOnTopoff(t *testing.T) {
	t.Run("shown when granted", func(t *testing.T) {
		var buf bytes.Buffer
		printSessionShowCard(&buf, &do.HostedAgentSession{
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID:      "sess_1",
				Name:           "demo",
				Status:         godo.HostedAgentSessionStatusReady,
				ResumeOnTopoff: true,
			},
		})
		assert.Contains(t, buf.String(), "Resume on top-off")
	})

	t.Run("absent when withheld", func(t *testing.T) {
		var buf bytes.Buffer
		printSessionShowCard(&buf, &do.HostedAgentSession{
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID: "sess_1",
				Name:      "demo",
				Status:    godo.HostedAgentSessionStatusReady,
			},
		})
		assert.False(t, strings.Contains(buf.String(), "top-off"))
	})
}
