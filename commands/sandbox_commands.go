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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// addSandboxCommandCmds adds `exec` and the `command` group to `doctl sandbox`.
func addSandboxCommandCmds(parent *Command) {
	cmdExec := CmdBuilder(parent, RunSandboxExec, "exec <sandbox-id> -- <command> [args...]",
		"Run a command in a sandbox",
		"Runs an executable in a sandbox, streams its output and exits with a non-zero status when the command does. "+
			"The command is an executable and its arguments, not a shell string: use `-- bash -lc '...'` for a shell. "+
			"Output can only be read while the command runs, so a command that exits before doctl attaches fails with a not-found error.",
		Writer)
	AddStringFlag(cmdExec, doctl.ArgSandboxCwd, "", "", "The working directory of the command")
	AddStringArrayFlag(cmdExec, doctl.ArgSandboxEnv, "", nil,
		"An environment variable for the command, in `KEY=VALUE` form. Repeatable")
	AddIntFlag(cmdExec, doctl.ArgSandboxTimeout, "", 0, "Kill the command after this many seconds. Zero means no timeout")
	AddBoolFlag(cmdExec, doctl.ArgSandboxStdin, "", false, "Send this terminal's standard input to the command")
	cmdExec.Example = "The following example runs a shell command: doctl sandbox exec 01a0ef1a-d873-793d-ae6a-5dd6239be62a -- bash -lc 'echo hello'\n\n" +
		"The following example pipes a file to a command: cat data.txt | doctl sandbox exec 01a0ef1a-d873-793d-ae6a-5dd6239be62a --stdin -- wc -l"

	cmd := &Command{
		Command: &cobra.Command{
			Use:     "command",
			Aliases: []string{"commands"},
			Short:   "Manage the commands running in a sandbox",
			Long: "The subcommands of `doctl sandbox command` list, attach to and stop the commands running in a sandbox. " +
				"Start one with `doctl sandbox exec`.",
		},
	}
	parent.AddCommand(cmd)

	cmdList := CmdBuilder(cmd, RunSandboxCommandList, "list <sandbox-id>",
		"List the commands running in a sandbox",
		"Lists the commands that are running in a sandbox: their process ID, executable and working directory.",
		Writer, aliasOpt("ls"), displayerType(&displayers.SandboxCommand{}))
	cmdList.Example = "The following example lists the running commands: doctl sandbox command list 01a0ef1a-d873-793d-ae6a-5dd6239be62a"

	cmdAttach := CmdBuilder(cmd, RunSandboxCommandAttach, "attach <sandbox-id> <pid>",
		"Stream the output of a running command",
		"Attaches to a running command and streams its output until it exits. Exits with a non-zero status when the command does.",
		Writer)
	cmdAttach.Example = "The following example attaches to process 42: doctl sandbox command attach 01a0ef1a-d873-793d-ae6a-5dd6239be62a 42"

	cmdKill := CmdBuilder(cmd, RunSandboxCommandKill, "kill <sandbox-id> <pid>",
		"Stop a running command",
		"Sends a signal to a running command. The default is `term` (SIGTERM); use `--signal kill` for SIGKILL.",
		Writer)
	AddStringFlag(cmdKill, doctl.ArgSandboxSignal, "", "term", "The signal to send: `term` or `kill`")
	cmdKill.Example = "The following example force-stops process 42: doctl sandbox command kill 01a0ef1a-d873-793d-ae6a-5dd6239be62a 42 --signal kill"
}

// RunSandboxExec starts a command and streams its output.
func RunSandboxExec(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id, argv := c.Args[0], c.Args[1:]

	cwd, err := c.Doit.GetString(c.NS, doctl.ArgSandboxCwd)
	if err != nil {
		return err
	}
	pairs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSandboxEnv)
	if err != nil {
		return err
	}
	env, err := parseEnvPairs(pairs)
	if err != nil {
		return err
	}
	timeout, err := c.Doit.GetInt(c.NS, doctl.ArgSandboxTimeout)
	if err != nil {
		return err
	}
	if timeout < 0 {
		return errors.New("--timeout must not be negative")
	}
	withStdin, err := c.Doit.GetBool(c.NS, doctl.ArgSandboxStdin)
	if err != nil {
		return err
	}

	req := &godo.SandboxCommandStartRequest{
		Cmd:            argv[0],
		Cwd:            cwd,
		Stdin:          withStdin,
		TimeoutSeconds: int64(timeout),
	}
	if len(argv) > 1 {
		req.Args = argv[1:]
	}
	if len(env) > 0 {
		req.EnvVars = env
	}

	svc := c.Sandboxes()
	pid, err := svc.StartCommand(id, req)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if withStdin {
		go forwardStdin(svc, id, pid, os.Stdin)
	}

	code, err := streamSandboxCommand(ctx, svc, id, pid, c.Out, os.Stderr)
	if err != nil {
		if ctx.Err() != nil {
			// Ctrl-C: ask the command to stop rather than leave it running.
			_ = svc.SignalCommand(id, pid, godo.SandboxCommandSignalTerm)
			return errors.New("interrupted; sent SIGTERM to the command")
		}
		return err
	}
	if code != 0 {
		return fmt.Errorf("command exited with code %d", code)
	}
	return nil
}

// RunSandboxCommandList lists the commands running in a sandbox.
func RunSandboxCommandList(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	list, err := c.Sandboxes().ListCommands(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.SandboxCommand{Commands: list})
}

// RunSandboxCommandAttach streams the output of a running command.
func RunSandboxCommandAttach(c *CmdConfig) error {
	id, pid, err := sandboxAndPID(c)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	code, err := streamSandboxCommand(ctx, c.Sandboxes(), id, pid, c.Out, os.Stderr)
	if err != nil {
		if ctx.Err() != nil {
			return errors.New("detached; the command is still running")
		}
		return err
	}
	if code != 0 {
		return fmt.Errorf("command exited with code %d", code)
	}
	return nil
}

// RunSandboxCommandKill signals a running command.
func RunSandboxCommandKill(c *CmdConfig) error {
	id, pid, err := sandboxAndPID(c)
	if err != nil {
		return err
	}
	flag, err := c.Doit.GetString(c.NS, doctl.ArgSandboxSignal)
	if err != nil {
		return err
	}
	sig, err := parseSandboxSignal(flag)
	if err != nil {
		return err
	}
	return c.Sandboxes().SignalCommand(id, pid, sig)
}

func sandboxAndPID(c *CmdConfig) (string, int64, error) {
	if len(c.Args) < 2 {
		return "", 0, doctl.NewMissingArgsErr(c.NS)
	}
	if len(c.Args) > 2 {
		return "", 0, doctl.NewTooManyArgsErr(c.NS)
	}
	pid, err := strconv.ParseInt(c.Args[1], 10, 64)
	if err != nil || pid < 0 {
		return "", 0, fmt.Errorf("invalid process ID %q: expected a number", c.Args[1])
	}
	return c.Args[0], pid, nil
}

// parseSandboxSignal turns the --signal flag into an API signal. It accepts
// term or kill, optionally prefixed with SIG, in any case. Empty means term.
func parseSandboxSignal(value string) (godo.SandboxCommandSignal, error) {
	v := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(value)), "SIG")
	switch v {
	case "", "TERM":
		return godo.SandboxCommandSignalTerm, nil
	case "KILL":
		return godo.SandboxCommandSignalKill, nil
	}
	return "", fmt.Errorf("invalid --signal %q: expected term or kill", value)
}

// streamSandboxCommand copies a command's output to stdout and stderr until it
// exits, and returns its exit code. A stream that ends without an exit event is
// an error: the outcome is unknown.
func streamSandboxCommand(ctx context.Context, svc do.SandboxesService, id string, pid int64, stdout, stderr io.Writer) (int32, error) {
	stream, err := svc.StreamCommand(ctx, id, pid)
	if err != nil {
		var apiErr *godo.ErrorResponse
		if errors.As(err, &apiErr) && apiErr.Response != nil && apiErr.Response.StatusCode == 404 {
			return 0, fmt.Errorf("command %d not found or already exited: output can only be read while it runs: %w", pid, err)
		}
		return 0, err
	}
	defer stream.Close()

	for {
		ev, err := stream.Next()
		if err == io.EOF {
			return 0, fmt.Errorf("the stream for command %d ended before the command exited; its outcome is unknown", pid)
		}
		if err != nil {
			return 0, err
		}
		switch {
		case len(ev.Stdout) > 0:
			if _, err := stdout.Write(ev.Stdout); err != nil {
				return 0, err
			}
		case len(ev.Stderr) > 0:
			if _, err := stderr.Write(ev.Stderr); err != nil {
				return 0, err
			}
		case ev.Exit != nil:
			if ev.Exit.Error != "" {
				fmt.Fprintf(stderr, "error: %s\n", ev.Exit.Error)
			}
			return ev.Exit.ExitCode, nil
		}
	}
}

// forwardStdin sends r to the command's stdin in chunks, then closes it.
func forwardStdin(svc do.SandboxesService, id string, pid int64, r io.Reader) {
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if werr := svc.WriteCommandStdin(id, pid, append([]byte(nil), buf[:n]...), false); werr != nil {
				fmt.Fprintf(os.Stderr, "warning: writing to the command's stdin failed: %v\n", werr)
				return
			}
		}
		if err != nil {
			_ = svc.WriteCommandStdin(id, pid, nil, true)
			return
		}
	}
}
