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
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	domocks "github.com/digitalocean/doctl/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	yaml "gopkg.in/yaml.v2"
)

const sampleManifest = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
metadata:
  name: test-agent
spec:
  adapter: opencode
`

// sampleFlatManifest is the flat-format equivalent of sampleManifest: no
// envelope, top-level agent key.
const sampleFlatManifest = `name: test-agent
agent: opencode
`

func testAttachStateFromPending(pending *pendingHITL) *attachState {
	if pending == nil {
		pending = &pendingHITL{}
	}
	return &attachState{pending: pending}
}

func TestAgentsCommand(t *testing.T) {
	cmd := Agents()
	assert.NotNil(t, cmd)

	assertCommandNames(t, cmd, "start", "attach", "list", "show", "logs", "approve", "destroy", "pause", "resume", "upload", "download", "start-proxy", "auth", "fork", "rollback", "checkpoint", "triggers")
}

func TestAgents_helpers(t *testing.T) {
	t.Run("hitlOutcomeFor", func(t *testing.T) {
		cases := []struct {
			in      string
			want    godo.HostedAgentHITLOutcome
			wantErr bool
		}{
			{"approve", godo.HostedAgentHITLOutcomeApprove, false},
			{"REJECT", godo.HostedAgentHITLOutcomeReject, false},
			{"defer", godo.HostedAgentHITLOutcomeDefer, false},
			{"maybe", "", true},
		}
		for _, tc := range cases {
			got, err := hitlOutcomeFor(tc.in)
			if tc.wantErr {
				assert.Error(t, err, "input=%q", tc.in)
				continue
			}
			assert.NoError(t, err, "input=%q", tc.in)
			assert.Equal(t, tc.want, got, "input=%q", tc.in)
		}
	})
}

func TestReadManifest(t *testing.T) {
	t.Run("from file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "agent.yaml")
		assert.NoError(t, os.WriteFile(path, []byte(sampleManifest), 0o644))

		raw, err := readManifest(nil, path)
		assert.NoError(t, err)
		assert.Equal(t, sampleManifest, string(raw))
	})

	t.Run("from stdin", func(t *testing.T) {
		raw, err := readManifest(strings.NewReader(sampleManifest), "-")
		assert.NoError(t, err)
		assert.Equal(t, sampleManifest, string(raw))
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readManifest(nil, filepath.Join(t.TempDir(), "nope.yaml"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("empty manifest", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.yaml")
		assert.NoError(t, os.WriteFile(path, []byte("   \n  \t\n"), 0o644))

		_, err := readManifest(nil, path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("expands env references", func(t *testing.T) {
		t.Setenv("DOCTL_TEST_API_KEY", "sk-test-123")
		raw, err := readManifest(strings.NewReader("env:\n  OPENAI_API_KEY: ${DOCTL_TEST_API_KEY}\n"), "-")
		assert.NoError(t, err)
		assert.Equal(t, "env:\n  OPENAI_API_KEY: sk-test-123\n", string(raw))
	})
}

func TestExpandManifestEnv(t *testing.T) {
	t.Run("expands set variables", func(t *testing.T) {
		t.Setenv("DOCTL_TEST_KEY", "value-1")
		t.Setenv("DOCTL_TEST_OTHER", "value-2")
		out, err := expandManifestEnv([]byte("a: ${DOCTL_TEST_KEY}\nb: ${DOCTL_TEST_OTHER}\n"))
		assert.NoError(t, err)
		assert.Equal(t, "a: value-1\nb: value-2\n", string(out))
	})

	t.Run("expands empty-but-set variables", func(t *testing.T) {
		t.Setenv("DOCTL_TEST_EMPTY", "")
		out, err := expandManifestEnv([]byte("a: '${DOCTL_TEST_EMPTY}'\n"))
		assert.NoError(t, err)
		assert.Equal(t, "a: ''\n", string(out))
	})

	t.Run("unset variable errors with its name", func(t *testing.T) {
		_, err := expandManifestEnv([]byte("a: ${DOCTL_TEST_DEFINITELY_UNSET}\n"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DOCTL_TEST_DEFINITELY_UNSET")
	})

	t.Run("reports all missing variables once", func(t *testing.T) {
		t.Setenv("DOCTL_TEST_KEY", "v")
		_, err := expandManifestEnv([]byte("a: ${DOCTL_TEST_MISSING_A}\nb: ${DOCTL_TEST_KEY}\nc: ${DOCTL_TEST_MISSING_B}\nd: ${DOCTL_TEST_MISSING_A}\n"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DOCTL_TEST_MISSING_A, DOCTL_TEST_MISSING_B")
		assert.Equal(t, 1, strings.Count(err.Error(), "DOCTL_TEST_MISSING_A"))
	})

	t.Run("escape produces a literal reference", func(t *testing.T) {
		out, err := expandManifestEnv([]byte("a: $${DOCTL_TEST_DEFINITELY_UNSET}\n"))
		assert.NoError(t, err)
		assert.Equal(t, "a: ${DOCTL_TEST_DEFINITELY_UNSET}\n", string(out))
	})

	t.Run("bare dollar forms are untouched", func(t *testing.T) {
		in := "script: |\n  echo $HOME $1 $(pwd) ${!indirect} ${no spaces allowed}\n"
		out, err := expandManifestEnv([]byte(in))
		assert.NoError(t, err)
		assert.Equal(t, in, string(out))
	})
}

func TestRunAgentsStart(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	assert.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest([]byte(sampleManifest), nil).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_test",
					AgentKind: godo.HostedAgentKindOpenCode,
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		assert.NoError(t, RunAgentsStart(config))
	})
}

func TestRunAgentsStart_WithName(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	assert.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// The manifest sent to the server must carry metadata.name = the flag.
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			DoAndReturn(func(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				var doc map[string]any
				assert.NoError(t, yaml.Unmarshal(manifest, &doc))
				meta, ok := doc["metadata"].(map[any]any)
				assert.True(t, ok, "metadata should be a mapping")
				assert.Equal(t, "my-session", meta["name"])
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_test", Name: "my-session"},
				}, nil
			})

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-session")
		assert.NoError(t, RunAgentsStart(config))
	})
}

func TestInjectManifestName(t *testing.T) {
	t.Run("empty name leaves manifest untouched", func(t *testing.T) {
		out, err := injectManifestName([]byte(sampleManifest), "")
		assert.NoError(t, err)
		assert.Equal(t, sampleManifest, string(out))
	})

	t.Run("sets metadata.name", func(t *testing.T) {
		out, err := injectManifestName([]byte(sampleManifest), "my-session")
		assert.NoError(t, err)
		var doc map[string]any
		assert.NoError(t, yaml.Unmarshal(out, &doc))
		meta, ok := doc["metadata"].(map[any]any)
		assert.True(t, ok)
		assert.Equal(t, "my-session", meta["name"])
	})

	t.Run("overrides an existing metadata.name", func(t *testing.T) {
		const withName = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
metadata:
  name: original
spec:
  adapter: opencode
`
		out, err := injectManifestName([]byte(withName), "override")
		assert.NoError(t, err)
		var doc map[string]any
		assert.NoError(t, yaml.Unmarshal(out, &doc))
		meta := doc["metadata"].(map[any]any)
		assert.Equal(t, "override", meta["name"])
		// unrelated fields survive
		spec := doc["spec"].(map[any]any)
		assert.Equal(t, "opencode", spec["adapter"])
	})

	t.Run("invalid yaml errors", func(t *testing.T) {
		_, err := injectManifestName([]byte("::: not yaml :::"), "x")
		assert.Error(t, err)
	})

	t.Run("flat manifest gets a top-level name", func(t *testing.T) {
		out, err := injectManifestName([]byte("agent: opencode\n"), "my-session")
		assert.NoError(t, err)
		var doc map[string]any
		assert.NoError(t, yaml.Unmarshal(out, &doc))
		assert.Equal(t, "my-session", doc["name"])
		assert.NotContains(t, doc, "metadata")
	})

	t.Run("overrides an existing flat name", func(t *testing.T) {
		out, err := injectManifestName([]byte(sampleFlatManifest), "override")
		assert.NoError(t, err)
		var doc map[string]any
		assert.NoError(t, yaml.Unmarshal(out, &doc))
		assert.Equal(t, "override", doc["name"])
		assert.Equal(t, "opencode", doc["agent"])
		assert.NotContains(t, doc, "metadata")
	})

	t.Run("preserves multi-line spec.skills instructions", func(t *testing.T) {
		const withSkills = `apiVersion: agents.digitalocean.com/v1alpha1
kind: Agent
spec:
  adapter: opencode
  skills:
    - name: example-skill
      description: A test skill with multi-line instructions
      instructions: |
        Step one: do the first thing.
        Step two: do the second thing.

        Step three: finish up.
`
		const wantInstructions = "Step one: do the first thing.\nStep two: do the second thing.\n\nStep three: finish up.\n"

		out, err := injectManifestName([]byte(withSkills), "my-session")
		assert.NoError(t, err)

		var doc map[string]any
		assert.NoError(t, yaml.Unmarshal(out, &doc))
		spec := doc["spec"].(map[any]any)
		skills, ok := spec["skills"].([]any)
		assert.True(t, ok)
		assert.Len(t, skills, 1)
		skill := skills[0].(map[any]any)
		assert.Equal(t, "example-skill", skill["name"])
		assert.Equal(t, wantInstructions, skill["instructions"])
	})
}

func TestManifestUsesLegacyEnvelope(t *testing.T) {
	assert.True(t, manifestUsesLegacyEnvelope([]byte(sampleManifest)))
	assert.False(t, manifestUsesLegacyEnvelope([]byte(sampleFlatManifest)))
	assert.False(t, manifestUsesLegacyEnvelope([]byte("agent: opencode\n")))
	// Unparsable YAML defers to the server for the authoritative error.
	assert.False(t, manifestUsesLegacyEnvelope([]byte("::: not yaml :::")))
}

func TestRunAgentsStart_FlatWithName(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	assert.NoError(t, os.WriteFile(specPath, []byte("agent: opencode\n"), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// The manifest sent to the server must carry a top-level name (flat
		// format), not the legacy metadata.name.
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			DoAndReturn(func(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				var doc map[string]any
				assert.NoError(t, yaml.Unmarshal(manifest, &doc))
				assert.Equal(t, "my-session", doc["name"])
				assert.NotContains(t, doc, "metadata")
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_test", Name: "my-session"},
				}, nil
			})

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-session")
		assert.NoError(t, RunAgentsStart(config))
	})
}

func TestRunAgentsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().ListSessions(nil).Return([]do.HostedAgentSession{}, "", nil)
		assert.NoError(t, RunAgentsList(config))
	})
}

func TestRunAgentsList_Pagination(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		want := &godo.HostedAgentSessionListOptions{
			PageSize: 2,
			Status:   godo.HostedAgentSessionStatusReady,
		}
		tm.hostedAgents.EXPECT().ListSessions(want).Return([]do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"}},
		}, "1561", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentPageSize, 2)
		config.Doit.Set(config.NS, doctl.ArgAgentStatus, string(godo.HostedAgentSessionStatusReady))

		assert.NoError(t, RunAgentsList(config))
		assert.Contains(t, buf.String(), "Next page token: 1561")
	})
}

// TestRunAgentsList_JSONPaginationCleanStdout pins the fix for MARSOHS-235: under
// -o json the pagination hint must not be written to stdout (it would follow the
// JSON array and break parsers like jq). It goes to stderr instead so the cursor
// is still available.
func TestRunAgentsList_JSONPaginationCleanStdout(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		want := &godo.HostedAgentSessionListOptions{PageSize: 2}
		tm.hostedAgents.EXPECT().ListSessions(want).Return([]do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1"}},
		}, "1561", nil)

		Output = "json"
		defer func() { Output = "text" }()

		// Capture stderr, where the human-readable cursor hint must land.
		oldStderr := os.Stderr
		r, w, err := os.Pipe()
		assert.NoError(t, err)
		os.Stderr = w
		defer func() { os.Stderr = oldStderr }()

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentPageSize, 2)

		runErr := RunAgentsList(config)

		assert.NoError(t, w.Close())
		os.Stderr = oldStderr
		stderrOut, _ := io.ReadAll(r)

		assert.NoError(t, runErr)

		stdout := buf.String()
		// stdout must stay clean, parseable JSON with no trailing hint.
		assert.NotContains(t, stdout, "Next page token")
		assert.True(t, json.Valid([]byte(stdout)), "stdout must be valid JSON, got: %q", stdout)
		// the cursor is still surfaced, on stderr.
		assert.Contains(t, string(stderrOut), "Next page token: 1561")
	})
}

func TestRunAgentsList_NameFilter(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		want := &godo.HostedAgentSessionListOptions{Name: "Named-E2E-Test"}
		tm.hostedAgents.EXPECT().ListSessions(want).Return([]do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1", Name: "Named-E2E-Test"}},
		}, "", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentName, "Named-E2E-Test")

		assert.NoError(t, RunAgentsList(config))
		assert.Contains(t, buf.String(), "Named-E2E-Test")
	})
}

func TestRunAgentsShow(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().GetSession("sess_test").Return(&do.HostedAgentSession{
			HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_test"},
		}, nil)
		config.Args = []string{"sess_test"}
		assert.NoError(t, RunAgentsShow(config))
	})
}

func TestRunAgentsDestroy(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().DestroySession("sess_test").Return(nil)
		config.Args = []string{"sess_test"}
		assert.NoError(t, RunAgentsDestroy(config))
	})
}

func TestRunAgentsPause(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().PauseSession("sess_test").Return(nil)
		config.Args = []string{"sess_test"}
		assert.NoError(t, RunAgentsPause(config))
	})
}

func TestRunAgentsResume(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().ResumeSession("sess_test").Return(nil)
		config.Args = []string{"sess_test"}
		assert.NoError(t, RunAgentsResume(config))
	})
}

func TestRunAgentsAuth(t *testing.T) {
	// Keep the wait loop from sleeping through the poll interval in tests.
	orig := agentsAuthPollInterval
	agentsAuthPollInterval = time.Millisecond
	defer func() { agentsAuthPollInterval = orig }()

	t.Run("already connected exits without polling", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				StartProviderAuth("github").
				Return(&godo.HostedAgentProviderAuthStart{Provider: "github", Status: "success"}, nil)
			config.Args = []string{"github"}
			assert.NoError(t, RunAgentsAuth(config))
		})
	})

	t.Run("no-wait prints URL and returns", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				StartProviderAuth("github").
				Return(&godo.HostedAgentProviderAuthStart{
					Provider:   "github",
					Status:     "pending",
					ConnectURL: "https://example.com/connect",
					PollURL:    "https://example.com/poll",
				}, nil)
			config.Args = []string{"github"}
			config.Doit.Set(config.NS, doctl.ArgAgentAuthNoBrowser, true)
			config.Doit.Set(config.NS, doctl.ArgAgentAuthNoWait, true)
			assert.NoError(t, RunAgentsAuth(config))
		})
	})

	t.Run("polls until authorized", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				StartProviderAuth("github").
				Return(&godo.HostedAgentProviderAuthStart{
					Provider:   "github",
					Status:     "pending",
					ConnectURL: "https://example.com/connect",
					PollURL:    "https://example.com/poll",
				}, nil)
			gomock.InOrder(
				tm.hostedAgents.EXPECT().
					PollProviderAuth("github", "https://example.com/poll").
					Return(&godo.HostedAgentProviderAuthPoll{Provider: "github", Status: "pending"}, nil),
				tm.hostedAgents.EXPECT().
					PollProviderAuth("github", "https://example.com/poll").
					Return(&godo.HostedAgentProviderAuthPoll{Provider: "github", Status: "success"}, nil),
			)
			config.Args = []string{"github"}
			config.Doit.Set(config.NS, doctl.ArgAgentAuthNoBrowser, true)
			assert.NoError(t, RunAgentsAuth(config))
		})
	})

	t.Run("requires a provider argument", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Args = []string{}
			assert.Error(t, RunAgentsAuth(config))
		})
	})
}

func TestResolveSessionRef(t *testing.T) {
	t.Run("uuid id passes through without a lookup", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			// No ListSessions expectation: a UUID ref must not trigger a name
			// lookup. Pins the regression where real UUID IDs were mistaken for
			// names.
			const id = "019f275e-96dc-7ea0-98bd-9ecf2a0834c3"
			got, err := resolveSessionRef(config.HostedAgents(), id)
			assert.NoError(t, err)
			assert.Equal(t, id, got)
		})
	})

	t.Run("prefixed id passes through without a lookup", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			got, err := resolveSessionRef(config.HostedAgents(), "sess_abc123")
			assert.NoError(t, err)
			assert.Equal(t, "sess_abc123", got)
		})
	})

	t.Run("empty ref errors", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			_, err := resolveSessionRef(config.HostedAgents(), "")
			assert.Error(t, err)
		})
	})

	t.Run("unique name resolves to id", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListSessions(&godo.HostedAgentSessionListOptions{Name: "my-agent"}).
				Return([]do.HostedAgentSession{
					{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_42", Name: "my-agent"}},
				}, "", nil)

			got, err := resolveSessionRef(config.HostedAgents(), "my-agent")
			assert.NoError(t, err)
			assert.Equal(t, "sess_42", got)
		})
	})

	t.Run("fuzzy server matches are filtered to exact name", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListSessions(&godo.HostedAgentSessionListOptions{Name: "demo"}).
				Return([]do.HostedAgentSession{
					{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_1", Name: "demo"}},
					{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_2", Name: "demo-2"}},
				}, "", nil)

			got, err := resolveSessionRef(config.HostedAgents(), "demo")
			assert.NoError(t, err)
			assert.Equal(t, "sess_1", got)
		})
	})

	t.Run("no match errors", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListSessions(&godo.HostedAgentSessionListOptions{Name: "ghost"}).
				Return([]do.HostedAgentSession{}, "", nil)

			_, err := resolveSessionRef(config.HostedAgents(), "ghost")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no agent session goes by the name")
		})
	})

	t.Run("case-insensitive name match", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListSessions(&godo.HostedAgentSessionListOptions{Name: "My-Agent"}).
				Return([]do.HostedAgentSession{
					{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_42", Name: "my-agent"}},
				}, "", nil)

			got, err := resolveSessionRef(config.HostedAgents(), "My-Agent")
			assert.NoError(t, err)
			assert.Equal(t, "sess_42", got)
		})
	})

	t.Run("terminal sessions are excluded so a live reuse wins", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListSessions(&godo.HostedAgentSessionListOptions{Name: "reused"}).
				Return([]do.HostedAgentSession{
					{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_dead1", Name: "reused", Status: godo.HostedAgentSessionStatusDestroyed}},
					{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_dead2", Name: "reused", Status: godo.HostedAgentSessionStatusFailed}},
					{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_live", Name: "reused", Status: godo.HostedAgentSessionStatusReady}},
				}, "", nil)

			got, err := resolveSessionRef(config.HostedAgents(), "reused")
			assert.NoError(t, err)
			assert.Equal(t, "sess_live", got)
		})
	})

	t.Run("only terminal matches errors as not found", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListSessions(&godo.HostedAgentSessionListOptions{Name: "gone"}).
				Return([]do.HostedAgentSession{
					{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_dead", Name: "gone", Status: godo.HostedAgentSessionStatusDestroyed}},
				}, "", nil)

			_, err := resolveSessionRef(config.HostedAgents(), "gone")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no agent session goes by the name")
		})
	})

	t.Run("ambiguous name errors and lists ids", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListSessions(&godo.HostedAgentSessionListOptions{Name: "dup"}).
				Return([]do.HostedAgentSession{
					{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_a", Name: "dup"}},
					{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_b", Name: "dup"}},
				}, "", nil)

			_, err := resolveSessionRef(config.HostedAgents(), "dup")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "many agent sessions go by the name")
			assert.Contains(t, err.Error(), "sess_a")
			assert.Contains(t, err.Error(), "sess_b")
		})
	})

	t.Run("list error is surfaced", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListSessions(&godo.HostedAgentSessionListOptions{Name: "boom"}).
				Return(nil, "", errors.New("network down"))

			_, err := resolveSessionRef(config.HostedAgents(), "boom")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "network down")
		})
	})
}

// TestRunAgentsDestroy_ByName verifies a session command resolves a name to an
// ID (via the name-filtered list) before acting on it.
func TestRunAgentsDestroy_ByName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListSessions(&godo.HostedAgentSessionListOptions{Name: "my-agent"}).
			Return([]do.HostedAgentSession{
				{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_resolved", Name: "my-agent"}},
			}, "", nil)
		tm.hostedAgents.EXPECT().DestroySession("sess_resolved").Return(nil)

		config.Args = []string{"my-agent"}
		assert.NoError(t, RunAgentsDestroy(config))
	})
}

func TestRunAgentsApprove(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		want := &godo.HostedAgentResolveHITLRequest{
			Outcome: godo.HostedAgentHITLOutcomeApprove,
			Source:  godo.HostedAgentResolutionSourceOutOfBand,
		}
		tm.hostedAgents.EXPECT().ResolveHITL("sess_test", "req_1", want).Return(nil)
		config.Args = []string{"sess_test", "req_1", "approve"}
		assert.NoError(t, RunAgentsApprove(config))
	})
}

func TestRunAgentsUpload(t *testing.T) {
	prevPoll := workspaceTransferPollInterval
	workspaceTransferPollInterval = 0
	defer func() { workspaceTransferPollInterval = prevPoll }()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "main.go")
	contents := []byte("package main\n\nfunc main() {}\n")
	assert.NoError(t, os.WriteFile(localPath, contents, 0o644))
	sha := sha256Hex(contents)

	partServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, contents, body)
		w.WriteHeader(http.StatusOK)
	}))
	defer partServer.Close()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectWorkspaceTransferUpload(t, tm, "sess_test", "src/main.go", int64(len(contents)), sha, false, partServer.URL)

		config.Args = []string{"sess_test"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspacePath, "src/main.go")
		config.Doit.Set(config.NS, doctl.ArgAgentLocalFile, localPath)
		assert.NoError(t, RunAgentsUpload(config))
	})
}

func TestRunAgentsUpload_Archive(t *testing.T) {
	prevPoll := workspaceTransferPollInterval
	workspaceTransferPollInterval = 0
	defer func() { workspaceTransferPollInterval = prevPoll }()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "bundle.tar")
	contents := mustTarBytes(t, "hello.txt", []byte("hello"))
	assert.NoError(t, os.WriteFile(localPath, contents, 0o644))
	sha := sha256Hex(contents)

	partServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer partServer.Close()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectWorkspaceTransferUpload(t, tm, "sess_test", "src", int64(len(contents)), sha, true, partServer.URL)

		config.Args = []string{"sess_test"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspacePath, "src")
		config.Doit.Set(config.NS, doctl.ArgAgentLocalFile, localPath)
		config.Doit.Set(config.NS, doctl.ArgAgentArchive, true)
		assert.NoError(t, RunAgentsUpload(config))
	})
}

func TestRunAgentsUpload_ArchiveRejectsGzip(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "bundle.tgz")
	// gzip magic 1f 8b — enough for validateArchiveUpload to reject before
	// attempting a full gzip/tar parse.
	assert.NoError(t, os.WriteFile(localPath, []byte{0x1f, 0x8b, 0x08, 0x00}, 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"sess_test"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspacePath, "src")
		config.Doit.Set(config.NS, doctl.ArgAgentLocalFile, localPath)
		config.Doit.Set(config.NS, doctl.ArgAgentArchive, true)
		err := RunAgentsUpload(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gzip-compressed")
		assert.Contains(t, err.Error(), "tar -cf")
	})
}

func TestRunAgentsUpload_ArchiveRejectsNonTar(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "bundle.tar")
	assert.NoError(t, os.WriteFile(localPath, []byte("not really a tar, but bytes are bytes"), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"sess_test"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspacePath, "src")
		config.Doit.Set(config.NS, doctl.ArgAgentLocalFile, localPath)
		config.Doit.Set(config.NS, doctl.ArgAgentArchive, true)
		err := RunAgentsUpload(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "uncompressed .tar")
	})
}

func TestValidateArchiveUpload(t *testing.T) {
	t.Run("accepts uncompressed tar", func(t *testing.T) {
		assert.NoError(t, validateArchiveUpload(bytes.NewReader(mustTarBytes(t, "a.txt", []byte("a")))))
	})
	t.Run("rejects gzip", func(t *testing.T) {
		err := validateArchiveUpload(bytes.NewReader([]byte{0x1f, 0x8b, 0x08}))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gzip-compressed")
	})
	t.Run("rejects zip", func(t *testing.T) {
		err := validateArchiveUpload(bytes.NewReader([]byte{'P', 'K', 0x03, 0x04}))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "zip archive")
	})
	t.Run("rejects empty", func(t *testing.T) {
		err := validateArchiveUpload(bytes.NewReader(nil))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

func mustTarBytes(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0o644,
		Size: int64(len(body)),
	}))
	_, err := tw.Write(body)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	return buf.Bytes()
}

func TestRunAgentsUpload_MissingFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"sess_test"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspacePath, "src/main.go")
		config.Doit.Set(config.NS, doctl.ArgAgentLocalFile, filepath.Join(t.TempDir(), "nope.go"))
		err := RunAgentsUpload(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestRunAgentsUpload_FileTooLarge(t *testing.T) {
	prevMax := maxWorkspaceTransferBytes
	maxWorkspaceTransferBytes = 10
	defer func() { maxWorkspaceTransferBytes = prevMax }()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "big.bin")
	assert.NoError(t, os.WriteFile(localPath, make([]byte, maxWorkspaceTransferBytes+1), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"sess_test"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspacePath, "big.bin")
		config.Doit.Set(config.NS, doctl.ArgAgentLocalFile, localPath)
		err := RunAgentsUpload(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "50 GiB")
	})
}

func TestRunAgentsUpload_LargeFile(t *testing.T) {
	prevPoll := workspaceTransferPollInterval
	workspaceTransferPollInterval = 0
	defer func() { workspaceTransferPollInterval = prevPoll }()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "big.bin")
	size := int64(51 << 20)
	contents := bytes.Repeat([]byte{0xab}, int(size))
	assert.NoError(t, os.WriteFile(localPath, contents, 0o644))
	sha := sha256Hex(contents)

	partServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer partServer.Close()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectWorkspaceTransferUpload(t, tm, "sess_test", "big.bin", size, sha, false, partServer.URL)

		config.Args = []string{"sess_test"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspacePath, "big.bin")
		config.Doit.Set(config.NS, doctl.ArgAgentLocalFile, localPath)
		assert.NoError(t, RunAgentsUpload(config))
	})
}

func TestRunAgentsUpload_BatchedPartURLs(t *testing.T) {
	prevPoll := workspaceTransferPollInterval
	workspaceTransferPollInterval = 0
	defer func() { workspaceTransferPollInterval = prevPoll }()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "multi.bin")
	contents := []byte{0x01, 0x02, 0x03}
	assert.NoError(t, os.WriteFile(localPath, contents, 0o644))
	sha := sha256Hex(contents)
	partSize := int64(1)

	partServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer partServer.Close()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateWorkspaceTransfer("sess_test", gomock.Any()).
			DoAndReturn(func(_ string, create *godo.HostedAgentWorkspaceTransferCreateRequest) (*godo.HostedAgentWorkspaceTransfer, error) {
				assert.Equal(t, int64(len(contents)), create.SizeBytes)
				return &godo.HostedAgentWorkspaceTransfer{
					TransferID: "xfer_batch",
					Status:     godo.HostedAgentWorkspaceTransferStatusPending,
					PartSize:   partSize,
				}, nil
			})
		// One batched request for all three parts (batch size 32 > 3).
		tm.hostedAgents.EXPECT().
			CreateWorkspaceTransferPartUploadURLs("sess_test", "xfer_batch", &godo.HostedAgentWorkspaceTransferPartUploadURLsRequest{
				PartNumbers: []int{1, 2, 3},
			}).
			Return(&godo.HostedAgentWorkspaceTransferPartUploadURLs{
				PartURLs: []godo.HostedAgentWorkspaceTransferPartUploadURL{
					{PartNumber: 1, UploadURL: partServer.URL},
					{PartNumber: 2, UploadURL: partServer.URL},
					{PartNumber: 3, UploadURL: partServer.URL},
				},
			}, nil)
		tm.hostedAgents.EXPECT().
			CommitWorkspaceTransfer("sess_test", "xfer_batch", &godo.HostedAgentWorkspaceTransferCommitRequest{SHA256: sha}).
			Return(&godo.HostedAgentWorkspaceTransfer{
				TransferID: "xfer_batch",
				Status:     godo.HostedAgentWorkspaceTransferStatusInProgress,
			}, nil)
		tm.hostedAgents.EXPECT().
			GetWorkspaceTransfer("sess_test", "xfer_batch").
			Return(&godo.HostedAgentWorkspaceTransfer{
				TransferID:   "xfer_batch",
				Status:       godo.HostedAgentWorkspaceTransferStatusCompleted,
				BytesWritten: int64(len(contents)),
			}, nil)

		config.Args = []string{"sess_test"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspacePath, "multi.bin")
		config.Doit.Set(config.NS, doctl.ArgAgentLocalFile, localPath)
		assert.NoError(t, RunAgentsUpload(config))
	})
}

func TestRunAgentsDownload(t *testing.T) {
	prevPoll := workspaceTransferPollInterval
	workspaceTransferPollInterval = 0
	defer func() { workspaceTransferPollInterval = prevPoll }()

	dir := t.TempDir()
	saveTo := filepath.Join(dir, "out.go")
	contents := []byte("package main\n\nfunc main() {}\n")
	wantSum := sha256.Sum256(contents)

	objServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(contents)
	}))
	defer objServer.Close()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateWorkspaceTransfer("sess_test", gomock.Any()).
			DoAndReturn(func(sessionID string, create *godo.HostedAgentWorkspaceTransferCreateRequest) (*godo.HostedAgentWorkspaceTransfer, error) {
				assert.Equal(t, godo.HostedAgentWorkspaceTransferDirectionDownload, create.Direction)
				assert.Equal(t, "src/main.go", create.Path)
				return &godo.HostedAgentWorkspaceTransfer{
					TransferID: "xfer_dl",
					Status:     godo.HostedAgentWorkspaceTransferStatusPending,
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetWorkspaceTransfer("sess_test", "xfer_dl").
			Return(&godo.HostedAgentWorkspaceTransfer{
				TransferID:  "xfer_dl",
				Status:      godo.HostedAgentWorkspaceTransferStatusCompleted,
				SHA256:      hex.EncodeToString(wantSum[:]),
				DownloadURL: objServer.URL,
			}, nil)

		config.Args = []string{"sess_test"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspacePath, "src/main.go")
		config.Doit.Set(config.NS, doctl.ArgAgentSaveTo, saveTo)
		assert.NoError(t, RunAgentsDownload(config))

		got, err := os.ReadFile(saveTo)
		assert.NoError(t, err)
		assert.Equal(t, contents, got)
	})
}

func TestRunAgentsDownload_DiscardOnChecksumMismatch(t *testing.T) {
	prevPoll := workspaceTransferPollInterval
	workspaceTransferPollInterval = 0
	defer func() { workspaceTransferPollInterval = prevPoll }()

	dir := t.TempDir()
	saveTo := filepath.Join(dir, "out.go")

	objServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("wrong bytes"))
	}))
	defer objServer.Close()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateWorkspaceTransfer("sess_test", gomock.Any()).
			Return(&godo.HostedAgentWorkspaceTransfer{
				TransferID: "xfer_dl",
				Status:     godo.HostedAgentWorkspaceTransferStatusPending,
			}, nil)
		wantSHA := sha256.Sum256([]byte("expected"))
		tm.hostedAgents.EXPECT().
			GetWorkspaceTransfer("sess_test", "xfer_dl").
			Return(&godo.HostedAgentWorkspaceTransfer{
				TransferID:  "xfer_dl",
				Status:      godo.HostedAgentWorkspaceTransferStatusCompleted,
				SHA256:      hex.EncodeToString(wantSHA[:]),
				DownloadURL: objServer.URL,
			}, nil)

		config.Args = []string{"sess_test"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspacePath, "src/main.go")
		config.Doit.Set(config.NS, doctl.ArgAgentSaveTo, saveTo)
		err := RunAgentsDownload(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")

		_, statErr := os.Stat(saveTo)
		assert.True(t, os.IsNotExist(statErr), "destination must not exist after a failed transfer")
	})
}

// TestHostedAgentEventDecodesSPIWire regression-guards the SPI envelope
// (type/data/timestamp/tenant_id) -> godo HostedAgentEvent mapping.
func TestHostedAgentEventDecodesSPIWire(t *testing.T) {
	const frame = `{"event_id":"01KTBXPBY60VYC5YKF6AKDX0ZS","run_id":"run-7f16719a-da1c-449d-a4ca-18e524bb63e3","tenant_id":"120","session_id":"sess_5a1ff33e","timestamp":"2026-06-05T12:56:24.774753219Z","seq":0,"type":"run.token_delta","data":{"text":"Paris"}}`

	var ev godo.HostedAgentEvent
	assert.NoError(t, json.Unmarshal([]byte(frame), &ev))

	assert.Equal(t, "01KTBXPBY60VYC5YKF6AKDX0ZS", ev.EventID)
	assert.Equal(t, "run-7f16719a-da1c-449d-a4ca-18e524bb63e3", ev.RunID)
	assert.Equal(t, "sess_5a1ff33e", ev.SessionID)
	assert.Equal(t, uint64(120), ev.TeamID)
	assert.Equal(t, godo.HostedAgentEventKindTokenChunk, ev.Kind)
	assert.False(t, ev.At.IsZero(), "timestamp should be parsed from the wire `timestamp` field")
	assert.JSONEq(t, `{"text":"Paris"}`, string(ev.Payload))

	var buf bytes.Buffer
	renderEvent(&buf, ev)
	assert.Equal(t, "Paris", buf.String())
}

func TestRenderEvent(t *testing.T) {
	cases := []struct {
		name    string
		kind    godo.HostedAgentEventKind
		runID   string
		payload string
		want    string
	}{
		{"token chunk", godo.HostedAgentEventKindTokenChunk, "", `{"text":"Paris"}`, "Paris"},
		{"run started", godo.HostedAgentEventKindRunStarted, "run-1", `{"agent":"codex"}`, "\n▶ run started\n"},
		{"run started no agent", godo.HostedAgentEventKindRunStarted, "run-2", `{}`, "\n▶ run started\n"},
		{"tool call started", godo.HostedAgentEventKindToolCallStarted, "", `{"tool_call_id":"t1","name":"bash"}`, "\n▸ bash\n"},
		{"tool call completed", godo.HostedAgentEventKindToolCallCompleted, "", `{"ok":true,"duration_ms":12,"summary":"ran ls"}`, "  ✓ ran ls (12ms)\n"},
		{"tool call failed", godo.HostedAgentEventKindToolCallCompleted, "", `{"ok":false,"duration_ms":3,"summary":"boom"}`, "  ✗ boom (3ms)\n"},
		{"run completed", godo.HostedAgentEventKindRunCompleted, "", `{"total_tokens_in":3,"total_tokens_out":5,"run_cost_micros":1234}`, "\n✓ run complete · 3 in / 5 out tokens · $0.0012\n" + runSeparator + "\n"},
		{"run completed no usage", godo.HostedAgentEventKindRunCompleted, "", `{"total_tokens_in":0,"total_tokens_out":0,"run_cost_micros":0}`, "\n✓ run complete\n" + runSeparator + "\n"},
		{"run failed", godo.HostedAgentEventKindRunFailed, "", `{"code":5,"message":"hitl rejected"}`, "\n✗ run failed: hitl rejected (code 5)\n" + runSeparator + "\n"},
		{"hitl resolved", godo.HostedAgentEventKindHITLResolved, "", `{"hitl_id":"hitl_1","outcome":1}`, "\nhitl_1 approve\n"},
		{"session updated", godo.HostedAgentEventKindSessionUpdated, "", `{}`, "\n• session updated\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			renderEvent(&buf, godo.HostedAgentEvent{
				Kind:    tc.kind,
				RunID:   tc.runID,
				Payload: json.RawMessage(tc.payload),
			})
			assert.Equal(t, tc.want, buf.String())
		})
	}
}

// TestErrorResponseSurfacesNestedMessage pins the {"error":{"code","message"}}
// decode that harness-api uses (vs godo's historical top-level {"message"}).
func TestErrorResponseSurfacesNestedMessage(t *testing.T) {
	const body = `{"error":{"code":400,"message":"forward input to OHR: ohr attach: connection error"}}`
	er := &godo.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Request:    httptest.NewRequest(http.MethodPost, "http://harness/v2/agents/sessions/sess_x/input", nil),
		},
	}
	assert.NoError(t, json.Unmarshal([]byte(body), er))
	assert.Contains(t, er.Error(), "forward input to OHR: ohr attach: connection error")
}

// TestRunAgentsStart_SkillsEnvSizeCapError pins that harness-api's
// HARNESS_SKILLS env-size-cap rejection (agentspec.validateSkillsEnvSize,
// returned as a 400 with the nested {"error":{"code":...,"message":...}}
// envelope) surfaces to the CLI user as the server's own readable message,
// not a raw JSON/HTTP dump.
func TestRunAgentsStart_SkillsEnvSizeCapError(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	assert.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	const wantMessage = `agentspec: spec.skills would encode to 65537 bytes as the HARNESS_SKILLS guest env value, exceeding the sandbox's 65536-byte limit; trim instructions/descriptions or reduce the number of skills (temporary limit while skill delivery rides an env var)`
	const body = `{"error":{"code":400,"message":"` + wantMessage + `"}}`
	er := &godo.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Request:    httptest.NewRequest(http.MethodPost, "http://harness/v2/agents/sessions", nil),
		},
	}
	require.NoError(t, json.Unmarshal([]byte(body), er))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			Return(nil, er)

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		err := RunAgentsStart(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), wantMessage)
		assert.NotContains(t, err.Error(), `{"error"`)
		assert.NotContains(t, err.Error(), `{"message"`)
	})
}

func TestRenderEventHITLRequested(t *testing.T) {
	const fullID = "019f1dfc-f017-70e2-9eac-2ea470a55ac2"
	var buf bytes.Buffer
	renderEvent(&buf, godo.HostedAgentEvent{
		Kind:    godo.HostedAgentEventKindHITLRequested,
		Payload: json.RawMessage(`{"hitl_id":"` + fullID + `","payload":{"command":"rm -rf /tmp/x"}}`),
	})
	out := buf.String()
	assert.Contains(t, out, "Approval required")
	assert.Contains(t, out, "rm -rf /tmp/x")
	assert.Contains(t, out, fullID)
	assert.NotContains(t, out, "#a55ac2")
	assert.NotContains(t, out, "approve / reject / defer")
	assert.NotContains(t, out, "Action requires approval")
}

// TestRenderEventHITLRequestedTopLevelDetails covers the harness-api wire
// shape: HITLRequest.action + details.command at the top level of the event
// data (no nested "payload"). This is what reattach re-injects.
func TestRenderEventHITLRequestedTopLevelDetails(t *testing.T) {
	const fullID = "019f1dfc-f017-70e2-9eac-2ea470a55ac2"
	var buf bytes.Buffer
	renderEvent(&buf, godo.HostedAgentEvent{
		Kind: godo.HostedAgentEventKindHITLRequested,
		Payload: json.RawMessage(`{
			"hitl_id":"` + fullID + `",
			"action":"HITL_ACTION_BASH",
			"details":{"command":"mkdir /tmp/hitl-test && echo done"}
		}`),
	})
	out := buf.String()
	assert.Contains(t, out, "Approval required")
	assert.Contains(t, out, "mkdir /tmp/hitl-test && echo done")
	assert.Contains(t, out, fullID)
	assert.NotContains(t, out, "action pending")
}

func TestHITLRequestedPayloadFields(t *testing.T) {
	t.Run("top-level details.command", func(t *testing.T) {
		var p hitlRequestedPayload
		assert.NoError(t, json.Unmarshal([]byte(`{
			"request_id":"req-1",
			"action":"HITL_ACTION_BASH",
			"details":{"command":"echo hi"}
		}`), &p))
		assert.Equal(t, "req-1", p.id())
		assert.Equal(t, "echo hi", p.commandSummary())
		assert.Equal(t, "HITL_ACTION_BASH", p.actionLabel())
	})
	t.Run("nested payload still preferred for command", func(t *testing.T) {
		var p hitlRequestedPayload
		assert.NoError(t, json.Unmarshal([]byte(`{
			"hitl_id":"h1",
			"details":{"command":"from-details"},
			"payload":{"command":"from-payload"}
		}`), &p))
		assert.Equal(t, "h1", p.id())
		assert.Equal(t, "from-payload", p.commandSummary())
	})
	t.Run("github action falls back to friendly label", func(t *testing.T) {
		var p hitlRequestedPayload
		assert.NoError(t, json.Unmarshal([]byte(`{
			"hitl_id":"h2",
			"action":"HITL_ACTION_GITHUB_CREATE_PR",
			"details":{"repo":"digitalocean/doctl","title":"fix stuff","base":"main","branch":"fix"}
		}`), &p))
		assert.Equal(t, "create a pull request", p.commandSummary())
	})
}

// TestHITLCommandSummaryNested covers the recursive command search used to pull
// a command out of adapter-specific nested payload shapes.
func TestHITLCommandSummaryNested(t *testing.T) {
	assert.Equal(t, "ls -la", hitlCommandSummary(map[string]any{
		"tool": map[string]any{"input": map[string]any{"command": "ls -la"}},
	}))
	assert.Equal(t, "git status", hitlCommandSummary(map[string]any{
		"bash": map[string]any{"cmd": "git status"},
	}))
}

// TestToolCommandLine verifies the tool-call line prefers the real command from
// arguments/input and falls back to the tool name.
func TestToolCommandLine(t *testing.T) {
	var p toolCallStartedPayload
	assert.NoError(t, json.Unmarshal([]byte(`{"name":"bash","arguments":{"command":"uname -a"}}`), &p))
	assert.Equal(t, "uname -a", p.commandLine())

	var p2 toolCallStartedPayload
	assert.NoError(t, json.Unmarshal([]byte(`{"name":"bash","input":{"cmd":"ls"}}`), &p2))
	assert.Equal(t, "ls", p2.commandLine())

	var p3 toolCallStartedPayload
	assert.NoError(t, json.Unmarshal([]byte(`{"name":"read_file"}`), &p3))
	assert.Equal(t, "read_file", p3.commandLine())
}

// TestRenderMarkdownColorizesCode confirms that, with styling enabled, a code
// block is rendered with ANSI escape sequences (syntax highlighting), not raw.
func TestRenderMarkdownColorizesCode(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = true
	defer func() { stylingEnabled = prev }()

	out := renderMarkdown("```python\nprint('hello')\n```\n")
	assert.Contains(t, out, "\x1b[", "code block should contain ANSI color escapes")
	assert.NotContains(t, out, "```", "markdown fences should be consumed, not printed literally")
}

// TestNormalizeCodeFences covers the salvage path for the malformed Markdown
// the agent commonly streams: fences glued to prose and/or an unclosed block.
func TestNormalizeCodeFences(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"fence glued to sentence gets its own line",
			"Here is code.```python\nprint(1)\n```\n",
			"Here is code.\n```python\nprint(1)\n```\n",
		},
		{
			"unclosed block gets a closing fence",
			"see:\n```python\nprint(1)",
			"see:\n```python\nprint(1)\n```\n",
		},
		{
			"well-formed input is unchanged",
			"```go\nfmt.Println()\n```\n",
			"```go\nfmt.Println()\n```\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeCodeFences(tc.in))
		})
	}
}

// TestRenderMarkdownSalvagesGluedFence is the end-to-end check: a fence glued to
// a sentence (as the agent emits) still renders as a highlighted code block.
func TestRenderMarkdownSalvagesGluedFence(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = true
	defer func() { stylingEnabled = prev }()

	out := renderMarkdown("It's straightforward.```python\nfor n in range(3):\n    print(n)\n```\n")
	assert.Contains(t, out, "\x1b[", "glued-fence code should still be syntax-highlighted")
	assert.NotContains(t, out, "```", "fences should be consumed, not printed literally")
}

func TestHITLCommandSummary(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]any
		want    string
	}{
		{"command key", map[string]any{"command": "git status"}, "git status"},
		{"cmd key", map[string]any{"cmd": "ls -la"}, "ls -la"},
		{"argv joins", map[string]any{"argv": []any{"git", "switch", "-c", "feat"}}, "git switch -c feat"},
		{"nested details", map[string]any{"details": map[string]any{"command": "rm -rf x"}}, "rm -rf x"},
		{"action fallback", map[string]any{"action": string(godo.HostedAgentHITLActionGitHubCreatePR)}, "create a pull request"},
		{"unknown action passthrough", map[string]any{"kind": "HITL_ACTION_CUSTOM"}, "HITL_ACTION_CUSTOM"},
		{"empty", map[string]any{}, ""},
		{"nil", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, hitlCommandSummary(tc.payload))
		})
	}
}

func TestHITLOutcomeForSelection(t *testing.T) {
	assert.Equal(t, godo.HostedAgentHITLOutcomeApprove, hitlOutcomeForSelection(0))
	assert.Equal(t, godo.HostedAgentHITLOutcomeReject, hitlOutcomeForSelection(1))
	assert.Equal(t, godo.HostedAgentHITLOutcomeDefer, hitlOutcomeForSelection(2))
	assert.Equal(t, godo.HostedAgentHITLOutcomeApprove, hitlOutcomeForSelection(99))
}

// TestHITLMenuPromptPlain verifies the arrow menu with styling disabled: the
// selected option is bracketed and the queue depth is surfaced.
func TestHITLMenuPromptPlain(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	defer func() { stylingEnabled = prev }()

	assert.Contains(t, hitlMenuPrompt(0, 1), "[Approve (y)]")
	assert.Contains(t, hitlMenuPrompt(1, 1), "[Reject (n)]")
	assert.Contains(t, hitlMenuPrompt(2, 1), "[Defer (d)]")
	// Unselected options still advertise their shortcut key.
	assert.Contains(t, hitlMenuPrompt(0, 1), "Reject (n)")
	assert.Contains(t, hitlMenuPrompt(0, 3), "3 pending")
}

// TestHandleAttachByteCursorMovement covers left/right caret motion and
// in-place insert/backspace on the text prompt (non-HITL path).
func TestHandleAttachByteCursorMovement(t *testing.T) {
	typewrite := func(t *testing.T, state *attachState, s string) {
		t.Helper()
		for i := 0; i < len(s); i++ {
			stop, err := handleAttachByte(nil, nil, "sess", s[i], state)
			assert.NoError(t, err)
			assert.False(t, stop)
		}
	}
	arrow := func(t *testing.T, state *attachState, dir byte) {
		t.Helper()
		for _, b := range []byte{0x1b, '[', dir} {
			stop, err := handleAttachByte(nil, nil, "sess", b, state)
			assert.NoError(t, err)
			assert.False(t, stop)
		}
	}

	t.Run("left/right move the caret and insert mid-line", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		typewrite(t, state, "abcde")
		assert.Equal(t, 5, state.cursor)

		arrow(t, state, 'D') // left
		arrow(t, state, 'D')
		assert.Equal(t, 3, state.cursor)

		typewrite(t, state, "X")
		assert.Equal(t, "abcXde", string(state.lineBuf))
		assert.Equal(t, 4, state.cursor)

		arrow(t, state, 'C') // right
		assert.Equal(t, 5, state.cursor)
	})

	t.Run("left clamps at start; right clamps at end", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		typewrite(t, state, "ab")
		for i := 0; i < 5; i++ {
			arrow(t, state, 'D')
		}
		assert.Equal(t, 0, state.cursor)
		for i := 0; i < 5; i++ {
			arrow(t, state, 'C')
		}
		assert.Equal(t, 2, state.cursor)
	})

	t.Run("backspace deletes before the caret", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		typewrite(t, state, "abcd")
		arrow(t, state, 'D')
		arrow(t, state, 'D') // caret before 'c'
		stop, err := handleAttachByte(nil, nil, "sess", 0x7f, state)
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "acd", string(state.lineBuf))
		assert.Equal(t, 1, state.cursor)
	})

	t.Run("enter clears the caret", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				SendInput("sess", &godo.HostedAgentSendInputRequest{Text: "hi"}).
				Return(&godo.HostedAgentSendInputResponse{RunID: "run_1"}, nil)

			state := newAttachState(io.Discard, &pendingHITL{})
			state.display.setRaw(true)
			typewrite(t, state, "hi")
			stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 0x0d, state)
			assert.NoError(t, err)
			assert.False(t, stop)
			assert.Equal(t, "", string(state.lineBuf))
			assert.Equal(t, 0, state.cursor)
		})
	})
}

// TestAttachStateHITLSelection covers the arrow-key selection wrap-around and
// that the prompt reflects the current selection.
func TestAttachStateHITLSelection(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	defer func() { stylingEnabled = prev }()

	pending := &pendingHITL{}
	s := newAttachState(io.Discard, pending)
	assert.Equal(t, "> ", s.promptString(), "no menu without a pending approval")

	pending.set("h1", "HITL_ACTION_BASH")
	assert.Contains(t, s.promptString(), "[Approve (y)]")

	s.moveHITLSelection(1)
	assert.Equal(t, 1, s.hitlSelection())
	assert.Contains(t, s.promptString(), "[Reject (n)]")

	s.moveHITLSelection(-1) // back to approve
	s.moveHITLSelection(-1) // wraps to defer
	assert.Equal(t, 2, s.hitlSelection())

	s.resetHITLSelection()
	assert.Equal(t, 0, s.hitlSelection())
}

// TestMsgAccumulatorPlain confirms buffered tokens flush as-is when styling is
// off (no markdown rewriting) and that flush resets the buffer.
func TestMsgAccumulatorPlain(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	defer func() { stylingEnabled = prev }()

	var buf bytes.Buffer
	acc := &msgAccumulator{}
	acc.add("## Title\n")
	acc.add("some **body**")
	acc.flush(&buf)
	assert.Equal(t, "## Title\nsome **body**\n", buf.String())

	buf.Reset()
	acc.flush(&buf) // nothing buffered
	assert.Equal(t, "", buf.String())
}

// TestAttachLoopAcknowledgesSend: a successful submit prints a "waiting"
// acknowledgement immediately so the multi-second wait for the first token
// doesn't read as a hang.
func TestAttachLoopAcknowledgesSend(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		tm.hostedAgents.EXPECT().
			SendInput("sess_x", &godo.HostedAgentSendInputRequest{Text: "What is the capital of France?"}).
			Return(&godo.HostedAgentSendInputResponse{RunID: "run-abc"}, nil)

		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("What is the capital of France?\n"), testAttachStateFromPending(nil))
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "waiting for the agent")
	})
}

// TestAttachLoopHITLLetterShortcut covers the line-mode HITL path. The
// raw-mode keystroke path needs a real PTY; see TestReadHITLKeystroke.
func TestAttachLoopHITLLetterShortcut(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		outcome godo.HostedAgentHITLOutcome
	}{
		{"approve y", "y\n", godo.HostedAgentHITLOutcomeApprove},
		{"approve YES", "YES\n", godo.HostedAgentHITLOutcomeApprove},
		{"approve a", "a\n", godo.HostedAgentHITLOutcomeApprove},
		{"reject n", "n\n", godo.HostedAgentHITLOutcomeReject},
		{"reject no", "no\n", godo.HostedAgentHITLOutcomeReject},
		{"reject r", "r\n", godo.HostedAgentHITLOutcomeReject},
		{"defer d", "d\n", godo.HostedAgentHITLOutcomeDefer},
		{"defer defer", "defer\n", godo.HostedAgentHITLOutcomeDefer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				var buf bytes.Buffer
				config.Out = &buf
				pending := &pendingHITL{}
				pending.set("hitl_42")

				tm.hostedAgents.EXPECT().
					ResolveHITL("sess_x", "hitl_42", &godo.HostedAgentResolveHITLRequest{
						Outcome: tc.outcome,
						Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
					}).Return(nil)

				err := attachLoop(config, config.HostedAgents(), "sess_x",
					strings.NewReader(tc.input), testAttachStateFromPending(pending))
				assert.NoError(t, err)
				assert.Contains(t, buf.String(), "[y/n/d] > ")
			})
		})
	}
}

// TestAttachLoopClearsPendingAfterResolve guards the "press y, then had to
// press Enter again" bug: after a successful ResolveHITL, attachLoop must
// clear `pending` client-side instead of waiting for the SSE HITLResolved
// echo. Otherwise the next iteration re-enters the HITL branch and blocks.
func TestAttachLoopClearsPendingAfterResolve(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		pending := &pendingHITL{}
		pending.set("hitl_42")

		tm.hostedAgents.EXPECT().
			ResolveHITL("sess_x", "hitl_42", &godo.HostedAgentResolveHITLRequest{
				Outcome: godo.HostedAgentHITLOutcomeApprove,
				Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
			}).Return(nil)

		// Line-mode `y\n` exercises the same resolve+clear path; the raw-mode
		// branch shares the clearIf call.
		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("y\n"), testAttachStateFromPending(pending))
		assert.NoError(t, err)
		assert.Equal(t, "", pending.get(), "pending must be cleared after successful resolve")
	})
}

// TestAttachLoopKeepsPendingOnResolveError: if ResolveHITL fails, we must NOT
// clear pending — otherwise the user loses their chance to retry the approval.
func TestAttachLoopKeepsPendingOnResolveError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		pending := &pendingHITL{}
		pending.set("hitl_42")

		tm.hostedAgents.EXPECT().
			ResolveHITL("sess_x", "hitl_42", &godo.HostedAgentResolveHITLRequest{
				Outcome: godo.HostedAgentHITLOutcomeApprove,
				Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
			}).Return(errors.New("boom"))

		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("y\n"), testAttachStateFromPending(pending))
		assert.NoError(t, err)
		assert.Equal(t, "hitl_42", pending.get(), "pending must survive a failed resolve")
		assert.Contains(t, buf.String(), "resolve failed: boom")
	})
}

// TestAttachLoopHITLShortcutIgnoredWithoutPending: a bare `y` with no pending
// HITL is sent as regular input, not silently swallowed as an approval.
func TestAttachLoopHITLShortcutIgnoredWithoutPending(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		tm.hostedAgents.EXPECT().
			SendInput("sess_x", &godo.HostedAgentSendInputRequest{Text: "y"}).
			Return(&godo.HostedAgentSendInputResponse{RunID: "run-1"}, nil)

		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("y\n"), testAttachStateFromPending(&pendingHITL{}))
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "waiting for the agent")
	})
}

func TestHitlLetterShortcut(t *testing.T) {
	cases := []struct {
		in   string
		want godo.HostedAgentHITLOutcome
		ok   bool
	}{
		{"y", godo.HostedAgentHITLOutcomeApprove, true},
		{"YES", godo.HostedAgentHITLOutcomeApprove, true},
		{"a", godo.HostedAgentHITLOutcomeApprove, true},
		{"A", godo.HostedAgentHITLOutcomeApprove, true},
		{"n", godo.HostedAgentHITLOutcomeReject, true},
		{"No", godo.HostedAgentHITLOutcomeReject, true},
		{"r", godo.HostedAgentHITLOutcomeReject, true},
		{"R", godo.HostedAgentHITLOutcomeReject, true},
		{"d", godo.HostedAgentHITLOutcomeDefer, true},
		{"defer", godo.HostedAgentHITLOutcomeDefer, true},
		{"yes please", "", false},
		{"", "", false},
		{"approve", "", false},
	}
	for _, tc := range cases {
		got, ok := hitlLetterShortcut(tc.in)
		assert.Equal(t, tc.ok, ok, "input=%q", tc.in)
		if ok {
			assert.Equal(t, tc.want, got, "input=%q", tc.in)
		}
	}
}

func TestAttachPrompt(t *testing.T) {
	p := &pendingHITL{}
	assert.Equal(t, "> ", attachPrompt(p), "empty queue")

	p.set("hitl_42")
	assert.Equal(t, "[y/n/d] > ", attachPrompt(p), "exactly one pending — no count noise")

	p.set("hitl_43")
	p.set("hitl_44")
	assert.Equal(t, "[y/n/d] (3 pending) > ", attachPrompt(p), "count surfaces when >1 queued")

	p.clearIf("hitl_43")
	assert.Equal(t, "[y/n/d] (2 pending) > ", attachPrompt(p), "count reflects mid-queue removal")

	p.clearIf("hitl_42")
	p.clearIf("hitl_44")
	assert.Equal(t, "> ", attachPrompt(p), "drained")
}

// TestPendingHITLQueue locks in the FIFO + dedupe + arbitrary-position-remove
// semantics that multi-HITL UX depends on.
func TestPendingHITLQueue(t *testing.T) {
	t.Run("FIFO order; get returns oldest", func(t *testing.T) {
		p := &pendingHITL{}
		p.set("h1")
		p.set("h2")
		p.set("h3")
		assert.Equal(t, "h1", p.get())
		assert.Equal(t, 3, p.len())

		p.clearIf("h1")
		assert.Equal(t, "h2", p.get(), "head advances to next oldest after clear")
		assert.Equal(t, 2, p.len())
	})

	t.Run("dedupe: re-setting same id is a no-op", func(t *testing.T) {
		p := &pendingHITL{}
		p.set("h1")
		p.set("h1")
		p.set("h1", "bash")
		assert.Equal(t, 1, p.len(), "duplicate set must not enlarge the queue (SSE replay safe)")
	})

	t.Run("empty id is rejected", func(t *testing.T) {
		p := &pendingHITL{}
		p.set("")
		assert.Equal(t, 0, p.len())
	})

	t.Run("clearIf removes from any position, not just head", func(t *testing.T) {
		p := &pendingHITL{}
		p.set("h1")
		p.set("h2")
		p.set("h3")

		p.clearIf("h2")
		assert.Equal(t, 2, p.len())
		assert.Equal(t, "h1", p.get(), "head is unaffected when middle is removed")

		ids := []string{}
		for _, e := range p.list() {
			ids = append(ids, e.id)
		}
		assert.Equal(t, []string{"h1", "h3"}, ids, "order preserved across mid-queue removal")
	})

	t.Run("clearIf on unknown id is a no-op", func(t *testing.T) {
		p := &pendingHITL{}
		p.set("h1")
		p.clearIf("h999")
		assert.Equal(t, 1, p.len())
	})

	t.Run("reset drains the whole queue and reports the count", func(t *testing.T) {
		p := &pendingHITL{}
		p.set("h1")
		p.set("h2")
		p.set("h3")

		assert.Equal(t, 3, p.reset(), "reset returns how many were cleared")
		assert.Equal(t, 0, p.len(), "queue is empty after reset")
		assert.Equal(t, "", p.get(), "no head after reset")

		assert.Equal(t, 0, p.reset(), "reset on an empty queue clears nothing")
	})

	t.Run("action label is carried through to /pending", func(t *testing.T) {
		p := &pendingHITL{}
		p.set("h1", "HITL_ACTION_BASH")
		p.set("h2") // no action provided
		list := p.list()
		assert.Equal(t, "HITL_ACTION_BASH", list[0].action)
		assert.Equal(t, "", list[1].action)
	})
}

// TestSingleKeystrokeAdvancesQueue: after resolving the head, the next
// keystroke must target the next-oldest entry without the user retyping
// the id. This is the core multi-HITL UX contract.
func TestSingleKeystrokeAdvancesQueue(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		pending := &pendingHITL{}
		pending.set("h1", "HITL_ACTION_BASH")
		pending.set("h2", "HITL_ACTION_BASH")

		// "y\n" approves head (h1); the loop sees pending still has h2 and
		// drops back into HITL keystroke mode. "n\n" rejects h2.
		tm.hostedAgents.EXPECT().ResolveHITL("sess_x", "h1", &godo.HostedAgentResolveHITLRequest{
			Outcome: godo.HostedAgentHITLOutcomeApprove,
			Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
		}).Return(nil)
		tm.hostedAgents.EXPECT().ResolveHITL("sess_x", "h2", &godo.HostedAgentResolveHITLRequest{
			Outcome: godo.HostedAgentHITLOutcomeReject,
			Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
		}).Return(nil)

		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("y\nn\n"), testAttachStateFromPending(pending))
		assert.NoError(t, err)
		assert.Equal(t, 0, pending.len(), "both HITLs must be drained after two keystrokes")

		out := buf.String()
		assert.Contains(t, out, "[y/n/d] (2 pending) > ", "first prompt shows the multi-pending count")
		assert.Contains(t, out, "[y/n/d] > ", "after resolving one, prompt drops to plain HITL prompt")
	})
}

// TestListPendingHITLs covers the /pending command output for empty, single,
// and multi-entry queues.
func TestListPendingHITLs(t *testing.T) {
	t.Run("empty queue", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
			var buf bytes.Buffer
			config.Out = &buf
			assert.NoError(t, listPendingHITLs(config, &pendingHITL{}))
			assert.Contains(t, buf.String(), "(no HITL approvals pending)")
		})
	})

	t.Run("multi-entry queue marks head with *", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
			var buf bytes.Buffer
			config.Out = &buf
			p := &pendingHITL{}
			p.set("h1", "HITL_ACTION_BASH")
			p.set("h2", "HITL_ACTION_GITHUB_COMMIT_PUSH")
			p.set("h3") // no action

			assert.NoError(t, listPendingHITLs(config, p))
			out := buf.String()
			assert.Contains(t, out, "3 HITL approval(s) pending")
			assert.Contains(t, out, "* h1  (HITL_ACTION_BASH)", "head marked with *")
			assert.Contains(t, out, "  h2  (HITL_ACTION_GITHUB_COMMIT_PUSH)", "non-head not marked")
			assert.Contains(t, out, "  h3\n", "missing action falls back to id only")
		})
	})
}

// TestHandleAttachCommandPending exercises the /pending verb dispatch.
func TestHandleAttachCommandPending(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		p := &pendingHITL{}
		p.set("h1", "HITL_ACTION_BASH")

		assert.NoError(t, handleAttachCommand(config, config.HostedAgents(), "sess_x", "/pending", p))
		assert.Contains(t, buf.String(), "1 HITL approval(s) pending")
		assert.Contains(t, buf.String(), "* h1  (HITL_ACTION_BASH)")
	})
}

// TestResolveFromAttachClearsQueue: an explicit /a <id> must remove that id
// from the queue (not just the head), mirroring keystroke-resolve semantics.
func TestResolveFromAttachClearsQueue(t *testing.T) {
	withTestClient(t, func(_ *CmdConfig, tm *tcMocks) {
		p := &pendingHITL{}
		p.set("h1")
		p.set("h2")
		p.set("h3")

		tm.hostedAgents.EXPECT().
			ResolveHITL("sess_x", "h2", &godo.HostedAgentResolveHITLRequest{
				Outcome: godo.HostedAgentHITLOutcomeApprove,
				Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
			}).Return(nil)

		err := resolveFromAttach(tm.hostedAgents, "sess_x",
			[]string{"/a", "h2"}, p, godo.HostedAgentHITLOutcomeApprove)
		assert.NoError(t, err)
		assert.Equal(t, 2, p.len(), "h2 must be gone from the middle of the queue")
		assert.Equal(t, "h1", p.get(), "head unchanged")
	})

	t.Run("server failure preserves the queue", func(t *testing.T) {
		withTestClient(t, func(_ *CmdConfig, tm *tcMocks) {
			p := &pendingHITL{}
			p.set("h1")

			tm.hostedAgents.EXPECT().
				ResolveHITL("sess_x", "h1", &godo.HostedAgentResolveHITLRequest{
					Outcome: godo.HostedAgentHITLOutcomeApprove,
					Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
				}).Return(errors.New("boom"))

			err := resolveFromAttach(tm.hostedAgents, "sess_x",
				[]string{"/a", "h1"}, p, godo.HostedAgentHITLOutcomeApprove)
			assert.Error(t, err)
			assert.Equal(t, 1, p.len(), "queue survives a failed resolve so the user can retry")
		})
	})
}

// TestHITLActionLabel: action label extraction tolerates the loose payload
// shape (any of action/kind/tool/name; otherwise empty).
func TestHITLActionLabel(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]any
		want    string
	}{
		{"action key", map[string]any{"action": "HITL_ACTION_BASH"}, "HITL_ACTION_BASH"},
		{"kind key", map[string]any{"kind": "tool_call"}, "tool_call"},
		{"tool key", map[string]any{"tool": "bash"}, "bash"},
		{"name key", map[string]any{"name": "git_push"}, "git_push"},
		{"prefers action over fallbacks", map[string]any{"action": "primary", "kind": "secondary"}, "primary"},
		{"non-string value is ignored", map[string]any{"action": 42}, ""},
		{"empty string is ignored", map[string]any{"action": ""}, ""},
		{"nil map", nil, ""},
		{"unknown keys", map[string]any{"foo": "bar"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, hitlActionLabel(tc.payload))
		})
	}
}

// TestReadHITLKeystroke pins the non-TTY fallback contract: no raw mode,
// no bytes consumed. The raw-mode path needs a real PTY and is verified live.
func TestReadHITLKeystroke(t *testing.T) {
	t.Run("non-file reader falls back", func(t *testing.T) {
		outcome, key, action := readHITLKeystroke(strings.NewReader("y"))
		assert.Equal(t, godo.HostedAgentHITLOutcome(""), outcome)
		assert.Equal(t, byte(0), key)
		assert.Equal(t, hitlKeyFallback, action)
	})

	t.Run("pipe (file but not a TTY) falls back without consuming input", func(t *testing.T) {
		r, w, err := os.Pipe()
		assert.NoError(t, err)
		t.Cleanup(func() { r.Close(); w.Close() })

		_, err = w.Write([]byte("y"))
		assert.NoError(t, err)

		outcome, key, action := readHITLKeystroke(r)
		assert.Equal(t, godo.HostedAgentHITLOutcome(""), outcome)
		assert.Equal(t, byte(0), key)
		assert.Equal(t, hitlKeyFallback, action)

		var buf [1]byte
		n, err := r.Read(buf[:])
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, byte('y'), buf[0], "byte must not be consumed when falling back")
	})
}

// TestPromptDisplay pins the prompt-aware writer: streaming tokens (no \n)
// must not get wiped, newline-terminated events drop to a fresh line, and
// the spinner draws above the prompt without disturbing it.
func TestPromptDisplay(t *testing.T) {
	t.Run("non-raw is pass-through", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		_, err := s.display.Write([]byte("hello\nworld\n"))
		assert.NoError(t, err)
		assert.Equal(t, "hello\nworld\n", buf.String())
	})

	t.Run("raw + newline-terminated event clears, CRLFs, redraws prompt+lineBuf", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.mu.Lock()
		s.lineBuf = []byte("partial input")
		s.cursor = len(s.lineBuf)
		s.mu.Unlock()
		s.display.setRaw(true)

		_, err := s.display.Write([]byte("event\n"))
		assert.NoError(t, err)
		assert.Equal(t, "\r\x1b[Kevent\r\n> partial input", buf.String())
	})

	t.Run("raw + redraw places caret mid-line when cursor is not at EOL", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.mu.Lock()
		s.lineBuf = []byte("abcdef")
		s.cursor = 3 // caret before 'd'
		s.mu.Unlock()
		s.display.setRaw(true)

		s.display.redraw()
		assert.Equal(t, "\r\x1b[K> abcdef\x1b[3D", buf.String())
	})

	t.Run("raw + streaming tokens preserve previous content (no clear, no redraw)", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.Write([]byte("I'll"))
		s.display.Write([]byte(" create"))
		s.display.Write([]byte(" hello.txt"))

		// First token clears the prompt line, later tokens append.
		assert.Equal(t, "\r\x1b[KI'll create hello.txt", buf.String())
	})

	t.Run("raw + event after stream lands on a fresh line below the stream", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.Write([]byte("I'll create"))
		s.display.Write([]byte("(thinking...)\n"))
		assert.Contains(t, buf.String(), "I'll create\r\n(thinking...)\r\n> ",
			"event must land on a fresh line, not glue onto the streaming text")
	})

	t.Run("raw mode redraw flips prompt when HITL becomes pending", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.redraw()
		assert.Equal(t, "\r\x1b[K> ", buf.String())

		buf.Reset()
		pending.set("hitl_99")
		s.display.redraw()
		// In raw mode the prompt flips to the arrow-navigable approve/reject/defer
		// menu (with per-option shortcut keys), not the plain "> " prompt.
		assert.Contains(t, buf.String(), "\r\x1b[K")
		assert.Contains(t, buf.String(), "[Approve (y)]")
		assert.Contains(t, buf.String(), "Reject (n)")
		assert.Contains(t, buf.String(), "Defer (d)")
	})

	t.Run("echo is silent mid-stream so keystrokes don't land on the agent's line", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.Write([]byte("streaming"))
		buf.Reset()
		s.display.echo([]byte("h"))
		assert.Equal(t, "", buf.String(), "echo must not paint onto the streaming line")
	})

	t.Run("spinnerFrame draws the line above the prompt via DECSC/DECRC", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.spinnerFrame("⠋", "thinking...")
		assert.Equal(t, "\x1b7\x1b[A\r\x1b[K⠋ thinking...\x1b8", buf.String())
	})

	t.Run("spinnerFrame is a no-op mid-stream", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.Write([]byte("tokens streaming"))
		buf.Reset()
		s.display.spinnerFrame("⠋", "thinking...")
		assert.Equal(t, "", buf.String(), "spinner must not animate while tokens stream")
	})

	t.Run("spinnerInit reserves a line and redraws the prompt below it", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.spinnerInit("⠋", "thinking...")
		// Spinner on its own line, then the prompt one row below it.
		assert.Equal(t, "\r\x1b[K⠋ thinking...\r\n> ", buf.String())
		assert.False(t, s.display.midLine)
	})
}

func TestEventCursor(t *testing.T) {
	var c eventCursor
	assert.Equal(t, "", c.get())

	c.set("evt_1")
	assert.Equal(t, "evt_1", c.get())

	c.set("evt_2")
	assert.Equal(t, "evt_2", c.get())

	// Empty must not reset — protects the cursor against events missing EventID.
	c.set("")
	assert.Equal(t, "evt_2", c.get())
}

// TestRunAgentsAttachAuthFailure: a 401 from pre-attach GetSession surfaces
// the friendly "Authentication failed" message, not the raw HTTP error.
func TestRunAgentsAttachAuthFailure(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		authErr := &godo.ErrorResponse{
			Response: &http.Response{
				StatusCode: http.StatusUnauthorized,
				Request:    httptest.NewRequest(http.MethodGet, "http://harness/v2/agents/sessions/sess_x", nil),
			},
			Message: "Unable to authenticate you",
		}
		tm.hostedAgents.EXPECT().GetSession("sess_x").Return(nil, authErr)

		config.Args = []string{"sess_x"}
		err := RunAgentsAttach(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Authentication failed")
		assert.Contains(t, err.Error(), "doctl auth init")
	})
}

// TestRunAgentsAttachTerminalSession: attach must fail fast (no banner, no
// interactive loop) when the session is already destroyed/destroying/failed,
// instead of connecting and only failing once the user sends input.
func TestRunAgentsAttachTerminalSession(t *testing.T) {
	cases := []struct {
		name   string
		status godo.HostedAgentSessionStatus
	}{
		{"destroyed", godo.HostedAgentSessionStatusDestroyed},
		{"destroying", godo.HostedAgentSessionStatusDestroying},
		{"failed", godo.HostedAgentSessionStatusFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.hostedAgents.EXPECT().GetSession("sess_x").Return(&do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_x",
						Status:    tc.status,
					},
				}, nil)

				config.Args = []string{"sess_x"}
				err := RunAgentsAttach(config)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot be attached")
				assert.Contains(t, err.Error(), humanSessionStatus(tc.status))
			})
		})
	}
}

// TestHumanSessionStatus pins the SESSION_STATUS_ -> lowercase mapping used in
// attach's terminal-session error message.
func TestHumanSessionStatus(t *testing.T) {
	assert.Equal(t, "destroyed", humanSessionStatus(godo.HostedAgentSessionStatusDestroyed))
	assert.Equal(t, "failed", humanSessionStatus(godo.HostedAgentSessionStatusFailed))
	assert.Equal(t, "ready", humanSessionStatus(godo.HostedAgentSessionStatusReady))
}

// TestClassifyStreamError pins the apierr contract: 401/403/404/409 are
// terminal (409 == V0 single-connection rejection); 5xx/other are transient.
func TestClassifyStreamError(t *testing.T) {
	mkErr := func(status int, message string) error {
		return &godo.ErrorResponse{
			Response: &http.Response{
				StatusCode: status,
				Request:    httptest.NewRequest(http.MethodGet, "http://harness/v2/agents/sessions/sess_x/stream", nil),
			},
			Message: message,
		}
	}

	cases := []struct {
		name        string
		err         error
		wantTermini bool
		wantSubstr  string
	}{
		{
			name:        "401 unauthorized is terminal",
			err:         mkErr(http.StatusUnauthorized, "token expired"),
			wantTermini: true,
			wantSubstr:  "Authentication failed",
		},
		{
			name:        "403 forbidden is terminal",
			err:         mkErr(http.StatusForbidden, "session does not belong to your team"),
			wantTermini: true,
			wantSubstr:  "Access denied",
		},
		{
			name:        "404 not found is terminal",
			err:         mkErr(http.StatusNotFound, "session not found"),
			wantTermini: true,
			wantSubstr:  "Session not found",
		},
		{
			name:        "409 conflict is the V0 single-connection rejection",
			err:         mkErr(http.StatusConflict, "already attached on device abc-123 since 2026-06-24T10:00:00Z"),
			wantTermini: true,
			wantSubstr:  "Session already attached on another device",
		},
		{
			name:        "500 is transient",
			err:         mkErr(http.StatusInternalServerError, "internal"),
			wantTermini: false,
		},
		{
			name:        "503 is transient",
			err:         mkErr(http.StatusServiceUnavailable, "unavailable"),
			wantTermini: false,
		},
		{
			name:        "non-godo error is transient",
			err:         errors.New("network: connection reset"),
			wantTermini: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg, terminal := classifyStreamError(tc.err)
			assert.Equal(t, tc.wantTermini, terminal)
			if tc.wantSubstr != "" {
				assert.Contains(t, msg, tc.wantSubstr)
			}
		})
	}
}

// TestIsRunTerminalErr pins the SendInput 409 case where the session's run is
// already terminal — only that specific error should trip the friendly
// detach-and-restart path, not other 409s or unrelated errors.
func TestIsRunTerminalErr(t *testing.T) {
	mkErr := func(status int, message string) error {
		return &godo.ErrorResponse{
			Response: &http.Response{StatusCode: status},
			Message:  message,
		}
	}

	mkNestedErr := func(status int, message string) error {
		return &godo.ErrorResponse{
			Response: &http.Response{StatusCode: status},
			NestedError: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: status, Message: message},
		}
	}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "409 run is terminal (top-level message) matches",
			err:  mkErr(http.StatusConflict, "invalid transition for run 019f...: run is terminal; create a new one"),
			want: true,
		},
		{
			name: "409 run is terminal (nested harness-api envelope) matches",
			err:  mkNestedErr(http.StatusConflict, "invalid transition for run 019f...: run is terminal; create a new one"),
			want: true,
		},
		{
			name: "409 single-connection rejection does not match",
			err:  mkErr(http.StatusConflict, "already attached on device abc-123"),
			want: false,
		},
		{
			name: "500 with terminal text does not match",
			err:  mkErr(http.StatusInternalServerError, "run is terminal"),
			want: false,
		},
		{
			name: "non-godo error does not match",
			err:  errors.New("network: connection reset"),
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isRunTerminalErr(tc.err))
		})
	}
}

// TestSessionLimitErr pins the `agents start` 409 returned when the team is at
// its active-session cap — only that specific 409 should trip the friendly
// "destroy a session" hint, not other 409s or unrelated errors.
func TestSessionLimitErr(t *testing.T) {
	mkErr := func(status int, message string) error {
		return &godo.ErrorResponse{
			Response: &http.Response{StatusCode: status},
			Message:  message,
		}
	}
	mkNestedErr := func(status int, message string) error {
		return &godo.ErrorResponse{
			Response: &http.Response{StatusCode: status},
			NestedError: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: status, Message: message},
		}
	}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "409 session limit (top-level) matches",
			err:  mkErr(http.StatusConflict, "team is at the limit of 4 active sessions"),
			want: true,
		},
		{
			name: "409 session limit (nested envelope) matches",
			err:  mkNestedErr(http.StatusConflict, "team is at the limit of 4 active sessions"),
			want: true,
		},
		{
			name: "409 run is terminal does not match",
			err:  mkErr(http.StatusConflict, "run is terminal; create a new one"),
			want: false,
		},
		{
			name: "non-godo error does not match",
			err:  errors.New("network: connection reset"),
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, sessionLimitErr(tc.err))
		})
	}
}

// TestTokenDeduper pins the reasoning-double-print fix: a consolidated delta
// that repeats the whole accumulated segment is suppressed, while distinct
// text, short repeats, and post-reset segments still render.
func TestTokenDeduper(t *testing.T) {
	t.Run("consolidated repeat of a streamed segment is suppressed", func(t *testing.T) {
		d := &tokenDeduper{}
		// Reasoning streamed as pieces.
		assert.True(t, d.allow("The user wants me to "))
		assert.True(t, d.allow("run six commands."))
		// Same block re-sent as one consolidated delta.
		assert.False(t, d.allow("The user wants me to run six commands."))
		// The real answer still renders.
		assert.True(t, d.allow("Sure, here goes."))
	})

	t.Run("single-delta block repeated once is suppressed", func(t *testing.T) {
		d := &tokenDeduper{}
		block := "The user said hello - a simple greeting."
		assert.True(t, d.allow(block))
		assert.False(t, d.allow(block))
	})

	t.Run("short repeats are not suppressed", func(t *testing.T) {
		d := &tokenDeduper{}
		assert.True(t, d.allow("ok\n"))
		assert.True(t, d.allow("ok\n"))
	})

	t.Run("empty text is always allowed and inert", func(t *testing.T) {
		d := &tokenDeduper{}
		assert.True(t, d.allow("a long enough block of text"))
		assert.True(t, d.allow(""))
		// The empty delta must not have altered the segment, so the real
		// repeat is still caught.
		assert.False(t, d.allow("a long enough block of text"))
	})

	t.Run("reset starts a fresh segment", func(t *testing.T) {
		d := &tokenDeduper{}
		block := "a long enough block of text"
		assert.True(t, d.allow(block))
		d.reset()
		// After a structural boundary the same text is legitimate again.
		assert.True(t, d.allow(block))
	})
}

func TestNextBackoff(t *testing.T) {
	cur := initialReconnectBackoff
	for i := 0; i < 10; i++ {
		cur = nextBackoff(cur)
		if cur > maxReconnectBackoff {
			t.Fatalf("backoff exceeded cap: %s > %s", cur, maxReconnectBackoff)
		}
	}
	assert.Equal(t, maxReconnectBackoff, cur)
}

// stubReconnectSleep replaces reconnect backoff with a no-op so streamWithReconnect
// tests finish instantly.
func stubReconnectSleep(t *testing.T) {
	t.Helper()
	old := reconnectSleepFn
	reconnectSleepFn = func(ctx context.Context, _ time.Duration) bool {
		return ctx.Err() == nil
	}
	t.Cleanup(func() { reconnectSleepFn = old })
}

// sseFrame builds one SPI-style SSE data frame for hosted-agent stream tests.
func sseFrame(eventID, eventType, dataJSON string) string {
	return fmt.Sprintf(
		"id: %s\ndata: {\"event_id\":\"%s\",\"type\":\"%s\",\"data\":%s}\n\n",
		eventID, eventID, eventType, dataJSON,
	)
}

// hostedAgentSSEHandler writes one SSE body and optionally fails mid-stream.
func hostedAgentSSEHandler(sse string, trailErr error) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if trailErr == nil {
			_, _ = io.WriteString(w, sse)
			return
		}
		_, _ = io.WriteString(w, sse)
		// Trailing corrupt frame forces a non-EOF stream error after the first event.
		_, _ = io.WriteString(w, "data: {not-json\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func openHostedAgentStream(t *testing.T, client *godo.Client, opt *godo.HostedAgentSessionStreamOptions) *godo.HostedAgentSessionStream {
	t.Helper()
	stream, _, err := client.HostedAgents.StreamSession(context.Background(), "sess_x", opt)
	assert.NoError(t, err)
	return stream
}

// terminalStreamErr is the 404 a gone session returns; it stops the reconnect
// loop. Tests append it as the final StreamSession result so a stream that ends
// on a clean EOF (which now reconnects) has a deterministic terminal stop.
func terminalStreamErr() *godo.ErrorResponse {
	return &godo.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
			Request:    httptest.NewRequest(http.MethodGet, "http://harness/v2/agents/sessions/sess_x/stream", nil),
		},
		Message: "session not found",
	}
}

// TestStreamWithReconnect_cleanEOFReconnectsUntilTerminal pins the idle-timeout
// fix: a clean EOF (how a server idle-timeout close looks, err == nil) is an
// unexpected drop for an interactive attach, so it must reconnect rather than
// silently exit. The attach stops only once the session is gone, which surfaces
// as a terminal error (404) on the next connect.
func TestStreamWithReconnect_cleanEOFReconnectsUntilTerminal(t *testing.T) {
	stubReconnectSleep(t)

	evt := sseFrame("evt-1", string(godo.HostedAgentEventKindSessionUpdated), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(evt, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	ctrl := gomock.NewController(t)
	mock := domocks.NewMockHostedAgentsService(ctrl)
	gomock.InOrder(
		mock.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				return openHostedAgentStream(t, client, opt), nil
			}),
		mock.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			Return(nil, terminalStreamErr()),
	)

	var buf bytes.Buffer
	streamWithReconnect(context.Background(), mock, "sess_x", &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil)

	out := buf.String()
	assert.Contains(t, out, "session updated")
	assert.Contains(t, out, msgReconnecting, "a clean EOF must trigger a visible reconnect")
	assert.NotContains(t, out, msgReconnectFailed)
}

// TestStreamWithReconnect_supersededStopsWithoutReconnect pins the one stream end
// that must NOT reconnect. The data-plane stream reports a same-device takeover
// as stream.state: superseded and then closes; reconnecting would supersede the
// connection that just superseded us, and the two windows would evict each other
// forever. Only one StreamSession call is expected, so a reconnect fails the test.
func TestStreamWithReconnect_supersededStopsWithoutReconnect(t *testing.T) {
	stubReconnectSleep(t)

	body := sseFrame("evt-1", string(godo.HostedAgentEventKindSessionUpdated), `{}`) +
		sseFrame("", string(godo.HostedAgentEventKindStreamState), `{"state":"superseded","cursor":""}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	ctrl := gomock.NewController(t)
	mock := domocks.NewMockHostedAgentsService(ctrl)
	mock.EXPECT().
		StreamSession(gomock.Any(), "sess_x", gomock.Any()).
		DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
			return openHostedAgentStream(t, client, opt), nil
		}).
		Times(1)

	var buf bytes.Buffer
	streamWithReconnect(context.Background(), mock, "sess_x", &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil)

	out := buf.String()
	assert.Contains(t, out, "session updated", "events before the takeover still render")
	assert.Contains(t, out, msgSuperseded)
	assert.NotContains(t, out, msgReconnecting)
	assert.NotContains(t, out, msgReconnectFailed)
}

// TestDrainStream_HITLReattachShowsCommand reproduces MARSOHS-648: on reattach
// the server re-injects only run.human_input_requested (no tool_call_started).
// The approval line must still show details.command, not "action pending".
func TestDrainStream_HITLReattachShowsCommand(t *testing.T) {
	const hitlID = "019f1dfc-f017-70e2-9eac-2ea470a55ac2"
	const cmd = "mkdir /tmp/hitl-test && echo done"
	body := sseFrame("evt-hitl", string(godo.HostedAgentEventKindHITLRequested),
		fmt.Sprintf(`{"hitl_id":%q,"action":"HITL_ACTION_BASH","details":{"command":%q}}`, hitlID, cmd))
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	pending := &pendingHITL{}
	superseded := drainStream(stream, &buf, pending, &eventCursor{}, newThinkingState(&buf), nil, &tokenDeduper{})

	assert.False(t, superseded)
	out := buf.String()
	assert.Contains(t, out, "Approval required")
	assert.Contains(t, out, cmd)
	assert.Contains(t, out, hitlID)
	assert.NotContains(t, out, "action pending")
	assert.Equal(t, 1, pending.len())
	assert.Equal(t, hitlID, pending.get())
}

// TestDrainStream_skipsStreamStateControlFrames pins that a live stream.state
// frame is transport bookkeeping: it renders nothing and must not become the
// reconnect cursor, or a reconnect would resume from a position no event holds.
func TestDrainStream_skipsStreamStateControlFrames(t *testing.T) {
	body := sseFrame("", string(godo.HostedAgentEventKindStreamState), `{"state":"live","cursor":""}`) +
		sseFrame("evt-7", string(godo.HostedAgentEventKindSessionUpdated), `{}`) +
		sseFrame("", string(godo.HostedAgentEventKindStreamState), `{"state":"catching_up","cursor":""}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	cursor := &eventCursor{}
	superseded := drainStream(stream, &buf, &pendingHITL{}, cursor, newThinkingState(&buf), nil, &tokenDeduper{})

	assert.False(t, superseded)
	assert.Equal(t, 1, strings.Count(buf.String(), "session updated"))
	assert.NotContains(t, buf.String(), "stream.state")
	assert.Equal(t, "evt-7", cursor.get(), "the cursor must track the last real event, not a control frame")
}

func TestStreamWithReconnect_successOnSecondAttempt(t *testing.T) {
	stubReconnectSleep(t)

	evt := sseFrame("evt-2", string(godo.HostedAgentEventKindSessionUpdated), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(evt, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	ctrl := gomock.NewController(t)
	mock := domocks.NewMockHostedAgentsService(ctrl)
	gomock.InOrder(
		mock.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			Return(nil, errors.New("connection reset by peer")),
		mock.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				stream := openHostedAgentStream(t, client, opt)
				return stream, nil
			}),
		// The successful connect ends on a clean EOF, which now reconnects;
		// a terminal error gives the loop a deterministic stop.
		mock.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			Return(nil, terminalStreamErr()),
	)

	var buf bytes.Buffer
	streamWithReconnect(context.Background(), mock, "sess_x", &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil)

	out := buf.String()
	assert.Contains(t, out, msgReconnecting)
	assert.Contains(t, out, "session updated")
	assert.NotContains(t, out, msgReconnectFailed)
}

func TestStreamWithReconnect_exhaustedRetries(t *testing.T) {
	stubReconnectSleep(t)

	ctrl := gomock.NewController(t)
	mock := domocks.NewMockHostedAgentsService(ctrl)
	mock.EXPECT().
		StreamSession(gomock.Any(), "sess_x", gomock.Any()).
		Return(nil, errors.New("connection reset by peer")).
		Times(maxAutoReconnectAttempts)

	var buf bytes.Buffer
	streamWithReconnect(context.Background(), mock, "sess_x", &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil)

	out := buf.String()
	assert.Equal(t, maxAutoReconnectAttempts-1, strings.Count(out, msgReconnecting))
	assert.Contains(t, out, msgReconnectFailed)
}

func TestStreamWithReconnect_terminalErrorNoRetry(t *testing.T) {
	stubReconnectSleep(t)

	authErr := &godo.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusUnauthorized,
			Request:    httptest.NewRequest(http.MethodGet, "http://harness/v2/agents/sessions/sess_x/stream", nil),
		},
		Message: "token expired",
	}

	ctrl := gomock.NewController(t)
	mock := domocks.NewMockHostedAgentsService(ctrl)
	mock.EXPECT().
		StreamSession(gomock.Any(), "sess_x", gomock.Any()).
		Return(nil, authErr).
		Times(1)

	var buf bytes.Buffer
	streamWithReconnect(context.Background(), mock, "sess_x", &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil)

	out := buf.String()
	assert.Contains(t, out, "Authentication failed")
	assert.NotContains(t, out, msgReconnecting)
	assert.NotContains(t, out, msgReconnectFailed)
}

func TestStreamWithReconnect_replayCursorAfterMidStreamDrop(t *testing.T) {
	stubReconnectSleep(t)

	evt1 := sseFrame("evt-1", string(godo.HostedAgentEventKindSessionUpdated), `{}`)
	evt2 := sseFrame("evt-2", string(godo.HostedAgentEventKindSessionUpdated), `{}`)

	var (
		mu    sync.Mutex
		calls int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		n := calls
		// The live stream carries the resume cursor in the standard SSE
		// Last-Event-ID header, not a replay_from query parameter.
		replayFrom := r.Header.Get("Last-Event-ID")
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		switch n {
		case 1:
			assert.Empty(t, replayFrom)
			_, _ = io.WriteString(w, evt1)
			_, _ = io.WriteString(w, "data: {not-json\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case 2:
			assert.Equal(t, "evt-1", replayFrom)
			_, _ = io.WriteString(w, evt2)
		default:
			t.Fatalf("unexpected stream call %d", n)
		}
	}))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	ctrl := gomock.NewController(t)
	mock := domocks.NewMockHostedAgentsService(ctrl)
	gomock.InOrder(
		mock.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				stream := openHostedAgentStream(t, client, opt)
				return stream, nil
			}),
		mock.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				stream := openHostedAgentStream(t, client, opt)
				return stream, nil
			}),
		// evt2 ends on a clean EOF, which now reconnects; stop deterministically.
		mock.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			Return(nil, terminalStreamErr()),
	)

	var buf bytes.Buffer
	streamWithReconnect(context.Background(), mock, "sess_x", &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil)

	out := buf.String()
	assert.Equal(t, 2, strings.Count(out, "session updated"), "both events should render after replay reconnect")
	assert.Contains(t, out, msgReconnecting)
	assert.NotContains(t, out, msgReconnectFailed)
}

// TestStreamWithReconnect_healthyDropsDoNotExhaustBudget pins the fix for the
// long-quiet-attach bug: a stream that stays connected past healthyStreamDuration
// before each idle drop must reset the reconnect budget, so an attach survives
// more than maxAutoReconnectAttempts idle timeouts over its lifetime.
func TestStreamWithReconnect_healthyDropsDoNotExhaustBudget(t *testing.T) {
	stubReconnectSleep(t)

	// Treat every drop as a healthy (long-lived) connection.
	oldDur := healthyStreamDuration
	healthyStreamDuration = 0
	t.Cleanup(func() { healthyStreamDuration = oldDur })

	// More idle drops than the consecutive-failure budget, then a terminal stop.
	idleDrops := maxAutoReconnectAttempts + 3

	var (
		mu    sync.Mutex
		calls int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		frame := sseFrame(fmt.Sprintf("evt-%d", n), string(godo.HostedAgentEventKindSessionUpdated), `{}`)
		_, _ = io.WriteString(w, frame)
		// Force a non-EOF mid-stream drop after delivering an event.
		_, _ = io.WriteString(w, "data: {not-json\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	ctrl := gomock.NewController(t)
	mock := domocks.NewMockHostedAgentsService(ctrl)
	gomock.InOrder(
		mock.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				return openHostedAgentStream(t, client, opt), nil
			}).
			Times(idleDrops),
		mock.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			Return(nil, terminalStreamErr()),
	)

	var buf bytes.Buffer
	streamWithReconnect(context.Background(), mock, "sess_x", &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil)

	out := buf.String()
	assert.NotContains(t, out, msgReconnectFailed, "healthy idle drops must not exhaust the reconnect budget")
	assert.Equal(t, idleDrops, strings.Count(out, "session updated"), "every reconnect should keep rendering events")
	// A healthy drop resets the failure budget to zero, but the user must still
	// see the reconnect notice on each reconnect attempt after the first.
	assert.Equal(t, idleDrops, strings.Count(out, msgReconnecting), "each healthy-drop reconnect must still show the notice")
}

// TestStreamWithReconnect_rapidDropsExhaustBudget pins the complementary case:
// back-to-back drops that never stay connected long enough still give up after
// maxAutoReconnectAttempts.
func TestStreamWithReconnect_rapidDropsExhaustBudget(t *testing.T) {
	stubReconnectSleep(t)

	// No connection is ever considered healthy.
	oldDur := healthyStreamDuration
	healthyStreamDuration = time.Hour
	t.Cleanup(func() { healthyStreamDuration = oldDur })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Immediately corrupt frame => mid-stream drop with no useful event.
		_, _ = io.WriteString(w, "data: {not-json\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	ctrl := gomock.NewController(t)
	mock := domocks.NewMockHostedAgentsService(ctrl)
	mock.EXPECT().
		StreamSession(gomock.Any(), "sess_x", gomock.Any()).
		DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
			return openHostedAgentStream(t, client, opt), nil
		}).
		Times(maxAutoReconnectAttempts)

	var buf bytes.Buffer
	streamWithReconnect(context.Background(), mock, "sess_x", &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil)

	out := buf.String()
	assert.Equal(t, maxAutoReconnectAttempts-1, strings.Count(out, msgReconnecting))
	assert.Contains(t, out, msgReconnectFailed)
}

func TestWarmupState_skipsOldSessions(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	w := newWarmupState(&buf, now.Add(-3*time.Minute))
	w.start()
	assert.Empty(t, buf.String(), "sessions older than warmupEligibleAge must not show a notice")
	assert.False(t, w.eligible)
}

func TestWarmupState_showsForYoungSessions(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	w := newWarmupState(&buf, now.Add(-30*time.Second))
	w.start()
	assert.Contains(t, buf.String(), msgAgentWarmup)
	assert.True(t, w.active)

	w.clear()
	assert.False(t, w.active)
	assert.True(t, w.dismissed)
	assert.NotContains(t, buf.String(), "You can type anytime")
}

func TestWarmupState_clearsOnTimeout(t *testing.T) {
	oldClock := warmupClock
	oldDur := warmupDuration
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	warmupDuration = 20 * time.Millisecond
	t.Cleanup(func() {
		warmupClock = oldClock
		warmupDuration = oldDur
	})

	var buf bytes.Buffer
	w := newWarmupState(&buf, now)
	w.start()
	assert.Contains(t, buf.String(), msgAgentWarmup)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		w.mu.Lock()
		done := !w.active && w.dismissed
		w.mu.Unlock()
		if done {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	w.mu.Lock()
	active, dismissed := w.active, w.dismissed
	w.mu.Unlock()
	assert.False(t, active)
	assert.True(t, dismissed)
	assert.NotContains(t, buf.String(), "You can type anytime")
}

func TestWarmupState_startAfterClearIsNoop(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	w := newWarmupState(&buf, now)
	w.clear()
	w.start()
	assert.Empty(t, buf.String())
}

func TestDrainStream_clearsWarmupOnRunStarted(t *testing.T) {
	evt := sseFrame("e1", string(godo.HostedAgentEventKindRunStarted), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(evt, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)

	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	warmup := newWarmupState(&buf, now)
	warmup.start()
	assert.Contains(t, buf.String(), msgAgentWarmup)

	drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), warmup, &tokenDeduper{})
	assert.True(t, warmup.dismissed)
	assert.False(t, warmup.active)
}

func TestDrainStream_sessionUpdatedDoesNotClearWarmup(t *testing.T) {
	evt := sseFrame("e1", string(godo.HostedAgentEventKindSessionUpdated), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(evt, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)

	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	warmup := newWarmupState(&buf, now)
	warmup.start()

	drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), warmup, &tokenDeduper{})
	assert.True(t, warmup.active, "session.updated must not dismiss the warm-up notice")
	warmup.clear()
}
