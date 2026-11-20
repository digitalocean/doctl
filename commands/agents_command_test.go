package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/digitalocean/doctl"
)

// parseCreateArgs runs args through a cobra flag set the way the real command
// line is parsed, so the position of `--` is recorded, and mirrors each string
// flag that was set into config.Doit, where the resolver reads values from.
func parseCreateArgs(t *testing.T, config *CmdConfig, args []string, stringFlags ...string) {
	t.Helper()
	cmd := &cobra.Command{Use: "create"}
	for _, f := range stringFlags {
		cmd.Flags().String(f, "", "")
	}
	require.NoError(t, cmd.Flags().Parse(args))
	config.Command = cmd
	config.Args = cmd.Flags().Args()
	for _, f := range stringFlags {
		if cmd.Flags().Changed(f) {
			v, err := cmd.Flags().GetString(f)
			require.NoError(t, err)
			config.Doit.Set(config.NS, f, v)
		}
	}
}

func manifestCommand(t *testing.T, raw []byte) map[any]any {
	t.Helper()
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	cmd, ok := doc["command"].(map[any]any)
	require.True(t, ok, "manifest has no command block:\n%s", raw)
	return cmd
}

func writeManifest(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sandbox.yaml")
	require.NoError(t, os.WriteFile(p, []byte(body), 0o600))
	return p
}

func TestResolveCreateSource_CommandAfterDash(t *testing.T) {
	t.Run("alone it creates a sandbox with no agent", func(t *testing.T) {
		t.Chdir(t.TempDir())
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			parseCreateArgs(t, config, []string{"--", "python", "train.py", "--epochs", "3"})

			src, err := resolveAgentCreationSource(config)

			require.NoError(t, err)
			assert.True(t, src.bareSandbox())
			cmd := manifestCommand(t, src.manifest)
			assert.Equal(t, []any{"python", "train.py", "--epochs", "3"}, cmd["argv"],
				"flags after -- belong to the program, not to doctl")
			assert.NotContains(t, cmd, "workdir")
		})
	})

	t.Run("arguments are passed unsplit", func(t *testing.T) {
		t.Chdir(t.TempDir())
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			parseCreateArgs(t, config, []string{"--", "sh", "-c", `echo "a b" > out`})

			src, err := resolveAgentCreationSource(config)

			require.NoError(t, err)
			assert.Equal(t, []any{"sh", "-c", `echo "a b" > out`}, manifestCommand(t, src.manifest)["argv"])
		})
	})

	t.Run("with --template", func(t *testing.T) {
		t.Chdir(t.TempDir())
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			parseCreateArgs(t, config, []string{"--template", "my-eval-image", "--workdir", "/srv", "--", "make", "test"},
				doctl.ArgAgentTemplate, doctl.ArgAgentExecWorkdir)

			src, err := resolveAgentCreationSource(config)

			require.NoError(t, err)
			var doc map[string]any
			require.NoError(t, yaml.Unmarshal(src.manifest, &doc))
			assert.Equal(t, "my-eval-image", doc["template"])
			cmd := manifestCommand(t, src.manifest)
			assert.Equal(t, []any{"make", "test"}, cmd["argv"])
			assert.Equal(t, "/srv", cmd["workdir"])
		})
	})

	t.Run("replaces the manifest's argv and keeps its workdir", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			path := writeManifest(t, "harness: none\nsize: mv-2vcpu-4gb\ncommand:\n  argv: [\"python\", \"old.py\"]\n  workdir: /workspace/app\n")
			parseCreateArgs(t, config, []string{path, "--", "python", "new.py"})

			src, err := resolveAgentCreationSource(config)

			require.NoError(t, err)
			cmd := manifestCommand(t, src.manifest)
			assert.Equal(t, []any{"python", "new.py"}, cmd["argv"])
			assert.Equal(t, "/workspace/app", cmd["workdir"])
			assert.Contains(t, string(src.manifest), "size: mv-2vcpu-4gb", "the rest of the manifest is untouched")
		})
	})

	t.Run("--workdir wins over the manifest's", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			path := writeManifest(t, "harness: sandbox\ncommand:\n  argv: [\"true\"]\n  workdir: /workspace/app\n")
			parseCreateArgs(t, config, []string{"--spec", path, "--workdir", "/tmp", "--", "ls"},
				doctl.ArgAgentSpec, doctl.ArgAgentExecWorkdir)

			src, err := resolveAgentCreationSource(config)

			require.NoError(t, err)
			cmd := manifestCommand(t, src.manifest)
			assert.Equal(t, []any{"ls"}, cmd["argv"])
			assert.Equal(t, "/tmp", cmd["workdir"])
		})
	})

	t.Run("legacy envelope gets spec.sandbox.command", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			path := writeManifest(t, "apiVersion: agents.digitalocean.com/v1alpha1\nkind: Agent\nmetadata:\n  name: x\nspec:\n  runtime:\n    adapter: none\n  sandbox:\n    size: mv-2vcpu-4gb\n")
			parseCreateArgs(t, config, []string{path, "--", "python", "train.py"})

			src, err := resolveAgentCreationSource(config)

			require.NoError(t, err)
			var doc map[string]any
			require.NoError(t, yaml.Unmarshal(src.manifest, &doc))
			sandbox := doc["spec"].(map[any]any)["sandbox"].(map[any]any)
			assert.Equal(t, "mv-2vcpu-4gb", sandbox["size"])
			assert.Equal(t, []any{"python", "train.py"}, sandbox["command"].(map[any]any)["argv"])
			assert.NotContains(t, doc, "command", "nothing is written at the top level of an envelope")
		})
	})

	t.Run("without -- the manifest is sent as written", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			body := "harness: none\ncommand:\n  argv: [\"python\", \"train.py\"]\n"
			parseCreateArgs(t, config, []string{writeManifest(t, body)})

			src, err := resolveAgentCreationSource(config)

			require.NoError(t, err)
			assert.Equal(t, body, string(src.manifest))
		})
	})
}

func TestResolveCreateSource_CommandRefusals(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		flags   []string
		chdir   bool
		wantErr string
	}{
		{
			name:    "manifest that runs an agent",
			args:    []string{"@agent", "--", "python", "train.py"},
			wantErr: "the manifest names one",
		},
		{
			name:    "--harness",
			args:    []string{"--harness", "opencode", "--", "python", "train.py"},
			flags:   []string{doctl.ArgAgentHarness},
			chdir:   true,
			wantErr: "--harness opencode names one",
		},
		{
			name:    "--from-config",
			args:    []string{"--from-config", "my-config", "--name", "x", "--", "python"},
			flags:   []string{doctl.ArgAgentFromConfig, doctl.ArgAgentName},
			chdir:   true,
			wantErr: "cannot be used with --from-config",
		},
		{
			name:    "--workdir with nothing to run",
			args:    []string{"--workdir", "/workspace", "@bare"},
			flags:   []string{doctl.ArgAgentExecWorkdir},
			wantErr: "give the command after --",
		},
		{
			name:    "relative --workdir",
			args:    []string{"--workdir", "app", "--", "ls"},
			flags:   []string{doctl.ArgAgentExecWorkdir},
			chdir:   true,
			wantErr: "must be an absolute guest path",
		},
		{
			name:    "a command without -- reads as a second manifest",
			args:    []string{"@bare", "python"},
			wantErr: "put it after --",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.chdir {
				t.Chdir(t.TempDir())
			}
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				args := make([]string, len(tc.args))
				for i, a := range tc.args {
					switch a {
					case "@agent":
						a = writeManifest(t, "harness: opencode\n")
					case "@bare":
						a = writeManifest(t, "harness: none\n")
					}
					args[i] = a
				}
				parseCreateArgs(t, config, args, tc.flags...)

				_, err := resolveAgentCreationSource(config)

				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			})
		})
	}
}

func TestCommandAfterDash_RefusedByValidateAndLaunch(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		parseCreateArgs(t, config, []string{writeManifest(t, "harness: none\n"), "--", "python"})

		err := RunAgentsValidate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "takes no command after --")

		err = RunAgentsLaunch(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "use `"+agentCLI+" create -- <command>`")
	})
}

func TestCreateDryRun_PrintsTheCommand(t *testing.T) {
	t.Chdir(t.TempDir())
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		parseCreateArgs(t, config, []string{"--", "python", "-m", "http.server"})
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)
		var out bytes.Buffer
		config.Out = &out

		require.NoError(t, RunAgentsCreate(config))

		assert.Equal(t, []any{"python", "-m", "http.server"}, manifestCommand(t, []byte(out.String()))["argv"])
	})
}

func TestInjectManifestCommand_NoCommandIsANoOp(t *testing.T) {
	in := []byte("harness: none\n")
	out, err := injectManifestCommand(in, commandOverlay{workdir: "/x"})
	require.NoError(t, err)
	assert.Equal(t, in, out)
}
