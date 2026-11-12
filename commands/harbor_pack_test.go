/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeHarborFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		mode := os.FileMode(0o644)
		if strings.HasSuffix(rel, ".sh") {
			mode = 0o755
		}
		require.NoError(t, os.WriteFile(p, []byte(content), mode))
	}
}

// harborDigestCases are task folders whose file lists and digests were
// produced by Harbor's own Packager.compute_content_hash (pathspec 1.1.1), so
// doctl publishes the same files under the same digest as `harbor publish`.
var harborDigestCases = []struct {
	name   string
	files  map[string]string
	want   []string
	digest string
}{
	{
		name: "default ignores and harborignore scopes",
		files: map[string]string{
			"task.toml":                     "[task]\nname = \"my-org/basic-task\"\n",
			"instruction.md":                "Fix the bug.\n",
			"README.md":                     "# Basic\n",
			"notes.txt":                     "not published\n",
			"environment/Dockerfile":        "FROM python:3.12\n",
			"environment/__pycache__/x.pyc": "bytecode",
			"environment/.DS_Store":         "junk",
			"environment/editor.swp":        "junk",
			"environment/backup~":           "junk",
			"environment/.harborignore":     "data.tmp\n",
			"environment/data.tmp":          "kept: .harborignore does not apply here",
			"tests/test.sh":                 "#!/bin/sh\nexit 0\n",
			"tests/.harborignore":           "fixtures/large/\n*.tmp\n!keep.tmp\n",
			"tests/fixtures/large/a.bin":    "big",
			"tests/fixtures/small/b.bin":    "small",
			"tests/a.tmp":                   "tmp",
			"tests/keep.tmp":                "keep",
			"solution/solve.sh":             "#!/bin/sh\necho fix\n",
			"solution/.harborignore":        "secret.txt\n",
			"solution/secret.txt":           "hidden",
			"solution/nested/secret.txt":    "hidden too",
			"solution/nested/public.txt":    "public",
		},
		want:   []string{"README.md", "environment/Dockerfile", "environment/data.tmp", "instruction.md", "solution/nested/public.txt", "solution/solve.sh", "task.toml", "tests/fixtures/small/b.bin", "tests/keep.tmp", "tests/test.sh"},
		digest: "sha256:0b01a6d838c3ff69b8e8187b40e2044d23239aed94377e79eabaa484b0346272",
	},
	{
		name: "root gitignore replaces the defaults",
		files: map[string]string{
			"task.toml":                     "[task]\nname = \"gitignore-task\"\n",
			"instruction.md":                "Do it.\n",
			".gitignore":                    "*.log\n!important.log\nbuild/\n/environment/cache\n# comment\n\n",
			"environment/Dockerfile":        "FROM alpine\n",
			"environment/app.log":           "log",
			"environment/important.log":     "important",
			"environment/build/out.o":       "obj",
			"environment/cache/c":           "cache",
			"environment/src/cache/d":       "nested cache",
			"environment/__pycache__/m.pyc": "kept: no default ignores when .gitignore exists",
			"tests/test.sh":                 "#!/bin/sh\n",
		},
		want:   []string{"environment/Dockerfile", "environment/__pycache__/m.pyc", "environment/important.log", "environment/src/cache/d", "instruction.md", "task.toml", "tests/test.sh"},
		digest: "sha256:e95f227cea7fe674ef2c8c29742b1ce23c86185a6213b9a90dee02ebaf7a483a",
	},
	{
		name: "multi-step task",
		files: map[string]string{
			"task.toml":                        "[task]\nname = \"steps-task\"\n",
			"steps/one/instruction.md":         "step one",
			"steps/one/tests/test.sh":          "#!/bin/sh\n",
			"steps/one/tests/.harborignore":    "*.snap\n",
			"steps/one/tests/x.snap":           "snapshot",
			"steps/one/tests/y.txt":            "kept",
			"steps/one/workdir/f.txt":          "workdir",
			"steps/one/workdir/z.snap":         "kept: outside the tests scope",
			"steps/two/solution/solve.sh":      "#!/bin/sh\n",
			"steps/two/solution/.harborignore": "/notes.md\n",
			"steps/two/solution/notes.md":      "notes",
			"steps/two/solution/sub/notes.md":  "kept: anchored pattern",
			"environment/Dockerfile":           "FROM alpine\n",
		},
		want:   []string{"environment/Dockerfile", "steps/one/instruction.md", "steps/one/tests/test.sh", "steps/one/tests/y.txt", "steps/one/workdir/f.txt", "steps/one/workdir/z.snap", "steps/two/solution/solve.sh", "steps/two/solution/sub/notes.md", "task.toml"},
		digest: "sha256:b7dd41e2f6abec5c0ba0e6700db164ee77f5969cd76d85c94080ca209670627f",
	},
	{
		name: "inherited rules and excluded folders",
		files: map[string]string{
			"task.toml":                "[task]\nname = \"inherit-task\"\n",
			"instruction.md":           "x",
			"environment/Dockerfile":   "FROM alpine\n",
			"tests/test.sh":            "#!/bin/sh\n",
			"tests/.harborignore":      "data/\n*.cache\n",
			"tests/data/.harborignore": "!keep\n",
			"tests/data/keep":          "excluded: its folder is excluded",
			"tests/sub/.harborignore":  "*.txt\n!b.cache\n",
			"tests/sub/a.txt":          "excluded by sub",
			"tests/sub/b.cache":        "re-included by sub",
			"tests/sub/c.cache":        "excluded by tests",
			"tests/sub/d.md":           "kept",
			"tests/other.txt":          "kept: sub rules do not apply here",
		},
		want:   []string{"environment/Dockerfile", "instruction.md", "task.toml", "tests/other.txt", "tests/sub/b.cache", "tests/sub/d.md", "tests/test.sh"},
		digest: "sha256:51da6a8810ea5b6f8b4b2ad2cc07f6d99ea85fd21c49a1a851a59e832a9b491b",
	},
	{
		name: "sorting and unicode",
		files: map[string]string{
			"task.toml":              "[task]\nname = \"sort-task\"\n",
			"environment/a.txt":      "a",
			"environment/B.txt":      "B",
			"environment/\u00c4.txt": "umlaut",
			"environment/z.txt":      "z",
			"environment/z/ab":       "z/ab",
			"environment/z-1":        "z-1",
			"tests/test.sh":          "#!/bin/sh\n",
		},
		want:   []string{"environment/B.txt", "environment/a.txt", "environment/z-1", "environment/z.txt", "environment/z/ab", "environment/\u00c4.txt", "task.toml", "tests/test.sh"},
		digest: "sha256:47d66ee84773be151b7e39f3085e828612f0575de42799b5bf9b9a51ed3f78c9",
	},
}

func TestHarborDigestMatchesHarbor(t *testing.T) {
	for _, tc := range harborDigestCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeHarborFixture(t, root, tc.files)

			pkg, err := packHarborTask(root, "")
			require.NoError(t, err)
			defer pkg.Close()

			got := make([]string, len(pkg.Files))
			for i, f := range pkg.Files {
				got[i] = f.Path
			}
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.digest, pkg.Digest)
		})
	}
}

func TestHarborArchiveFormat(t *testing.T) {
	root := t.TempDir()
	writeHarborFixture(t, root, map[string]string{
		"task.toml":              "[task]\nname = \"fmt\"\n",
		"instruction.md":         "hi",
		"environment/Dockerfile": "FROM alpine\n",
		"tests/test.sh":          "#!/bin/sh\n",
	})

	pkg, err := packHarborTask(root, "")
	require.NoError(t, err)
	defer pkg.Close()

	again, err := packHarborTask(root, "")
	require.NoError(t, err)
	defer again.Close()
	a, err := os.ReadFile(pkg.ArchivePath)
	require.NoError(t, err)
	b, err := os.ReadFile(again.ArchivePath)
	require.NoError(t, err)
	assert.Equal(t, a, b, "packing the same folder twice gives the same bytes")

	f, err := os.Open(pkg.ArchivePath)
	require.NoError(t, err)
	defer f.Close()
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	assert.True(t, gz.ModTime.IsZero() || gz.ModTime.Unix() == 0)

	tr := tar.NewReader(gz)
	modes := map[string]int64{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		assert.Equal(t, byte(tar.TypeReg), hdr.Typeflag)
		assert.Zero(t, hdr.Uid)
		assert.Zero(t, hdr.Gid)
		assert.Empty(t, hdr.Uname)
		assert.Empty(t, hdr.Gname)
		assert.Equal(t, int64(0), hdr.ModTime.Unix())
		modes[hdr.Name] = hdr.Mode
	}
	assert.Equal(t, map[string]int64{
		"environment/Dockerfile": 0o644,
		"instruction.md":         0o644,
		"task.toml":              0o644,
		"tests/test.sh":          0o755,
	}, modes)

	sums, err := extractHarborArchive(pkg.ArchivePath, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, pkg.Digest, harborContentDigest(sums), "the digest can be recomputed from the archive contents")
}

func TestResolveHarborTaskName(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		toml     string
		override string
		want     string
		warns    int
		err      string
	}{
		{name: "flag wins", dir: "folder", toml: "[task]\nname = \"org/from-toml\"\n", override: "from-flag", want: "from-flag"},
		{name: "toml with org prefix", dir: "folder", toml: "[task]\nname = \"org/from-toml\"\n", want: "from-toml", warns: 1},
		{name: "toml without prefix", dir: "folder", toml: "[task]\nname = \"plain\"\n", want: "plain"},
		{name: "folder name", dir: "my-task", toml: "[metadata]\n", want: "my-task"},
		{name: "invalid folder name", dir: "My Task", toml: "", err: "is invalid"},
		{name: "invalid flag", dir: "folder", toml: "", override: "-bad", err: "from --name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), tt.dir)
			writeHarborFixture(t, root, map[string]string{"task.toml": tt.toml})

			pkg, err := packHarborTask(root, tt.override)
			if tt.err != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.err)
				return
			}
			require.NoError(t, err)
			defer pkg.Close()
			assert.Equal(t, tt.want, pkg.Name)
			assert.Len(t, pkg.Warnings, tt.warns)
		})
	}
}

func TestPackHarborTaskRejects(t *testing.T) {
	t.Run("no task.toml", func(t *testing.T) {
		_, err := packHarborTask(t.TempDir(), "x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task.toml not found")
	})

	t.Run("not a folder", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "task.tar.gz")
		require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
		_, err := packHarborTask(file, "x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not a folder")
	})

	t.Run("symlink out of the task", func(t *testing.T) {
		root := t.TempDir()
		outside := filepath.Join(t.TempDir(), "secret")
		require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o644))
		writeHarborFixture(t, root, map[string]string{"task.toml": ""})
		require.NoError(t, os.MkdirAll(filepath.Join(root, "tests"), 0o755))
		require.NoError(t, os.Symlink(outside, filepath.Join(root, "tests", "leak")))

		_, err := packHarborTask(root, "x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "outside the task folder")
	})

	// Checked against Harbor's Packager: a symlinked top-level folder is
	// walked, a symlinked folder below it is not, and a symlinked file is
	// packed with its target's content.
	t.Run("symlinks inside the task", func(t *testing.T) {
		root := t.TempDir()
		writeHarborFixture(t, root, map[string]string{"task.toml": "", "tests/real.txt": "real", "solution/inner/a": "x"})
		require.NoError(t, os.Symlink("real.txt", filepath.Join(root, "tests", "link.txt")))
		require.NoError(t, os.Symlink(filepath.Join(root, "tests"), filepath.Join(root, "environment")))
		require.NoError(t, os.Symlink("inner", filepath.Join(root, "solution", "linkdir")))

		pkg, err := packHarborTask(root, "x")
		require.NoError(t, err)
		defer pkg.Close()
		var paths []string
		for _, f := range pkg.Files {
			paths = append(paths, f.Path)
		}
		assert.Equal(t, []string{"environment/link.txt", "environment/real.txt", "solution/inner/a", "task.toml", "tests/link.txt", "tests/real.txt"}, paths)
	})

	t.Run("too many files", func(t *testing.T) {
		root := t.TempDir()
		files := map[string]string{"task.toml": ""}
		for i := 0; i <= harborMaxFiles; i++ {
			files[fmt.Sprintf("environment/f%06d", i)] = ""
		}
		writeHarborFixture(t, root, files)
		_, err := packHarborTask(root, "x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "the limit is 10000")
	})
}

// harborPatternCases were checked against pathspec 1.1.1: basic is
// PathSpec.from_lines("gitignore", ...), git is GitIgnoreSpec.from_lines.
var harborPatternCases = []struct {
	flavour  harborIgnoreFlavour
	patterns []string
	path     string
	excluded bool
}{
	{harborIgnoreBasic, []string{"*.log"}, "a.log", true},
	{harborIgnoreBasic, []string{"*.log"}, "dir/a.log", true},
	{harborIgnoreBasic, []string{"*.log"}, "a.log/x", true},
	{harborIgnoreBasic, []string{"*.log"}, "notes", false},
	{harborIgnoreBasic, []string{"/root.txt"}, "root.txt", true},
	{harborIgnoreBasic, []string{"/root.txt"}, "sub/root.txt", false},
	{harborIgnoreBasic, []string{"build/"}, "build", false},
	{harborIgnoreBasic, []string{"build/"}, "build/x", true},
	{harborIgnoreBasic, []string{"build/"}, "a/build/x", true},
	{harborIgnoreBasic, []string{"build/"}, "build.txt", false},
	{harborIgnoreBasic, []string{"doc/*.md"}, "doc/a.md", true},
	{harborIgnoreBasic, []string{"doc/*.md"}, "doc/sub/a.md", false},
	{harborIgnoreBasic, []string{"doc/*.md"}, "x/doc/a.md", false},
	{harborIgnoreBasic, []string{"**/logs"}, "logs", true},
	{harborIgnoreBasic, []string{"**/logs"}, "a/logs", true},
	{harborIgnoreBasic, []string{"**/logs"}, "a/b/logs/x", true},
	{harborIgnoreBasic, []string{"a/**/b"}, "a/b", true},
	{harborIgnoreBasic, []string{"a/**/b"}, "a/x/b", true},
	{harborIgnoreBasic, []string{"a/**/b"}, "a/x/y/b", true},
	{harborIgnoreBasic, []string{"a/**/b"}, "xa/b", false},
	{harborIgnoreBasic, []string{"foo/**"}, "foo", false},
	{harborIgnoreBasic, []string{"foo/**"}, "foo/", true},
	{harborIgnoreBasic, []string{"foo/**"}, "foo/x", true},
	{harborIgnoreBasic, []string{"foo/**"}, "foo/x/y", true},
	{harborIgnoreBasic, []string{"*"}, "a", true},
	{harborIgnoreBasic, []string{"*"}, "a/b", true},
	{harborIgnoreBasic, []string{"*/"}, "a", false},
	{harborIgnoreBasic, []string{"*/"}, "a/", true},
	{harborIgnoreBasic, []string{"*/"}, "a/b", true},
	{harborIgnoreBasic, []string{"**"}, "a", true},
	{harborIgnoreBasic, []string{"**"}, "a/b", true},
	{harborIgnoreBasic, []string{"**/"}, "a", false},
	{harborIgnoreBasic, []string{"**/"}, "a/b", true},
	{harborIgnoreBasic, []string{"/"}, "a", true},
	{harborIgnoreBasic, []string{"/"}, "a/b", true},
	{harborIgnoreBasic, []string{"?.txt"}, "a.txt", true},
	{harborIgnoreBasic, []string{"?.txt"}, "ab.txt", false},
	{harborIgnoreBasic, []string{"?.txt"}, "/.txt", false},
	{harborIgnoreBasic, []string{"[ab].txt"}, "a.txt", true},
	{harborIgnoreBasic, []string{"[ab].txt"}, "c.txt", false},
	{harborIgnoreBasic, []string{"[!ab].txt"}, "a.txt", false},
	{harborIgnoreBasic, []string{"[!ab].txt"}, "c.txt", true},
	{harborIgnoreBasic, []string{"[]a].txt"}, "].txt", true},
	{harborIgnoreBasic, []string{"[]a].txt"}, "a.txt", true},
	{harborIgnoreBasic, []string{"[]a].txt"}, "b.txt", false},
	{harborIgnoreBasic, []string{"\\#hash"}, "#hash", true},
	{harborIgnoreBasic, []string{"\\!bang"}, "!bang", true},
	{harborIgnoreBasic, []string{"trail \\ "}, "trail  ", true},
	{harborIgnoreBasic, []string{"trail \\ "}, "trail ", false},
	{harborIgnoreBasic, []string{"trailing   "}, "trailing", true},
	{harborIgnoreBasic, []string{"trailing   "}, "trailing   ", false},
	{harborIgnoreBasic, []string{"*.tmp", "!keep.tmp"}, "a.tmp", true},
	{harborIgnoreBasic, []string{"*.tmp", "!keep.tmp"}, "keep.tmp", false},
	{harborIgnoreBasic, []string{"*.tmp", "!keep.tmp"}, "d/keep.tmp", false},
	{harborIgnoreBasic, []string{"dir/", "!dir/keep"}, "dir/keep", false},
	{harborIgnoreBasic, []string{"dir/", "!dir/keep"}, "dir/other", true},
	{harborIgnoreBasic, []string{"!dir/keep", "dir/"}, "dir/keep", true},
	{harborIgnoreBasic, []string{"dir", "!dir/keep"}, "dir/keep", false},
	{harborIgnoreBasic, []string{"*.txt", "!*.txt", "a.txt"}, "a.txt", true},
	{harborIgnoreBasic, []string{"*.txt", "!*.txt", "a.txt"}, "b.txt", false},
	{harborIgnoreBasic, []string{"# comment", "", "x"}, "x", true},
	{harborIgnoreBasic, []string{"# comment", "", "x"}, "# comment", false},
	{harborIgnoreBasic, []string{"a/b/"}, "a/b/c", true},
	{harborIgnoreBasic, []string{"a/b/"}, "x/a/b/c", false},
	{harborIgnoreBasic, []string{"*.MD"}, "a.md", false},
	{harborIgnoreBasic, []string{"*.MD"}, "a.MD", true},
	{harborIgnoreGit, []string{"*.log"}, "a.log", true},
	{harborIgnoreGit, []string{"*.log"}, "dir/a.log", true},
	{harborIgnoreGit, []string{"*.log"}, "a.log/x", true},
	{harborIgnoreGit, []string{"*.log"}, "notes", false},
	{harborIgnoreGit, []string{"/root.txt"}, "root.txt", true},
	{harborIgnoreGit, []string{"/root.txt"}, "sub/root.txt", false},
	{harborIgnoreGit, []string{"build/"}, "build", false},
	{harborIgnoreGit, []string{"build/"}, "build/x", true},
	{harborIgnoreGit, []string{"build/"}, "a/build/x", true},
	{harborIgnoreGit, []string{"build/"}, "build.txt", false},
	{harborIgnoreGit, []string{"doc/*.md"}, "doc/a.md", true},
	{harborIgnoreGit, []string{"doc/*.md"}, "doc/sub/a.md", false},
	{harborIgnoreGit, []string{"doc/*.md"}, "x/doc/a.md", false},
	{harborIgnoreGit, []string{"**/logs"}, "logs", true},
	{harborIgnoreGit, []string{"**/logs"}, "a/logs", true},
	{harborIgnoreGit, []string{"**/logs"}, "a/b/logs/x", true},
	{harborIgnoreGit, []string{"a/**/b"}, "a/b", true},
	{harborIgnoreGit, []string{"a/**/b"}, "a/x/b", true},
	{harborIgnoreGit, []string{"a/**/b"}, "a/x/y/b", true},
	{harborIgnoreGit, []string{"a/**/b"}, "xa/b", false},
	{harborIgnoreGit, []string{"foo/**"}, "foo", false},
	{harborIgnoreGit, []string{"foo/**"}, "foo/", true},
	{harborIgnoreGit, []string{"foo/**"}, "foo/x", true},
	{harborIgnoreGit, []string{"foo/**"}, "foo/x/y", true},
	{harborIgnoreGit, []string{"*"}, "a", true},
	{harborIgnoreGit, []string{"*"}, "a/b", true},
	{harborIgnoreGit, []string{"*/"}, "a", false},
	{harborIgnoreGit, []string{"*/"}, "a/", true},
	{harborIgnoreGit, []string{"*/"}, "a/b", true},
	{harborIgnoreGit, []string{"**"}, "a", true},
	{harborIgnoreGit, []string{"**"}, "a/b", true},
	{harborIgnoreGit, []string{"**/"}, "a", false},
	{harborIgnoreGit, []string{"**/"}, "a/b", true},
	{harborIgnoreGit, []string{"/"}, "a", false},
	{harborIgnoreGit, []string{"/"}, "a/b", false},
	{harborIgnoreGit, []string{"?.txt"}, "a.txt", true},
	{harborIgnoreGit, []string{"?.txt"}, "ab.txt", false},
	{harborIgnoreGit, []string{"?.txt"}, "/.txt", false},
	{harborIgnoreGit, []string{"[ab].txt"}, "a.txt", true},
	{harborIgnoreGit, []string{"[ab].txt"}, "c.txt", false},
	{harborIgnoreGit, []string{"[!ab].txt"}, "a.txt", false},
	{harborIgnoreGit, []string{"[!ab].txt"}, "c.txt", true},
	{harborIgnoreGit, []string{"[]a].txt"}, "].txt", true},
	{harborIgnoreGit, []string{"[]a].txt"}, "a.txt", true},
	{harborIgnoreGit, []string{"[]a].txt"}, "b.txt", false},
	{harborIgnoreGit, []string{"\\#hash"}, "#hash", true},
	{harborIgnoreGit, []string{"\\!bang"}, "!bang", true},
	{harborIgnoreGit, []string{"trail \\ "}, "trail  ", true},
	{harborIgnoreGit, []string{"trail \\ "}, "trail ", false},
	{harborIgnoreGit, []string{"trailing   "}, "trailing", true},
	{harborIgnoreGit, []string{"trailing   "}, "trailing   ", false},
	{harborIgnoreGit, []string{"*.tmp", "!keep.tmp"}, "a.tmp", true},
	{harborIgnoreGit, []string{"*.tmp", "!keep.tmp"}, "keep.tmp", false},
	{harborIgnoreGit, []string{"*.tmp", "!keep.tmp"}, "d/keep.tmp", false},
	{harborIgnoreGit, []string{"dir/", "!dir/keep"}, "dir/keep", false},
	{harborIgnoreGit, []string{"dir/", "!dir/keep"}, "dir/other", true},
	{harborIgnoreGit, []string{"!dir/keep", "dir/"}, "dir/keep", false},
	{harborIgnoreGit, []string{"dir", "!dir/keep"}, "dir/keep", false},
	{harborIgnoreGit, []string{"*.txt", "!*.txt", "a.txt"}, "a.txt", true},
	{harborIgnoreGit, []string{"*.txt", "!*.txt", "a.txt"}, "b.txt", false},
	{harborIgnoreGit, []string{"# comment", "", "x"}, "x", true},
	{harborIgnoreGit, []string{"# comment", "", "x"}, "# comment", false},
	{harborIgnoreGit, []string{"a/b/"}, "a/b/c", true},
	{harborIgnoreGit, []string{"a/b/"}, "x/a/b/c", false},
	{harborIgnoreGit, []string{"*.MD"}, "a.md", false},
	{harborIgnoreGit, []string{"*.MD"}, "a.MD", true},
}

func TestHarborIgnorePatterns(t *testing.T) {
	for _, tc := range harborPatternCases {
		spec, err := newHarborIgnoreSpec(tc.flavour, tc.patterns)
		require.NoError(t, err, tc.patterns)
		assert.Equal(t, tc.excluded, spec.excludes(tc.path), "flavour %d patterns %q path %q", tc.flavour, tc.patterns, tc.path)
	}
}

func TestHarborIgnoreErrors(t *testing.T) {
	_, err := newHarborIgnoreSpec(harborIgnoreGit, []string{`foo\`})
	assert.Error(t, err)

	spec, err := newHarborIgnoreSpec(harborIgnoreGit, []string{"[abc"})
	require.NoError(t, err, "the Git flavour discards an invalid range")
	assert.Empty(t, spec.patterns)

	spec, err = newHarborIgnoreSpec(harborIgnoreBasic, []string{"[abc"})
	require.NoError(t, err, "the basic flavour takes the bracket literally")
	assert.True(t, spec.excludes("[abc"))
}

func TestHarborSplitLines(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c", "", "d"}, harborSplitLines("a\r\nb\rc\n\nd"))
	assert.Equal(t, []string{"a"}, harborSplitLines("a\n"))
	assert.Nil(t, harborSplitLines(""))
}
