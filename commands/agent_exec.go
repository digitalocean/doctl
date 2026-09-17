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
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// execStderr is where a guest command's stderr is mirrored. c.Out only covers
// stdout, and the two streams must stay separate for redirection to behave; a
// var so tests can capture it.
var execStderr io.Writer = os.Stderr

// execExit ends doctl with the guest command's status. Exiting here rather than
// returning an error is deliberate: doctl's shared checkErr can only ever exit
// 1, and teaching it a second status would change behaviour for every other
// command. A var so tests can observe the code instead of ending the test
// binary.
var execExit = os.Exit

// RunAgentsExec runs one command in the session's sandbox and reproduces its
// result locally: guest stdout and stderr are passed through unchanged and the
// guest's exit code becomes doctl's, so the command composes in a pipeline or an
// `&&` chain.
func RunAgentsExec(c *CmdConfig) error {
	// args[0] is the session; everything after it is the guest command. Callers
	// separate the two with `--`, without which cobra would try to parse the
	// guest command's own flags (`ls -la`) as doctl flags.
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	argv := c.Args[1:]

	// Resolution above is a plain GET and keeps the shared retrying client; the
	// exec itself must not.
	if err := disableRetriesForExec(c); err != nil {
		return err
	}

	workdir, err := c.Doit.GetString(c.NS, doctl.ArgAgentExecWorkdir)
	if err != nil {
		return err
	}
	timeout, err := c.Doit.GetInt(c.NS, doctl.ArgAgentExecTimeout)
	if err != nil {
		return err
	}

	// SIGTERM alongside SIGINT so Ctrl-C or a plain `kill` stops the wait
	// instead of hanging until the server's own timeout. Giving up locally does
	// NOT yet stop the guest process — it runs on until its timeout.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	resp, err := c.HostedAgents().ExecInSandbox(ctx, sessionID, &godo.HostedAgentSandboxExecRequest{
		Argv:           argv,
		Workdir:        workdir,
		TimeoutSeconds: int64(timeout),
	})
	if err != nil {
		return err
	}

	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentSandboxExec{
			Execs:  []*godo.HostedAgentSandboxExecResponse{resp},
			Single: true,
		}); err != nil {
			return err
		}
		return exitWithGuestStatus(resp.ExitCode)
	}

	// Verbatim, and with no added newline: the guest's bytes are the output
	// contract, so anything appended here would corrupt a piped payload.
	if _, err := io.WriteString(c.Out, resp.Stdout); err != nil {
		return err
	}
	if _, err := io.WriteString(execStderr, resp.Stderr); err != nil {
		return err
	}
	return exitWithGuestStatus(resp.ExitCode)
}

// disableRetriesForExec swaps in a client that does not retry, following the
// same pattern as `doctl account ratelimit`.
//
// doctl retries 429s and 5xx up to --http-retry-max (5 by default). That is
// right for the idempotent calls it was written for and wrong for exec: a retry
// runs the guest command a second time, so a `make install` answered with a 5xx
// would be replayed up to six times. Losing the retry costs little here, since
// the caller sees the failure and can re-run deliberately.
func disableRetriesForExec(c *CmdConfig) error {
	if RetryMax <= 0 {
		return nil
	}
	godoClient, err := c.Doit.GetGodoClient(Trace, false, c.getContextAccessToken())
	if err != nil {
		return fmt.Errorf("unable to initialize DigitalOcean API client: %s", err)
	}
	c.HostedAgents = func() do.HostedAgentsService { return do.NewHostedAgentsService(godoClient) }
	return nil
}

// exitWithGuestStatus reproduces the guest's exit code as doctl's own, so the
// command composes in `&&` chains and `if` tests. A command that ran and failed
// is not a doctl error, so nothing is printed on top of the output the guest
// already produced.
//
// The guest's stdout and stderr have been written to file descriptors by this
// point, so os.Exit cannot truncate them.
func exitWithGuestStatus(code int) error {
	if code == 0 {
		return nil
	}
	execExit(code)
	// Unreachable in a real run — os.Exit does not return. Under test the stub
	// does, and the sentinel keeps the runner honest about having failed
	// without printing a second error on top of the guest's.
	return ErrExitSilently
}
