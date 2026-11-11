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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// expectRepoClone expects the existence check and then the clone, in order,
// for owner/repo in the session. checkExit is the exit status of the check
// (0 means the folder exists, so no clone follows); cloneStderr non-empty
// makes the clone fail.
func expectRepoClone(tm *tcMocks, sessionID, owner, repo string, checkExit int, cloneStderr string) *gomock.Call {
	_, clone := expectRepoCalls(tm, sessionID, owner, repo, checkExit, cloneStderr)
	return clone
}

// expectRepoCalls is expectRepoClone returning the check call too, so a test can
// order the check after an earlier call. The clone call is the check call when
// the folder exists, since nothing follows.
func expectRepoCalls(tm *tcMocks, sessionID, owner, repo string, checkExit int, cloneStderr string) (check, clone *gomock.Call) {
	dest := "/workspace/" + repo
	check = tm.hostedAgents.EXPECT().
		ExecInSandbox(gomock.Any(), sessionID, &godo.HostedAgentSandboxExecRequest{
			Argv:           []string{"test", "-e", dest},
			TimeoutSeconds: repoCheckTimeoutSeconds,
		}).
		Return(&godo.HostedAgentSandboxExecResponse{ExitCode: checkExit}, nil)
	if checkExit == 0 {
		return check, check
	}
	resp := &godo.HostedAgentSandboxExecResponse{}
	if cloneStderr != "" {
		resp = &godo.HostedAgentSandboxExecResponse{ExitCode: 128, Stderr: cloneStderr}
	}
	clone = tm.hostedAgents.EXPECT().
		ExecInSandbox(gomock.Any(), sessionID, &godo.HostedAgentSandboxExecRequest{
			Argv:           []string{"sh", "-c", repoCloneScript, "sh", "https://github.com/" + owner + "/" + repo + ".git", dest},
			TimeoutSeconds: repoCloneTimeoutSeconds,
		}).
		After(check).
		Return(resp, nil)
	return check, clone
}

func repoTestSession(id string) *do.HostedAgentSession {
	return &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
		SessionID: id, Name: "demo", Status: godo.HostedAgentSessionStatusProvisioning,
	}}
}

// expectRepoSession stubs create, readiness and the GitHub connect check.
func expectRepoSession(tm *tcMocks, id string) {
	tm.hostedAgents.EXPECT().CreateSessionFromManifest(gomock.Any(), gomock.Any()).Return(repoTestSession(id), nil)
	ready := repoTestSession(id)
	ready.Status = godo.HostedAgentSessionStatusReady
	tm.hostedAgents.EXPECT().GetSession(id).Return(ready, nil)
	tm.hostedAgents.EXPECT().
		StartProviderAuth("github").
		Return(&godo.HostedAgentProviderAuthStart{Provider: "github", Status: "success"}, nil)
}

func repoTestSetup(t *testing.T) {
	prev := sessionReadyPollInterval
	sessionReadyPollInterval = time.Millisecond
	t.Cleanup(func() { sessionReadyPollInterval = prev })
	// Registering the root flags can leave retries on, which makes the clone
	// swap in a real client; these tests drive the mock.
	prevRetry := RetryMax
	RetryMax = 0
	t.Cleanup(func() { RetryMax = prevRetry })
}

func setRepoCreateFlags(config *CmdConfig, repo string) {
	config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
	config.Doit.Set(config.NS, doctl.ArgAgentRepo, repo)
	config.Doit.Set(config.NS, doctl.ArgAgentTriggerPrompt, "Review README")
	config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
}

func TestRunAgentsCreate_RepoClonesBeforePrompt(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectRepoSession(tm, "sess_clone_1")
		clone := expectRepoClone(tm, "sess_clone_1", "OWASP", "railsgoat", 1, "")
		tm.hostedAgents.EXPECT().
			SendInput("sess_clone_1", &godo.HostedAgentSendInputRequest{Text: "Review README"}).
			After(clone).
			Return(nil, nil)

		var buf bytes.Buffer
		config.Out = &buf
		setRepoCreateFlags(config, "OWASP/railsgoat")

		require.NoError(t, RunAgentsCreate(config))
		out := buf.String()
		assert.Contains(t, out, "Cloning OWASP/railsgoat…")
		assert.Contains(t, out, "✓ Cloned OWASP/railsgoat into /workspace/railsgoat (0s)")
		assert.Contains(t, out, "OWASP/railsgoat → /workspace/railsgoat")
		t.Logf("success output:\n%s", out)
	})
}

func TestRunAgentsCreate_RepoAlreadyPresentSkipsClone(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectRepoSession(tm, "sess_clone_2")
		// Only the check is expected: a clone call would fail the test.
		expectRepoClone(tm, "sess_clone_2", "OWASP", "railsgoat", 0, "")
		tm.hostedAgents.EXPECT().
			SendInput("sess_clone_2", &godo.HostedAgentSendInputRequest{Text: "Review README"}).
			Return(nil, nil)

		var buf bytes.Buffer
		config.Out = &buf
		setRepoCreateFlags(config, "OWASP/railsgoat")

		require.NoError(t, RunAgentsCreate(config))
		assert.Contains(t, buf.String(), "Using existing /workspace/railsgoat")
		assert.NotContains(t, buf.String(), "Cloning")
		t.Logf("already-present output:\n%s", buf.String())
	})
}

func TestRunAgentsCreate_RepoCloneFailureKeepsSessionAndSkipsPrompt(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectRepoSession(tm, "sess_clone_3")
		// No SendInput and no RemoveSession expectation: either call fails the test.
		expectRepoClone(tm, "sess_clone_3", "owner", "private", 1,
			"Cloning into '/workspace/private'...\nfatal: could not read Username for 'https://github.com': terminal prompts disabled\n")

		var buf bytes.Buffer
		config.Out = &buf
		setRepoCreateFlags(config, "owner/private")

		err := RunAgentsCreate(config)
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, "Could not clone owner/private into /workspace/private")
		assert.Contains(t, msg, "terminal prompts disabled")
		assert.Contains(t, msg, "The session demo is still running and the prompt was not sent")
		assert.Contains(t, msg, "doctl harness-runtime exec demo -- git clone --depth 1 https://github.com/owner/private.git /workspace/private")
		assert.Contains(t, msg, "token may not have access", "the connected create gave the session a token")
		t.Logf("failure error:\n%s", msg)
	})
}

func TestRunAgentsCreate_RepoCloneFailureNotAuthRelatedOmitsPrivateHint(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectRepoSession(tm, "sess_clone_4")
		expectRepoClone(tm, "sess_clone_4", "owner", "repo", 1, "fatal: unable to access: Could not resolve host\n")

		var buf bytes.Buffer
		config.Out = &buf
		setRepoCreateFlags(config, "owner/repo")

		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "If the repository is private")
	})
}

func TestRunAgentsCreate_JSONRepoWaitsAndClones(t *testing.T) {
	repoTestSetup(t)
	prev := Output
	Output = "json"
	t.Cleanup(func() { Output = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectRepoSession(tm, "sess_clone_5")
		expectRepoClone(tm, "sess_clone_5", "owner", "repo", 1, "")

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentRepo, "owner/repo")

		var stderr bytes.Buffer
		prevErr := execStderr
		execStderr = &stderr
		t.Cleanup(func() { execStderr = prevErr })

		require.NoError(t, RunAgentsCreate(config))
		var doc any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &doc), "stdout must be one JSON document, got %q", buf.String())
		assert.Contains(t, buf.String(), "sess_clone_5")
		assert.Contains(t, stderr.String(), "GitHub already connected", "status lines go to stderr in json mode")
	})
}

func TestRunAgentsCreate_RepoNameValidatedBeforeCreate(t *testing.T) {
	for _, repo := range []string{"own er/repo", "owner/..", "owner/re$po", "-owner/repo", "owner/re;po", "owner/repo/extra", "owner/.", "https://github.com/owner/.."} {
		t.Run(repo, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				// No expectations: any API call fails the test.
				config.Out = &bytes.Buffer{}
				config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
				config.Doit.Set(config.NS, doctl.ArgAgentRepo, repo)
				err := RunAgentsCreate(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid --gh-repo")
			})
		})
	}
}

func TestValidateGitHubOwnerRepoAcceptsGitHubNames(t *testing.T) {
	for _, ref := range []string{"user_short/repo", "OWASP/railsgoat", "a-b/c.d_e-f", "o/.github", "o/repo.js"} {
		got, err := validateGitHubOwnerRepo(ref)
		require.NoError(t, err, ref)
		assert.Equal(t, ref, got)
	}
}

// The token lives only in the sandbox: the request carries a fixed script, and
// nothing git prints can bring a token back into doctl's output.
// fakeGitHubToken builds a token-shaped value at run time, so the source holds
// no literal that secret scanners report. kind is the letter after "gh".
func fakeGitHubToken(kind string) string {
	return "gh" + kind + "_" + strings.Repeat("a1B2c3", 6)
}

func TestCloneRequestedRepo_NeverLeaksTokens(t *testing.T) {
	repoTestSetup(t)
	localToken := fakeGitHubToken("p") + "local"
	urlToken := fakeGitHubToken("p")
	bearerToken := fakeGitHubToken("s")
	patToken := "github" + "_pat_" + "11" + strings.Repeat("A1b2", 8) + "_" + strings.Repeat("c3D4", 6)
	t.Setenv("GITHUB_TOKEN", localToken)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tokens := "fatal: unable to access 'https://x-access-token:" + urlToken + "@github.com/o/r.git/'\n" +
			"remote: Authorization: " + "Bearer " + bearerToken + "\n" +
			patToken + "\n"
		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), "sess_tok", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, req *godo.HostedAgentSandboxExecRequest) (*godo.HostedAgentSandboxExecResponse, error) {
				for _, a := range req.Argv {
					assert.NotContains(t, a, localToken, "the local token must never travel in argv")
				}
				if req.Argv[0] == "test" {
					return &godo.HostedAgentSandboxExecResponse{ExitCode: 1}, nil
				}
				assert.Equal(t, repoCloneScript, req.Argv[2])
				assert.Contains(t, req.Argv[2], "$GITHUB_TOKEN", "the script reads the token inside the sandbox")
				assert.Contains(t, req.Argv[2], "credential.helper=!f()")
				assert.NotContains(t, req.Argv[2], "http.extraHeader")
				return &godo.HostedAgentSandboxExecResponse{ExitCode: 128, Stderr: tokens}, nil
			}).Times(2)

		var buf bytes.Buffer
		prog := newCreationProgress(&buf)
		sess := repoTestSession("sess_tok")
		_, err := cloneRequestedRepo(config, sess, &agentCreationSource{repo: "o/r"}, prog)
		require.Error(t, err)
		all := err.Error() + buf.String()
		for _, secret := range []string{urlToken, bearerToken, patToken, "x-access-token:" + urlToken} {
			assert.NotContains(t, all, secret)
		}
		assert.Contains(t, all, "[redacted]")
	})
}

func TestCloneRequestedRepo_NoRepoIsNoop(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, err := cloneRequestedRepo(config, repoTestSession("s"), &agentCreationSource{}, nil)
		require.NoError(t, err)
	})
}

// launch calls the same helper at the same point, between ready and prompt;
// exercising it directly covers launch without needing a terminal.
func TestCloneRequestedRepo_ExecErrorIsLoud(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), "s", gomock.Any()).
			Return(nil, io.ErrUnexpectedEOF).Times(2)
		_, err := cloneRequestedRepo(config, repoTestSession("s"), &agentCreationSource{repo: "o/r"}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is still running.")
	})
}

// launch and create share finishSessionStart, so this pins the order for both:
// ready, existence check, clone, then the prompt.
func TestFinishSessionStart_OrderIsReadyCloneThenPrompt(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ready := repoTestSession("sess_ord")
		ready.Status = godo.HostedAgentSessionStatusReady
		get := tm.hostedAgents.EXPECT().GetSession("sess_ord").Return(ready, nil)
		check, clone := expectRepoCalls(tm, "sess_ord", "o", "r", 1, "")
		check.After(get)
		tm.hostedAgents.EXPECT().
			SendInput("sess_ord", &godo.HostedAgentSendInputRequest{Text: "go"}).
			After(clone).Return(nil, nil)

		_, _, err := finishSessionStart(config, repoTestSession("sess_ord"),
			&agentCreationSource{repo: "o/r", prompt: "go"}, nil)
		require.NoError(t, err)
	})
}

// launch itself, on the clone-failure path: the error comes back, no prompt is
// sent and nothing attaches (an attach would call the mock for events, which has
// no expectation, and fail the test).
func TestLaunchNewSession_CloneFailureSkipsPromptAndAttach(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stubInteractiveTerminal(t, true)
		t.Chdir(t.TempDir())
		expectRepoSession(tm, "sess_launch")
		expectRepoClone(tm, "sess_launch", "owner", "private", 1, "fatal: repository not found\n")

		config.Out = &bytes.Buffer{}
		setRepoCreateFlags(config, "owner/private")

		err := launchNewSession(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Could not clone owner/private")
		assert.Contains(t, err.Error(), "prompt was not sent")
	})
}

func TestRunAgentsCreate_RepoCloneAPIErrorHasNoPrivateHintOrTransportNoise(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectRepoSession(tm, "sess_api")
		tm.hostedAgents.EXPECT().ExecInSandbox(gomock.Any(), "sess_api", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, req *godo.HostedAgentSandboxExecRequest) (*godo.HostedAgentSandboxExecResponse, error) {
				if req.Argv[0] == "test" {
					return &godo.HostedAgentSandboxExecResponse{ExitCode: 1}, nil
				}
				return nil, errors.New(`POST https://api.digitalocean.com/v2/x/exec: 404 (request "abc") session not found`)
			}).Times(2)

		config.Out = &bytes.Buffer{}
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentRepo, "owner/repo")

		err := RunAgentsCreate(config)
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, "session not found")
		assert.NotContains(t, msg, "POST https")
		assert.NotContains(t, msg, "If the repository is private", "an API error says nothing about the repository")
		assert.NotContains(t, msg, "prompt was not sent", "no --prompt was given")
		assert.Contains(t, msg, "is still running.")
		t.Logf("API error failure:\n%s", msg)
	})
}

// The clone's exec calls must reach the host the rest of the command uses,
// including after the no-retry client is rebuilt.
func TestCloneRequestedRepo_ExecReachesRegionalHost(t *testing.T) {
	prevRetry := RetryMax
	RetryMax = 5 // retries on, so the no-retry swap really runs
	t.Cleanup(func() { RetryMax = prevRetry })

	var hosts []string
	cfg := &regionalTestConfig{TestConfig: doctl.NewTestConfig(), seen: &hosts}
	sess := repoTestSession("sess_reg")
	sess.Location = "https://api-mem1.example.test/v2/agents/sessions/sess_reg"

	config := &CmdConfig{NS: "test", Doit: cfg, Out: &bytes.Buffer{}, getContextAccessToken: func() string { return "tok" }}
	config.HostedAgents = func() do.HostedAgentsService { return nil }
	require.NoError(t, useSessionLocationHost(config, sess))
	require.Equal(t, "https://api-mem1.example.test/", config.hostedAgentsBase)

	_, err := cloneRequestedRepo(config, sess, &agentCreationSource{repo: "o/r"}, nil)
	require.NoError(t, err)

	require.NotEmpty(t, hosts)
	for _, h := range hosts {
		assert.Equal(t, "api-mem1.example.test", h, "every exec call must reach the session's host")
	}
	// check (retrying client) + clone (no-retry client) both went regional.
	assert.GreaterOrEqual(t, len(hosts), 2)
}

func TestHostedAgentsHostOpts(t *testing.T) {
	c := &CmdConfig{}
	opts, err := hostedAgentsHostOpts(c)
	require.NoError(t, err)
	assert.Empty(t, opts)

	c.hostedAgentsBase = "https://api-mem1.example.test/"
	opts, err = hostedAgentsHostOpts(c)
	require.NoError(t, err)
	cl, err := godo.New(nil, opts...)
	require.NoError(t, err)
	assert.Equal(t, "https://api-mem1.example.test/", cl.BaseURL.String())
}

// regionalTestConfig builds real godo clients (honouring the host options) over
// a transport that records the host and answers every exec with success.
type regionalTestConfig struct {
	*doctl.TestConfig
	seen  *[]string
	allOK bool // every exec exits 0, instead of "not there" on the first
}

func (r *regionalTestConfig) GetGodoClient(trace, allowRetries bool, accessToken string, opts ...godo.ClientOpt) (*godo.Client, error) {
	hc := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/exec") {
			*r.seen = append(*r.seen, req.URL.Host)
			if len(*r.seen) == 1 && !r.allOK { // the existence check: not there, so the clone follows
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(`{"exit_code":1}`)), Request: req}, nil
			}
		}
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"exit_code":0}`)),
			Request:    req,
		}, nil
	})}
	return godo.New(hc, opts...)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGitHubTokenSlot(t *testing.T) {
	flat := []byte("harness: opencode\nname: demo\n")

	t.Run("connected adds the slot to the create body", func(t *testing.T) {
		repoTestSetup(t)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			var stderr bytes.Buffer
			prev := execStderr
			execStderr = &stderr
			t.Cleanup(func() { execStderr = prev })

			tm.hostedAgents.EXPECT().CreateSessionFromManifest(gomock.Any(), gomock.Any()).
				DoAndReturn(func(m []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
					assert.Contains(t, string(m), "GITHUB_TOKEN: oauth/github")
					return repoTestSession("s"), nil
				})
			tm.hostedAgents.EXPECT().StartProviderAuth("github").
				Return(&godo.HostedAgentProviderAuthStart{Provider: "github", Status: "success"}, nil)
			config.Out = &bytes.Buffer{}
			_, err := createAgentSession(config, &agentCreationSource{repo: "o/r", manifest: flat}, nil)
			require.NoError(t, err)
			assert.Contains(t, stderr.String(), "can also push with it")
		})
	})

	t.Run("not connected adds nothing", func(t *testing.T) {
		repoTestSetup(t)
		prevOut := Output
		Output = "json" // no prompt; the team is simply not connected
		t.Cleanup(func() { Output = prevOut })
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().CreateSessionFromManifest(gomock.Any(), gomock.Any()).
				DoAndReturn(func(m []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
					assert.NotContains(t, string(m), "GITHUB_TOKEN")
					return repoTestSession("s"), nil
				})
			tm.hostedAgents.EXPECT().StartProviderAuth("github").
				Return(&godo.HostedAgentProviderAuthStart{Provider: "github", Status: "pending"}, nil)
			config.Out = &bytes.Buffer{}
			_, err := createAgentSession(config, &agentCreationSource{repo: "o/r", manifest: flat}, nil)
			require.NoError(t, err)
		})
	})

	t.Run("already declared is untouched", func(t *testing.T) {
		declared := []byte("harness: opencode\nsecrets:\n  GITHUB_TOKEN: my-own-value\n")
		src := &agentCreationSource{repo: "o/r", manifest: declared}
		added, _, err := addGitHubTokenSlot(src)
		require.NoError(t, err)
		assert.False(t, added)
		assert.Equal(t, declared, src.manifest)
	})

	t.Run("legacy envelope is left alone", func(t *testing.T) {
		legacy := []byte("apiVersion: v1\nkind: Agent\nspec: {}\n")
		src := &agentCreationSource{repo: "o/r", manifest: legacy}
		added, _, err := addGitHubTokenSlot(src)
		require.NoError(t, err)
		assert.False(t, added)
	})

	t.Run("existing secrets are kept", func(t *testing.T) {
		src := &agentCreationSource{repo: "o/r", manifest: []byte("harness: opencode\nsecrets:\n  OTHER: x\n")}
		added, _, err := addGitHubTokenSlot(src)
		require.NoError(t, err)
		assert.True(t, added)
		assert.Contains(t, string(src.manifest), "OTHER: x")
		assert.Contains(t, string(src.manifest), "GITHUB_TOKEN: oauth/github")
	})
}

func TestLastRepoErrorLinesIsRuneSafe(t *testing.T) {
	out := lastRepoErrorLines(strings.Repeat("é", 1500), 5)
	assert.True(t, utf8.ValidString(out))
	assert.Equal(t, 1000, utf8.RuneCountInString(out))
}

func TestRepoCloneScriptKeepsTokenOutOfArgv(t *testing.T) {
	const helper = `-c 'credential.helper=!f() { echo username=x-access-token; echo "password=$GITHUB_TOKEN"; }; f'`
	assert.Contains(t, repoCloneScript, helper, "the helper must stay single-quoted so $GITHUB_TOKEN expands only inside it")
	assert.Contains(t, repoCloneScript, `[ -n "${GITHUB_TOKEN:+x}" ]`)
}

// Runs the real script under sh with a stub git that records its argv.
func TestRepoCloneScriptRunsWithoutTokenInArgv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX sh")
	}
	run := func(t *testing.T, token string) []string {
		dir := t.TempDir()
		out := filepath.Join(dir, "argv")
		stub := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > \"$ARGV_OUT\"\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, "git"), []byte(stub), 0o755))
		cmd := exec.Command("sh", "-c", repoCloneScript, "sh", "https://github.com/o/r.git", "/workspace/r")
		cmd.Env = []string{"PATH=" + dir + ":/usr/bin:/bin", "ARGV_OUT=" + out, "GITHUB_TOKEN=" + token}
		require.NoError(t, cmd.Run())
		raw, err := os.ReadFile(out)
		require.NoError(t, err)
		return strings.Split(strings.TrimSpace(string(raw)), "\n")
	}

	secretToken := fakeGitHubToken("p")
	withToken := run(t, secretToken)
	for _, a := range withToken {
		assert.NotContains(t, a, secretToken, "the token must not be expanded into git's argv")
	}
	assert.Contains(t, strings.Join(withToken, "\n"), "$GITHUB_TOKEN", "the helper body reads it from the environment")
	assert.Equal(t, []string{"clone", "--depth", "1", "https://github.com/o/r.git", "/workspace/r"}, withToken[len(withToken)-5:])

	anon := run(t, "")
	assert.Equal(t, []string{"clone", "--depth", "1", "https://github.com/o/r.git", "/workspace/r"}, anon)
}

func TestRepoCloneErrorHints(t *testing.T) {
	authErr := "fatal: could not read Username for 'https://github.com': terminal prompts disabled"
	mk := func(text string, kind cloneFailure, hasToken bool) string {
		return repoCloneError("o/r", "https://github.com/o/r.git", "/workspace/r", "demo", "demo", text, kind, hasToken, true).Error()
	}
	assert.Contains(t, mk(authErr, cloneGitError, false), "connect GitHub (`doctl harness-runtime auth github`) and run the command again")

	withToken := mk(authErr, cloneGitError, true)
	assert.Contains(t, withToken, "token may not have access")
	assert.NotContains(t, withToken, "connect GitHub")

	noGit := mk("sh: git: not found", cloneGitMissing, false)
	assert.Contains(t, noGit, "git is not available in this session's image")
	assert.NotContains(t, noGit, "private")

	assert.NotContains(t, mk("repository not found", cloneAPIError, false), "private")
}

func TestCloneRequestedRepo_MissingGitSaysSo(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectRepoClone(tm, "s", "o", "r", 1, "sh: 1: exec: git: not found\n")
		_, err := cloneRequestedRepo(config, repoTestSession("s"), &agentCreationSource{repo: "o/r"}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git is not available in this session's image")
		assert.NotContains(t, err.Error(), "connect GitHub")
	})
}

func TestCloneRequestedRepo_TokenAwareHintFromManifest(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectRepoClone(tm, "s", "o", "r", 1, "fatal: repository 'https://github.com/o/r.git/' not found\n")
		src := &agentCreationSource{repo: "o/r", manifest: []byte("harness: opencode\nsecrets:\n  GITHUB_TOKEN: oauth/github\n")}
		_, err := cloneRequestedRepo(config, repoTestSession("s"), src, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token may not have access")
	})
}

func TestStripAPIErrorNoise(t *testing.T) {
	assert.Equal(t, "dial tcp: lookup api: no such host",
		stripAPIErrorNoise(`Post "https://api.example.test/v2/x/exec": dial tcp: lookup api: no such host`))
}

func TestRetryCommandCarriesRegion(t *testing.T) {
	mk := func(region string) *do.HostedAgentSession {
		s := repoTestSession("s")
		s.Region = region
		return s
	}
	assert.Equal(t, " --region mem1", regionFlagFor(mk("mem1")), "the region the server reported")
	assert.Equal(t, "", regionFlagFor(mk(defaultAgentRegion)), "the default region needs no flag")
	assert.Equal(t, "", regionFlagFor(mk("")))

	setAgentRegion(t, "nyc9")
	assert.Equal(t, " --region nyc9", regionFlagFor(mk("")), "falls back to --region")

	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectRepoClone(tm, "s", "o", "r", 1, "fatal: boom\n")
		_, err := cloneRequestedRepo(config, mk("mem1"), &agentCreationSource{repo: "o/r"}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "doctl harness-runtime exec demo --region mem1 -- git clone")
	})
}

func TestHostedAgentsHostOptsUsesRegionFlag(t *testing.T) {
	setAgentRegion(t, "mem1")
	opts, err := hostedAgentsHostOpts(&CmdConfig{})
	require.NoError(t, err)
	cl, err := godo.New(nil, opts...)
	require.NoError(t, err)
	assert.Equal(t, godo.HostedAgentsRegionalBaseURL("mem1"), cl.BaseURL.String())
}

// With retries on and --region set, the rebuilt no-retry client must still go to
// the regional host.
func TestDisableRetriesForExecKeepsRegionFlagHost(t *testing.T) {
	prevRetry := RetryMax
	RetryMax = 5
	t.Cleanup(func() { RetryMax = prevRetry })
	setAgentRegion(t, "mem1")

	var hosts []string
	config := &CmdConfig{NS: "test", Doit: &regionalTestConfig{TestConfig: doctl.NewTestConfig(), seen: &hosts},
		Out: &bytes.Buffer{}, getContextAccessToken: func() string { return "tok" }}
	config.HostedAgents = func() do.HostedAgentsService { return nil }
	require.NoError(t, disableRetriesForExec(config))
	_, err := config.HostedAgents().ExecInSandbox(context.Background(), "s", &godo.HostedAgentSandboxExecRequest{Argv: []string{"true"}})
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, strings.TrimSuffix(strings.TrimPrefix(godo.HostedAgentsRegionalBaseURL("mem1"), "https://"), "/"), hosts[0])
}

// The existence check runs on the retrying client (the one already in place);
// only the clone swaps to the no-retry client, and the retrying one is back
// afterwards for the prompt.
func TestCloneRequestedRepo_CheckKeepsRetryingClientAndItIsRestored(t *testing.T) {
	prevRetry := RetryMax
	RetryMax = 5
	t.Cleanup(func() { RetryMax = prevRetry })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var hosts []string
		config.Doit = &regionalTestConfig{TestConfig: config.Doit.(*doctl.TestConfig), seen: &hosts, allOK: true}
		original := config.HostedAgents()

		// Only the check is on the mock (the retrying client); the clone goes
		// through the rebuilt client and shows up in hosts.
		tm.hostedAgents.EXPECT().ExecInSandbox(gomock.Any(), "s", &godo.HostedAgentSandboxExecRequest{
			Argv: []string{"test", "-e", "/workspace/r"}, TimeoutSeconds: repoCheckTimeoutSeconds,
		}).Return(&godo.HostedAgentSandboxExecResponse{ExitCode: 1}, nil)

		_, err := cloneRequestedRepo(config, repoTestSession("s"), &agentCreationSource{repo: "o/r"}, nil)
		require.NoError(t, err)
		assert.Len(t, hosts, 1, "the clone ran on the rebuilt no-retry client")
		assert.Same(t, original, config.HostedAgents(), "the retrying client is restored for SendInput")
	})
}

func TestMaybeOfferGitHubAuthConnectedResult(t *testing.T) {
	pending := &godo.HostedAgentProviderAuthStart{Provider: "github", Status: "pending",
		ConnectURL: "https://example.com/connect", PollURL: "https://example.com/poll"}
	cases := []struct {
		name  string
		setup func(tm *tcMocks)
		want  bool
	}{
		{"already connected", func(tm *tcMocks) {
			tm.hostedAgents.EXPECT().StartProviderAuth("github").
				Return(&godo.HostedAgentProviderAuthStart{Status: "success"}, nil)
		}, true},
		{"status check error", func(tm *tcMocks) {
			tm.hostedAgents.EXPECT().StartProviderAuth("github").Return(nil, errors.New("boom"))
		}, false},
		{"declined", func(tm *tcMocks) {
			tm.hostedAgents.EXPECT().StartProviderAuth("github").Return(pending, nil)
			askConnectGitHub = func() (bool, error) { return false, nil }
		}, false},
		{"accepted but nothing to poll", func(tm *tcMocks) {
			noPoll := *pending
			noPoll.PollURL = ""
			tm.hostedAgents.EXPECT().StartProviderAuth("github").Return(&noPoll, nil)
			askConnectGitHub = func() (bool, error) { return true, nil }
		}, false},
		{"accepted and poll succeeds", func(tm *tcMocks) {
			tm.hostedAgents.EXPECT().StartProviderAuth("github").Return(pending, nil)
			tm.hostedAgents.EXPECT().PollProviderAuth("github", "https://example.com/poll").
				Return(&godo.HostedAgentProviderAuthPoll{Provider: "github", Status: "success"}, nil)
			askConnectGitHub = func() (bool, error) { return true, nil }
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				prevAsk, prevInterval := askConnectGitHub, agentsAuthPollInterval
				t.Cleanup(func() { askConnectGitHub, agentsAuthPollInterval = prevAsk, prevInterval })
				agentsAuthPollInterval = time.Millisecond
				tc.setup(tm)
				config.Out = &bytes.Buffer{}
				got, err := maybeOfferGitHubAuth(config)
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			})
		})
	}
}

func TestLegacyEnvelopeSkipSaysSo(t *testing.T) {
	repoTestSetup(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var stderr bytes.Buffer
		prev := execStderr
		execStderr = &stderr
		t.Cleanup(func() { execStderr = prev })
		tm.hostedAgents.EXPECT().CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			DoAndReturn(func(m []byte, _ *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				assert.NotContains(t, string(m), "oauth/github")
				return repoTestSession("s"), nil
			})
		tm.hostedAgents.EXPECT().StartProviderAuth("github").
			Return(&godo.HostedAgentProviderAuthStart{Status: "success"}, nil)
		config.Out = &bytes.Buffer{}
		_, err := createAgentSession(config, &agentCreationSource{repo: "o/r", manifest: []byte("apiVersion: agents.digitalocean.com/v1alpha1\nkind: Agent\nmetadata:\n  name: demo\nspec:\n  runtime:\n    adapter: opencode\n")}, nil)
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "add GITHUB_TOKEN: oauth/github under secrets in a flat manifest")
	})
}

// The slot is added before the manifest is expanded, so a ${GITHUB_TOKEN}
// reference elsewhere must not resolve to the literal reference.
func TestOAuthReferenceIsNotAnExpansionValue(t *testing.T) {
	m := []byte("harness: opencode\nsecrets:\n  GITHUB_TOKEN: oauth/github\n  OTHER: real\n")
	got := resolvedManifestSecretValues(m)
	assert.NotContains(t, got, "GITHUB_TOKEN")
	assert.Equal(t, "real", got["OTHER"])
}

func TestDryRunWithRepoSaysWhatItWouldDo(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var stderr bytes.Buffer
		prev := execStderr
		execStderr = &stderr
		t.Cleanup(func() { execStderr = prev })

		config.Out = &bytes.Buffer{}
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentRepo, "owner/repo")
		config.Doit.Set(config.NS, doctl.ArgAgentDryRun, true)

		require.NoError(t, RunAgentsCreate(config)) // no mock expectations: any API call fails
		assert.Contains(t, stderr.String(), "Would clone owner/repo into /workspace/repo")
		assert.Contains(t, stderr.String(), "If GitHub is connected, create also adds GITHUB_TOKEN: oauth/github.")
	})
}
