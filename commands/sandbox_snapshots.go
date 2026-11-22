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
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const sandboxSnapshotWaitInterval = 5 * time.Second

var sandboxSnapshotNameRE = regexp.MustCompile(`^[a-z0-9-]{1,63}$`)

// addSandboxSnapshotCmds adds the `snapshot` group to `doctl sandbox`.
func addSandboxSnapshotCmds(parent *Command) {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "snapshot",
			Aliases: []string{"snapshots"},
			Short:   "Manage sandbox snapshots",
			Long: "The subcommands of `doctl sandbox snapshot` take and manage snapshots of your sandboxes. " +
				"A snapshot captures the boot disk and memory of a running or paused sandbox, not its workspace data disk. " +
				"A snapshot is created asynchronously and is ready when its state is complete.",
		},
	}
	parent.AddCommand(cmd)

	cmdCreate := CmdBuilder(cmd, RunSandboxSnapshotCreate, "create <sandbox-id>",
		"Take a snapshot of a sandbox",
		"Starts a snapshot of a running or paused sandbox and returns as soon as the capture starts. "+
			"Add `--wait` to wait until the snapshot is complete.",
		Writer, aliasOpt("c"), displayerType(&displayers.SandboxSnapshot{}))
	AddStringFlag(cmdCreate, doctl.ArgSandboxSnapshotName, "", "",
		"A name for the snapshot: 1 to 63 characters, lowercase letters, digits and hyphens")
	AddStringFlag(cmdCreate, doctl.ArgSandboxSnapshotTTL, "", "",
		"How long to keep the snapshot, as a duration such as `168h`. When unset, the platform default applies")
	AddWaitFlagsWithTimeout(cmdCreate, false, "Wait for the snapshot to be complete before returning", 15*time.Minute)
	cmdCreate.Example = "The following example snapshots a sandbox, keeps the snapshot for a week and waits for it: " +
		"doctl sandbox snapshot create 01a0ef1a-d873-793d-ae6a-5dd6239be62a --name my-snapshot --ttl 168h --wait"

	cmdList := CmdBuilder(cmd, RunSandboxSnapshotList, "list",
		"List your snapshots",
		"Lists your team's snapshots, newest first. Add `--sandbox-id` to list the snapshots of one sandbox.",
		Writer, aliasOpt("ls"), displayerType(&displayers.SandboxSnapshot{}))
	AddStringFlag(cmdList, doctl.ArgSandboxID, "", "", "List only the snapshots of this sandbox")
	cmdList.Example = "The following example lists the snapshots of one sandbox: " +
		"doctl sandbox snapshot list --sandbox-id 01a0ef1a-d873-793d-ae6a-5dd6239be62a"

	cmdGet := CmdBuilder(cmd, RunSandboxSnapshotGet, "get <snapshot-id>",
		"Retrieve a snapshot",
		"Retrieves a snapshot: its state, size, and for a failed snapshot, the error. Add `--format Error` to show the error.",
		Writer, aliasOpt("g"), displayerType(&displayers.SandboxSnapshot{}))
	cmdGet.Example = "The following example shows only the state and error of a snapshot: " +
		"doctl sandbox snapshot get 33333333-4444-4555-8666-777777777777 --format State,Error"

	cmdDelete := CmdBuilder(cmd, RunSandboxSnapshotDelete, "delete <snapshot-id>...",
		"Permanently delete one or more snapshots",
		"Permanently deletes the snapshots you name. Sandboxes already started from a snapshot are not affected. This is irreversible.",
		Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the snapshot(s) without a confirmation prompt")
	cmdDelete.Example = "The following example deletes a snapshot without asking: " +
		"doctl sandbox snapshot delete 33333333-4444-4555-8666-777777777777 --force"
}

// RunSandboxSnapshotCreate takes a snapshot of a sandbox.
func RunSandboxSnapshotCreate(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgSandboxSnapshotName)
	if err != nil {
		return err
	}
	if name != "" && !sandboxSnapshotNameRE.MatchString(name) {
		return fmt.Errorf("--%s %q must be 1 to 63 characters: lowercase letters, digits and hyphens", doctl.ArgSandboxSnapshotName, name)
	}
	ttl, err := c.Doit.GetString(c.NS, doctl.ArgSandboxSnapshotTTL)
	if err != nil {
		return err
	}
	if ttl != "" {
		if d, perr := time.ParseDuration(ttl); perr != nil || d <= 0 {
			return fmt.Errorf("--%s %q must be a positive duration such as 168h", doctl.ArgSandboxSnapshotTTL, ttl)
		}
	}
	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	svc := c.Sandboxes()
	snap, err := svc.CreateSnapshot(c.Args[0], &godo.SandboxSnapshotCreateRequest{Name: name, TTL: ttl})
	if err != nil {
		return err
	}
	if wait {
		if snap, err = waitForSandboxSnapshot(c, svc, snap); err != nil {
			return err
		}
	}
	return c.Display(&displayers.SandboxSnapshot{Snapshots: do.SandboxSnapshots{*snap}})
}

// RunSandboxSnapshotList lists snapshots.
func RunSandboxSnapshotList(c *CmdConfig) error {
	if len(c.Args) != 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	sandboxID, err := c.Doit.GetString(c.NS, doctl.ArgSandboxID)
	if err != nil {
		return err
	}
	list, err := c.Sandboxes().ListSnapshots(sandboxID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.SandboxSnapshot{Snapshots: list})
}

// RunSandboxSnapshotGet retrieves a snapshot.
func RunSandboxSnapshotGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	snap, err := c.Sandboxes().GetSnapshot(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.SandboxSnapshot{Snapshots: do.SandboxSnapshots{*snap}})
}

// RunSandboxSnapshotDelete deletes one or more snapshots.
func RunSandboxSnapshotDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if err := confirmDelete(force, "sandbox snapshot", len(c.Args)); err != nil {
		return err
	}
	svc := c.Sandboxes()
	for _, id := range c.Args {
		if err := svc.DeleteSnapshot(id); err != nil {
			return fmt.Errorf("deleting snapshot %s: %w", id, err)
		}
	}
	return nil
}

func waitForSandboxSnapshot(c *CmdConfig, svc do.SandboxesService, snap *do.SandboxSnapshot) (*do.SandboxSnapshot, error) {
	w, err := newWaiter(c)
	if err != nil {
		return nil, err
	}
	return waitForSnapshot(w, svc, snap, sandboxSnapshotWaitInterval)
}

// waitForSnapshot polls a snapshot until it is complete. A snapshot that
// failed is reported as a failure, with the API's error message.
func waitForSnapshot(w waiter, svc do.SandboxesService, snap *do.SandboxSnapshot, interval time.Duration) (*do.SandboxSnapshot, error) {
	label := snap.ID
	if snap.Name != "" {
		label = snap.Name
	}
	current := snap
	err := w.wait(waitOp{
		Activity: fmt.Sprintf("Taking snapshot (%s)", label),
		Subject:  fmt.Sprintf("snapshot (%s) to be complete", label),
		Success:  fmt.Sprintf("Snapshot (%s) is complete", label),
		Interval: interval,
	}, func() (bool, string, error) {
		s, err := svc.GetSnapshot(snap.ID)
		if err != nil {
			return false, "", err
		}
		current = s
		switch s.State {
		case godo.SandboxSnapshotStateComplete:
			return true, "", nil
		case godo.SandboxSnapshotStateFailed:
			msg := fmt.Sprintf("snapshot %s failed", s.ID)
			if m := strings.TrimSpace(s.ErrorMessage); m != "" {
				return false, "", fmt.Errorf("%s: %s", msg, m)
			}
			return false, "", errors.New(msg)
		}
		return false, displayers.FormatSandboxSnapshotState(s.State), nil
	})
	if err != nil {
		return nil, err
	}
	return current, nil
}
