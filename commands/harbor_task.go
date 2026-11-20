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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/digitalocean/doctl"
)

const (
	harborDefaultLintTimeout  = 30 * time.Minute
	harborDefaultLintInterval = 5 * time.Second
)

// harborWait is how a command waits for the server.
type harborWait struct {
	Enabled  bool
	Timeout  time.Duration
	Interval time.Duration
}

// harborWaitDurations reads --wait-timeout and --poll-interval for a command
// that always waits.
func harborWaitDurations(c *CmdConfig) (harborWait, error) {
	timeout, err := c.Doit.GetDuration(c.NS, doctl.ArgWaitTimeout)
	if err != nil {
		return harborWait{}, err
	}
	interval, err := c.Doit.GetDuration(c.NS, doctl.ArgHarborPollInterval)
	if err != nil {
		return harborWait{}, err
	}
	if interval <= 0 {
		interval = harborDefaultLintInterval
	}
	return harborWait{Enabled: true, Timeout: timeout, Interval: interval}, nil
}

// moveHarborDir moves src into place at dest. A rename across the same
// directory is atomic, but Windows refuses to rename onto an existing
// directory, so it falls back to copying and then removes src.
func moveHarborDir(src, dest string) error {
	if err := os.Rename(src, dest); err == nil {
		return nil
	}
	if err := copyHarborDir(src, dest); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

// copyHarborDir copies the files of src into dest, keeping each file's mode.
func copyHarborDir(src, dest string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		info, err := in.Stat()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		return errors.Join(copyErr, out.Close())
	})
}

// harborProgress reports progress on stderr so stdout stays parseable.
func harborProgress(format string, args ...any) {
	fmt.Fprintf(uiEnv().ErrWriter(), format+"\n", args...)
}
