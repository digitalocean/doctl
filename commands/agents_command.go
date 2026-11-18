package commands

import (
	"fmt"
	"strings"

	"github.com/digitalocean/doctl"
	"gopkg.in/yaml.v2"
)

// manifestArgs returns the positional arguments before a `--`. Everything after
// it is the argv of the sandbox's command (commandArgv), so a manifest path and
// the program it runs can share one invocation:
//
//	create sandbox.yaml -- python train.py --epochs 3
func manifestArgs(c *CmdConfig) []string {
	if dash := argsLenAtDash(c); dash >= 0 {
		return c.Args[:dash]
	}
	return c.Args
}

// commandArgv returns the arguments after a `--`, or nil when there was none.
// They are passed through verbatim: an argv is never split or re-quoted, so an
// argument containing spaces or quotes reaches the program as typed.
func commandArgv(c *CmdConfig) []string {
	dash := argsLenAtDash(c)
	if dash < 0 || dash >= len(c.Args) {
		return nil
	}
	return append([]string(nil), c.Args[dash:]...)
}

func argsLenAtDash(c *CmdConfig) int {
	if c == nil || c.Command == nil {
		return -1
	}
	dash := c.Command.ArgsLenAtDash()
	if dash > len(c.Args) {
		return -1
	}
	return dash
}

// commandOverlay is the sandbox command given on the command line: an argv
// after `--` and an optional --workdir.
type commandOverlay struct {
	argv    []string
	workdir string
}

func (o commandOverlay) set() bool { return len(o.argv) > 0 }

// resolveCommandOverlay reads the argv after `--` and --workdir, refusing a
// --workdir with nothing to run.
func resolveCommandOverlay(c *CmdConfig) (commandOverlay, error) {
	workdir, err := invocationString(c, doctl.ArgAgentExecWorkdir)
	if err != nil {
		return commandOverlay{}, err
	}
	o := commandOverlay{argv: commandArgv(c), workdir: strings.TrimSpace(workdir)}
	if o.workdir != "" && !o.set() {
		return commandOverlay{}, fmt.Errorf("--%s sets the working directory of the sandbox's command; give the command after --, e.g. `%s create --%s %s -- python train.py`",
			doctl.ArgAgentExecWorkdir, agentCLI, doctl.ArgAgentExecWorkdir, o.workdir)
	}
	if o.workdir != "" && !strings.HasPrefix(o.workdir, "/") {
		return commandOverlay{}, fmt.Errorf("--%s must be an absolute guest path, got %q", doctl.ArgAgentExecWorkdir, o.workdir)
	}
	return o, nil
}

// injectManifestCommand writes o into the manifest's command block: top-level
// `command` in the flat format, spec.sandbox.command in the legacy envelope.
// The argv replaces the manifest's; the manifest's workdir is kept unless
// --workdir is given, since it describes the image rather than the program.
func injectManifestCommand(manifest []byte, o commandOverlay) ([]byte, error) {
	if !o.set() {
		return manifest, nil
	}
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		return nil, fmt.Errorf("parsing manifest to apply the command after --: %w", err)
	}
	if doc == nil {
		doc = map[string]any{}
	}

	argv := make([]any, len(o.argv))
	for i, a := range o.argv {
		argv[i] = a
	}
	apply := func(parent map[any]any) map[any]any {
		cmd, _ := parent["command"].(map[any]any)
		if cmd == nil {
			cmd = map[any]any{}
		}
		cmd["argv"] = argv
		if o.workdir != "" {
			cmd["workdir"] = o.workdir
		}
		return cmd
	}

	if _, legacy := doc["apiVersion"]; legacy {
		// yaml.v2 decodes nested mappings as map[any]any.
		spec, _ := doc["spec"].(map[any]any)
		if spec == nil {
			spec = map[any]any{}
		}
		sandbox, _ := spec["sandbox"].(map[any]any)
		if sandbox == nil {
			sandbox = map[any]any{}
		}
		sandbox["command"] = apply(sandbox)
		spec["sandbox"] = sandbox
		doc["spec"] = spec
	} else {
		top := map[any]any{"command": doc["command"]}
		doc["command"] = apply(top)
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("applying the command after -- to the manifest: %w", err)
	}
	return out, nil
}

// commandNeedsBareSandboxErr is the refusal for a command on a session that
// runs an agent: the agent is that session's program.
func commandNeedsBareSandboxErr(source string) error {
	return fmt.Errorf("a command after -- runs in a sandbox with no agent, and %s names one; drop the agent (`harness: none` in a manifest, or `--%s %s` with no --%s), or drop the command",
		source, doctl.ArgAgentTemplate, templateAliasSandbox, doctl.ArgAgentHarness)
}
