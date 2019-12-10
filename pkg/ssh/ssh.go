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

package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/digitalocean/doctl/pkg/runner"
	"github.com/fatih/color"
)

// Options is the type used to specify options passed to the SSH command
type Options map[string]interface{}

// Runner runs ssh commands.
type Runner struct {
	User            string
	Host            string
	KeyPath         string
	Port            int
	AgentForwarding bool
	Command         string
}

var _ runner.Runner = &Runner{}

// Run ssh.
func (r *Runner) Run() error {
	args := []string{}
	if r.KeyPath != "" {
		args = append(args, "-i", r.KeyPath)
	}

	sshHost := r.Host
	if r.User != "" {
		sshHost = r.User + "@" + sshHost
	}

	if r.Port > 0 {
		args = append(args, "-p", strconv.Itoa(r.Port))
	}

	if r.AgentForwarding {
		args = append(args, "-A")
	}

	args = append(args, sshHost)
	if r.Command != "" {
		args = append(args, r.Command)
	}

	cmd := exec.Command("ssh", args...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		_, isSnap := os.LookupEnv("SNAP")

		if os.IsPermission(err) && isSnap {
			msg := "Using the doctl Snap? Grant access to the ssh-keys interface with this command: sudo snap connect doctl:ssh-keys"
			fmt.Fprintf(color.Error, "%s: %s\n", color.YellowString("Warning"), msg)
			return err
		}

		return err
	}

	return nil
}
