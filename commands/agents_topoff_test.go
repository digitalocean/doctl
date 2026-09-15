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
// on the manifest create path and as a body field on the config create path,
// because the manifest itself is immutable and shared through Agent Configs
// while the consent is per-session. These tests pin both wire forms, and the
// create-only-ness that the absent server-side update surface implies.

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

// TestRunAgentsLaunch_ResumeOnTopoffRejectedForExistingSession: the server sets
// the consent at create time only, so accepting the flag while attaching would
// leave the user believing an existing session had been enrolled.
func TestRunAgentsLaunch_ResumeOnTopoffRejectedForExistingSession(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "my-session")
		config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, true)

		err := RunAgentsLaunch(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), doctl.ArgAgentResumeOnTopoff)
		assert.Contains(t, err.Error(), "only applies when creating a new session")
	})
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
