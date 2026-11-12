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
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	harborTaskNS = harborNS + ".task"

	harborDefaultLintTimeout  = 30 * time.Minute
	harborDefaultLintInterval = 5 * time.Second
)

// HarborTaskCmd manages Harbor tasks.
func HarborTaskCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "task",
			Aliases: []string{"tasks"},
			Short:   "Display commands that manage Harbor tasks",
			Long:    "The subcommands of `doctl genai harbor task` publish Harbor task folders to your team and manage the published tasks.",
		},
	}

	cmdPublish := CmdBuilder(
		cmd,
		RunHarborTaskPublish,
		"publish <task-dir>",
		"Publish a Harbor task folder",
		`Packs a local Harbor task folder exactly as `+"`harbor publish`"+` does and uploads it to your team. Publishing again replaces the task; jobs already submitted keep running the version they started with.

The same files are included as with the harbor CLI: task.toml, instruction.md, README.md, trajectory.json, and the environment/, tests/, solution/, and steps/ folders, filtered by the task's .gitignore and .harborignore files. The digest is computed the same way too, so it matches the one the harbor CLI reports for the same folder. If the task is unchanged since it was last published, nothing is uploaded.

The task's name comes from `+"`--name`"+`, then from `+"`[task] name`"+` in task.toml (without any org prefix), then from the folder name.

The server lints every upload before it replaces the current version. Use `+"`--wait`"+` to wait for the result and print the lint findings; the command fails if the upload is rejected.`,
		Writer, aliasOpt("p"),
		displayerType(&displayers.HarborPublish{}),
		overrideCmdNS(harborTaskNS),
	)
	AddStringFlag(cmdPublish, doctl.ArgHarborName, "", "", "The name to publish the task under (default: the name in task.toml, then the folder name)")
	addHarborWaitFlags(cmdPublish, "Waits for the server to lint the upload and prints the findings", harborDefaultLintTimeout, harborDefaultLintInterval)
	cmdPublish.Example = "The following example publishes the task in `./fix-login-bug` and waits for lint: " +
		"`doctl genai harbor task publish ./fix-login-bug --wait`"

	cmdList := CmdBuilder(
		cmd,
		RunHarborTaskList,
		"list",
		"List published Harbor tasks",
		"Lists your team's published tasks with their current digest, archive size, and publish time.",
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HarborTasks{}),
		overrideCmdNS(harborTaskNS),
	)
	AddStringFlag(cmdList, doctl.ArgHarborQuery, "", "", "Only lists tasks whose name matches this search term")
	cmdList.Example = "The following example lists tasks whose name contains `login`: " +
		"`doctl genai harbor task list --query login`"

	cmdGet := CmdBuilder(
		cmd,
		RunHarborTaskGet,
		"get <task-name>",
		"Retrieve a published Harbor task",
		"Retrieves a task's current digest, archive size, image, lint report, and who published it and when. Lint findings are listed below the task.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.HarborTasks{}),
		overrideCmdNS(harborTaskNS),
	)
	cmdGet.Example = "The following example retrieves a task: `doctl genai harbor task get fix-login-bug`"

	cmdPull := CmdBuilder(
		cmd,
		RunHarborTaskPull,
		"pull <task-name>",
		"Download a published Harbor task",
		"Downloads the current archive of a task and unpacks it into `<out>/<task-name>`. The unpacked files are checked against the task's digest.",
		Writer,
		displayerType(&displayers.HarborPull{}),
		overrideCmdNS(harborTaskNS),
	)
	AddStringFlag(cmdPull, doctl.ArgHarborOut, "", ".", "The folder to unpack the task into")
	AddBoolFlag(cmdPull, doctl.ArgForce, doctl.ArgShortForce, false, "Replaces the task folder if it already exists")
	cmdPull.Example = "The following example unpacks a task into `./tasks/fix-login-bug`: " +
		"`doctl genai harbor task pull fix-login-bug --out ./tasks`"

	cmdDelete := CmdBuilder(
		cmd,
		RunHarborTaskDelete,
		"delete <task-name>",
		"Delete a published Harbor task",
		"Deletes a task. A task that a dataset refers to cannot be deleted until it is removed from the dataset. Jobs that already ran the task keep their results.",
		Writer, aliasOpt("rm"),
		overrideCmdNS(harborTaskNS),
	)
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the task without a confirmation prompt")
	cmdDelete.Example = "The following example deletes a task: `doctl genai harbor task delete fix-login-bug --force`"

	return cmd
}

func addHarborWaitFlags(cmd *Command, desc string, timeout, interval time.Duration) {
	AddWaitFlagsWithTimeout(cmd, false, desc, timeout)
	AddDurationFlag(cmd, doctl.ArgHarborPollInterval, "", interval, "How often to check for progress while waiting")
}

// RunHarborTaskPublish packs, uploads, and optionally waits for lint.
func RunHarborTaskPublish(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgHarborName)
	if err != nil {
		return err
	}

	wait, err := harborWaitSettings(c)
	if err != nil {
		return err
	}

	result, err := publishHarborTask(c, c.Args[0], name, wait)
	if err != nil {
		return err
	}
	if err := displayHarborPublish(c, result); err != nil {
		return err
	}
	return harborPublishOutcome(result)
}

// publishHarborTask runs the publish flow for one folder: announce the
// digest, upload unless unchanged, complete, and wait for lint if asked to.
// It is shared with dataset publish.
func publishHarborTask(c *CmdConfig, dir, name string, wait harborWait) (*do.HarborPublishResult, error) {
	pkg, err := packHarborTask(dir, name)
	if err != nil {
		return nil, err
	}
	defer pkg.Close()
	for _, w := range pkg.Warnings {
		warn("%s", w)
	}
	harborProgress("Packed %d files from %s into %s (%s)", len(pkg.Files), dir, pkg.Name,
		displayers.BytesToHumanReadableUnitBinary(uint64(pkg.ArchiveSize)))

	result := &do.HarborPublishResult{
		Name:             pkg.Name,
		Digest:           pkg.Digest,
		Files:            len(pkg.Files),
		ArchiveSizeBytes: pkg.ArchiveSize,
	}

	svc := c.Harbor()
	start, err := svc.StartTaskUpload(pkg.Name, &do.HarborTaskUploadRequest{
		Digest:      pkg.Digest,
		SizeBytes:   pkg.ArchiveSize,
		FileDigests: pkg.FileDigests,
		Manifest:    pkg.Manifest,
	})
	if err != nil {
		return nil, err
	}
	if start.Unchanged {
		result.State = do.HarborPublishUnchanged
		return result, nil
	}
	if start.UploadUUID == "" || start.UploadURL == "" {
		return nil, fmt.Errorf("the upload response for %s has no upload link", pkg.Name)
	}
	result.UploadUUID = start.UploadUUID

	if err := putHarborArchive(start.UploadURL, pkg.ArchivePath, pkg.ArchiveSize); err != nil {
		return nil, err
	}
	upload, err := svc.CompleteTaskUpload(pkg.Name, start.UploadUUID)
	if err != nil {
		return nil, fmt.Errorf("completing the upload of %s: %w", pkg.Name, err)
	}

	upload, err = waitForHarborUpload(c, pkg.Name, upload, wait)
	if err != nil {
		return nil, err
	}
	result.State = upload.State
	result.LintReport = upload.LintReport
	result.ErrorDescription = upload.ErrorDescription
	return result, nil
}

func waitForHarborUpload(c *CmdConfig, name string, upload *do.HarborTaskUpload, wait harborWait) (*do.HarborTaskUpload, error) {
	if !wait.Enabled {
		return upload, nil
	}

	harborProgress("Waiting for the server to lint %s", name)
	deadline := time.Now().Add(wait.Timeout)
	for !do.HarborUploadTerminal(upload.State) {
		if wait.Timeout > 0 && time.Now().After(deadline) {
			return nil, fmt.Errorf("lint of %s did not finish within %s; the upload is still %s", name, wait.Timeout, upload.State)
		}
		harborSleep(wait.Interval)
		next, err := c.Harbor().GetTaskUpload(name, upload.UploadUUID)
		if err != nil {
			return nil, err
		}
		upload = next
	}
	return upload, nil
}

// harborWait is how a command waits for the server.
type harborWait struct {
	Enabled  bool
	Timeout  time.Duration
	Interval time.Duration
}

// harborWaitSettings reads --wait, --wait-timeout, and --poll-interval.
func harborWaitSettings(c *CmdConfig) (harborWait, error) {
	enabled, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil || !enabled {
		return harborWait{}, err
	}
	return harborWaitDurations(c)
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

func displayHarborPublish(c *CmdConfig, results ...*do.HarborPublishResult) error {
	if err := c.Display(&displayers.HarborPublish{Results: results}); err != nil {
		return err
	}
	if Output == "json" {
		return nil
	}
	for _, r := range results {
		if r.LintReport != nil && len(r.LintReport.Findings) > 0 {
			fmt.Fprintf(c.Out, "\nLint findings for %s:\n", r.Name)
			if err := c.Display(&displayers.HarborLintFindings{Findings: r.LintReport.Findings}); err != nil {
				return err
			}
		}
	}
	return nil
}

// harborPublishOutcome turns a rejected upload into a failing exit status.
func harborPublishOutcome(r *do.HarborPublishResult) error {
	switch r.State {
	case do.HarborUploadRejected:
		msg := fmt.Sprintf("the server rejected %s", r.Name)
		if r.ErrorDescription != "" {
			msg += ": " + r.ErrorDescription
		} else {
			msg += "; see the lint findings above"
		}
		return errors.New(msg)
	case do.HarborUploadSuperseded:
		warn("A newer upload of %s was accepted first, so this one was not applied.", r.Name)
	case do.HarborUploadValidating:
		notice("The server is linting %s. Check the result with `doctl genai harbor task get %s`, or publish with --wait.", r.Name, r.Name)
	}
	return nil
}

// RunHarborTaskList lists published tasks.
func RunHarborTaskList(c *CmdConfig) error {
	query, err := c.Doit.GetString(c.NS, doctl.ArgHarborQuery)
	if err != nil {
		return err
	}
	tasks, err := c.Harbor().ListTasks(query)
	if err != nil {
		return err
	}
	return c.Display(&displayers.HarborTasks{Tasks: tasks})
}

// RunHarborTaskGet retrieves a task and lists its lint findings.
func RunHarborTaskGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	task, err := c.Harbor().GetTask(c.Args[0])
	if err != nil {
		return err
	}
	if err := c.Display(&displayers.HarborTasks{Tasks: []do.HarborTask{*task}, Single: true}); err != nil {
		return err
	}
	if Output != "json" && task.LintReport != nil && len(task.LintReport.Findings) > 0 {
		fmt.Fprintln(c.Out, "\nLint findings:")
		return c.Display(&displayers.HarborLintFindings{Findings: task.LintReport.Findings})
	}
	return nil
}

// RunHarborTaskPull downloads and unpacks a task, checking its digest.
func RunHarborTaskPull(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	name := c.Args[0]
	out, err := c.Doit.GetString(c.NS, doctl.ArgHarborOut)
	if err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	dest, err := harborSafePath(out, name)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(dest); err == nil && !force {
		return fmt.Errorf("%s already exists; pass --force to replace it", dest)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	svc := c.Harbor()
	archive, err := svc.GetTaskArchive(name)
	if err != nil {
		return err
	}
	digest := archive.Digest
	if digest == "" {
		task, err := svc.GetTask(name)
		if err != nil {
			return err
		}
		digest = task.Digest
	}

	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(out, ".doctl-pull-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	archivePath := filepath.Join(staging, "archive.tar.gz")
	if _, err := downloadHarborFile(archive.URL, archivePath, archive.SizeBytes, ""); err != nil {
		return err
	}
	unpacked := filepath.Join(staging, "task")
	sums, err := extractHarborArchive(archivePath, unpacked)
	if err != nil {
		return err
	}
	got := harborContentDigest(sums)
	if digest != "" && got != digest {
		return fmt.Errorf("the downloaded files of %s have digest %s, but the task's digest is %s; try again", name, got, digest)
	}

	if force {
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
	}
	if err := os.Rename(unpacked, dest); err != nil {
		return err
	}
	return c.Display(&displayers.HarborPull{Result: &do.HarborPullResult{Name: name, Digest: got, Path: dest, Files: len(sums)}})
}

// RunHarborTaskDelete deletes a task.
func RunHarborTaskDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if !force {
		if err := AskForConfirmDelete("Harbor task", 1); err != nil {
			return err
		}
	}
	err = c.Harbor().DeleteTask(c.Args[0])
	if harborStatus(err) == http.StatusConflict {
		return fmt.Errorf("%w\nThe task is used by a dataset; remove it from the dataset (or delete the dataset) first", err)
	}
	return err
}

func harborStatus(err error) int {
	var apiErr *godo.ErrorResponse
	if errors.As(err, &apiErr) && apiErr.Response != nil {
		return apiErr.Response.StatusCode
	}
	return 0
}

// harborProgress reports progress on stderr so stdout stays parseable.
func harborProgress(format string, args ...any) {
	fmt.Fprintf(uiEnv().ErrWriter(), format+"\n", args...)
}
