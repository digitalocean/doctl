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
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	domocks "github.com/digitalocean/doctl/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
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
  runtime:
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

	assertCommandNames(t, cmd, "create", "validate", "launch", "list", "show", "logs", "approve", "remove", "pause", "resume", "upload", "download", "start-proxy", "port-forward", "auth", "fork", "rollback", "checkpoint", "triggers", "config", "sizes", "exec")
}

// start, run and attach are gone outright rather than kept as deprecated
// aliases: leaving them registered would preserve the three-near-synonym
// problem the split exists to remove.
func TestAgentsOldVerbsAreGone(t *testing.T) {
	cmd := Agents()
	for _, name := range []string{"start", "run"} {
		found, _, err := cmd.Find([]string{name})
		if err == nil {
			assert.NotEqual(t, name, found.Name(), "%q should no longer be a command", name)
		}
	}
}

// attach survives only as an alias of launch, for muscle memory.
func TestAgentsLaunchAliases(t *testing.T) {
	cmd := Agents()
	for _, name := range []string{"launch", "up", "chat", "attach"} {
		found, _, err := cmd.Find([]string{name})
		require.NoError(t, err, "alias %q", name)
		require.NotNil(t, found, "alias %q", name)
		assert.Equal(t, "launch", found.Name(), "alias %q should resolve to launch", name)
	}
}

func TestAgentsPrimaryNameIsOpenHarnessRuntime(t *testing.T) {
	cmd := Agents()
	assert.Equal(t, agentCmdName, cmd.Name())
	assert.Contains(t, cmd.Aliases, "agent")
	assert.Contains(t, cmd.Aliases, "agents")
	assert.Contains(t, cmd.Aliases, "ohr")

	found, _, err := cmd.Find([]string{"launch"})
	require.NoError(t, err)
	assert.Equal(t, "launch", found.Name())

	var create *Command
	for _, child := range cmd.ChildCommands() {
		if child.Name() == "create" {
			create = child
			break
		}
	}
	require.NotNil(t, create)
	assert.Equal(t, "agents.create", cmdNS(create), "viper keys must stay under agents.*")
}

func TestAgentsRemoveAliases(t *testing.T) {
	cmd := Agents()
	require.NotNil(t, cmd)

	for _, name := range []string{"remove", "destroy", "rm"} {
		found, _, err := cmd.Find([]string{name})
		require.NoError(t, err, "alias %q", name)
		require.NotNil(t, found, "alias %q", name)
		assert.Equal(t, "remove", found.Name(), "alias %q should resolve to remove", name)
	}
}

// TestAgentsUnknownSubcommandFails is MARSOHS-1075: unknown nested subcommands must
// exit non-zero, not print parent help and exit 0. ValidateArgs checks the NoArgs
// guard directly; avoid Execute() here because it runs cobra.OnInitialize
// (initConfig) and pollutes viper for other tests in the package.
func TestAgentsUnknownSubcommandFails(t *testing.T) {
	cmd := Agents()
	require.NoError(t, cmd.ValidateArgs(nil))

	err := cmd.ValidateArgs([]string{"frobnicate"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown command "frobnicate"`)

	config := AgentConfigs()
	require.NoError(t, config.ValidateArgs(nil))

	err = config.ValidateArgs([]string{"frobnicate"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown command "frobnicate"`)
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

	t.Run("setBracketedPasteMode", func(t *testing.T) {
		var buf bytes.Buffer
		setBracketedPasteMode(&buf, true)
		setBracketedPasteMode(&buf, false)
		assert.Equal(t, "\x1b[?2004h\x1b[?2004l", buf.String())
	})
}

func TestNamedManifestPath(t *testing.T) {
	t.Run("flag wins when it is the only source", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgAgentSpec, "from-flag.yaml")
			path, err := namedManifestPath(config)
			assert.NoError(t, err)
			assert.Equal(t, "from-flag.yaml", path)
		})
	})

	t.Run("positional path is accepted", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Args = []string{"from-arg.yaml"}
			path, err := namedManifestPath(config)
			assert.NoError(t, err)
			assert.Equal(t, "from-arg.yaml", path)
		})
	})

	t.Run("flag and positional together are rejected", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgAgentSpec, "from-flag.yaml")
			config.Args = []string{"from-arg.yaml"}
			_, err := namedManifestPath(config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "given twice")
		})
	})

	t.Run("more than one positional is rejected", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Args = []string{"a.yaml", "b.yaml"}
			_, err := namedManifestPath(config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "at most one manifest path")
		})
	})

	t.Run("nothing named returns empty", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			path, err := namedManifestPath(config)
			assert.NoError(t, err)
			assert.Empty(t, path)
		})
	})

	t.Run("stale required mark does not block empty spec", func(t *testing.T) {
		requiredKey := "required.agents.start.spec"
		viper.Set(requiredKey, true)
		t.Cleanup(func() { viper.Set(requiredKey, false) })

		config := &CmdConfig{
			NS:   "agents.start",
			Doit: &doctl.LiveConfig{},
		}
		path, err := namedManifestPath(config)
		assert.NoError(t, err)
		assert.Empty(t, path)

		_, err = config.Doit.GetString(config.NS, doctl.ArgAgentSpec)
		assert.Error(t, err, "LiveConfig still enforces required; namedManifestPath must bypass it")
	})
}

// Attaching used to be a flag on start/run, decided by attachAfterStart. It is
// now the difference between two commands: create never attaches and launch
// always does. These pin that boundary, since it is the whole point of the
// split and a regression would silently reintroduce the ambiguity.
func TestCreateNeverAttaches(t *testing.T) {
	// A terminal is the condition under which the old code would have attached.
	stubInteractiveTerminal(t, true)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_no_attach",
					Name:      "demo",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)
		// Exactly one GetSession, from the readiness wait. Attaching would call
		// it again to fetch the session it is about to stream.
		tm.hostedAgents.EXPECT().
			GetSession("sess_no_attach").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_no_attach",
					Name:      "demo",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil).
			Times(1)

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")

		require.NoError(t, RunAgentsCreate(config))
		out := buf.String()
		assert.Contains(t, out, "Agent is ready")
		assert.Contains(t, out, "doctl harness-runtime launch demo",
			"the ready card should point at launch, the command that attaches")
	})
}

func TestRunAgentsLaunch_RefusesWithoutATerminal(t *testing.T) {
	stubInteractiveTerminal(t, false)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		err := RunAgentsLaunch(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "needs a terminal")
		assert.Contains(t, err.Error(), "create", "the error must name the command that works in a pipeline")
	})
}

func TestRunAgentsLaunch_RejectsJSONOutput(t *testing.T) {
	prev := Output
	Output = "json"
	t.Cleanup(func() { Output = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunAgentsLaunch(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no JSON form")
	})
}

func TestRunAgentsLaunch_RejectsCreateOnlyFlags(t *testing.T) {
	for _, flag := range []string{doctl.ArgAgentDryRun, doctl.ArgAgentOnHITL} {
		t.Run(flag, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, flag, "approve")
				err := RunAgentsLaunch(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), flag)
				assert.Contains(t, err.Error(), "create")
			})
		})
	}
}

func TestDiscoverManifestFile(t *testing.T) {
	t.Run("finds agents.yaml in the working directory", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "agents.yaml"), []byte(sampleFlatManifest), 0o644))
		t.Chdir(dir)
		assert.Equal(t, "agents.yaml", discoverManifestFile())
	})

	t.Run("falls back to the .yml spelling", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "agents.yml"), []byte(sampleFlatManifest), 0o644))
		t.Chdir(dir)
		assert.Equal(t, "agents.yml", discoverManifestFile())
	})

	t.Run("ignores a directory of that name", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "agents.yaml"), 0o755))
		t.Chdir(dir)
		assert.Empty(t, discoverManifestFile())
	})

	t.Run("empty when there is nothing to find", func(t *testing.T) {
		t.Chdir(t.TempDir())
		assert.Empty(t, discoverManifestFile())
	})
}

func TestRunAgentsCreate_DiscoversAgentsYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agents.yaml"), []byte(sampleFlatManifest), 0o644))
	t.Chdir(dir)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			DoAndReturn(func(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				assert.Contains(t, string(manifest), "agent: opencode")
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess-discovered",
						Name:      "test-agent",
						Status:    godo.HostedAgentSessionStatusProvisioning,
					},
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetSession("sess-discovered").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess-discovered",
					Name:      "test-agent",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil).
			AnyTimes()

		assert.NoError(t, RunAgentsCreate(config))
	})
}

func TestRunAgentsCreate_RejectsManifestArgWithHarness(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"agents.yaml"}
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		err := RunAgentsCreate(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be combined with")
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

func TestRunAgentsCreate(t *testing.T) {
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
					Status:    godo.HostedAgentSessionStatusProvisioning,
				},
			}, nil)
		tm.hostedAgents.EXPECT().
			GetSession("sess_test").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_test",
					AgentKind: godo.HostedAgentKindOpenCode,
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		assert.NoError(t, RunAgentsCreate(config))
	})
}

func TestRunAgentsCreate_FromHarness(t *testing.T) {
	t.Setenv(anthropicAPIKeyEnv, "sk-ant-test")
	origValidate := validateAnthropicAPIKey
	t.Cleanup(func() { validateAnthropicAPIKey = origValidate })
	validateAnthropicAPIKey = func(ctx context.Context, apiKey string) error { return nil }

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			DoAndReturn(func(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				var doc map[string]any
				assert.NoError(t, yaml.Unmarshal(manifest, &doc))
				assert.Equal(t, "claude-code", doc["agent"])
				env, ok := doc["env"].(map[any]any)
				assert.True(t, ok)
				assert.Equal(t, "sk-ant-test", env["ANTHROPIC_API_KEY"])
				return &do.HostedAgentSession{
					HostedAgentSession: &godo.HostedAgentSession{
						SessionID: "sess_harness",
						Name:      "harness-demo",
						Status:    godo.HostedAgentSessionStatusProvisioning,
					},
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetSession("sess_harness").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_harness",
					Name:      "harness-demo",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "claude-code")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "harness-demo")
		assert.NoError(t, RunAgentsCreate(config))
	})
}

func TestRunAgentsCreate_WithName(t *testing.T) {
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
		tm.hostedAgents.EXPECT().
			GetSession("sess_test").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_test",
					Name:      "my-session",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-session")
		assert.NoError(t, RunAgentsCreate(config))
	})
}

func TestAgentsCreate_SpecNotRequiredForFromConfig(t *testing.T) {
	cmd, _, err := DoitCmd.Find([]string{"agents", "create"})
	require.NoError(t, err)
	require.NotNil(t, cmd.Flags().Lookup(doctl.ArgAgentFromConfig))

	// LiveConfig.GetString is what the real CLI uses; TestConfig skips the
	// required-flag check, so the --from-config runner tests cannot catch this.
	require.False(t, viper.GetBool("required.agents.create.spec"),
		"spec is still marked required; `doctl harness-runtime create --from-config` fails with (agents.create.spec) command is missing required arguments")
	_, err = (&doctl.LiveConfig{}).GetString("agents.create", doctl.ArgAgentSpec)
	require.NoError(t, err)
}

func TestRunAgentsCreate_SpecAndConfigMutuallyExclusive(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, "agent.yaml")
		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		err := RunAgentsCreate(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mutually exclusive")
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

func TestRunAgentsCreate_FlatWithName(t *testing.T) {
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
		tm.hostedAgents.EXPECT().
			GetSession("sess_test").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_test",
					Name:      "my-session",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		prev := sessionReadyPollInterval
		sessionReadyPollInterval = time.Millisecond
		defer func() { sessionReadyPollInterval = prev }()

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentName, "my-session")
		assert.NoError(t, RunAgentsCreate(config))
	})
}

func TestRunAgentsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().ListSessions(nil).Return([]do.HostedAgentSession{}, "", nil)
		var buf bytes.Buffer
		config.Out = &buf
		assert.NoError(t, RunAgentsList(config))
		assert.Contains(t, buf.String(), "No sessions")
	})
}

func TestRunAgentsList_StyledText(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().ListSessions(nil).Return([]do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{
				SessionID: "sess_1",
				Name:      "demo",
				AgentKind: godo.HostedAgentKindOpenCode,
				Status:    godo.HostedAgentSessionStatusReady,
			}},
		}, "", nil)
		var buf bytes.Buffer
		config.Out = &buf
		assert.NoError(t, RunAgentsList(config))
		got := buf.String()
		assert.Contains(t, got, "1 session")
		assert.Contains(t, got, "demo")
		assert.Contains(t, got, "ready")
		assert.NotContains(t, got, "SESSION_STATUS_")
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
			HostedAgentSession: &godo.HostedAgentSession{
				SessionID: "sess_test",
				Name:      "demo",
				AgentKind: godo.HostedAgentKindOpenCode,
				Status:    godo.HostedAgentSessionStatusReady,
			},
		}, nil)
		config.Args = []string{"sess_test"}
		var buf bytes.Buffer
		config.Out = &buf
		assert.NoError(t, RunAgentsShow(config))
		got := buf.String()
		assert.Contains(t, got, "demo")
		assert.Contains(t, got, "ready")
		assert.Contains(t, got, "doctl harness-runtime launch demo")
		assert.NotContains(t, got, "SESSION_STATUS_")
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
// (type/data/timestamp) -> godo HostedAgentEvent mapping.
func TestHostedAgentEventDecodesSPIWire(t *testing.T) {
	const frame = `{"event_id":"01KTBXPBY60VYC5YKF6AKDX0ZS","run_id":"run-7f16719a-da1c-449d-a4ca-18e524bb63e3","session_id":"sess_5a1ff33e","timestamp":"2026-06-05T12:56:24.774753219Z","seq":0,"type":"run.token_delta","data":{"text":"Paris"}}`

	var ev godo.HostedAgentEvent
	assert.NoError(t, json.Unmarshal([]byte(frame), &ev))

	assert.Equal(t, "01KTBXPBY60VYC5YKF6AKDX0ZS", ev.EventID)
	assert.Equal(t, "run-7f16719a-da1c-449d-a4ca-18e524bb63e3", ev.RunID)
	assert.Equal(t, "sess_5a1ff33e", ev.SessionID)
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
		{"tool call started file change", godo.HostedAgentEventKindToolCallStarted, "", `{"tool_call_id":"t1","name":"file_change","arguments":{"path":"index.html","operation":"add"}}`, "\n▸ file_change index.html (add)\n"},
		{"tool call started file change no operation", godo.HostedAgentEventKindToolCallStarted, "", `{"tool_call_id":"t1","name":"file_change","arguments":{"path":"index.html"}}`, "\n▸ file_change index.html\n"},
		{"tool call completed", godo.HostedAgentEventKindToolCallCompleted, "", `{"ok":true,"duration_ms":12,"summary":"ran ls"}`, "  ✓ ran ls (12ms)\n"},
		{"tool call failed", godo.HostedAgentEventKindToolCallCompleted, "", `{"ok":false,"duration_ms":3,"summary":"boom"}`, "  ✗ boom (3ms)\n"},
		{"tool call completed no duration", godo.HostedAgentEventKindToolCallCompleted, "", `{"ok":true,"duration_ms":0,"summary":"ran ls"}`, "  ✓ ran ls\n"},
		{"tool call completed no summary", godo.HostedAgentEventKindToolCallCompleted, "", `{"ok":true,"duration_ms":12}`, "  ✓ done (12ms)\n"},
		{"tool call completed multiline summary", godo.HostedAgentEventKindToolCallCompleted, "", `{"ok":true,"duration_ms":12,"summary":"HTTP/2 200\nserver: nginx"}`, "  ✓ HTTP/2 200 (12ms)\n"},
		{"run completed", godo.HostedAgentEventKindRunCompleted, "", `{"total_tokens_in":3,"total_tokens_out":5,"run_cost_micros":1234}`, "\n✓ run complete · 3 in / 5 out tokens · $0.0012\n" + runSeparator + "\n"},
		{"run completed no cost", godo.HostedAgentEventKindRunCompleted, "", `{"total_tokens_in":133328,"total_tokens_out":5414,"run_cost_micros":0}`, "\n✓ run complete · 133328 in / 5414 out tokens\n" + runSeparator + "\n"},
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

// TestRunAgentsCreate_SkillsEnvSizeCapError pins that harness-api's
// HARNESS_SKILLS env-size-cap rejection (agentspec.validateSkillsEnvSize,
// returned as a 400 with the nested {"error":{"code":...,"message":...}}
// envelope) surfaces to the CLI user as the server's own readable message,
// not a raw JSON/HTTP dump.
func TestRunAgentsCreate_SkillsEnvSizeCapError(t *testing.T) {
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
		err := RunAgentsCreate(config)
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

// visibleText drops SGR sequences so a test can assert on what a reader sees.
func visibleText(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}

// TestRenderMarkdownDropsInlineCodePadding pins the removal of glamour's
// literal space on each side of an inline-code span, which read as a typo
// ("in  /workspace , and") wherever the highlight color is unavailable.
func TestRenderMarkdownDropsInlineCodePadding(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = true
	defer func() { stylingEnabled = prev }()

	out := visibleText(renderMarkdown("Set up in `/workspace`, archived as `site.tar.gz`.\n"))
	assert.Contains(t, out, "Set up in /workspace, archived as site.tar.gz.")
	assert.NotContains(t, out, " /workspace ,")
	assert.NotContains(t, out, "  /workspace")
}

// TestRenderMarkdownTrimsPaddingAndBlankEdges covers the two invisible sources
// of whitespace: glamour pads every wrapped line out to the wrap column, and
// wraps the document in leading/trailing newlines that stack with the newlines
// the surrounding event lines already print.
func TestRenderMarkdownTrimsPaddingAndBlankEdges(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = true
	defer func() { stylingEnabled = prev }()

	out := renderMarkdown("This paragraph is deliberately long enough that it must wrap across more " +
		"than one terminal line, which is when the padding shows up.\n")

	assert.False(t, strings.HasPrefix(out, "\n"), "block should not open with a blank line")
	assert.False(t, strings.HasSuffix(out, "\n"), "block should not close with a blank line")
	for i, line := range strings.Split(visibleText(out), "\n") {
		assert.Equal(t, strings.TrimRight(line, " "), line, "line %d keeps trailing padding", i)
	}
}

func TestTrimLinePaddingKeepsBackgroundFill(t *testing.T) {
	// A code-block line whose padding carries a background fill must survive
	// intact, or the block renders with a ragged right edge.
	filled := "\x1b[48;5;236m  print('hi')      \x1b[0m"
	assert.Equal(t, filled, trimLinePadding(filled))

	// Foreground-only padding is glamour's wrap filler and gets dropped.
	padded := "  and the\x1b[38;5;252m \x1b[0m\x1b[38;5;252m \x1b[0m\x1b[0m"
	assert.Equal(t, "  and the\x1b[38;5;252m\x1b[0m\x1b[38;5;252m\x1b[0m\x1b[0m", trimLinePadding(padded))
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
			stop, err := handleAttachByte(nil, nil, "sess", s[i], state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)
		}
	}
	arrow := func(t *testing.T, state *attachState, dir byte) {
		t.Helper()
		for _, b := range []byte{0x1b, '[', dir} {
			stop, err := handleAttachByte(nil, nil, "sess", b, state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)
		}
	}
	escSeq := func(t *testing.T, state *attachState, seq []byte) {
		t.Helper()
		for _, b := range seq {
			stop, err := handleAttachByte(nil, nil, "sess", b, state, nil, nil)
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
		stop, err := handleAttachByte(nil, nil, "sess", 0x7f, state, nil, nil)
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "acd", string(state.lineBuf))
		assert.Equal(t, 1, state.cursor)
	})

	t.Run("emacs line editing shortcuts", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		typewrite(t, state, "hello world")
		assert.Equal(t, 11, state.cursor)

		stop, err := handleAttachByte(nil, nil, "sess", 0x01, state, nil, nil) // Ctrl-A
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, 0, state.cursor)

		stop, err = handleAttachByte(nil, nil, "sess", 0x05, state, nil, nil) // Ctrl-E
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, 11, state.cursor)

		stop, err = handleAttachByte(nil, nil, "sess", 0x17, state, nil, nil) // Ctrl-W at EOL
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "hello ", string(state.lineBuf))
		assert.Equal(t, 6, state.cursor)
	})

	t.Run("kill line and forward delete shortcuts", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)

		typewrite(t, state, "hello world")
		stop, err := handleAttachByte(nil, nil, "sess", 0x15, state, nil, nil) // Ctrl-U at EOL
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "", string(state.lineBuf))

		typewrite(t, state, "hello world")
		stop, err = handleAttachByte(nil, nil, "sess", 0x01, state, nil, nil) // Ctrl-A
		assert.NoError(t, err)
		stop, err = handleAttachByte(nil, nil, "sess", 0x0b, state, nil, nil) // Ctrl-K
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "", string(state.lineBuf))

		typewrite(t, state, "wxyz")
		arrow(t, state, 'D')
		arrow(t, state, 'D')                                                  // caret before 'y'
		stop, err = handleAttachByte(nil, nil, "sess", 0x04, state, nil, nil) // Ctrl-D
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "wxz", string(state.lineBuf))

		stop, err = handleAttachByte(nil, nil, "sess", 0x05, state, nil, nil) // Ctrl-E
		assert.NoError(t, err)
		typewrite(t, state, "q")
		escSeq(t, state, []byte{0x1b, '[', '1', '~'}) // Home
		escSeq(t, state, []byte{0x1b, '[', '3', '~'}) // Delete
		assert.Equal(t, "xzq", string(state.lineBuf))
	})

	t.Run("home/end and tab completion", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		typewrite(t, state, "/he")
		stop, err := handleAttachByte(nil, nil, "sess", 0x09, state, nil, nil) // Tab
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "/help ", string(state.lineBuf))

		typewrite(t, state, "xyz")
		escSeq(t, state, []byte{0x1b, '[', 'H'}) // Home
		assert.Equal(t, 0, state.cursor)
		escSeq(t, state, []byte{0x1b, '[', 'F'}) // End
		assert.Equal(t, len(state.lineBuf), state.cursor)
	})

	t.Run("esc clears input and exits history browse", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		state.pushHistory("saved")
		state.historyUp()
		assert.Equal(t, "saved", string(state.lineBuf))

		state.escSeq = []byte{0x1b}
		assert.True(t, state.handlePendingEscTimeout())
		assert.Equal(t, "", string(state.lineBuf))
	})

	t.Run("ctrl-D detaches on empty prompt even after pending esc", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		state.sessionRef = "demo"
		state.escSeq = []byte{0x1b}

		stop, err := handleAttachByte(nil, nil, "sess", 0x04, state, nil, nil)
		assert.NoError(t, err)
		assert.True(t, stop)
	})

	t.Run("slash exit closes attach loop", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
			state := newAttachState(io.Discard, &pendingHITL{})
			state.sessionRef = "demo"
			assert.True(t, processAttachLine(config, nil, "sess", "/exit", state, nil, nil))
		})
	})

	t.Run("up/down recalls submitted input history", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				SendInput("sess", &godo.HostedAgentSendInputRequest{Text: "first"}).
				Return(&godo.HostedAgentSendInputResponse{RunID: "run_1"}, nil)
			tm.hostedAgents.EXPECT().
				SendInput("sess", &godo.HostedAgentSendInputRequest{Text: "second"}).
				Return(&godo.HostedAgentSendInputResponse{RunID: "run_2"}, nil)

			state := newAttachState(io.Discard, &pendingHITL{})
			state.display.setRaw(true)

			typewrite(t, state, "first")
			stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 0x0d, state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)

			typewrite(t, state, "second")
			stop, err = handleAttachByte(config, tm.hostedAgents, "sess", 0x0d, state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)

			arrow(t, state, 'A') // up
			assert.Equal(t, "second", string(state.lineBuf))
			arrow(t, state, 'A') // up
			assert.Equal(t, "first", string(state.lineBuf))
			arrow(t, state, 'B') // down
			assert.Equal(t, "second", string(state.lineBuf))
			arrow(t, state, 'B') // down to draft
			assert.Equal(t, "", string(state.lineBuf))
		})
	})

	t.Run("enter clears the caret", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				SendInput("sess", &godo.HostedAgentSendInputRequest{Text: "hi"}).
				Return(&godo.HostedAgentSendInputResponse{RunID: "run_1"}, nil)

			state := newAttachState(io.Discard, &pendingHITL{})
			state.display.setRaw(true)
			typewrite(t, state, "hi")
			stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 0x0d, state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)
			assert.Equal(t, "", string(state.lineBuf))
			assert.Equal(t, 0, state.cursor)
		})
	})

	t.Run("bracketed paste buffers multiline input into one submit", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				SendInput("sess", &godo.HostedAgentSendInputRequest{Text: "first line\nsecond line"}).
				Return(&godo.HostedAgentSendInputResponse{RunID: "run_1"}, nil)

			state := newAttachState(io.Discard, &pendingHITL{})
			state.display.setRaw(true)
			for _, b := range []byte("\x1b[200~first line\r\nsecond line\x1b[201~") {
				stop, err := handleAttachByte(config, tm.hostedAgents, "sess", b, state, nil, nil)
				assert.NoError(t, err)
				assert.False(t, stop)
			}
			assert.Equal(t, "first line\nsecond line", string(state.lineBuf))

			stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 0x0d, state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)
			assert.Equal(t, "", string(state.lineBuf))
			assert.Equal(t, 0, state.cursor)
		})
	})

	t.Run("large multiline paste requires confirmation", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			text := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6"
			tm.hostedAgents.EXPECT().
				SendInput("sess", &godo.HostedAgentSendInputRequest{Text: text}).
				Return(&godo.HostedAgentSendInputResponse{RunID: "run_1"}, nil)

			state := newAttachState(io.Discard, &pendingHITL{})
			state.display.setRaw(true)
			for _, b := range []byte("\x1b[200~Line 1\r\nLine 2\r\nLine 3\r\nLine 4\r\nLine 5\r\nLine 6\x1b[201~") {
				stop, err := handleAttachByte(config, tm.hostedAgents, "sess", b, state, nil, nil)
				assert.NoError(t, err)
				assert.False(t, stop)
			}

			stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 0x0d, state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)
			require.NotNil(t, state.largePasteConfirmation())

			stop, err = handleAttachByte(config, tm.hostedAgents, "sess", 'y', state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)
			assert.Nil(t, state.largePasteConfirmation())
		})
	})

	t.Run("large multiline paste can fall back to line by line", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			for i := 1; i <= 6; i++ {
				tm.hostedAgents.EXPECT().
					SendInput("sess", &godo.HostedAgentSendInputRequest{Text: fmt.Sprintf("Line %d", i)}).
					Return(&godo.HostedAgentSendInputResponse{RunID: fmt.Sprintf("run_%d", i)}, nil)
			}

			state := newAttachState(io.Discard, &pendingHITL{})
			state.display.setRaw(true)
			for _, b := range []byte("\x1b[200~Line 1\r\nLine 2\r\nLine 3\r\nLine 4\r\nLine 5\r\nLine 6\x1b[201~") {
				stop, err := handleAttachByte(config, tm.hostedAgents, "sess", b, state, nil, nil)
				assert.NoError(t, err)
				assert.False(t, stop)
			}

			stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 0x0d, state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)
			require.NotNil(t, state.largePasteConfirmation())

			stop, err = handleAttachByte(config, tm.hostedAgents, "sess", 'n', state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)
			assert.Nil(t, state.largePasteConfirmation())
		})
	})
}

// TestInsertNewlineAtCursor covers appending at the end of the buffer and
// splicing in the middle, mirroring the character-insert splice used
// elsewhere for regular typed bytes.
func TestInsertNewlineAtCursor(t *testing.T) {
	s := &attachState{lineBuf: []byte("abcd"), cursor: 4}
	s.insertNewlineAtCursor()
	assert.Equal(t, "abcd\n", string(s.lineBuf))
	assert.Equal(t, 5, s.cursor)

	s = &attachState{lineBuf: []byte("abcd"), cursor: 2}
	s.insertNewlineAtCursor()
	assert.Equal(t, "ab\ncd", string(s.lineBuf))
	assert.Equal(t, 3, s.cursor)
}

func TestWordBoundaryScanners(t *testing.T) {
	const line = "docs/agents.md is ready"
	cases := []struct {
		name string
		fn   func([]byte, int) int
		pos  int
		want int
	}{
		{"word start skips separators then the word", wordStartBefore, len(line), 18},
		{"word start from mid-word", wordStartBefore, 4, 0},
		{"word start stops before punctuation run", wordStartBefore, 12, 5},
		{"word start at line start", wordStartBefore, 0, 0},
		{"word end from line start", wordEndAfter, 0, 4},
		{"word end skips leading separator", wordEndAfter, 4, 11},
		{"word end at line end", wordEndAfter, len(line), len(line)},
		{"token start takes the whole path", tokenStartBefore, 14, 0},
		{"token start from line end", tokenStartBefore, len(line), 18},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.fn([]byte(line), tc.pos))
		})
	}
}

// TestWordScannersAreRuneSafe covers pasted multibyte text: a boundary must
// never land inside a rune, or the buffer is corrupted on the next edit.
func TestWordScannersAreRuneSafe(t *testing.T) {
	buf := []byte("héllo wörld")
	assert.Equal(t, len("héllo "), wordStartBefore(buf, len(buf)))
	assert.Equal(t, len("héllo"), wordEndAfter(buf, 0))
	assert.True(t, utf8.Valid(buf[wordStartBefore(buf, len(buf)):]))
}

func feedEscape(t *testing.T, s *attachState, seq ...byte) {
	t.Helper()
	for i, b := range seq {
		assert.True(t, handleAttachEscapeSequence(b, s),
			"byte %d (%q) must be consumed by the escape parser, not leak to the text path", i, b)
	}
	assert.Empty(t, s.escSeq, "parser should not leave a partial sequence buffered")
}

func newTestAttachState(line string) *attachState {
	s := newAttachState(io.Discard, &pendingHITL{})
	s.lineBuf = []byte(line)
	s.cursor = len(line)
	return s
}

// TestMetaKeyWordMotion pins Meta+B / Meta+F / Meta+D / Meta+Backspace. Before
// these bindings existed the ESC prefix was discarded and the trailing byte
// reached the text path, so Meta+B typed a literal "b".
func TestMetaKeyWordMotion(t *testing.T) {
	s := newTestAttachState("hello world foo")
	feedEscape(t, s, 0x1b, 'b')
	assert.Equal(t, len("hello world "), s.cursor)
	feedEscape(t, s, 0x1b, 'b')
	assert.Equal(t, len("hello "), s.cursor)
	feedEscape(t, s, 0x1b, 'f')
	assert.Equal(t, len("hello world"), s.cursor)
	assert.Equal(t, "hello world foo", string(s.lineBuf), "motion must not edit the buffer")

	s = newTestAttachState("hello world")
	feedEscape(t, s, 0x1b, 0x7f) // Meta+Backspace
	assert.Equal(t, "hello ", string(s.lineBuf))
	assert.Equal(t, len("hello "), s.cursor)

	s = newTestAttachState("hello world")
	s.cursor = len("hello ")
	feedEscape(t, s, 0x1b, 'd') // Meta+D
	assert.Equal(t, "hello ", string(s.lineBuf))
}

// TestCtrlWKillsWholeToken pins the deliberate difference from Meta+Backspace:
// Ctrl-W takes the whitespace-delimited token, as it does in bash.
func TestCtrlWKillsWholeToken(t *testing.T) {
	s := newTestAttachState("edit docs/agents.md")
	assert.True(t, handleAttachEditingKey(0x17, s))
	assert.Equal(t, "edit ", string(s.lineBuf))

	s = newTestAttachState("edit docs/agents.md")
	feedEscape(t, s, 0x1b, 0x7f)
	assert.Equal(t, "edit docs/agents.", string(s.lineBuf))
}

func TestAttachEditingControlKeys(t *testing.T) {
	s := newTestAttachState("hello world")
	assert.True(t, handleAttachEditingKey(0x01, s)) // Ctrl-A
	assert.Equal(t, 0, s.cursor)
	assert.True(t, handleAttachEditingKey(0x05, s)) // Ctrl-E
	assert.Equal(t, len("hello world"), s.cursor)

	s = newTestAttachState("hello world")
	s.cursor = len("hello ")
	assert.True(t, handleAttachEditingKey(0x0b, s)) // Ctrl-K
	assert.Equal(t, "hello ", string(s.lineBuf))

	s = newTestAttachState("hello world")
	s.cursor = len("hello ")
	assert.True(t, handleAttachEditingKey(0x15, s)) // Ctrl-U
	assert.Equal(t, "world", string(s.lineBuf))
	assert.Equal(t, 0, s.cursor)

	assert.False(t, handleAttachEditingKey('x', newTestAttachState("")),
		"printable bytes belong to the text path")
	assert.False(t, handleAttachEditingKey(0x03, newTestAttachState("")),
		"Ctrl-C must stay with the detach handler")
}

// TestCSIKeysDoNotLeak covers the CSI family: bound keys act, and unbound ones
// are swallowed instead of typing their final byte (Home used to insert "H",
// Delete used to insert "3~").
func TestCSIKeysDoNotLeak(t *testing.T) {
	s := newTestAttachState("hello")
	feedEscape(t, s, 0x1b, '[', 'H')
	assert.Equal(t, 0, s.cursor, "Home moves to the line start")
	feedEscape(t, s, 0x1b, '[', 'F')
	assert.Equal(t, len("hello"), s.cursor, "End moves to the line end")

	s = newTestAttachState("hello")
	s.cursor = 0
	feedEscape(t, s, 0x1b, '[', '3', '~') // Delete
	assert.Equal(t, "ello", string(s.lineBuf))

	s = newTestAttachState("hello world")
	feedEscape(t, s, 0x1b, '[', '1', ';', '5', 'D') // Ctrl+Left
	assert.Equal(t, len("hello "), s.cursor)
	feedEscape(t, s, 0x1b, '[', '1', ';', '3', 'C') // Alt+Right
	assert.Equal(t, len("hello world"), s.cursor)

	s = newTestAttachState("hello")
	feedEscape(t, s, 0x1b, '[', '2', '0', ';', '4', 'R') // unbound report
	assert.Equal(t, "hello", string(s.lineBuf), "an unbound CSI sequence must not type anything")
	assert.Equal(t, len("hello"), s.cursor)
}

// TestEscapeSequenceRegressions keeps the behavior that predates the parser
// rewrite: plain arrows, Alt+Enter composing a newline, and bracketed paste.
func TestEscapeSequenceRegressions(t *testing.T) {
	s := newTestAttachState("abc")
	feedEscape(t, s, 0x1b, '[', 'D')
	assert.Equal(t, 2, s.cursor, "Left moves one byte")
	feedEscape(t, s, 0x1b, 'O', 'C')
	assert.Equal(t, 3, s.cursor, "SS3 Right moves one byte")

	s = newTestAttachState("abc")
	feedEscape(t, s, 0x1b, 0x0d)
	assert.Equal(t, "abc\n", string(s.lineBuf), "Alt+Enter composes a multi-line message")

	s = newTestAttachState("")
	feedEscape(t, s, bracketedPasteStart...)
	assert.True(t, s.pasting)
}

func TestCSIModifierParsing(t *testing.T) {
	assert.Equal(t, 3, csiParam("3", 0))
	assert.Equal(t, 5, csiParam("1;5", 1))
	assert.Equal(t, 0, csiParam("1", 1), "absent parameter reads as 0")
	assert.Equal(t, 0, csiParam("", 0))

	assert.True(t, csiJumpsWord("1;5"), "ctrl")
	assert.True(t, csiJumpsWord("1;3"), "alt")
	assert.True(t, csiJumpsWord("1;7"), "ctrl+alt")
	assert.False(t, csiJumpsWord("1;2"), "shift alone is not word motion")
	assert.False(t, csiJumpsWord(""), "unmodified")
}

// TestDisplayInputBufferShowsNewlineMarker is a regression test: the
// flattened single-row prompt used to collapse an embedded newline to a
// plain space, which made Option/Alt+Enter look like a no-op until the
// message was actually submitted. It must render a visible marker instead.
func TestDisplayInputBufferShowsNewlineMarker(t *testing.T) {
	assert.Equal(t, "first line ↵ second line", displayInputBuffer([]byte("first line\nsecond line")))
	assert.Equal(t, "first line ↵ second line", displayInputBuffer([]byte("first line\r\nsecond line")))
	assert.Equal(t, "", displayInputBuffer(nil))
}

// TestHandleAttachEscapeSequenceOptionEnterInsertsNewline pins Option/Alt+
// Enter (ESC followed by CR, the standard meta-key encoding used by iTerm2
// and Terminal.app's default settings) as "insert a newline, don't submit" —
// so a multi-line message can be composed and sent as one input, the same
// way a pasted multi-line block already works.
func TestHandleAttachEscapeSequenceOptionEnterInsertsNewline(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			SendInput("sess", &godo.HostedAgentSendInputRequest{Text: "first line\nsecond line"}).
			Return(&godo.HostedAgentSendInputResponse{RunID: "run_1"}, nil)

		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		send := func(bs ...byte) {
			for _, b := range bs {
				stop, err := handleAttachByte(config, tm.hostedAgents, "sess", b, state, nil, nil)
				assert.NoError(t, err)
				assert.False(t, stop)
			}
		}
		send([]byte("first line")...)
		send(0x1b, 0x0d) // Option/Alt+Enter
		send([]byte("second line")...)
		assert.Equal(t, "first line\nsecond line", string(state.lineBuf))

		// A real Enter submits the whole multi-line buffer as one message.
		send(0x0d)
		assert.Equal(t, "", string(state.lineBuf))
	})
}

// TestHandleAttachEscapeSequenceOptionEnterIgnoredDuringHITL confirms
// Option/Alt+Enter is swallowed (not left to fall through as stray bytes)
// but does nothing while a HITL approval is pending — there's no text input
// to insert a newline into at that point.
func TestHandleAttachEscapeSequenceOptionEnterIgnoredDuringHITL(t *testing.T) {
	state := newAttachState(io.Discard, &pendingHITL{})
	state.display.setRaw(true)
	state.pending.set("req-1", "approve something")

	for _, b := range []byte{0x1b, 0x0d} {
		stop, err := handleAttachByte(nil, nil, "sess", b, state, nil, nil)
		assert.NoError(t, err)
		assert.False(t, stop)
	}
	assert.Equal(t, "", string(state.lineBuf))
}

// TestHandlePastedByteOnlyRedrawsOnceAtEnd is a regression test for a bug
// where every single pasted byte triggered its own full prompt repaint.
// paintPromptLocked only erases the terminal's *current* row (\x1b[K); once
// the growing pasted line got long enough to wrap, each of those hundreds of
// incremental repaints left the previous, shorter wrapped block behind as a
// stale duplicate instead of clearing it — turning one paste into dozens of
// near-identical truncated lines in the scrollback. Buffering silently and
// painting once when the paste ends fixes it.
func TestHandlePastedByteOnlyRedrawsOnceAtEnd(t *testing.T) {
	var buf bytes.Buffer
	state := newAttachState(&buf, &pendingHITL{})
	state.display.setRaw(true)

	pasted := strings.Repeat("x", 300)
	input := "\x1b[200~" + pasted + "\x1b[201~"
	for _, b := range []byte(input) {
		stop, err := handleAttachByte(nil, nil, "sess", b, state, nil, nil)
		assert.NoError(t, err)
		assert.False(t, stop)
	}

	assert.Equal(t, pasted, string(state.lineBuf))
	// The clear-and-repaint escape must appear exactly once — for the single
	// redraw when the paste ends — not once per pasted byte.
	assert.Equal(t, 1, strings.Count(buf.String(), "\x1b[K"))
}

// TestHandlePastedByteDuringWarmupDoesNotRepaintEveryByte is MARSOHS-1095:
// pasting while the attach warm-up banner is visible used to call noteQueued
// (and thus warmupPaintLocked) on every pasted byte, leaving the same
// truncated-duplicate scrollback corruption as a bare per-byte redraw.
func TestHandlePastedByteDuringWarmupDoesNotRepaintEveryByte(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	state := newAttachState(&buf, &pendingHITL{})
	state.display.setRaw(true)
	w := newWarmupState(state.display, now.Add(-30*time.Second))
	w.start()
	buf.Reset()

	pasted := strings.Repeat("x", 300)
	input := "\x1b[200~" + pasted + "\x1b[201~"
	for _, b := range []byte(input) {
		stop, err := handleAttachByte(nil, nil, "sess", b, state, w, nil)
		assert.NoError(t, err)
		assert.False(t, stop)
	}

	assert.Equal(t, pasted, string(state.lineBuf))
	assert.Contains(t, buf.String(), msgAgentWarmupQueued)
	// One paint to reveal the queued notice + one final paint when paste ends.
	// Hundreds of clears means the warm-up path is still rewriting every byte.
	clears := strings.Count(buf.String(), "\x1b[K")
	assert.LessOrEqual(t, clears, 12, "paste during warm-up repainted too often (%d clears)", clears)
	assert.Contains(t, buf.String(), pasted, "final prompt should include the full pasted line")
}

// TestPromptDisplayClearsAllWrappedRowsOnRedraw is the Enter/spinner half of
// MARSOHS-1095: a long prompt that wraps across several terminal rows must be
// fully erased before the next paint, otherwise only the last row is cleared
// and truncated copies of the line pile up in scrollback.
func TestPromptDisplayClearsAllWrappedRowsOnRedraw(t *testing.T) {
	var buf bytes.Buffer
	state := newAttachState(&buf, &pendingHITL{})
	state.display.setRaw(true)
	state.display.termCols = 40

	state.mu.Lock()
	state.lineBuf = []byte(strings.Repeat("x", 100))
	state.cursor = len(state.lineBuf)
	state.mu.Unlock()
	state.display.redraw()
	require.GreaterOrEqual(t, state.display.promptRows, 3)

	buf.Reset()
	state.mu.Lock()
	state.lineBuf = nil
	state.cursor = 0
	state.mu.Unlock()
	state.display.redraw()

	ups := strings.Count(buf.String(), "\x1b[A")
	assert.GreaterOrEqual(t, ups, 2, "expected cursor-up clears for wrapped rows, got %d in %q", ups, buf.String())
	assert.Equal(t, 1, state.display.promptRows)
}

// TestSpinnerFrameMovesAboveWrappedPrompt ensures the run-in-progress spinner
// climbs over every prompt row, not just one — otherwise each tick overwrites
// the middle of a wrapped paste and leaves truncated "> …" ghosts.
func TestSpinnerFrameMovesAboveWrappedPrompt(t *testing.T) {
	var buf bytes.Buffer
	state := newAttachState(&buf, &pendingHITL{})
	state.display.setRaw(true)
	state.display.termCols = 40

	state.mu.Lock()
	state.lineBuf = []byte(strings.Repeat("y", 90))
	state.cursor = len(state.lineBuf)
	state.mu.Unlock()
	state.display.redraw()
	rows := state.display.promptRows
	require.GreaterOrEqual(t, rows, 2)

	buf.Reset()
	state.display.spinnerFrame("⠋", "Run in progress")
	assert.Contains(t, buf.String(), fmt.Sprintf("\x1b[%dA", rows))
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

// TestHandleAttachByteCtrlCDuringRequest pins the reason attachLoopTTY runs API
// calls on a worker: raw mode makes Ctrl-C a byte the input loop has to read,
// so a request in flight must not be holding that loop hostage.
func TestHandleAttachByteCtrlCDuringRequest(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		inFlight := make(chan struct{})
		release := make(chan struct{})
		tm.hostedAgents.EXPECT().
			SendInput("sess", &godo.HostedAgentSendInputRequest{Text: "hi"}).
			DoAndReturn(func(string, *godo.HostedAgentSendInputRequest) (*godo.HostedAgentSendInputResponse, error) {
				close(inFlight)
				<-release
				return &godo.HostedAgentSendInputResponse{RunID: "run_1"}, nil
			})

		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)

		// Stand-in for attachLoopTTY's worker.
		work := make(chan func() (detach bool), 1)
		state.dispatch = func(fn func() (detach bool)) { work <- fn }
		workerDone := make(chan struct{})
		go func() {
			defer close(workerDone)
			for fn := range work {
				fn()
			}
		}()

		for _, b := range []byte("hi") {
			_, err := handleAttachByte(config, tm.hostedAgents, "sess", b, state, nil, nil)
			require.NoError(t, err)
		}
		stop, err := handleAttachByte(config, tm.hostedAgents, "sess", 0x0d, state, nil, nil)
		require.NoError(t, err)
		require.False(t, stop)

		<-inFlight

		stop, err = handleAttachByte(config, tm.hostedAgents, "sess", 0x03, state, nil, nil)
		require.NoError(t, err)
		assert.True(t, stop, "Ctrl-C must detach while a request is still in flight")

		close(release)
		close(work)
		<-workerDone
	})
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

// TestReasoningStreamer confirms the shared reasoning-block printer: a
// leading label appears once per block, stop/start bracket the thinking
// spinner around the block, end() is a no-op if nothing is active, and empty
// chunks never trigger the leading label on their own.
func TestReasoningStreamer(t *testing.T) {
	var buf bytes.Buffer
	thinking := newThinkingState(&buf)
	r := &reasoningStreamer{out: &buf, thinking: thinking}

	r.end() // no-op: nothing streamed yet
	assert.Equal(t, "", buf.String())

	r.stream("")
	assert.Equal(t, "", buf.String(), "empty chunks don't open a block")

	r.stream("thinking about it")
	r.stream(" some more")
	out := buf.String()
	assert.Contains(t, out, "reasoning")
	assert.Contains(t, out, "thinking about it some more")

	buf.Reset()
	r.end()
	// end() closes the block with a trailing newline, then resumes the
	// thinking spinner — for a non-promptDisplay writer that's a one-shot
	// "(Run in progress)" line.
	assert.Equal(t, "\n(Run in progress)\n", buf.String())
	thinking.stop()

	// A second end() without an intervening stream() is a no-op.
	buf.Reset()
	r.end()
	assert.Equal(t, "", buf.String())
}

// TestTokenChunkPayloadIsReasoning confirms the SPI TokenChunk.is_reasoning
// field round-trips through doctl's local payload struct, including the
// common case where a harness never sets it (defaults to false, i.e. final
// answer).
func TestTokenChunkPayloadIsReasoning(t *testing.T) {
	var p tokenChunkPayload
	require.NoError(t, json.Unmarshal([]byte(`{"text":"hmm","is_reasoning":true}`), &p))
	assert.True(t, p.IsReasoning)
	assert.Equal(t, "hmm", p.Text)

	var q tokenChunkPayload
	require.NoError(t, json.Unmarshal([]byte(`{"text":"answer"}`), &q))
	assert.False(t, q.IsReasoning)
}

// TestMsgAccumulatorPreviewTail confirms the bounded tail used for the live
// "thinking" preview tracks the most recent content and stays bounded
// regardless of how many small chunks built up the message, and that flush
// clears it for the next turn.
func TestMsgAccumulatorPreviewTail(t *testing.T) {
	acc := &msgAccumulator{}
	assert.Equal(t, "", acc.previewTail())

	acc.add("hello ")
	acc.add("world")
	assert.Equal(t, "hello world", acc.previewTail())

	// Feed enough chunks to exceed previewTailMaxRunes; only the tail survives.
	long := strings.Repeat("x", previewTailMaxRunes+50)
	acc.add(long)
	assert.Len(t, []rune(acc.previewTail()), previewTailMaxRunes)
	assert.True(t, strings.HasSuffix(acc.previewTail(), "xxxx"))

	var buf bytes.Buffer
	acc.flush(&buf)
	assert.Equal(t, "", acc.previewTail())
}

func TestTrimTailRunes(t *testing.T) {
	assert.Equal(t, "abc", trimTailRunes("abc", 5))
	assert.Equal(t, "abc", trimTailRunes("abc", 3))
	assert.Equal(t, "bc", trimTailRunes("abc", 2))
	assert.Equal(t, "", trimTailRunes("abc", 0))
	// Multi-byte runes must not be split.
	assert.Equal(t, "🎉b", trimTailRunes("a🎉b", 2))
}

func TestThinkingPreviewLabel(t *testing.T) {
	assert.Equal(t, "", thinkingPreviewLabel(""))
	assert.Equal(t, "", thinkingPreviewLabel("   \n\t "))
	assert.Equal(t, "hello world", thinkingPreviewLabel("hello  world"))
	// Embedded newlines collapse to single spaces so the label stays one line.
	assert.Equal(t, "line one line two", thinkingPreviewLabel("line one\nline two"))

	long := strings.Repeat("a", thinkingPreviewMaxRunes+20)
	got := thinkingPreviewLabel(long)
	assert.True(t, strings.HasPrefix(got, "…"))
	assert.Equal(t, thinkingPreviewMaxRunes+1, len([]rune(got))) // +1 for the ellipsis
}

// TestThinkingStateLivePreview confirms setLabel changes what the animated
// spinner shows (falling back to defaultThinkingLabel until the first
// preview arrives), and that a fresh start() clears any label left over from
// a previous turn.
func TestThinkingStateLivePreview(t *testing.T) {
	var buf bytes.Buffer
	pending := &pendingHITL{}
	s := newAttachState(&buf, pending)
	s.display.setRaw(true)

	thinking := newThinkingState(s.display)
	assert.Equal(t, defaultThinkingLabel, thinking.currentLabel())

	thinking.setLabel("some text stream")
	assert.Equal(t, "some text stream", thinking.currentLabel())

	thinking.start()
	defer thinking.stop()
	// start() resets any stale label from a prior turn.
	assert.Equal(t, defaultThinkingLabel, thinking.currentLabel())

	thinking.setLabel("…streaming preview")
	assert.Equal(t, "…streaming preview", thinking.currentLabel())
}

// TestThinkingStateTurnRunning confirms turnRunning is a distinct signal from
// active: it's what the input loop checks to warn a user their message will
// be queued behind an already-running turn, and it must not be perturbed by
// stop()/start() cycling for sub-turn events like tool calls.
func TestThinkingStateTurnRunning(t *testing.T) {
	var buf bytes.Buffer
	thinking := newThinkingState(&buf)
	assert.False(t, thinking.isTurnRunning())

	thinking.setTurnRunning(true)
	assert.True(t, thinking.isTurnRunning())

	// A tool call mid-turn stops/starts the spinner but the turn is still running.
	thinking.start()
	thinking.stop()
	assert.True(t, thinking.isTurnRunning())

	thinking.setTurnRunning(false)
	assert.False(t, thinking.isTurnRunning())
}

// TestPrintAttachSendAck pins the three messages a user can see after
// sending: queued behind warm-up, queued behind an already-running turn, or
// (the common case) just waiting for the first response. There's no
// server-side "queued" signal (see OHR queueing research — a queued send
// returns the same response as one that starts immediately), so the "queued
// behind a running turn" case is inferred entirely from local state.
func TestPrintAttachSendAck(t *testing.T) {
	t.Run("no warmup, no run active: generic waiting message", func(t *testing.T) {
		var buf bytes.Buffer
		printAttachSendAck(&buf, nil, nil)
		assert.Contains(t, buf.String(), "waiting for the agent")
	})

	t.Run("run already active: queued-behind-run message, not the generic one", func(t *testing.T) {
		var buf bytes.Buffer
		thinking := newThinkingState(&buf)
		thinking.setTurnRunning(true)
		printAttachSendAck(&buf, nil, thinking)
		assert.Contains(t, buf.String(), "queued")
		assert.NotContains(t, buf.String(), "waiting for the agent")
	})

	t.Run("run not active: generic waiting message even with thinking wired up", func(t *testing.T) {
		var buf bytes.Buffer
		thinking := newThinkingState(&buf)
		printAttachSendAck(&buf, nil, thinking)
		assert.Contains(t, buf.String(), "waiting for the agent")
	})

	t.Run("warm-up takes priority over run-active", func(t *testing.T) {
		var buf bytes.Buffer
		warmup := newWarmupState(&buf, time.Now())
		warmup.start()
		thinking := newThinkingState(&buf)
		thinking.setTurnRunning(true)
		printAttachSendAck(&buf, warmup, thinking)
		assert.Contains(t, buf.String(), "will send when agent is ready")
		assert.NotContains(t, buf.String(), "queued behind")
	})
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
			strings.NewReader("What is the capital of France?\n"), testAttachStateFromPending(nil), nil)
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "waiting for the agent")
	})
}

func TestAttachLoopBatchesRapidMultilineInput(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		prev := attachLineBatchWindow
		attachLineBatchWindow = 5 * time.Millisecond
		defer func() { attachLineBatchWindow = prev }()

		var buf bytes.Buffer
		config.Out = &buf
		tm.hostedAgents.EXPECT().
			SendInput("sess_x", &godo.HostedAgentSendInputRequest{Text: "Line A\nLine B\nLine C"}).
			Return(&godo.HostedAgentSendInputResponse{RunID: "run-abc"}, nil)

		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("Line A\nLine B\nLine C\n"), testAttachStateFromPending(nil), nil)
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "Detected rapid multiline input")
		assert.Contains(t, buf.String(), "waiting for the agent")
	})
}

func TestConfirmLargePasteLineMode(t *testing.T) {
	lines := make(chan attachLineRead, 1)
	lines <- attachLineRead{line: "y\n"}
	close(lines)
	var pending *attachLineRead
	var buf bytes.Buffer
	decision, err := confirmLargePasteLineMode(&buf, 6, lines, &pending)
	require.NoError(t, err)
	assert.Equal(t, largePasteSendTogether, decision)
	assert.Contains(t, buf.String(), "You pasted 6 lines. Send them together as one message?")
}

func TestConfirmLargePasteLineModeDefaultSeparate(t *testing.T) {
	lines := make(chan attachLineRead, 1)
	lines <- attachLineRead{line: "n\n"}
	close(lines)
	var pending *attachLineRead
	var buf bytes.Buffer
	decision, err := confirmLargePasteLineMode(&buf, 6, lines, &pending)
	require.NoError(t, err)
	assert.Equal(t, largePasteSendSeparately, decision)
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
					strings.NewReader(tc.input), testAttachStateFromPending(pending), nil)
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
			strings.NewReader("y\n"), testAttachStateFromPending(pending), nil)
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
			strings.NewReader("y\n"), testAttachStateFromPending(pending), nil)
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
			strings.NewReader("y\n"), testAttachStateFromPending(&pendingHITL{}), nil)
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
			strings.NewReader("y\nn\n"), testAttachStateFromPending(pending), nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, pending.len(), "both HITLs must be drained after two keystrokes")

		out := buf.String()
		assert.Contains(t, out, "[y/n/d] (2 pending) > ", "first prompt shows the multi-pending count")
		assert.Contains(t, out, "[y/n/d] > ", "after resolving one, prompt drops to plain HITL prompt")
	})
}

// TestHandleAttachByteTabCompletion covers Tab-completion of slash commands:
// a unique prefix fills in the verb, an ambiguous one lists candidates, and
// completion never fires once an argument has started.
func TestHandleAttachByteTabCompletion(t *testing.T) {
	typewrite := func(t *testing.T, state *attachState, s string) {
		t.Helper()
		for i := 0; i < len(s); i++ {
			stop, err := handleAttachByte(nil, nil, "sess", s[i], state, nil, nil)
			assert.NoError(t, err)
			assert.False(t, stop)
		}
	}

	t.Run("unique prefix completes with a trailing space", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		typewrite(t, state, "/pa")

		stop, err := handleAttachByte(nil, nil, "sess", 0x09, state, nil, nil)
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "/pause ", string(state.lineBuf))
		assert.Equal(t, len("/pause "), state.cursor)
	})

	t.Run("ambiguous prefix lists matches without altering the buffer", func(t *testing.T) {
		var buf bytes.Buffer
		state := newAttachState(&buf, &pendingHITL{})
		state.display.setRaw(true)
		typewrite(t, state, "/")

		config := &CmdConfig{Out: state.display}
		stop, err := handleAttachByte(config, nil, "sess", 0x09, state, nil, nil)
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "/", string(state.lineBuf))
		out := buf.String()
		assert.Contains(t, out, "/help")
		assert.Contains(t, out, "/download")
	})

	t.Run("/exit completes uniquely", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		typewrite(t, state, "/e")

		stop, err := handleAttachByte(nil, nil, "sess", 0x09, state, nil, nil)
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "/exit ", string(state.lineBuf))
	})

	t.Run("no match leaves the buffer untouched", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		typewrite(t, state, "/zz")

		stop, err := handleAttachByte(nil, nil, "sess", 0x09, state, nil, nil)
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "/zz", string(state.lineBuf))
	})

	t.Run("does not fire once an argument has started", func(t *testing.T) {
		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		typewrite(t, state, "/a foo")

		stop, err := handleAttachByte(nil, nil, "sess", 0x09, state, nil, nil)
		assert.NoError(t, err)
		assert.False(t, stop)
		assert.Equal(t, "/a foo", string(state.lineBuf))
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

// TestAttachLoopExitCommand: /exit in line mode (piped input) detaches like
// Ctrl-D / EOF — it prints the disconnect notice and returns without error,
// leaving the hosted session untouched.
func TestAttachLoopExitCommand(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf
		state := testAttachStateFromPending(nil)
		state.sessionRef = "my-session"

		err := attachLoop(config, config.HostedAgents(), "sess_x",
			strings.NewReader("/exit\n"), state, nil)
		assert.NoError(t, err)
		out := buf.String()
		assert.Contains(t, out, "Disconnected from session locally")
		assert.Contains(t, out, "still active in the cloud")
	})
}

// TestProcessAttachLineExitCommand covers the TTY-mode dispatch: /exit must
// report detach=true so the raw-mode loop stops reading bytes.
func TestProcessAttachLineExitCommand(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		var buf bytes.Buffer
		state := newAttachState(&buf, &pendingHITL{})
		state.sessionRef = "my-session"
		config.Out = state.display

		detach := processAttachLine(config, config.HostedAgents(), "sess_x", "/exit", state, nil, nil)
		assert.True(t, detach)
		assert.Contains(t, buf.String(), "Disconnected from session locally")
	})
}

// TestIsAttachExitCommand pins the exact-match contract: only a bare /exit
// counts, not other slash commands or /exit with trailing arguments.
func TestIsAttachExitCommand(t *testing.T) {
	assert.True(t, isAttachExitCommand("/exit"))
	assert.False(t, isAttachExitCommand("/exit now"))
	assert.False(t, isAttachExitCommand("/exiting"))
	assert.False(t, isAttachExitCommand("/help"))
	assert.False(t, isAttachExitCommand(""))
}

// TestHandleAttachCommandHelp exercises the /help card, and pins the
// tabwriter-aligned layout: every row's description column must start at the
// same offset within its section (hand-counted spaces used to drift out of
// alignment; tabwriter can't).
func TestHandleAttachCommandHelp(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf

		assert.NoError(t, handleAttachCommand(config, config.HostedAgents(), "sess_x", "/help", &pendingHITL{}))
		out := buf.String()

		assert.Contains(t, out, "Attach help")
		assert.Contains(t, out, "Detach (session keeps running)")
		assert.Contains(t, out, "/exit")
		assert.Contains(t, out, "Session controls")
		assert.Contains(t, out, "/pause")
		assert.Contains(t, out, "/upload")
		assert.Contains(t, out, "/download")
		assert.Contains(t, out, "Approvals pending")
		assert.Contains(t, out, "/pending")
		assert.Contains(t, out, "Tab")
		assert.Contains(t, out, "attach <session>")
		assert.Contains(t, out, "remove <session>")

		sectionLines := func(header string) []string {
			lines := strings.Split(out, "\n")
			var section []string
			started := false
			for _, l := range lines {
				if strings.Contains(l, header) {
					started = true
					continue
				}
				if !started {
					continue
				}
				if strings.TrimSpace(l) == "" {
					break
				}
				section = append(section, l)
			}
			return section
		}

		gapRE := regexp.MustCompile(`\S(\s{2,})\S`)
		descColumn := func(line string) int {
			// The first run of 2+ spaces between two non-space runs is the
			// gap tabwriter inserted before the description column; report
			// where the description text itself starts, in runes (tabwriter
			// aligns by rune count, but FindStringSubmatchIndex is byte-based
			// — some keys here contain multi-byte glyphs like "↑/↓").
			loc := gapRE.FindStringSubmatchIndex(line)
			require.NotNil(t, loc, "line %q has no column gap", line)
			return utf8.RuneCountInString(line[:loc[3]])
		}

		for _, header := range []string{"Session controls", "Approvals pending"} {
			lines := sectionLines(header)
			require.NotEmpty(t, lines, "section %q not found", header)
			want := descColumn(lines[0])
			for _, l := range lines[1:] {
				assert.Equal(t, want, descColumn(l), "misaligned row in %q section: %q", header, l)
			}
		}
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

// TestHandleAttachCommandPauseResume exercises the /pause and /resume verbs.
func TestHandleAttachCommandPauseResume(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var buf bytes.Buffer
		config.Out = &buf

		tm.hostedAgents.EXPECT().PauseSession("sess_x").Return(nil)
		assert.NoError(t, handleAttachCommand(config, config.HostedAgents(), "sess_x", "/pause", &pendingHITL{}))
		assert.Contains(t, buf.String(), "Session paused")

		buf.Reset()
		tm.hostedAgents.EXPECT().ResumeSession("sess_x").Return(nil)
		assert.NoError(t, handleAttachCommand(config, config.HostedAgents(), "sess_x", "/resume", &pendingHITL{}))
		assert.Contains(t, buf.String(), "Session resumed")
	})
}

// TestHandleAttachCommandUpload exercises the /upload verb, mirroring
// `doctl agents upload` but driven from the interactive attach REPL.
func TestHandleAttachCommandUpload(t *testing.T) {
	prevPoll := workspaceTransferPollInterval
	workspaceTransferPollInterval = 0
	defer func() { workspaceTransferPollInterval = prevPoll }()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "main.go")
	contents := []byte("package main\n\nfunc main() {}\n")
	assert.NoError(t, os.WriteFile(localPath, contents, 0o644))
	sha := sha256Hex(contents)

	partServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer partServer.Close()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectWorkspaceTransferUpload(t, tm, "sess_x", "src/main.go", int64(len(contents)), sha, false, partServer.URL)

		var buf bytes.Buffer
		config.Out = &buf
		line := fmt.Sprintf("/upload %s src/main.go", localPath)
		assert.NoError(t, handleAttachCommand(config, config.HostedAgents(), "sess_x", line, &pendingHITL{}))
		assert.Contains(t, buf.String(), "Upload complete")
	})
}

// TestHandleAttachCommandUpload_UsageError requires one or two positional args
// (workspace-path is optional — see TestHandleAttachCommandUpload_DefaultsWorkspacePath).
func TestHandleAttachCommandUpload_UsageError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		err := handleAttachCommand(config, config.HostedAgents(), "sess_x", "/upload", &pendingHITL{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "usage: /upload")

		err = handleAttachCommand(config, config.HostedAgents(), "sess_x", "/upload a b c", &pendingHITL{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "usage: /upload")
	})
}

// TestHandleAttachCommandUpload_DefaultsWorkspacePath confirms omitting
// workspace-path uploads to the local file's basename at the workspace root —
// like `curl -O`, retyping an identical filename twice is just noise.
func TestHandleAttachCommandUpload_DefaultsWorkspacePath(t *testing.T) {
	prevPoll := workspaceTransferPollInterval
	workspaceTransferPollInterval = 0
	defer func() { workspaceTransferPollInterval = prevPoll }()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "main.go")
	contents := []byte("package main\n\nfunc main() {}\n")
	assert.NoError(t, os.WriteFile(localPath, contents, 0o644))
	sha := sha256Hex(contents)

	partServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer partServer.Close()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// Only the basename ("main.go"), not the full local path, must be used
		// as the workspace destination.
		expectWorkspaceTransferUpload(t, tm, "sess_x", "main.go", int64(len(contents)), sha, false, partServer.URL)

		var buf bytes.Buffer
		config.Out = &buf
		line := fmt.Sprintf("/upload %s", localPath)
		assert.NoError(t, handleAttachCommand(config, config.HostedAgents(), "sess_x", line, &pendingHITL{}))
		assert.Contains(t, buf.String(), "Upload complete")
	})
}

// TestHandleAttachCommandDownload exercises the /download verb, mirroring
// `doctl agents download` but driven from the interactive attach REPL.
func TestHandleAttachCommandDownload(t *testing.T) {
	prevPoll := workspaceTransferPollInterval
	workspaceTransferPollInterval = 0
	defer func() { workspaceTransferPollInterval = prevPoll }()

	dir := t.TempDir()
	saveTo := filepath.Join(dir, "out.go")
	contents := []byte("package main\n\nfunc main() {}\n")
	wantSum := sha256Hex(contents)

	objServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(contents)
	}))
	defer objServer.Close()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateWorkspaceTransfer("sess_x", gomock.Any()).
			DoAndReturn(func(sessionID string, create *godo.HostedAgentWorkspaceTransferCreateRequest) (*godo.HostedAgentWorkspaceTransfer, error) {
				assert.Equal(t, godo.HostedAgentWorkspaceTransferDirectionDownload, create.Direction)
				assert.Equal(t, "src/main.go", create.Path)
				return &godo.HostedAgentWorkspaceTransfer{
					TransferID: "xfer_dl",
					Status:     godo.HostedAgentWorkspaceTransferStatusPending,
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetWorkspaceTransfer("sess_x", "xfer_dl").
			Return(&godo.HostedAgentWorkspaceTransfer{
				TransferID:  "xfer_dl",
				Status:      godo.HostedAgentWorkspaceTransferStatusCompleted,
				SHA256:      wantSum,
				DownloadURL: objServer.URL,
			}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		line := fmt.Sprintf("/download src/main.go %s", saveTo)
		assert.NoError(t, handleAttachCommand(config, config.HostedAgents(), "sess_x", line, &pendingHITL{}))
		assert.Contains(t, buf.String(), "Downloaded")

		got, err := os.ReadFile(saveTo)
		assert.NoError(t, err)
		assert.Equal(t, contents, got)
	})
}

// TestHandleAttachCommandDownload_UsageError requires one or two positional
// args (local-file is optional — see TestHandleAttachCommandDownload_DefaultsLocalFile).
func TestHandleAttachCommandDownload_UsageError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		err := handleAttachCommand(config, config.HostedAgents(), "sess_x", "/download", &pendingHITL{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "usage: /download")

		err = handleAttachCommand(config, config.HostedAgents(), "sess_x", "/download a b c", &pendingHITL{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "usage: /download")
	})
}

// TestHandleAttachCommandDownload_DefaultsLocalFile reproduces the reported
// devex bug: `/download /workspace/Plano-One-Pager.pdf` (one positional arg)
// used to fail with a usage error even though the intent — save it under
// that same name in the current directory — was unambiguous. Like `curl -O`,
// omitting local-file now defaults to the workspace path's basename.
func TestHandleAttachCommandDownload_DefaultsLocalFile(t *testing.T) {
	prevPoll := workspaceTransferPollInterval
	workspaceTransferPollInterval = 0
	defer func() { workspaceTransferPollInterval = prevPoll }()

	t.Chdir(t.TempDir())

	contents := []byte("%PDF-1.4 fake one-pager\n")
	wantSum := sha256Hex(contents)

	objServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(contents)
	}))
	defer objServer.Close()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateWorkspaceTransfer("sess_x", gomock.Any()).
			DoAndReturn(func(sessionID string, create *godo.HostedAgentWorkspaceTransferCreateRequest) (*godo.HostedAgentWorkspaceTransfer, error) {
				assert.Equal(t, "/workspace/Plano-One-Pager.pdf", create.Path)
				return &godo.HostedAgentWorkspaceTransfer{
					TransferID: "xfer_dl2",
					Status:     godo.HostedAgentWorkspaceTransferStatusPending,
				}, nil
			})
		tm.hostedAgents.EXPECT().
			GetWorkspaceTransfer("sess_x", "xfer_dl2").
			Return(&godo.HostedAgentWorkspaceTransfer{
				TransferID:  "xfer_dl2",
				Status:      godo.HostedAgentWorkspaceTransferStatusCompleted,
				SHA256:      wantSum,
				DownloadURL: objServer.URL,
			}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		err := handleAttachCommand(config, config.HostedAgents(), "sess_x", "/download /workspace/Plano-One-Pager.pdf", &pendingHITL{})
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "Downloaded")

		got, err := os.ReadFile("Plano-One-Pager.pdf")
		assert.NoError(t, err)
		assert.Equal(t, contents, got)
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
		assert.Equal(t, "\x1b7\x1b[1A\r\x1b[K⠋ thinking...\x1b8", buf.String())
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

	t.Run("warmupInit reserves spinner and queued rows above the prompt", func(t *testing.T) {
		var buf bytes.Buffer
		pending := &pendingHITL{}
		s := newAttachState(&buf, pending)
		s.display.setRaw(true)

		s.display.warmupInit("⠋", msgAgentWarmup)
		assert.Contains(t, buf.String(), msgAgentWarmup)
		assert.Contains(t, buf.String(), "> ")

		buf.Reset()
		s.display.warmupSetPhase("provisioning sandbox")
		assert.Contains(t, buf.String(), "provisioning sandbox")
		assert.Contains(t, buf.String(), "> ")

		buf.Reset()
		s.display.warmupSetQueued(msgAgentWarmupQueued)
		assert.Contains(t, buf.String(), msgAgentWarmupQueued)
		assert.Contains(t, buf.String(), "> ")
		assert.Equal(t, 3, s.display.warmupStatusLines)
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

// TestRunAgentsLaunch_AttachAuthFailure: a 401 from pre-attach GetSession surfaces
// a styled agentPrettyError, not the raw godo METHOD/URL dump.
func TestRunAgentsLaunch_AttachAuthFailure(t *testing.T) {
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
		err := RunAgentsLaunch(config)
		assert.Error(t, err)
		var pretty *agentPrettyError
		require.True(t, errors.As(err, &pretty))
		assert.Equal(t, "Authentication failed", pretty.title)
		assert.Contains(t, pretty.tips, "doctl auth init")
		display := pretty.DisplayError()
		assert.Contains(t, display, "Authentication failed")
		assert.NotContains(t, display, "GET http://")
	})
}

// TestRunAgentsLaunch_AttachTerminalSession: attach must fail fast (no banner, no
// interactive loop) when the session is already destroyed/destroying/failed,
// instead of connecting and only failing once the user sends input.
func TestRunAgentsLaunch_AttachTerminalSession(t *testing.T) {
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
				err := RunAgentsLaunch(config)
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
			wantSubstr:  "Session already attached elsewhere",
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

// TestRenderApprovalResolvedLine confirms the resolved line shares
// renderApprovalLine's exact "status  ·  label  ·  id" structure (just with a
// different status), for every outcome, and degrades to omitting the label
// segment (rather than "action pending", which no longer applies once
// resolved) when no label was ever captured.
func TestRenderApprovalResolvedLine(t *testing.T) {
	cases := []struct {
		outcome int32
		want    string
	}{
		{1, "Approved"},
		{2, "Rejected"},
		{3, "Deferred"},
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		renderApprovalResolvedLine(&buf, "hitl-123", "tool_permission", tc.outcome)
		out := buf.String()
		assert.Contains(t, out, tc.want)
		assert.Contains(t, out, "tool_permission")
		assert.Contains(t, out, "hitl-123")
	}

	var buf bytes.Buffer
	renderApprovalResolvedLine(&buf, "hitl-123", "", 1)
	assert.NotContains(t, buf.String(), "action pending")
	assert.Contains(t, buf.String(), "hitl-123")
}

// TestDrainStream_HITLResolvedReprintsRequestLabel reproduces a devex bug: the
// resolution of a HITL approval used to print a disconnected "<id> approve"
// line with no relation to the "● Approval required · <label> · <id>" line
// above it, leaving that line looking stale. The resolution must now reuse
// the same label and read as that line updating (status  ·  label  ·  id),
// not a brand new unrelated entry.
func TestDrainStream_HITLResolvedReprintsRequestLabel(t *testing.T) {
	const hitlID = "01a025a0-e622-7153-a23d-d18b4dde462f"
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindRunStarted), `{"agent":"claude-code"}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindHITLRequested), fmt.Sprintf(`{"hitl_id":%q,"action":"tool_permission"}`, hitlID)) +
		// No paired tool_call_started arrives for this generic permission
		// check; the next discrete event flushes the unpaired approval line.
		sseFrame("evt-3", string(godo.HostedAgentEventKindSessionUpdated), `{}`) +
		sseFrame("evt-4", string(godo.HostedAgentEventKindHITLResolved), fmt.Sprintf(`{"hitl_id":%q,"outcome":1}`, hitlID))
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil, &tokenDeduper{})

	out := buf.String()
	assert.Contains(t, out, "Approval required")
	assert.Contains(t, out, "tool_permission")
	assert.Contains(t, out, "Approved")
	// The resolved line must carry the same label forward and use the id — but
	// never as the old bare "<id> approve" pairing with no shared structure.
	assert.NotContains(t, out, hitlID+" approve")
	idx := strings.Index(out, "Approved")
	require.GreaterOrEqual(t, idx, 0)
	lineEnd := idx + strings.Index(out[idx:], "\n")
	approvedLine := out[idx:lineEnd]
	assert.Contains(t, approvedLine, "tool_permission")
	assert.Contains(t, approvedLine, hitlID)
}

func TestMsgAccumulatorStreamLive(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	defer func() { stylingEnabled = prev }()

	var buf bytes.Buffer
	acc := &msgAccumulator{}
	acc.streamLive(&buf, "Hello ")
	acc.streamLive(&buf, "world")
	assert.Equal(t, "Hello world", buf.String())

	acc.flush(&buf)
	assert.Equal(t, "Hello world\n", buf.String(), "flush must only seal the line, not reprint")

	buf.Reset()
	acc.add("buffered only")
	acc.flush(&buf)
	assert.Equal(t, "buffered only\n", buf.String())
}

func TestDrainStream_TokenChunksStreamLive(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindRunStarted), `{"agent":"claude-code"}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindTokenChunk), `{"text":"Let me look "}`) +
		sseFrame("evt-3", string(godo.HostedAgentEventKindTokenChunk), `{"text":"at the file."}`) +
		sseFrame("evt-4", string(godo.HostedAgentEventKindRunCompleted), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	thinking := newThinkingState(&buf)
	drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, thinking, nil, &tokenDeduper{})

	out := buf.String()
	assert.Contains(t, out, "Let me look at the file.")
	assert.Equal(t, 1, strings.Count(out, "Let me look at the file."))
}

// TestDrainStream_RunStaysStickyThroughToolCall pins that the "Run in
// progress" spinner restarts after a discrete line (like a tool-call start)
// prints, instead of staying off for the remainder of the run — a tool call
// can take many seconds with no other output, and without this the run would
// look identical to a hung session. Uses a *promptDisplay-backed thinking so
// isSticky() is true and the restart actually takes effect.
func TestDrainStream_RunStaysStickyThroughToolCall(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindRunStarted), `{"agent":"claude-code"}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindToolCallStarted), `{"tool_call_id":"t1","name":"bash"}`) +
		sseFrame("evt-3", string(godo.HostedAgentEventKindRunCompleted), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	pending := &pendingHITL{}
	state := newAttachState(&buf, pending)
	state.display.setRaw(true)
	thinking := newThinkingState(state.display)
	require.True(t, thinking.isSticky())

	drainStream(stream, state.display, pending, &eventCursor{}, thinking, nil, &tokenDeduper{})

	out := buf.String()
	// The run ended with the call still in flight, so its line commits with no
	// result rather than disappearing.
	assert.Contains(t, out, "▸ ")
	assert.Contains(t, out, "no result")
	// The spinner came back after the tool call instead of staying dark for the
	// rest of the run — captioned with the command, which is what an in-flight
	// deferred call uses the spinner for.
	assert.Contains(t, out, spinnerFrames[0]+" bash",
		"spinner should restart carrying the in-flight command as its caption")
}

// TestDrainStream_ToolCallRendersAsOneLine pins the merged tool line: a start
// paired with its completion commits exactly one "▸ cmd  ✓ …" row, with no
// separate result line and no dead row between them.
func TestDrainStream_ToolCallRendersAsOneLine(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindRunStarted), `{"agent":"codex"}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindToolCallStarted),
			`{"tool_call_id":"t1","name":"bash","arguments":{"command":"/bin/bash -lc \"wc -l /workspace/styles.css\""}}`) +
		sseFrame("evt-3", string(godo.HostedAgentEventKindToolCallCompleted),
			`{"tool_call_id":"t1","ok":true,"duration_ms":12,"summary":"482 styles.css"}`) +
		sseFrame("evt-4", string(godo.HostedAgentEventKindRunCompleted), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var buf bytes.Buffer
	pending := &pendingHITL{}
	state := newAttachState(&buf, pending)
	state.display.setRaw(true)
	thinking := newThinkingState(state.display)
	require.True(t, thinking.isSticky())

	drainStream(stream, state.display, pending, &eventCursor{}, thinking, nil, &tokenDeduper{})

	out := buf.String()
	assert.Contains(t, out, "▸ wc -l styles.css  ✓ 482 styles.css · 12ms")
	assert.Equal(t, 1, strings.Count(out, "▸ "), "the pair must commit exactly one line")
}

// TestDrainStream_ToolCallCompletionWithoutStartStandsAlone pins the reattach
// path: the server replays a completion whose start we never saw, so there's no
// command to merge it into and it has to render on its own rather than vanish.
func TestDrainStream_ToolCallCompletionWithoutStartStandsAlone(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindToolCallCompleted),
		`{"tool_call_id":"t9","ok":true,"summary":"ran ls"}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var buf bytes.Buffer
	drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil, &tokenDeduper{})

	assert.Contains(t, buf.String(), "  ✓ ran ls")
}

// TestDrainStream_ToolCallNotDeferredWithoutSpinner pins that piped output (no
// sticky spinner to carry the command) keeps the two-line shape: the start has
// to print when it happens, since nothing else would show what is running.
func TestDrainStream_ToolCallNotDeferredWithoutSpinner(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindToolCallStarted),
		`{"tool_call_id":"t1","name":"bash","arguments":{"command":"ls -la"}}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindToolCallCompleted),
			`{"tool_call_id":"t1","ok":true,"summary":"4 files"}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var buf bytes.Buffer
	thinking := newThinkingState(&buf)
	require.False(t, thinking.isSticky())
	drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, thinking, nil, &tokenDeduper{})

	out := buf.String()
	assert.Contains(t, out, "▸ ls -la")
	assert.Contains(t, out, "  ✓ 4 files")
}

func TestDrainStream_ReasoningTokensStreamDistinctlyFromFinalAnswer(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindRunStarted), `{"agent":"claude-code"}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindTokenChunk), `{"text":"Let me think... ","is_reasoning":true}`) +
		sseFrame("evt-3", string(godo.HostedAgentEventKindTokenChunk), `{"text":"this looks right.","is_reasoning":true}`) +
		sseFrame("evt-4", string(godo.HostedAgentEventKindTokenChunk), `{"text":"The answer is 42.","is_reasoning":false}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	thinking := newThinkingState(&buf)
	drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, thinking, nil, &tokenDeduper{})

	out := buf.String()
	assert.Contains(t, out, "reasoning")
	assert.Contains(t, out, "Let me think... this looks right.")
	assert.Contains(t, out, "The answer is 42.")
	assert.Equal(t, 1, strings.Count(out, "The answer is 42."))
	assert.NotEqual(t, "The answer is 42.", thinking.currentLabel())
}

func TestDrainStream_ReasoningToAnswerTransitionDoesNotFlashDefaultLabel(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindRunStarted), `{"agent":"claude-code"}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindTokenChunk), `{"text":"Let me think... ","is_reasoning":true}`) +
		sseFrame("evt-3", string(godo.HostedAgentEventKindTokenChunk), `{"text":"The answer is 42.","is_reasoning":false}`) +
		sseFrame("evt-4", string(godo.HostedAgentEventKindRunCompleted), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	pending := &pendingHITL{}
	state := newAttachState(&buf, pending)
	state.display.setRaw(true)
	thinking := newThinkingState(state.display)
	require.True(t, thinking.isSticky())

	drainStream(stream, state.display, pending, &eventCursor{}, thinking, nil, &tokenDeduper{})

	out := buf.String()
	assert.Equal(t, 1, strings.Count(out, defaultThinkingLabel))
	assert.Contains(t, out, "The answer is 42.")
}

// TestDrainStream_StickySpinnerDoesNotInterruptReasoningStream is a
// regression test for a bug where the sticky "Run in progress" restart (see
// TestDrainStream_RunStaysStickyThroughToolCall) fired after every single
// TokenChunk event, including the reasoning deltas that reasoningStreamer
// deliberately leaves the spinner stopped for across a whole block. That
// reinit landed mid-line, injecting a stray "Run in progress" line (and a
// spurious newline) into the middle of the streamed reasoning text. Uses a
// *promptDisplay-backed thinking (isSticky() true) — the bug never reproduced
// against a plain io.Writer because the sticky restart is skipped when
// !isSticky().
func TestDrainStream_StickySpinnerDoesNotInterruptReasoningStream(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindRunStarted), `{"agent":"claude-code"}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindTokenChunk), `{"text":"Let me think... ","is_reasoning":true}`) +
		sseFrame("evt-3", string(godo.HostedAgentEventKindTokenChunk), `{"text":"this looks right.","is_reasoning":true}`) +
		sseFrame("evt-4", string(godo.HostedAgentEventKindTokenChunk), `{"text":"The answer is 42.","is_reasoning":false}`) +
		sseFrame("evt-5", string(godo.HostedAgentEventKindRunCompleted), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	pending := &pendingHITL{}
	state := newAttachState(&buf, pending)
	state.display.setRaw(true)
	thinking := newThinkingState(state.display)
	require.True(t, thinking.isSticky())

	drainStream(stream, state.display, pending, &eventCursor{}, thinking, nil, &tokenDeduper{})

	out := buf.String()
	// The reasoning text must stay contiguous across both chunks. The bug
	// this pins: the sticky restart used to fire after the first reasoning
	// chunk too (thinking.active goes false the moment reasoningStreamer
	// pauses it for the block), reiniting the spinner — and forcing a
	// newline to do it, since the write landed mid-line — right between
	// "Let me think... " and "this looks right.".
	assert.Contains(t, out, "Let me think... this looks right.")
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

func TestWarmupState_noteQueuedUpdatesNotice(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	state := newAttachState(&buf, &pendingHITL{})
	state.display.setRaw(true)
	w := newWarmupState(state.display, now.Add(-30*time.Second))

	w.start()
	assert.Contains(t, buf.String(), msgAgentWarmup)
	assert.NotContains(t, buf.String(), msgAgentWarmupQueued)

	buf.Reset()
	w.noteQueued()
	assert.Contains(t, buf.String(), msgAgentWarmupQueued)

	// Further typing keeps the prompt in sync with the warm-up block.
	buf.Reset()
	stop, err := handleAttachByte(nil, nil, "sess", 'i', state, w, nil)
	assert.NoError(t, err)
	assert.False(t, stop)
	assert.Contains(t, buf.String(), "> i")
}

// Every message typed during warm-up is sent and counted: the guest queues
// turns in order once it accepts input, so dropping later lines only lost the
// user's typing.
func TestWarmupState_queuesEverySubmittedMessage(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		for _, text := range []string{"hello", "hello again"} {
			tm.hostedAgents.EXPECT().
				SendInput("sess", &godo.HostedAgentSendInputRequest{Text: text}).
				Return(&godo.HostedAgentSendInputResponse{RunID: "run_" + text}, nil)
		}

		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		w := newWarmupState(state.display, now.Add(-30*time.Second))
		w.start()

		typeLine := func(text string) {
			for _, b := range append([]byte(text), 0x0d) {
				stop, err := handleAttachByte(config, tm.hostedAgents, "sess", b, state, w, nil)
				assert.NoError(t, err)
				assert.False(t, stop)
			}
		}

		typeLine("hello")
		assert.Equal(t, 1, w.queuedMessages())

		typeLine("hello again")
		assert.Equal(t, 2, w.queuedMessages())
	})
}

// A failed send must hand the text back rather than swallowing it.
func TestWarmupState_failedSendRestoresInput(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			SendInput("sess", &godo.HostedAgentSendInputRequest{Text: "hello"}).
			Return(nil, errors.New("boom"))

		state := newAttachState(io.Discard, &pendingHITL{})
		state.display.setRaw(true)
		w := newWarmupState(state.display, now.Add(-30*time.Second))
		w.start()

		for _, b := range append([]byte("hello"), 0x0d) {
			_, err := handleAttachByte(config, tm.hostedAgents, "sess", b, state, w, nil)
			assert.NoError(t, err)
		}

		assert.Equal(t, 0, w.queuedMessages())
		state.mu.Lock()
		defer state.mu.Unlock()
		assert.Equal(t, "hello", string(state.lineBuf), "failed send should restore the typed text")
		assert.Equal(t, len("hello"), state.cursor)
	})
}

func TestWarmupQueuedLabel(t *testing.T) {
	assert.Equal(t, msgAgentWarmupQueued, warmupQueuedLabel(0))
	assert.Equal(t, "1 message queued until agent is ready", warmupQueuedLabel(1))
	assert.Equal(t, "3 messages queued until agent is ready", warmupQueuedLabel(3))
}

func TestHandleOpenAIAttachByte_warmupQueuedNotice(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	state := newAttachState(&buf, &pendingHITL{})
	state.display.setRaw(true)
	w := newWarmupState(state.display, now.Add(-30*time.Second))
	w.start()

	stop, err := handleOpenAIAttachByte(nil, context.Background(), nil, "", "sess_openai", '4', state, nil, w)
	assert.NoError(t, err)
	assert.False(t, stop)
	assert.Contains(t, buf.String(), msgAgentWarmupQueued)
	assert.Contains(t, buf.String(), "> 4")
}

func TestWarmupState_markInputQueuedDismissesBanner(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	state := newAttachState(&buf, &pendingHITL{})
	state.display.setRaw(true)
	w := newWarmupState(state.display, now.Add(-30*time.Second))
	w.start()
	assert.True(t, w.isBannerVisible())

	w.markInputQueued()
	assert.Equal(t, 1, w.queuedMessages())
	assert.True(t, w.isBannerVisible())
	assert.True(t, w.isActive())
	assert.Contains(t, buf.String(), warmupQueuedLabel(1))
}

func TestWarmupState_clearsOnTimeout(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	w := newWarmupState(&buf, now)
	w.timeout = 20 * time.Millisecond
	w.start()
	assert.Contains(t, buf.String(), msgAgentWarmup)

	require.Eventually(t, func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()
		return !w.active && w.dismissed
	}, time.Second, 5*time.Millisecond)

	assert.NotContains(t, buf.String(), "You can type anytime")
}

func TestWarmupState_startAfterClearIsNoop(t *testing.T) {
	oldClock := warmupClock
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	warmupClock = func() time.Time { return now }
	t.Cleanup(func() { warmupClock = oldClock })

	var buf bytes.Buffer
	w := newWarmupState(&buf, now)
	w.start()
	w.clear()
	buf.Reset()
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
	assert.Contains(t, buf.String(), "syncing session")
	assert.NotContains(t, buf.String(), "• session updated")
	warmup.clear()
}

func TestDrainStream_sandboxAllocatedUpdatesWarmup(t *testing.T) {
	evt := sseFrame("e1", string(godo.HostedAgentEventKindRunSandboxAllocated), `{}`)
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
	assert.True(t, warmup.active, "sandbox_allocated must not dismiss warm-up")
	assert.Contains(t, buf.String(), "sandbox allocated")
	warmup.clear()
}

// TestDrainStream_runLogDoesNotSplitStreamedMessage pins that a run.log
// arriving between two token chunks leaves the buffered message intact. The
// runtime interleaves run.log with token deltas and renderEvent prints nothing
// for it, so flushing on it used to cut one sentence into two separately
// rendered blocks with blank lines where the invisible event was.
func TestDrainStream_runLogDoesNotSplitStreamedMessage(t *testing.T) {
	body := sseFrame("e1", string(godo.HostedAgentEventKindRunStarted), `{"agent":"codex"}`) +
		sseFrame("e2", string(godo.HostedAgentEventKindTokenChunk), `{"text":"The theme is now quieter"}`) +
		sseFrame("e3", string(godo.HostedAgentEventKindRunLog), `{"level":"info","message":"patch applied"}`) +
		sseFrame("e4", string(godo.HostedAgentEventKindTokenChunk), `{"text":" and the hero name is smaller."}`) +
		sseFrame("e5", string(godo.HostedAgentEventKindRunCompleted), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	assert.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	cursor := &eventCursor{}
	drainStream(stream, &buf, &pendingHITL{}, cursor, newThinkingState(&buf), nil, &tokenDeduper{})

	assert.Contains(t, buf.String(), "The theme is now quieter and the hero name is smaller.")
	assert.Equal(t, "e5", cursor.get(), "run.log must still advance the resume cursor")
}

// TestMsgAccumulatorWhitespaceOnlyFlushIsSilent pins that flushing a buffer
// holding only whitespace writes nothing. Mid-message flushes routinely catch
// the buffer between paragraphs, and spending the block's leading and trailing
// newlines on it piles up blank lines.
func TestMsgAccumulatorWhitespaceOnlyFlushIsSilent(t *testing.T) {
	old := stylingEnabled
	stylingEnabled = true
	t.Cleanup(func() { stylingEnabled = old })

	var buf bytes.Buffer
	acc := &msgAccumulator{}
	acc.add("\n\n")
	acc.flush(&buf)

	assert.Empty(t, buf.String())
}

func TestPrettyCommandLabel(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain command untouched", "ls -la", "ls -la"},
		{"strips bash -lc and single quotes", `/bin/bash -lc 'ls -la /workspace'`, "ls -la /workspace"},
		{"strips double quotes", `/bin/bash -lc "wc -c index.html"`, "wc -c index.html"},
		{"unescapes inner double quotes", `/bin/bash -lc "rg -n \"h1|clamp\" styles.css"`, `rg -n "h1|clamp" styles.css`},
		{"keeps inner single quotes", `/bin/bash -lc "sed -n '1,260p' styles.css"`, `sed -n '1,260p' styles.css`},
		{"handles sh -c", `sh -c 'echo hi'`, "echo hi"},
		{"handles zsh -ic", `/usr/bin/zsh -ic 'echo hi'`, "echo hi"},
		{"drops workspace prefix", `/bin/bash -lc "sed -n '1,9p' /workspace/.agents/skills/x/SKILL.md"`, `sed -n '1,9p' .agents/skills/x/SKILL.md`},
		{"keeps bare workspace", `/bin/bash -lc 'ls -la /workspace'`, "ls -la /workspace"},
		{"collapses newlines", "echo one\n\necho two", "echo one echo two"},
		{"leaves several quoted words alone", `/bin/bash -lc 'a' 'b'`, `'a' 'b'`},
		{"tolerates missing command after wrapper", "/bin/bash -lc", "/bin/bash -lc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, prettyCommandLabel(tt.in))
		})
	}
}

// TestToolCallStartedCommandLineShowsSubject pins that a tool which runs
// neither a command nor a file edit still says what it was asked to do. Plano
// forwards an MCP call's arguments verbatim, so the query is all there is to go
// on — a bare tool name leaves you unable to tell what the agent searched for.
func TestToolCallStartedCommandLineShowsSubject(t *testing.T) {
	tests := []struct {
		name string
		p    toolCallStartedPayload
		want string
	}{
		{
			"mcp query",
			toolCallStartedPayload{Name: "web_search", Arguments: json.RawMessage(`{"query":"latest news this week"}`)},
			"web_search latest news this week",
		},
		{
			"url argument",
			toolCallStartedPayload{Name: "fetch", Arguments: json.RawMessage(`{"url":"https://example.com/a"}`)},
			"fetch https://example.com/a",
		},
		{
			"nested arguments",
			toolCallStartedPayload{Name: "search", Input: json.RawMessage(`{"params":{"pattern":"TODO"}}`)},
			"search TODO",
		},
		{
			"command still wins over subject",
			toolCallStartedPayload{Name: "bash", Arguments: json.RawMessage(`{"command":"ls","query":"ignored"}`)},
			"ls",
		},
		{
			"bare name when nothing else is known",
			toolCallStartedPayload{Name: "web_search", Arguments: json.RawMessage(`{"opaque":{"n":1}}`)},
			"web_search",
		},
		{
			"placeholder when even the name is missing",
			toolCallStartedPayload{},
			"tool call",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.p.commandLine())
		})
	}
}

func TestToolCallStartedCommandLineUnwrapsShell(t *testing.T) {
	p := toolCallStartedPayload{
		Name:      "bash",
		Arguments: json.RawMessage(`{"command":"/bin/bash -lc \"wc -c /workspace/index.html\""}`),
	}
	assert.Equal(t, "wc -c index.html", p.commandLine())
}

// TestToolResultSuffix pins that the suffix reports only what the adapter
// actually sent, and that its width excludes ANSI escapes so the command
// budget in renderToolLine isn't computed against invisible bytes.
func TestToolResultSuffix(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	tests := []struct {
		name      string
		payload   toolCallCompletedPayload
		want      string
		wantWidth int
	}{
		{"bare mark when nothing reported", toolCallCompletedPayload{OK: true}, "✓", 1},
		{"failure mark", toolCallCompletedPayload{OK: false}, "✗", 1},
		{"duration only", toolCallCompletedPayload{OK: true, DurationMS: 12}, "✓ 12ms", 6},
		{"summary only", toolCallCompletedPayload{OK: true, Summary: "4 files"}, "✓ 4 files", 9},
		{"both", toolCallCompletedPayload{OK: true, Summary: "4 files", DurationMS: 12}, "✓ 4 files · 12ms", 16},
		{"first line only", toolCallCompletedPayload{OK: true, Summary: "HTTP/2 200\nserver: nginx"}, "✓ HTTP/2 200", 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, width := tt.payload.resultSuffix()
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantWidth, width)
		})
	}
}

// TestRenderToolLineTruncatesToWidth pins that a long command is cut to fit
// rather than wrapping, which would break the ▸/✓ column.
func TestRenderToolLineTruncatesToWidth(t *testing.T) {
	prev := stylingEnabled
	stylingEnabled = false
	t.Cleanup(func() { stylingEnabled = prev })

	var buf bytes.Buffer
	renderToolLine(&buf, strings.Repeat("x", 400), "✓ 12ms", 6)

	line := strings.Trim(buf.String(), "\n")
	assert.LessOrEqual(t, utf8.RuneCountInString(line), mdWrapWidth())
	assert.True(t, strings.HasSuffix(line, "…  ✓ 12ms"), "got %q", line)
}

func TestBackendPhaseFromStatus(t *testing.T) {
	assert.Equal(t, "provisioning sandbox", backendPhaseFromStatus(godo.HostedAgentSessionStatusProvisioning))
	assert.Equal(t, "sandbox ready · starting agent", backendPhaseFromStatus(godo.HostedAgentSessionStatusReady))
}

func TestSessionUpdatedPhase(t *testing.T) {
	assert.Equal(t, "provisioning sandbox", sessionUpdatedPhase([]byte(`{"status":"SESSION_STATUS_PROVISIONING"}`)))
	assert.Equal(t, "cloning repo", sessionUpdatedPhase([]byte(`{"message":"cloning repo"}`)))
}
