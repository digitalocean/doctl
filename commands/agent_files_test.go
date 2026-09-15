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
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// A listing of one directory and one file, in the order find walks them (which
// is not the order they are displayed in). find leads with the directory it was
// pointed at, here ".", which a listing is not supposed to show.
const testListingStdout = "d\t4096\t1756400000.0000000000\t.\n" +
	"d\t4096\t1756500000.0000000000\t./src\n" +
	"f\t1240\t1756586400.0000000000\t./go.mod\n"

// withTextOutput pins the global output format, which is otherwise only set as
// a side effect of registering the root flags.
func withTextOutput(t *testing.T) {
	t.Helper()
	prev := Output
	Output = "text"
	t.Cleanup(func() { Output = prev })
}

func TestAgentFilesCommand(t *testing.T) {
	cmd := AgentFiles()
	require.NotNil(t, cmd)
	assertCommandNames(t, cmd, "ls", "cat")
}

// `ls` lives under `files` because the flat tree already spells `list` as `ls`
// (sessions), and one name cannot mean both.
func TestAgentFilesLsDoesNotShadowSessionList(t *testing.T) {
	cmd := Agents()

	found, _, err := cmd.Find([]string{"ls"})
	require.NoError(t, err)
	assert.Equal(t, "list", found.Name(), "the flat `ls` must still list sessions")

	found, _, err = cmd.Find([]string{"files", "ls"})
	require.NoError(t, err)
	assert.Equal(t, "ls", found.Name())
}

func TestAgentFilesLs(t *testing.T) {
	withTextOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, req *godo.HostedAgentSandboxExecRequest) (*godo.HostedAgentSandboxExecResponse, error) {
				assert.Equal(t, []string{
					"find", ".", "-maxdepth", "1", "-printf", workspaceListFormat,
				}, req.Argv, "with no path the workspace root is listed, one level deep")
				// Left unset so the sandbox resolves "." against the workspace
				// root, the same base `--workspace-path` uses.
				assert.Empty(t, req.Workdir)
				return &godo.HostedAgentSandboxExecResponse{Stdout: testListingStdout}, nil
			})

		config.Args = []string{testExecSessionID}
		require.NoError(t, RunAgentsFilesLs(config))

		body := out.String()
		assert.Contains(t, body, "Type")
		assert.Contains(t, body, "Modified")
		// The directory being listed is not one of its own contents.
		assert.NotContains(t, body, "\t.\n")
		// Sorted by path, so the file precedes the directory find listed first.
		assert.Less(t, indexOf(body, "go.mod"), indexOf(body, "src"))
		assert.Contains(t, body, "dir")
		assert.Contains(t, body, "file")
		assert.Contains(t, body, "4096")
		assert.Contains(t, body, time.Unix(1756500000, 0).UTC().Format("2006-01-02T15:04:05Z"))
	})
}

func TestAgentFilesLsListsAGivenDirectory(t *testing.T) {
	withTextOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = io.Discard

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, req *godo.HostedAgentSandboxExecRequest) (*godo.HostedAgentSandboxExecResponse, error) {
				assert.Equal(t, "src", req.Argv[1], "the path argument is the directory to list")
				assert.Equal(t, int64(15), req.TimeoutSeconds)
				return &godo.HostedAgentSandboxExecResponse{}, nil
			})

		config.Args = []string{testExecSessionID, "src"}
		config.Doit.Set(config.NS, doctl.ArgAgentExecTimeout, 15)
		require.NoError(t, RunAgentsFilesLs(config))
	})
}

// Pointing `ls` at a file answers with that file, the way `ls file` does.
// find reports the path it was given and nothing below it, so dropping that row
// unconditionally would answer a file with an empty table and exit 0 — a result
// indistinguishable from an empty directory.
func TestAgentFilesLsOnAFileListsThatFile(t *testing.T) {
	withTextOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(&godo.HostedAgentSandboxExecResponse{
				Stdout: "f\t169\t1756500000.0000000000\topencode.log\n",
			}, nil)

		config.Args = []string{testExecSessionID, "opencode.log"}
		require.NoError(t, RunAgentsFilesLs(config))

		assert.Contains(t, out.String(), "opencode.log")
		assert.Contains(t, out.String(), "169")
	})
}

// --recursive drops the depth limit rather than raising it, so a whole subtree
// comes back in one call.
func TestAgentFilesLsRecursive(t *testing.T) {
	withTextOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = io.Discard

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, req *godo.HostedAgentSandboxExecRequest) (*godo.HostedAgentSandboxExecResponse, error) {
				assert.Equal(t, []string{
					"find", "src", "-printf", workspaceListFormat,
				}, req.Argv)
				return &godo.HostedAgentSandboxExecResponse{}, nil
			})

		config.Args = []string{testExecSessionID, "src"}
		config.Doit.Set(config.NS, doctl.ArgAgentLsRecursive, true)
		require.NoError(t, RunAgentsFilesLs(config))
	})
}

// A listing is structured output, not a stream to reproduce, so a guest failure
// becomes a doctl error carrying the guest's own explanation.
func TestAgentFilesLsSurfacesGuestFailure(t *testing.T) {
	withTextOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = io.Discard

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(&godo.HostedAgentSandboxExecResponse{
				ExitCode: 1,
				Stderr:   "find: 'nope': No such file or directory\n",
			}, nil)

		config.Args = []string{testExecSessionID, "nope"}
		err := RunAgentsFilesLs(config)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "nope")
		assert.Contains(t, err.Error(), "No such file or directory")
		assert.NotErrorIs(t, err, ErrExitSilently, "a listing failure explains itself")
	})
}

func TestAgentFilesLsReportsUnreadableListings(t *testing.T) {
	withTextOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = io.Discard

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(&godo.HostedAgentSandboxExecResponse{Stdout: "not a listing line\n"}, nil)

		config.Args = []string{testExecSessionID}
		err := RunAgentsFilesLs(config)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a listing line")
	})
}

func TestAgentFilesLsJSONOutput(t *testing.T) {
	prev := Output
	Output = "json"
	t.Cleanup(func() { Output = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(&godo.HostedAgentSandboxExecResponse{Stdout: testListingStdout}, nil)

		config.Args = []string{testExecSessionID}
		require.NoError(t, RunAgentsFilesLs(config))

		var got []struct {
			Type       string    `json:"type"`
			SizeBytes  int64     `json:"size_bytes"`
			ModifiedAt time.Time `json:"modified_at"`
			Path       string    `json:"path"`
		}
		require.NoError(t, json.Unmarshal(out.Bytes(), &got), "body: %q", out.String())
		require.Len(t, got, 2)
		assert.Equal(t, "file", got[0].Type)
		assert.Equal(t, "go.mod", got[0].Path)
		assert.Equal(t, int64(1240), got[0].SizeBytes)
		assert.Equal(t, "dir", got[1].Type)
		assert.Equal(t, "src", got[1].Path)
		assert.Equal(t, time.Unix(1756500000, 0).UTC(), got[1].ModifiedAt.UTC())
	})
}

func TestAgentFilesLsValidatesArgs(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = io.Discard

		// No session to list, and more paths than one directory: neither can
		// be run, so the API is never called.
		config.Args = []string{}
		assert.Error(t, RunAgentsFilesLs(config))

		config.Args = []string{testExecSessionID, "src", "extra"}
		assert.Error(t, RunAgentsFilesLs(config))
	})
}

func TestAgentFilesCat(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, stderr, exitCode := captureExec(t, config)

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, req *godo.HostedAgentSandboxExecRequest) (*godo.HostedAgentSandboxExecResponse, error) {
				// `--` so a path beginning with a dash stays a path.
				assert.Equal(t, []string{"cat", "--", "src/main.go"}, req.Argv)
				assert.Empty(t, req.Workdir)
				return &godo.HostedAgentSandboxExecResponse{Stdout: "package main\n"}, nil
			})

		config.Args = []string{testExecSessionID, "src/main.go"}
		require.NoError(t, RunAgentsFilesCat(config))

		assert.Equal(t, "package main\n", stdout.String())
		assert.Empty(t, stderr.String())
		assert.Equal(t, noExit, *exitCode, "reading a file that exists must not force an exit status")
	})
}

// The file's bytes are the contract: they arrive verbatim, with the streams kept
// apart and no newline added by doctl, so `cat` composes in a pipeline.
func TestAgentFilesCatPassesContentThroughVerbatim(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, stderr, _ := captureExec(t, config)

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(&godo.HostedAgentSandboxExecResponse{
				Stdout: "no trailing newline",
				Stderr: "cat: warning\n",
			}, nil)

		config.Args = []string{testExecSessionID, "notes.txt"}
		require.NoError(t, RunAgentsFilesCat(config))

		assert.Equal(t, "no trailing newline", stdout.String())
		assert.Equal(t, "cat: warning\n", stderr.String())
	})
}

// A missing file is a non-zero exit rather than a doctl error, so `&&` chains
// and `if` tests behave and the guest's message is not printed over.
func TestAgentFilesCatPropagatesGuestExitCode(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, stderr, exitCode := captureExec(t, config)

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(&godo.HostedAgentSandboxExecResponse{
				ExitCode: 1,
				Stderr:   "cat: nope.txt: No such file or directory\n",
			}, nil)

		config.Args = []string{testExecSessionID, "nope.txt"}
		err := RunAgentsFilesCat(config)

		assert.Equal(t, 1, *exitCode)
		assert.ErrorIs(t, err, ErrExitSilently)
		assert.Empty(t, stdout.String())
		assert.Equal(t, "cat: nope.txt: No such file or directory\n", stderr.String())
	})
}

func TestAgentFilesCatJSONOutput(t *testing.T) {
	prev := Output
	Output = "json"
	t.Cleanup(func() { Output = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, _, exitCode := captureExec(t, config)

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(&godo.HostedAgentSandboxExecResponse{Stdout: "package main\n"}, nil)

		config.Args = []string{testExecSessionID, "src/main.go"}
		require.NoError(t, RunAgentsFilesCat(config))

		var got struct {
			Path      string `json:"path"`
			Content   string `json:"content"`
			SizeBytes int    `json:"size_bytes"`
		}
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &got), "body: %q", stdout.String())
		assert.Equal(t, "src/main.go", got.Path)
		assert.Equal(t, "package main\n", got.Content)
		assert.Equal(t, len("package main\n"), got.SizeBytes)
		assert.Equal(t, noExit, *exitCode)
	})
}

func TestAgentFilesCatRequiresAPath(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = io.Discard

		// A session with no path names no file to read, so the API is never
		// called.
		config.Args = []string{testExecSessionID}
		assert.Error(t, RunAgentsFilesCat(config))

		config.Args = []string{}
		assert.Error(t, RunAgentsFilesCat(config))
	})
}

func TestAgentFilesSurfacesTransportErrors(t *testing.T) {
	withTextOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = io.Discard

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(nil, errors.New("session is not operable"))

		config.Args = []string{testExecSessionID}
		err := RunAgentsFilesLs(config)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "session is not operable")
	})
}

func TestWorkspaceEntryType(t *testing.T) {
	assert.Equal(t, "file", workspaceEntryType("f"))
	assert.Equal(t, "dir", workspaceEntryType("d"))
	assert.Equal(t, "symlink", workspaceEntryType("l"))
	// An unrecognized code is passed through rather than dropped, so a listing
	// never hides an entry.
	assert.Equal(t, "U", workspaceEntryType("U"))
}

// A name containing spaces is common enough that the listing has to survive it:
// the path is the rest of the record, not a field.
func TestParseWorkspaceListingKeepsPathsIntact(t *testing.T) {
	entries, err := parseWorkspaceListing("f\t12\t1756500000.0000000000\tnotes/my notes.txt\n", ".")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "notes/my notes.txt", entries[0].Path)
	assert.Equal(t, int64(12), entries[0].SizeBytes)
}

// Only the listed directory's own row is dropped: a subdirectory that happens
// to share its name with the target, and the target itself when it is a file,
// both belong in the listing.
func TestParseWorkspaceListingDropsOnlyTheListedDirectory(t *testing.T) {
	entries, err := parseWorkspaceListing(
		"d\t4096\t1756500000.0\tsrc\n"+
			"d\t4096\t1756500000.0\tsrc/src\n"+
			"f\t12\t1756500000.0\tsrc/main.go\n",
		"src",
	)
	require.NoError(t, err)

	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	assert.Equal(t, []string{"src/main.go", "src/src"}, paths)

	// The same path as a file is the answer, not the container of the answer.
	entries, err = parseWorkspaceListing("f\t12\t1756500000.0\tsrc\n", "src")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "src", entries[0].Path)
}

// Output cut off at the exec cap ends mid-record. Reporting that is the whole
// point: the entries that did arrive look like a complete listing, so handing
// them back would have a caller act on a workspace it only half saw.
func TestParseWorkspaceListingRejectsTruncatedOutput(t *testing.T) {
	_, err := parseWorkspaceListing(
		"f\t12\t1756500000.0\tone.txt\n"+
			"f\t34\t1756500000.0\ttwo",
		".",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "truncated")
	assert.Contains(t, err.Error(), "--recursive", "the message has to say what to do about it")

	// A listing that ends cleanly is not truncated, and neither is an empty one.
	_, err = parseWorkspaceListing("f\t12\t1756500000.0\tone.txt\n", ".")
	assert.NoError(t, err)
	_, err = parseWorkspaceListing("", ".")
	assert.NoError(t, err)
}

// The sub-second digits are the guest's, not float64's nearest approximation
// of them: an mtime of .588 must not come back as .588000059.
func TestParseWorkspaceListingKeepsReportedPrecision(t *testing.T) {
	for _, tc := range []struct {
		field string
		want  time.Time
	}{
		{"1756500000.588000000", time.Unix(1756500000, 588000000).UTC()},
		{"1756500000.5", time.Unix(1756500000, 500000000).UTC()},
		{"1756500000", time.Unix(1756500000, 0).UTC()},
		// Finer than doctl can render, so truncated rather than rejected.
		{"1756500000.1234567891234", time.Unix(1756500000, 123456789).UTC()},
	} {
		got, err := parseWorkspaceListingTime(tc.field)
		require.NoError(t, err, tc.field)
		assert.Equal(t, tc.want, got, tc.field)
	}

	_, err := parseWorkspaceListingTime("not-a-time")
	assert.Error(t, err)
}

// Listing the root has find echo "." back as a prefix on every row, which is
// noise in a column meant to be read and pasted into the next command.
func TestParseWorkspaceListingDropsTheCwdPrefix(t *testing.T) {
	entries, err := parseWorkspaceListing("f\t169\t1756500000.0000000000\t./opencode.log\n", ".")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "opencode.log", entries[0].Path)
}

// indexOf is strings.Index, named for what the assertions above are checking.
func indexOf(haystack, needle string) int {
	return bytes.Index([]byte(haystack), []byte(needle))
}
