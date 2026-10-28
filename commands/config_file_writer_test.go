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
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// useConfigFile points doctl at path for the duration of the test.
func useConfigFile(t *testing.T, path string) {
	t.Helper()
	prev := viper.GetString("config")
	t.Cleanup(func() { viper.Set("config", prev) })
	viper.Set("config", path)
}

func requireConfig(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(b))
}

func requireNoTempFiles(t *testing.T, dir string, want int) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, want, "unexpected files in %s", dir)
}

func TestConfigFileWriterReplacesFileOnClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("access-token: old\n"), 0644))
	useConfigFile(t, path)

	w, err := defaultConfigFileWriter()
	require.NoError(t, err)
	_, err = w.Write([]byte("access-token: new\n"))
	require.NoError(t, err)

	// A concurrent reader still sees the complete old file.
	requireConfig(t, path, "access-token: old\n")

	require.NoError(t, w.Close())
	requireConfig(t, path, "access-token: new\n")
	requireNoTempFiles(t, dir, 1)

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
}

// A reader that opened the file before the write keeps reading the complete
// old configuration: the file is replaced, not truncated and rewritten.
func TestConfigFileWriterDoesNotRewriteOpenFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("an open file cannot be replaced on Windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("access-token: old\n"), 0600))
	useConfigFile(t, path)

	reader, err := os.Open(path)
	require.NoError(t, err)
	defer reader.Close()

	require.NoError(t, writeConfigBytes([]byte("access-token: new\n")))

	b, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, "access-token: old\n", string(b))
	requireConfig(t, path, "access-token: new\n")
}

func TestConfigFileWriterCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	useConfigFile(t, path)

	require.NoError(t, writeConfigBytes([]byte("access-token: new\n")))
	requireConfig(t, path, "access-token: new\n")
	requireNoTempFiles(t, dir, 1)
}

// A file that cannot be replaced, such as one bind-mounted into a container,
// is written in place instead.
func TestConfigFileWriterFallsBackWhenRenameFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("access-token: old\n"), 0644))
	useConfigFile(t, path)

	prev := renameConfigFile
	t.Cleanup(func() { renameConfigFile = prev })
	renameConfigFile = func(string, string) error {
		return &os.LinkError{Op: "rename", Err: syscall.EBUSY}
	}

	require.NoError(t, writeConfigBytes([]byte("access-token: new\n")))
	requireConfig(t, path, "access-token: new\n")
	requireNoTempFiles(t, dir, 1)

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
}

// A writable file in a directory the user cannot write to is written in place.
func TestConfigFileWriterFallsBackInReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions enforced for the current user")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("access-token: old\n"), 0600))
	require.NoError(t, os.Chmod(dir, 0500))
	t.Cleanup(func() { os.Chmod(dir, 0700) })
	useConfigFile(t, path)

	require.NoError(t, writeConfigBytes([]byte("access-token: new\n")))
	requireConfig(t, path, "access-token: new\n")
}

func TestConfigFileWriterKeepsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs extra privileges on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles-config.yaml")
	link := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(target, []byte("access-token: old\n"), 0600))
	require.NoError(t, os.Symlink(target, link))
	useConfigFile(t, link)

	require.NoError(t, writeConfigBytes([]byte("access-token: new\n")))

	info, err := os.Lstat(link)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&os.ModeSymlink, "config symlink was replaced by a file")
	requireConfig(t, target, "access-token: new\n")
}

func TestConfigFileWriterWritesThroughDanglingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs extra privileges on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles-config.yaml")
	link := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.Symlink(target, link))
	useConfigFile(t, link)

	require.NoError(t, writeConfigBytes([]byte("access-token: new\n")))

	info, err := os.Lstat(link)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&os.ModeSymlink, "config symlink was replaced by a file")
	requireConfig(t, target, "access-token: new\n")
}

func TestConfigFileWriterReportsCloseError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "config.yaml")
	useConfigFile(t, path)

	err := writeConfigBytes([]byte("access-token: new\n"))
	require.Error(t, err)
	require.True(t, errors.Is(err, os.ErrNotExist), "got %v", err)
}
