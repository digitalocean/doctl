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
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	yaml "gopkg.in/yaml.v2"
)

// stubInteractiveTerminal forces the tty check, which is false under `go test`
// and would otherwise make every launch path unreachable.
func stubInteractiveTerminal(t *testing.T, interactive bool) {
	t.Helper()
	prev := isInteractiveTerminal
	isInteractiveTerminal = func() bool { return interactive }
	t.Cleanup(func() { isInteractiveTerminal = prev })
}

// assertCalledErr is returned from a mock to stop a runner at the exact call
// under test, so a dispatch assertion does not have to stub out the whole
// downstream ready-wait and attach.
var assertCalledErr = errors.New("reached the expected API call")

// --- launch's dual-mode dispatch --------------------------------------------

func TestIsReadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleFlatManifest), 0o644))

	assert.True(t, isReadableFile(path))
	assert.True(t, isReadableFile("-"), `"-" names stdin, which is always readable`)
	assert.False(t, isReadableFile(filepath.Join(dir, "nope.yaml")))
	assert.False(t, isReadableFile(dir), "a directory is not a manifest")
	assert.False(t, isReadableFile("my-session-name"))
}

// A positional argument that is a readable file means "create from this
// manifest", and one that is not means "attach to this session". This is the
// only piece of inference in the new command surface, so it is pinned from both
// sides.
func TestRunAgentsLaunch_ManifestArgCreates(t *testing.T) {
	stubInteractiveTerminal(t, true)

	dir := t.TempDir()
	path := filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleFlatManifest), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// No ListSessions expectation: reading this as a session name would
		// have to look one up, and gomock fails the test if it does.
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			Return(nil, assertCalledErr)

		config.Args = []string{path}
		err := RunAgentsLaunch(config)
		require.ErrorIs(t, err, assertCalledErr)
	})
}

func TestRunAgentsLaunch_SessionRefAttaches(t *testing.T) {
	stubInteractiveTerminal(t, true)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// No CreateSessionFromManifest expectation: treating a session name as
		// a manifest path must not reach the create call.
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
			Return(nil, assertCalledErr)

		config.Args = []string{"my-session"}
		err := RunAgentsLaunch(config)
		require.Error(t, err)
	})
}

// A ref matching neither reading is the one case where the user needs both
// possibilities spelled out, since they cannot tell which way doctl read it.
func TestRunAgentsLaunch_UnresolvableRefExplainsBothReadings(t *testing.T) {
	stubInteractiveTerminal(t, true)
	t.Chdir(t.TempDir())

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListSessions(&godo.HostedAgentSessionListOptions{Name: "pr-reveiwer-test"}).
			Return(nil, "", nil)

		config.Args = []string{"pr-reveiwer-test"}
		err := RunAgentsLaunch(config)
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, "isn't an existing session")
		assert.Contains(t, msg, "no readable file exists at that path")
		assert.Contains(t, msg, "launch pr-reveiwer-test")
		assert.Contains(t, msg, "create --harness")
	})
}

// An ambiguous name is a different failure from a missing one and must not be
// reworded into "maybe it's a file".
func TestRunAgentsLaunch_AmbiguousNameSurfacesAsIs(t *testing.T) {
	stubInteractiveTerminal(t, true)
	t.Chdir(t.TempDir())

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListSessions(&godo.HostedAgentSessionListOptions{Name: "dupe"}).
			Return([]do.HostedAgentSession{
				{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_a", Name: "dupe", Status: godo.HostedAgentSessionStatusReady}},
				{HostedAgentSession: &godo.HostedAgentSession{SessionID: "sess_b", Name: "dupe", Status: godo.HostedAgentSessionStatusReady}},
			}, "", nil)

		config.Args = []string{"dupe"}
		err := RunAgentsLaunch(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "many agent sessions go by the name")
		assert.NotContains(t, err.Error(), "no readable file")
	})
}

// Creation flags settle the mode before the filesystem is consulted, so a
// session that happens to share a name with a local file is not misread.
func TestRunAgentsLaunch_CreationFlagWinsOverFilesystem(t *testing.T) {
	stubInteractiveTerminal(t, true)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
				Name:     "demo",
				ConfigID: "cfg_abc123",
			}).
			Return(nil, assertCalledErr)

		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		err := RunAgentsLaunch(config)
		require.ErrorIs(t, err, assertCalledErr)
	})
}

// --- launch's resume-if-paused ----------------------------------------------

func TestRunAgentsLaunch_ResumesPausedSession(t *testing.T) {
	stubInteractiveTerminal(t, true)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		gomock.InOrder(
			tm.hostedAgents.EXPECT().
				GetSession("sess_paused").
				Return(&do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_paused",
						Name:      "napping",
						Status:    godo.HostedAgentSessionStatusPaused,
					},
				}, nil),
			tm.hostedAgents.EXPECT().ResumeSession("sess_paused").Return(nil),
			// Re-read after the resume: attaching with the stale paused
			// session would banner and stream as though it never happened.
			tm.hostedAgents.EXPECT().
				GetSession("sess_paused").
				Return(nil, assertCalledErr),
		)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_paused"}
		err := RunAgentsLaunch(config)
		require.Error(t, err)
		assert.Contains(t, buf.String(), "is paused — resuming")
	})
}

// Only paused triggers the auto-resume; a gone session is an error, not
// something to wake up.
func TestRunAgentsLaunch_DoesNotResumeTerminalSession(t *testing.T) {
	stubInteractiveTerminal(t, true)

	for _, status := range []godo.HostedAgentSessionStatus{
		godo.HostedAgentSessionStatusDestroyed,
		godo.HostedAgentSessionStatusDestroying,
		godo.HostedAgentSessionStatusFailed,
	} {
		t.Run(humanSessionStatus(status), func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				// No ResumeSession expectation: gomock fails if it is called.
				tm.hostedAgents.EXPECT().
					GetSession("sess_gone").
					Return(&do.HostedAgentSession{
						HostedAgentSession: &godo.HostedAgentSession{
							SessionID: "sess_gone",
							Status:    status,
						},
					}, nil)

				config.Args = []string{"sess_gone"}
				err := RunAgentsLaunch(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "cannot be attached")
			})
		})
	}
}

// A session that is already awake attaches with the single GetSession the old
// standalone `attach` used, not two.
func TestRunAgentsLaunch_ReadySessionFetchedOnce(t *testing.T) {
	stubInteractiveTerminal(t, true)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			GetSession("sess_ready").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_ready",
					Status:    godo.HostedAgentSessionStatusDestroyed,
				},
			}, nil).
			Times(1)

		config.Args = []string{"sess_ready"}
		require.Error(t, RunAgentsLaunch(config))
	})
}

// --- launch rejects creation-only flags on an existing session --------------

func TestRunAgentsLaunch_RejectsCreationFlagsWithExistingRef(t *testing.T) {
	stubInteractiveTerminal(t, true)

	cases := map[string]any{
		doctl.ArgAgentSecret:        []string{"TOKEN=abc"},
		doctl.ArgAgentName:          "renamed",
		doctl.ArgAgentRepo:          "org/repo",
		doctl.ArgAgentTriggerPrompt: "do the thing",
		doctl.ArgAgentWaitTimeout:   60,
	}

	for flag, value := range cases {
		t.Run(flag, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Args = []string{"my-session"}
				config.Doit.Set(config.NS, flag, value)
				err := RunAgentsLaunch(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "--"+flag)
				assert.Contains(t, err.Error(), "create")
			})
		})
	}
}

// --- --dry-run --------------------------------------------------------------

func TestRunAgentsCreate_DryRunPrintsManifestWithoutCreating(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// No API expectations at all: --dry-run must be free of side effects.
		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "printed-only")
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)

		require.NoError(t, RunAgentsCreate(config))

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(buf.Bytes(), &doc), "output must be valid YAML so it can be piped")
		assert.Equal(t, "printed-only", doc["name"])
		assert.True(t, strings.HasSuffix(buf.String(), "\n"))
	})
}

func TestRunAgentsCreate_DryRunRedactsSecrets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: demo\nagent: opencode\n"), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{path}
		config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"ANTHROPIC_API_KEY=sk-ant-super-secret"})
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)

		require.NoError(t, RunAgentsCreate(config))
		out := buf.String()
		assert.NotContains(t, out, "sk-ant-super-secret", "a printed manifest must never carry a plaintext secret")
		assert.Contains(t, out, "ANTHROPIC_API_KEY", "the slot itself stays visible so the output is still usable")
		assert.Contains(t, out, redactedSecretValue)
	})
}

// ${VAR} is expanded on the real create path, so it must be expanded here too
// or --dry-run would print something the server never sees.
func TestRunAgentsCreate_DryRunExpandsEnv(t *testing.T) {
	t.Setenv("DOCTL_TEST_MODEL", "sonnet")
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(path,
		[]byte("name: demo\nagent: opencode\nenv:\n  MODEL: ${DOCTL_TEST_MODEL}\n"), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{path}
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)

		require.NoError(t, RunAgentsCreate(config))
		assert.Contains(t, buf.String(), "sonnet")
		assert.NotContains(t, buf.String(), "${DOCTL_TEST_MODEL}")
	})
}

// The reason --dry-run exists: a working --harness invocation can be promoted
// into a durable Agent Config without anyone hand-writing YAML, and without a
// manifest ever touching disk. Redaction is what makes the pipe safe, so the
// real value has to arrive separately via --secret on the receiving end.
func TestDryRunOutputPipesIntoConfigCreate(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(
		"agent: claude-code\nrepos:\n  - adamdev/orders-api\nsecrets:\n  - name: ANTHROPIC_API_KEY\n    source: tenantSecret\n    value: \"\"\n"), 0o644))

	var manifest bytes.Buffer
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = &manifest
		config.Args = []string{specPath}
		config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"ANTHROPIC_API_KEY=sk-ant-local"})
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)

		require.NoError(t, RunAgentsCreate(config))
	})
	require.Contains(t, manifest.String(), redactedSecretValue)
	require.NotContains(t, manifest.String(), "sk-ant-local")

	stdin, w, err := os.Pipe()
	require.NoError(t, err)
	prevStdin := os.Stdin
	os.Stdin = stdin
	t.Cleanup(func() { os.Stdin = prevStdin; stdin.Close() })
	go func() {
		defer w.Close()
		_, _ = w.Write(manifest.Bytes())
	}()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateAgentConfig(gomock.Any()).
			DoAndReturn(func(req *godo.HostedAgentConfigCreateRequest) (*godo.HostedAgentConfig, error) {
				assert.Equal(t, "kimi-k3-reviewer", req.Name)
				assert.Contains(t, req.ManifestYAML, "adamdev/orders-api")
				assert.Contains(t, req.ManifestYAML, "sk-ant-real",
					"--secret on this side has to fill the slot the dry run redacted")
				assert.NotContains(t, req.ManifestYAML, redactedSecretValue)
				return &godo.HostedAgentConfig{ID: "cfg_piped", Name: "kimi-k3-reviewer"}, nil
			})

		config.Out = io.Discard
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, "-")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "kimi-k3-reviewer")
		config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"ANTHROPIC_API_KEY=sk-ant-real"})

		require.NoError(t, RunAgentsConfigCreate(config))
	})
}

// The redacted placeholder is what makes the pipe safe, but it is also a
// plausible-looking string, so forgetting --secret on the receiving end must not
// quietly store "REDACTED" as the credential and fail later inside a sandbox.
func TestRedactedSecretIsRejectedInsteadOfStored(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(
		"agent: claude-code\nsecrets:\n  - name: ANTHROPIC_API_KEY\n    source: tenantSecret\n    value: "+redactedSecretValue+"\n"), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// No CreateAgentConfig expectation: it must never be reached.
		config.Out = io.Discard
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentName, "reviewer")

		err := RunAgentsConfigCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY")
		assert.Contains(t, err.Error(), doctl.ArgAgentSecret)
	})

	// --dry-run stores nothing, so it must still be able to re-print a manifest
	// it had already redacted rather than refusing to show it.
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{specPath}
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)

		require.NoError(t, RunAgentsCreate(config))
		assert.Contains(t, buf.String(), redactedSecretValue)
	})
}

// --harness claude-code asks for its key as an env reference rather than a
// secret slot, and --secret fills slots. So the key must be present locally to
// create for real; --dry-run instead leaves the reference standing and says so,
// which is what lets the manifest be generated on a machine that has no key.
func TestRunAgentsCreate_DryRunLeavesUnsetEnvAsPlaceholder(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "claude-code")
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)

		require.NoError(t, RunAgentsCreate(config))
		assert.Contains(t, buf.String(), "${ANTHROPIC_API_KEY}",
			"an unset reference stays a placeholder so the template can still be produced")
	})
}

func TestRunAgentsCreate_DryRunRejectsFromConfig(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)

		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no manifest to print")
	})
}

func TestRunAgentsCreate_DryRunAndOnHITLConflict(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)
		config.Doit.Set(config.NS, doctl.ArgAgentOnHITL, "approve")

		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "creates nothing")
	})
}

// --- --secret ---------------------------------------------------------------

func TestRunAgentsCreate_SecretReachesTheManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: demo\nagent: opencode\n"), 0o644))

	keyFile := filepath.Join(dir, "key")
	require.NoError(t, os.WriteFile(keyFile, []byte("from-a-file\n"), 0o600))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			DoAndReturn(func(manifest []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				assert.Contains(t, string(manifest), "from-a-file",
					"the trailing newline should be trimmed off a @file value")
				return nil, assertCalledErr
			})

		config.Args = []string{path}
		config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"ANTHROPIC_API_KEY=@" + keyFile})
		require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
	})
}

func TestRunAgentsCreate_SecretRejectedWithFromConfig(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentSecret, []string{"TOKEN=abc"})

		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already holds its secret values")
	})
}

func TestInjectManifestSecrets(t *testing.T) {
	slots := func(t *testing.T, manifest []byte, path ...string) []any {
		t.Helper()
		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(manifest, &doc))
		container := doc
		for _, key := range path {
			nested, ok := yamlMap(container[key])
			require.True(t, ok, "expected a mapping at %q", key)
			container = nested
		}
		list, ok := yamlList(container["secrets"])
		require.True(t, ok, "expected a list of secret slots")
		return list
	}

	t.Run("no secrets leaves the manifest byte-for-byte", func(t *testing.T) {
		in := []byte("name: demo\nagent: opencode\n")
		out, err := injectManifestSecrets(in, nil)
		require.NoError(t, err)
		assert.Equal(t, string(in), string(out))
	})

	t.Run("appends a tenantSecret slot to a flat manifest", func(t *testing.T) {
		out, err := injectManifestSecrets([]byte("name: demo\nagent: opencode\n"),
			map[string]string{"TOKEN": "abc"})
		require.NoError(t, err)

		list := slots(t, out)
		require.Len(t, list, 1)
		slot, _ := yamlMap(list[0])
		assert.Equal(t, "TOKEN", slot["name"])
		assert.Equal(t, tenantSecretSource, slot["source"])
		assert.Equal(t, "abc", slot["value"])
	})

	t.Run("nests under spec on a legacy envelope", func(t *testing.T) {
		out, err := injectManifestSecrets([]byte(sampleManifest), map[string]string{"TOKEN": "abc"})
		require.NoError(t, err)

		list := slots(t, out, "spec")
		require.Len(t, list, 1)
		slot, _ := yamlMap(list[0])
		assert.Equal(t, "TOKEN", slot["name"])
		assert.Equal(t, "abc", slot["value"])

		// The envelope must survive: rewriting it as flat would change how the
		// server parses the whole manifest.
		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(out, &doc))
		assert.Contains(t, doc, "apiVersion")
	})

	t.Run("overrides a value declared in the file", func(t *testing.T) {
		in := []byte(`name: demo
agent: opencode
secrets:
  - name: TOKEN
    source: tenantSecret
    value: from-the-file
`)
		out, err := injectManifestSecrets(in, map[string]string{"TOKEN": "from-the-flag"})
		require.NoError(t, err)

		list := slots(t, out)
		require.Len(t, list, 1, "overriding must not duplicate the slot")
		slot, _ := yamlMap(list[0])
		assert.Equal(t, "from-the-flag", slot["value"])
	})

	t.Run("preserves the map shorthand a manifest already uses", func(t *testing.T) {
		in := []byte("name: demo\nagent: opencode\nsecrets:\n  EXISTING: keep-me\n")
		out, err := injectManifestSecrets(in, map[string]string{"TOKEN": "abc"})
		require.NoError(t, err)

		var doc map[string]any
		require.NoError(t, yaml.Unmarshal(out, &doc))
		got, ok := yamlStringMap(doc["secrets"])
		require.True(t, ok, "the shorthand should not be rewritten into a list")
		assert.Equal(t, "keep-me", got["EXISTING"])
		assert.Equal(t, "abc", got["TOKEN"])
	})

	t.Run("multiple secrets are appended in a stable order", func(t *testing.T) {
		out, err := injectManifestSecrets([]byte("name: demo\nagent: opencode\n"),
			map[string]string{"B_TOKEN": "b", "A_TOKEN": "a", "C_TOKEN": "c"})
		require.NoError(t, err)

		var names []string
		for _, slot := range slots(t, out) {
			m, _ := yamlMap(slot)
			names = append(names, m["name"].(string))
		}
		assert.Equal(t, []string{"A_TOKEN", "B_TOKEN", "C_TOKEN"}, names)
	})
}

func TestRedactManifestSecrets(t *testing.T) {
	t.Run("list form", func(t *testing.T) {
		in := []byte("name: demo\nsecrets:\n  - name: TOKEN\n    source: tenantSecret\n    value: hunter2\n")
		out := string(redactManifestSecrets(in))
		assert.NotContains(t, out, "hunter2")
		assert.Contains(t, out, "TOKEN")
		assert.Contains(t, out, redactedSecretValue)
	})

	t.Run("map shorthand", func(t *testing.T) {
		in := []byte("name: demo\nsecrets:\n  TOKEN: hunter2\n")
		out := string(redactManifestSecrets(in))
		assert.NotContains(t, out, "hunter2")
		assert.Contains(t, out, "TOKEN")
	})

	t.Run("legacy envelope", func(t *testing.T) {
		in := []byte("apiVersion: agents.digitalocean.com/v1alpha1\nspec:\n  secrets:\n    - name: TOKEN\n      value: hunter2\n")
		out := string(redactManifestSecrets(in))
		assert.NotContains(t, out, "hunter2")
		assert.Contains(t, out, "TOKEN")
	})

	t.Run("no secrets is a no-op", func(t *testing.T) {
		in := []byte("name: demo\nagent: opencode\n")
		assert.Equal(t, string(in), string(redactManifestSecrets(in)))
	})

	t.Run("unparsable input is returned as-is rather than swallowed", func(t *testing.T) {
		in := []byte("\tnot: valid: yaml:\n")
		assert.Equal(t, string(in), string(redactManifestSecrets(in)))
	})
}

// --- --from-config resolution ----------------------------------------------

func TestResolveConfigRef(t *testing.T) {
	t.Run("an ID is used directly with no lookup", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			// No ListAgentConfigs expectation: a prefixed ID must not cost a
			// round-trip.
			got, err := resolveConfigRef(config.HostedAgents(), "cfg_abc123")
			require.NoError(t, err)
			assert.Equal(t, "cfg_abc123", got)
		})
	})

	t.Run("a UUID is used directly too", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			id := "019f275e-96dc-7ea0-98bd-9ecf2a0834c3"
			got, err := resolveConfigRef(config.HostedAgents(), id)
			require.NoError(t, err)
			assert.Equal(t, id, got)
		})
	})

	t.Run("a name resolves case-insensitively", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListAgentConfigs(gomock.Any()).
				Return([]godo.HostedAgentConfigSummary{
					{ID: "cfg_1", Name: "other"},
					{ID: "cfg_2", Name: "Reviewer"},
				}, "", nil)

			got, err := resolveConfigRef(config.HostedAgents(), "reviewer")
			require.NoError(t, err)
			assert.Equal(t, "cfg_2", got)
		})
	})

	// A name that exists must not report "not found" because it sat on a later
	// page; there is no server-side name filter to lean on.
	t.Run("scans past the first page", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			gomock.InOrder(
				tm.hostedAgents.EXPECT().
					ListAgentConfigs(gomock.Any()).
					Return([]godo.HostedAgentConfigSummary{{ID: "cfg_1", Name: "other"}}, "page-2", nil),
				tm.hostedAgents.EXPECT().
					ListAgentConfigs(&godo.HostedAgentConfigListOptions{
						PageSize:  configRefPageSize,
						PageToken: "page-2",
					}).
					Return([]godo.HostedAgentConfigSummary{{ID: "cfg_9", Name: "reviewer"}}, "", nil),
			)

			got, err := resolveConfigRef(config.HostedAgents(), "reviewer")
			require.NoError(t, err)
			assert.Equal(t, "cfg_9", got)
		})
	})

	t.Run("no match names the command that lists them", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListAgentConfigs(gomock.Any()).
				Return([]godo.HostedAgentConfigSummary{{ID: "cfg_1", Name: "other"}}, "", nil)

			_, err := resolveConfigRef(config.HostedAgents(), "missing")
			require.Error(t, err)
			assert.Contains(t, err.Error(), `no agent config goes by the name "missing"`)
			assert.Contains(t, err.Error(), "config list")
		})
	})

	t.Run("an ambiguous name lists the candidate IDs", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				ListAgentConfigs(gomock.Any()).
				Return([]godo.HostedAgentConfigSummary{
					{ID: "cfg_1", Name: "dupe"},
					{ID: "cfg_2", Name: "dupe"},
				}, "", nil)

			_, err := resolveConfigRef(config.HostedAgents(), "dupe")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cfg_1")
			assert.Contains(t, err.Error(), "cfg_2")
		})
	})

	t.Run("an empty ref is an error, not a lookup", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			_, err := resolveConfigRef(config.HostedAgents(), "  ")
			require.Error(t, err)
		})
	})
}

func TestRunAgentsCreate_FromConfigByName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListAgentConfigs(gomock.Any()).
			Return([]godo.HostedAgentConfigSummary{{ID: "cfg_resolved", Name: "reviewer"}}, "", nil)
		tm.hostedAgents.EXPECT().
			CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
				Name:     "demo",
				ConfigID: "cfg_resolved",
			}).
			Return(nil, assertCalledErr)

		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "reviewer")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
	})
}

func TestLooksLikeConfigID(t *testing.T) {
	assert.True(t, looksLikeConfigID("cfg_abc123"))
	assert.True(t, looksLikeConfigID("019f275e-96dc-7ea0-98bd-9ecf2a0834c3"))
	assert.False(t, looksLikeConfigID("reviewer"))
	assert.False(t, looksLikeConfigID("sess_abc123"))
}

// --- --on-hitl --------------------------------------------------------------

func TestAgentOnHITLOutcome(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		outcome, err := agentOnHITLOutcome(config)
		require.NoError(t, err)
		assert.Empty(t, outcome, "unset means the run is not unattended")
	})

	for input, want := range map[string]godo.HostedAgentHITLOutcome{
		"approve": godo.HostedAgentHITLOutcomeApprove,
		"REJECT":  godo.HostedAgentHITLOutcomeReject,
		"defer":   godo.HostedAgentHITLOutcomeDefer,
	} {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgAgentOnHITL, input)
			outcome, err := agentOnHITLOutcome(config)
			require.NoError(t, err)
			assert.Equal(t, want, outcome)
		})
	}

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentOnHITL, "maybe")
		_, err := agentOnHITLOutcome(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), doctl.ArgAgentOnHITL)
		assert.Contains(t, err.Error(), "approve, reject, or defer")
	})
}

func TestHumanHITLOutcome(t *testing.T) {
	assert.Equal(t, "approve", humanHITLOutcome(godo.HostedAgentHITLOutcomeApprove))
	assert.Equal(t, "reject", humanHITLOutcome(godo.HostedAgentHITLOutcomeReject))
	assert.Equal(t, "defer", humanHITLOutcome(godo.HostedAgentHITLOutcomeDefer))
}

func TestHitlWatchTerminalRun(t *testing.T) {
	assert.True(t, hitlWatchTerminalRun(godo.HostedAgentEventKindRunCompleted))
	assert.True(t, hitlWatchTerminalRun(godo.HostedAgentEventKindRunFailed))
	assert.False(t, hitlWatchTerminalRun(godo.HostedAgentEventKindRunStarted))
	assert.False(t, hitlWatchTerminalRun(godo.HostedAgentEventKindHITLRequested))
	assert.False(t, hitlWatchTerminalRun(godo.HostedAgentEventKindRunPaused))
}

// The whole point of --on-hitl: an approval arrives with nobody watching, gets
// resolved with the fixed policy, and the watch ends when the run does.
func TestWatchSessionHeadless_ResolvesAndStopsAtTerminalRun(t *testing.T) {
	stubReconnectSleep(t)

	body := sseFrame("evt-1", string(godo.HostedAgentEventKindRunStarted), `{}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindHITLRequested), `{"hitl_id":"hitl_1","action":"shell","details":{"command":"rm -rf build"}}`) +
		sseFrame("evt-3", string(godo.HostedAgentEventKindRunCompleted), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				return openHostedAgentStream(t, client, opt), nil
			}).
			Times(1)
		tm.hostedAgents.EXPECT().
			ResolveHITL("sess_x", "hitl_1", &godo.HostedAgentResolveHITLRequest{
				Outcome: godo.HostedAgentHITLOutcomeApprove,
				Source:  godo.HostedAgentResolutionSourceOutOfBand,
			}).
			Return(nil).
			Times(1)

		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, watchSessionHeadless(config, "sess_x", godo.HostedAgentHITLOutcomeApprove))

		out := buf.String()
		assert.Contains(t, out, "approvals will be approve automatically")
		assert.Contains(t, out, "rm -rf build", "the command being approved has to be in the log")
		assert.Contains(t, out, "auto-approve hitl_1")
		assert.Contains(t, out, string(godo.HostedAgentEventKindRunCompleted))
	})
}

// SSE replay after a reconnect re-delivers the same approval. Resolving twice is
// harmless server-side but would double the log lines and the API calls, so the
// second sighting is dropped.
func TestWatchSessionHeadless_ReplayedApprovalResolvedOnce(t *testing.T) {
	stubReconnectSleep(t)

	hitl := sseFrame("evt-1", string(godo.HostedAgentEventKindHITLRequested), `{"hitl_id":"hitl_1","action":"shell"}`)
	first := httptest.NewServer(hostedAgentSSEHandler(hitl, nil))
	t.Cleanup(first.Close)
	replay := httptest.NewServer(hostedAgentSSEHandler(hitl+sseFrame("evt-2", string(godo.HostedAgentEventKindRunCompleted), `{}`), nil))
	t.Cleanup(replay.Close)

	firstClient, err := godo.New(nil, godo.SetBaseURL(first.URL+"/"))
	require.NoError(t, err)
	replayClient, err := godo.New(nil, godo.SetBaseURL(replay.URL+"/"))
	require.NoError(t, err)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		gomock.InOrder(
			tm.hostedAgents.EXPECT().
				StreamSession(gomock.Any(), "sess_x", gomock.Any()).
				DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
					return openHostedAgentStream(t, firstClient, opt), nil
				}),
			tm.hostedAgents.EXPECT().
				StreamSession(gomock.Any(), "sess_x", gomock.Any()).
				DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
					assert.Equal(t, "evt-1", opt.ReplayFrom, "the reconnect must resume from the last event seen")
					return openHostedAgentStream(t, replayClient, opt), nil
				}),
		)
		tm.hostedAgents.EXPECT().
			ResolveHITL("sess_x", "hitl_1", gomock.Any()).
			Return(nil).
			Times(1)

		config.Out = io.Discard
		require.NoError(t, watchSessionHeadless(config, "sess_x", godo.HostedAgentHITLOutcomeApprove))
	})
}

// A failed resolve must not be swallowed: an unattended run that quietly stops
// getting approvals is indistinguishable from one that hung.
func TestWatchSessionHeadless_ResolveFailureSurfaces(t *testing.T) {
	stubReconnectSleep(t)

	body := sseFrame("evt-1", string(godo.HostedAgentEventKindHITLRequested), `{"hitl_id":"hitl_1","action":"shell"}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				return openHostedAgentStream(t, client, opt), nil
			}).
			Times(1)
		tm.hostedAgents.EXPECT().
			ResolveHITL("sess_x", "hitl_1", gomock.Any()).
			Return(errors.New("403 forbidden"))

		config.Out = io.Discard
		err := watchSessionHeadless(config, "sess_x", godo.HostedAgentHITLOutcomeReject)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hitl_1")
		assert.Contains(t, err.Error(), "reject")
		assert.Contains(t, err.Error(), "403 forbidden")
	})
}

// A gone session ends the watch cleanly rather than retrying to the failure cap:
// there is nothing left to wait for.
func TestWatchSessionHeadless_TerminalStreamErrorStopsCleanly(t *testing.T) {
	stubReconnectSleep(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			Return(nil, terminalStreamErr()).
			Times(1)

		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, watchSessionHeadless(config, "sess_x", godo.HostedAgentHITLOutcomeApprove))
	})
}

// Token deltas would flood a CI log one word at a time, and stream.state is
// connection bookkeeping rather than session activity. Both are dropped; the
// assistant's full output stays recoverable with `logs`.
func TestWatchSessionHeadless_SkipsNoiseEvents(t *testing.T) {
	stubReconnectSleep(t)

	body := sseFrame("evt-1", string(godo.HostedAgentEventKindTokenChunk), `{"text":"thinking out loud"}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindStreamState), `{"state":"live","cursor":""}`) +
		sseFrame("evt-3", string(godo.HostedAgentEventKindRunCompleted), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				return openHostedAgentStream(t, client, opt), nil
			}).
			Times(1)

		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, watchSessionHeadless(config, "sess_x", godo.HostedAgentHITLOutcomeApprove))

		out := buf.String()
		assert.NotContains(t, out, "thinking out loud")
		assert.NotContains(t, out, string(godo.HostedAgentEventKindStreamState))
		assert.Contains(t, out, string(godo.HostedAgentEventKindRunCompleted))
	})
}

// End to end: without --on-hitl, create returns at ready. With it, create keeps
// going and watches the run instead — no TTY involved either way, which is the
// whole reason the flag exists.
func TestRunAgentsCreate_OnHITLWatchesPastReady(t *testing.T) {
	stubReconnectSleep(t)
	stubInteractiveTerminal(t, false)

	body := sseFrame("evt-1", string(godo.HostedAgentEventKindHITLRequested), `{"hitl_id":"hitl_1","action":"shell"}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindRunCompleted), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dir := t.TempDir()
		manifestPath := filepath.Join(dir, "agents.yaml")
		require.NoError(t, os.WriteFile(manifestPath, []byte(sampleFlatManifest), 0o644))

		ready := &do.HostedAgentSession{
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID: "sess_x",
				Name:      "demo",
				Status:    godo.HostedAgentSessionStatusReady,
			},
		}
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			Return(ready, nil)
		tm.hostedAgents.EXPECT().
			GetSession("sess_x").
			Return(ready, nil)
		tm.hostedAgents.EXPECT().
			StreamSession(gomock.Any(), "sess_x", gomock.Any()).
			DoAndReturn(func(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				return openHostedAgentStream(t, client, opt), nil
			}).
			Times(1)
		tm.hostedAgents.EXPECT().
			ResolveHITL("sess_x", "hitl_1", &godo.HostedAgentResolveHITLRequest{
				Outcome: godo.HostedAgentHITLOutcomeApprove,
				Source:  godo.HostedAgentResolutionSourceOutOfBand,
			}).
			Return(nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{manifestPath}
		config.Doit.Set(config.NS, doctl.ArgAgentOnHITL, "approve")

		require.NoError(t, RunAgentsCreate(config))

		out := buf.String()
		assert.Contains(t, out, "auto-approve hitl_1")
		assert.NotContains(t, out, "doctl "+agentCLI+" launch demo",
			"the ready card belongs to the returning form of create, not the watching one")
	})
}

func TestHeadlessEventDetail(t *testing.T) {
	detail := headlessEventDetail(godo.HostedAgentEvent{
		Kind:    godo.HostedAgentEventKindRunFailed,
		Payload: json.RawMessage(`{"message":"the build broke"}`),
	})
	assert.Equal(t, " the build broke", detail)

	detail = headlessEventDetail(godo.HostedAgentEvent{
		Kind:    godo.HostedAgentEventKindHITLRequested,
		Payload: json.RawMessage(`{"hitl_id":"hitl_1","action":"shell","details":{"command":"git push --force"}}`),
	})
	assert.Contains(t, detail, "git push --force")

	// An event with nothing worth adding, and one with unreadable JSON, both
	// have to degrade to a bare kind line rather than breaking the log.
	assert.Empty(t, headlessEventDetail(godo.HostedAgentEvent{Kind: godo.HostedAgentEventKindRunStarted}))
	assert.Empty(t, headlessEventDetail(godo.HostedAgentEvent{
		Kind:    godo.HostedAgentEventKindRunFailed,
		Payload: json.RawMessage(`{not-json`),
	}))
}
