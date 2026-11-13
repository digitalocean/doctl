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
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/digitalocean/godo"
	yaml "gopkg.in/yaml.v2"

	"github.com/digitalocean/doctl/do"
)

const (
	// repoCloneRoot is where --gh-repo lands. The workspace root already holds
	// folders of its own, so the repository gets a subfolder named after it
	// instead of being cloned into the root.
	repoCloneRoot = "/workspace"

	// repoCloneTimeoutSeconds bounds the clone itself. Ten minutes covers a
	// shallow clone of a very large repository over a slow link.
	repoCloneTimeoutSeconds = 600

	// repoCheckTimeoutSeconds bounds the existence check, which is a stat.
	repoCheckTimeoutSeconds = 30

	// githubTokenSecret is the secret slot that hands the session the user's
	// GitHub connection as $GITHUB_TOKEN.
	githubTokenSecret = "GITHUB_TOKEN"

	// githubOAuthRef is the slot value that resolves, server-side, to the
	// team's GitHub connection. It is a reference, not a credential.
	githubOAuthRef = "oauth/github"

	// repoCloneScript runs inside the sandbox. It is fixed text with no user
	// input: the URL and destination arrive as $1 and $2. The token is read from
	// the sandbox's own environment and reaches git only through a credential
	// helper whose body is single-quoted, so $GITHUB_TOKEN expands inside the
	// helper and never in any process's argv. GIT_TERMINAL_PROMPT=0 makes a
	// private repository fail fast rather than wait on a prompt nobody can answer.
	repoCloneScript = `GIT_TERMINAL_PROMPT=0; export GIT_TERMINAL_PROMPT; ` +
		`if [ -n "${GITHUB_TOKEN:+x}" ]; then ` +
		`exec git -c credential.helper= ` +
		`-c 'credential.helper=!f() { echo username=x-access-token; echo "password=$GITHUB_TOKEN"; }; f' ` +
		`clone --depth 1 "$1" "$2"; ` +
		`fi; exec git clone --depth 1 "$1" "$2"`
)

// repoClonePaths returns the clone URL and destination for an OWNER/REPO
// reference that normalizeHarnessRepoRef has already validated.
func repoClonePaths(ref string) (cloneURL, dest string) {
	name := ref[strings.LastIndex(ref, "/")+1:]
	return "https://github.com/" + ref + ".git", repoCloneRoot + "/" + name
}

// finishSessionStart is the sequence both create and launch run on a new
// session: wait until it is ready, clone --gh-repo, then send --prompt. The
// order is the point: the agent never starts the task on an empty workspace,
// and a clone failure stops before the prompt.
func finishSessionStart(c *CmdConfig, sess *do.HostedAgentSession, src *agentCreationSource, prog *creationProgress) (*do.HostedAgentSession, creationTimings, error) {
	var t creationTimings
	readyAt := creationClock()
	sess, err := waitForAgentSessionReady(c, sess, prog)
	if err != nil {
		return nil, t, err
	}
	t.Ready = creationClock().Sub(readyAt)

	if t.Clone, err = cloneRequestedRepo(c, sess, src, prog); err != nil {
		return nil, t, err
	}

	promptAt := creationClock()
	if err := sendInitialPrompt(c, sess.SessionID, src); err != nil {
		return nil, t, err
	}
	if src.prompt != "" {
		t.Prompt = creationClock().Sub(promptAt)
	}
	return sess, t, nil
}

// promptHeldBack reports whether doctl itself delivers --prompt after the
// clone, which is what "the prompt was not sent" can truthfully refer to.
func promptHeldBack(src *agentCreationSource) bool {
	return src.prompt != "" && !manifestIncludesPrompt(src.manifest, src.prompt)
}

// warnPromptBeforeClone tells the user, before create, that a harness taking
// its prompt at create time may start before the clone finishes. doctl cannot
// hold it back.
func warnPromptBeforeClone(src *agentCreationSource) {
	if src.repo == "" || src.prompt == "" || len(src.manifest) == 0 {
		return
	}
	doc, err := parseAgentManifest(src.manifest)
	if err != nil || !isOpenAISandboxAdapter(doc.adapter()) {
		return
	}
	warn("this harness receives --prompt when the session is created, so it may start before %s is cloned", src.repo)
}

// cloneRequestedRepo clones --gh-repo into the session's workspace so the agent
// never starts on an empty one, and returns how long that took. It is a no-op
// without --gh-repo.
//
// It runs through the sandbox exec API with an argument list. A failure is an
// error: the caller must not send the prompt, and the session is left running.
func cloneRequestedRepo(c *CmdConfig, sess *do.HostedAgentSession, src *agentCreationSource, prog *creationProgress) (time.Duration, error) {
	if src == nil || strings.TrimSpace(src.repo) == "" {
		return 0, nil
	}
	ref, err := normalizeHarnessRepoRef(src.repo)
	if err != nil {
		return 0, err
	}
	cloneURL, dest := repoClonePaths(ref)
	sessionRef := displaySessionRef(sess)
	heldBack := promptHeldBack(src)
	retryRef := sessionRef + regionFlagFor(sess)
	hasToken := manifestDeclaresName(src.manifest, githubTokenSecret)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// The check is idempotent, so it uses the normal retrying client: a
	// transient error must not look like a reused workspace. An exec error here
	// (as opposed to a non-zero exit) is not proof the folder is missing, so it
	// falls through to the clone, which fails loudly if the folder is there.
	check, checkErr := c.HostedAgents().ExecInSandbox(ctx, sess.SessionID, &godo.HostedAgentSandboxExecRequest{
		Argv:           []string{"test", "-e", dest},
		TimeoutSeconds: repoCheckTimeoutSeconds,
	})
	if ctx.Err() != nil {
		return 0, repoCloneInterrupted(sessionRef, heldBack)
	}
	if checkErr == nil && check != nil && check.ExitCode == 0 {
		prog.step("Using existing " + dest)
		return 0, nil
	}

	// The clone is not idempotent: a retried one would fail with "already
	// exists". Swap in the no-retry client for it only, and restore the normal
	// one because the prompt that follows is safe to retry.
	prevService := c.HostedAgents
	defer func() { c.HostedAgents = prevService }()
	if err := disableRetriesForExec(c); err != nil {
		return 0, err
	}

	prog.wait(fmt.Sprintf("Cloning %s…", ref))
	started := creationClock()
	resp, err := c.HostedAgents().ExecInSandbox(ctx, sess.SessionID, &godo.HostedAgentSandboxExecRequest{
		Argv:           []string{"sh", "-c", repoCloneScript, "sh", cloneURL, dest},
		TimeoutSeconds: repoCloneTimeoutSeconds,
	})
	if ctx.Err() != nil {
		prog.stopSpin()
		return 0, repoCloneInterrupted(sessionRef, heldBack)
	}
	if err != nil {
		// A DigitalOcean API error says nothing about git, so it carries no
		// private-repository advice.
		prog.stopSpin()
		return 0, repoCloneError(ref, cloneURL, dest, sessionRef, retryRef, stripAPIErrorNoise(err.Error()), cloneAPIError, hasToken, heldBack)
	}
	if resp == nil || resp.ExitCode != 0 {
		prog.stopSpin()
		out := ""
		if resp != nil {
			out = resp.Stderr
			if strings.TrimSpace(out) == "" {
				out = resp.Stdout
			}
			if strings.TrimSpace(out) == "" {
				out = fmt.Sprintf("git exited with status %d", resp.ExitCode)
			}
		}
		kind := cloneGitError
		if resp != nil && (resp.ExitCode == 127 || gitMissingRE.MatchString(out)) {
			kind = cloneGitMissing
		}
		return 0, repoCloneError(ref, cloneURL, dest, sessionRef, retryRef, out, kind, hasToken, heldBack)
	}

	elapsed := creationClock().Sub(started)
	prog.ok(fmt.Sprintf("Cloned %s into %s (%s)", ref, dest, elapsed.Truncate(time.Second)))
	return elapsed, nil
}

// repoCloneInterrupted reports a Ctrl-C during the clone. The session stays up;
// the exit status matches the other interrupted commands.
func repoCloneInterrupted(sessionRef string, heldBack bool) error {
	fmt.Fprintf(execStderr, "Interrupted. The session %s is still running%s.\n", sessionRef, promptNote(heldBack))
	execExit(promptInterruptExit)
	return ErrExitSilently
}

// promptNote is the clause saying the prompt was withheld, present only when
// there was a prompt for doctl to withhold.
func promptNote(heldBack bool) string {
	if heldBack {
		return " and the prompt was not sent"
	}
	return ""
}

var (
	// Token shapes git or a proxy could echo: GitHub token prefixes, bearer
	// values, the header itself, and credentials embedded in a URL.
	repoSecretPatterns = []*regexp.Regexp{
		regexp.MustCompile(`github_pat_[A-Za-z0-9_]+`),
		regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]+`),
		regexp.MustCompile(`(?i)authorization:[^\n]*`),
		regexp.MustCompile(`(?i)bearer\s+\S+`),
		regexp.MustCompile(`://[^/@\s]+@`),
	}
	repoAuthHintRE = regexp.MustCompile(`(?i)authentication|could not read username|terminal prompts disabled|not found|invalid credentials|permission denied|\b40[134]\b`)
)

// redactRepoSecrets masks anything in git's output that looks like a credential.
func redactRepoSecrets(s string) string {
	for _, re := range repoSecretPatterns {
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			if strings.HasPrefix(m, "://") {
				return "://***@"
			}
			return "[redacted]"
		})
	}
	return s
}

// lastRepoErrorLines returns up to n of the last non-empty lines of s, redacted
// and capped at 1000 characters without splitting a character.
func lastRepoErrorLines(s string, n int) string {
	var lines []string
	for _, l := range strings.Split(redactRepoSecrets(s), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	out := strings.Join(lines, "\n")
	if r := []rune(out); len(r) > 1000 {
		out = string(r[len(r)-1000:])
	}
	return out
}

// cloneFailure says what kind of failure the reason text describes, which
// decides what advice follows it.
type cloneFailure int

const (
	cloneAPIError   cloneFailure = iota // the exec call itself failed; says nothing about git
	cloneGitError                       // git ran and exited non-zero
	cloneGitMissing                     // the image has no git
)

var (
	gitMissingRE = regexp.MustCompile(`(?i)git: (command )?not found|git: no such file`)
	// urlErrorRE matches the `Post "https://…": ` prefix a *url.Error carries.
	urlErrorRE = regexp.MustCompile(`^(?:Get|Post|Put|Patch|Delete|Head) "[^"]*": `)
)

// stripAPIErrorNoise drops transport detail (METHOD URL: status and the
// *url.Error prefix) from an API error, leaving the message a person can use.
func stripAPIErrorNoise(s string) string {
	return urlErrorRE.ReplaceAllString(stripGodoTransportNoise(s), "")
}

// regionFlagFor is the ` --region X` suffix the retry command needs when the
// session lives away from the default host: the standalone exec resolves the
// session on the host it is pointed at. It uses the region the server reported
// for the session, else the --region flag.
func regionFlagFor(sess *do.HostedAgentSession) string {
	region := ""
	if sess != nil && sess.HostedAgentSession != nil {
		region = sess.Region
	}
	if region == "" {
		region, _ = agentRegionFlag()
	}
	if region == "" || region == defaultAgentRegion {
		return ""
	}
	return " --region " + region
}

// repoCloneError builds the failure message. sessionRef names the session;
// retryRef is the same plus any --region the retry command needs.
func repoCloneError(ref, cloneURL, dest, sessionRef, retryRef, reasonText string, kind cloneFailure, hasToken, heldBack bool) error {
	reason := lastRepoErrorLines(reasonText, 5)
	var b strings.Builder
	fmt.Fprintf(&b, "Could not clone %s into %s.\n", ref, dest)
	if reason != "" {
		fmt.Fprintf(&b, "%s\n", reason)
	}
	fmt.Fprintf(&b, "The session %s is still running%s.\n", sessionRef, promptNote(heldBack))
	fmt.Fprintf(&b, "Retry the clone with: %s exec %s -- git clone --depth 1 %s %s", agentCLI, retryRef, cloneURL, dest)
	switch {
	case kind == cloneGitMissing:
		b.WriteString("\ngit is not available in this session's image, so the repository cannot be cloned there.")
	case kind == cloneGitError && repoAuthHintRE.MatchString(reasonText) && hasToken:
		b.WriteString("\nIf the repository is private, the GitHub connection's token may not have access to it; check the connection's permissions for this repository.")
	case kind == cloneGitError && repoAuthHintRE.MatchString(reasonText):
		fmt.Fprintf(&b, "\nIf the repository is private, connect GitHub (`%s auth github`) and run the command again; doctl then gives the session your GitHub token.", agentCLI)
	}
	return &agentPrettyError{title: "Repository clone failed", reason: b.String()}
}

// manifestDeclaresName reports whether a flat or legacy manifest already
// declares name as a secret or an env entry.
func manifestDeclaresName(manifest []byte, name string) bool {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil || doc == nil {
		return false
	}
	container := doc
	if _, legacy := doc["apiVersion"]; legacy {
		spec, ok := yamlMap(doc["spec"])
		if !ok {
			return false
		}
		container = spec
	}
	if env, ok := yamlStringMap(container["env"]); ok {
		if _, found := env[name]; found {
			return true
		}
	}
	raw := container["secrets"]
	if m, ok := yamlMap(raw); ok && isYAMLMapping(raw) {
		_, found := m[name]
		return found
	}
	for _, slot := range yamlListOrNil(raw) {
		if m, ok := yamlMap(slot); ok {
			if n, _ := yamlString(m["name"]); strings.TrimSpace(n) == name {
				return true
			}
		}
	}
	return false
}

// addGitHubTokenSlot declares GITHUB_TOKEN: oauth/github in a flat manifest so
// the session receives the user's GitHub connection. It reports whether it
// added the slot, and whether it skipped only because the manifest uses the
// deprecated envelope format, whose slots are lists with a separate source
// field this shorthand does not describe. A manifest that already declares the
// name is left alone.
func addGitHubTokenSlot(src *agentCreationSource) (added, skippedLegacy bool, err error) {
	if src.configID != "" || len(src.manifest) == 0 || manifestDeclaresName(src.manifest, githubTokenSecret) {
		return false, false, nil
	}
	var probe map[string]any
	if err := yaml.Unmarshal(src.manifest, &probe); err != nil {
		return false, false, nil
	}
	if _, legacy := probe["apiVersion"]; legacy {
		return false, true, nil
	}
	out, err := injectManifestSecrets(src.manifest, map[string]string{githubTokenSecret: githubOAuthRef})
	if err != nil {
		return false, false, err
	}
	src.manifest = out
	return true, false, nil
}
