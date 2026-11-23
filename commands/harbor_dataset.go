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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

const harborDatasetNS = harborNS + ".dataset"

// HarborDatasetCmd manages Harbor datasets.
func HarborDatasetCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "dataset",
			Aliases: []string{"datasets", "ds"},
			Short:   "Display commands that manage Harbor datasets",
			Long:    "The subcommands of `doctl genai harbor dataset` manage datasets: named lists of your team's tasks, for repeatable benchmark runs.",
		},
	}

	cmdPublish := CmdBuilder(
		cmd,
		RunHarborDatasetPublish,
		"publish <dataset-dir>",
		"Publish a Harbor dataset folder",
		`Packs a Harbor dataset folder into one archive and uploads it. The archive has dataset.toml at the root and each task under tasks/<task-name>/. The server validates the archive when the upload completes and stores the dataset as ready, or failed with findings.

Each `+"`[[tasks]]`"+` entry must be a task folder inside the dataset (any folder with a task.toml, up to two levels deep). An org prefix (`+"`my-org/`"+`) is dropped because datasets belong to a team. The name stored in the archive matches the name the dataset is published under.

The dataset's name comes from `+"`--name`"+`, then from `+"`[dataset] name`"+` without its org prefix, then from the folder name.`,
		Writer, aliasOpt("p"),
		displayerType(&displayers.HarborDatasets{}),
		overrideCmdNS(harborDatasetNS),
	)
	AddStringFlag(cmdPublish, doctl.ArgHarborName, "", "", "The name to publish the dataset under (default: the name in dataset.toml, then the folder name)")
	AddDurationFlag(cmdPublish, doctl.ArgWaitTimeout, "", harborDefaultLintTimeout, waitTimeoutDesc)
	AddDurationFlag(cmdPublish, doctl.ArgHarborPollInterval, "", harborDefaultLintInterval, "How often to check whether the dataset upload has finished")
	cmdPublish.Example = "The following example publishes the dataset in `./backend-bugs`: " +
		"`doctl genai harbor dataset publish ./backend-bugs`"

	cmdUpload := &Command{
		Command: &cobra.Command{
			Use:   "upload",
			Short: "Display commands for an in-progress Harbor dataset upload",
			Long:  "An upload stays invisible to `dataset get` until validation finishes and the dataset is ready. `dataset upload get` reads that in-progress upload by the id `dataset publish` prints.",
		},
	}
	cmd.AddCommand(cmdUpload)
	cmdUploadGet := CmdBuilder(
		cmdUpload,
		RunHarborDatasetUploadGet,
		"get <dataset-name> <upload-uuid>",
		"Show the state of a Harbor dataset upload",
		"Shows one dataset upload, including while it is still validating. `dataset get` does not see the dataset until this upload is ready.",
		Writer,
		displayerType(&displayers.HarborDatasetUploads{}),
		overrideCmdNS(harborDatasetNS),
	)
	cmdUploadGet.Example = "The following example checks an upload that `dataset publish` printed: " +
		"`doctl genai harbor dataset upload get checkout-ops e163bfe3-eef1-435a-b1bd-59e18927a97c`"

	cmdList := CmdBuilder(
		cmd,
		RunHarborDatasetList,
		"list",
		"List Harbor datasets",
		"Lists your team's ready datasets with the number of tasks in each. `--query` limits the list to names that contain the term.",
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HarborDatasets{}),
		overrideCmdNS(harborDatasetNS),
	)
	AddStringFlag(cmdList, doctl.ArgHarborQuery, "", "", "Only lists datasets whose name contains this term")
	cmdList.Example = "The following example lists datasets whose name contains `bugs`: " +
		"`doctl genai harbor dataset list --query bugs`"

	cmdGet := CmdBuilder(
		cmd,
		RunHarborDatasetGet,
		"get <dataset-name>",
		"Retrieve a Harbor dataset",
		"Retrieves a dataset's description and the names of its tasks.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.HarborDatasets{}),
		overrideCmdNS(harborDatasetNS),
	)
	cmdGet.Example = "The following example retrieves a dataset: `doctl genai harbor dataset get backend-bugs`"

	cmdPull := CmdBuilder(
		cmd,
		RunHarborDatasetPull,
		"pull <dataset-name>",
		"Download a Harbor dataset",
		"Downloads the ready archive of a dataset and unpacks it into `<out>/<dataset-name>`. The archive is checked against the dataset's digest.",
		Writer,
		displayerType(&displayers.HarborPull{}),
		overrideCmdNS(harborDatasetNS),
	)
	AddStringFlag(cmdPull, doctl.ArgHarborOut, "", ".", "The folder to unpack the dataset into")
	AddBoolFlag(cmdPull, doctl.ArgForce, doctl.ArgShortForce, false, "Replaces the dataset folder if it already exists")
	cmdPull.Example = "The following example unpacks a dataset into `./backend-bugs`: " +
		"`doctl genai harbor dataset pull backend-bugs`"

	cmdDelete := CmdBuilder(
		cmd,
		RunHarborDatasetDelete,
		"delete <dataset-name>",
		"Delete a Harbor dataset",
		"Deletes a dataset. Its tasks are not deleted.",
		Writer, aliasOpt("rm"),
		overrideCmdNS(harborDatasetNS),
	)
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the dataset without a confirmation prompt")
	cmdDelete.Example = "The following example deletes a dataset: `doctl genai harbor dataset delete backend-bugs --force`"

	return cmd
}

// harborDatasetManifest is the part of Harbor's dataset.toml doctl uses.
type harborDatasetManifest struct {
	Dataset struct {
		Name        string `toml:"name"`
		Description string `toml:"description"`
	} `toml:"dataset"`
	Tasks []struct {
		Name   string `toml:"name"`
		Digest string `toml:"digest"`
	} `toml:"tasks"`
	Files []struct {
		Path string `toml:"path"`
	} `toml:"files"`
}

// harborDatasetPlan is a dataset.toml resolved into what doctl will publish.
type harborDatasetPlan struct {
	Name        string
	Description string
	Tasks       []string
	// LocalTasks maps a task name to the folder it is published from.
	LocalTasks map[string]string
	Warnings   []string
}

func planHarborDataset(dir, nameOverride string) (*harborDatasetPlan, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if root, err = filepath.EvalSymlinks(root); err != nil {
		return nil, fmt.Errorf("reading dataset folder: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "dataset.toml"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("dataset.toml not found in %s", dir)
		}
		return nil, err
	}
	var m harborDatasetManifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing dataset.toml: %w", err)
	}

	plan := &harborDatasetPlan{Description: m.Dataset.Description, LocalTasks: map[string]string{}}

	name, source := strings.TrimSpace(nameOverride), "--name"
	if name == "" && strings.TrimSpace(m.Dataset.Name) != "" {
		name, source = harborStripOrg(strings.TrimSpace(m.Dataset.Name)), "[dataset] name in dataset.toml"
		if name != strings.TrimSpace(m.Dataset.Name) {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("Publishing dataset %q as %q: datasets belong to your team, so the org prefix is dropped.", m.Dataset.Name, name))
		}
	}
	if name == "" {
		name, source = filepath.Base(root), "the folder name"
	}
	if !harborNameRE.MatchString(name) {
		return nil, fmt.Errorf("dataset name %q (from %s) is invalid: use 1-128 lowercase letters, digits, '.', '_' or '-', starting with a letter or digit, or pass --name", name, source)
	}
	plan.Name = name

	if len(m.Tasks) == 0 {
		return nil, fmt.Errorf("dataset.toml lists no [[tasks]]")
	}
	seen := map[string]bool{}
	prefixed, digests := 0, 0
	for _, t := range m.Tasks {
		full := strings.TrimSpace(t.Name)
		short := harborStripOrg(full)
		if short != full {
			prefixed++
		}
		if t.Digest != "" {
			digests++
		}
		if !harborNameRE.MatchString(short) {
			return nil, fmt.Errorf("dataset.toml lists the task %q, which is not a valid task name", t.Name)
		}
		if seen[short] {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("dataset.toml lists %q more than once; it is included once.", short))
			continue
		}
		seen[short] = true
		plan.Tasks = append(plan.Tasks, short)
	}
	if prefixed > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("Dropped the org prefix from %d task name(s); tasks belong to your team.", prefixed))
	}
	if digests > 0 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("Ignored the digest of %d task(s): tasks are not versioned, so jobs run each task's current version.", digests))
	}
	if len(m.Files) > 0 {
		paths := make([]string, len(m.Files))
		for i, f := range m.Files {
			paths[i] = f.Path
		}
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("Dataset files (%s) are not uploaded yet; the dataset is published without them.", strings.Join(paths, ", ")))
	}

	local, err := findHarborTaskDirs(root)
	if err != nil {
		return nil, err
	}
	for _, t := range plan.Tasks {
		if d, ok := local[t]; ok {
			plan.LocalTasks[t] = d
		}
	}
	return plan, nil
}

// findHarborTaskDirs maps task names to folders with a task.toml, one or two
// levels below root (for example ./task-a or ./tasks/task-a).
func findHarborTaskDirs(root string) (map[string]string, error) {
	found := map[string]string{}
	var scan func(dir string, depth int) error
	scan = func(dir string, depth int) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			sub := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(filepath.Join(sub, "task.toml"))
			if err != nil {
				if depth < 2 {
					if err := scan(sub, depth+1); err != nil {
						return err
					}
				}
				continue
			}
			manifest := map[string]any{}
			if err := toml.Unmarshal(data, &manifest); err != nil {
				return fmt.Errorf("parsing %s: %w", filepath.Join(sub, "task.toml"), err)
			}
			var ignored []string
			name, err := resolveHarborTaskName(sub, manifest, "", &ignored)
			if err != nil {
				return err
			}
			if prev, dup := found[name]; dup {
				return fmt.Errorf("both %s and %s are the task %q", prev, sub, name)
			}
			found[name] = sub
		}
		return nil
	}
	return found, scan(root, 1)
}

func harborStripOrg(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

// harborDatasetArchive is one dataset folder packed for the upload API.
type harborDatasetArchive struct {
	Path   string
	Size   int64
	Digest string
	Files  int
}

// Close removes the temporary archive.
func (a *harborDatasetArchive) Close() {
	if a != nil && a.Path != "" {
		os.Remove(a.Path)
	}
}

// packHarborDataset writes a .tar.gz with dataset.toml and tasks/<name>/, then
// hashes those exact bytes. The API checks that hash, not a hash of the files.
func packHarborDataset(plan *harborDatasetPlan) (*harborDatasetArchive, error) {
	type entry struct {
		name string
		file harborFile
	}
	var entries []entry
	for _, name := range plan.Tasks {
		dir, ok := plan.LocalTasks[name]
		if !ok {
			return nil, fmt.Errorf("task %s is listed in dataset.toml but there is no folder for it under the dataset", name)
		}
		files, err := collectHarborFiles(dir)
		if err != nil {
			return nil, fmt.Errorf("task %s: %w", name, err)
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("task %s has no files to publish", name)
		}
		for _, f := range files {
			entries = append(entries, entry{name: path.Join("tasks", name, f.Path), file: f})
		}
	}
	if len(entries) > harborMaxFiles {
		return nil, fmt.Errorf("the dataset has %d files; the limit is %d", len(entries), harborMaxFiles)
	}

	config := "[dataset]\nname = " + strconv.Quote(plan.Name) + "\n"
	if plan.Description != "" {
		config += "description = " + strconv.Quote(plan.Description) + "\n"
	}

	tmp, err := os.CreateTemp("", "doctl-harbor-dataset-*.tar.gz")
	if err != nil {
		return nil, err
	}
	arch := &harborDatasetArchive{Path: tmp.Name(), Files: len(entries) + 1}
	gz := gzip.NewWriter(tmp)
	gz.Header.ModTime = time.Unix(0, 0)
	tw := tar.NewWriter(gz)
	if err := writeHarborTarBytes(tw, "dataset.toml", []byte(config)); err != nil {
		arch.Close()
		tmp.Close()
		return nil, err
	}
	for _, e := range entries {
		mode := int64(0o644)
		if e.file.Exec {
			mode = 0o755
		}
		hdr := &tar.Header{Typeflag: tar.TypeReg, Name: e.name, Size: e.file.Size, Mode: mode, ModTime: time.Unix(0, 0)}
		if err := tw.WriteHeader(hdr); err != nil {
			arch.Close()
			tmp.Close()
			return nil, err
		}
		if _, err := copyHarborFile(tw, e.file); err != nil {
			arch.Close()
			tmp.Close()
			return nil, err
		}
	}
	if err := errors.Join(tw.Close(), gz.Close(), tmp.Close()); err != nil {
		arch.Close()
		return nil, err
	}
	sum, size, err := hashHarborFile(arch.Path)
	if err != nil {
		arch.Close()
		return nil, err
	}
	if size > harborMaxArchiveBytes {
		arch.Close()
		return nil, fmt.Errorf("the dataset archive is %s; the limit is 100 MiB", displayers.BytesToHumanReadableUnitBinary(uint64(size)))
	}
	arch.Size = size
	arch.Digest = "sha256:" + sum
	return arch, nil
}

func writeHarborTarBytes(tw *tar.Writer, name string, body []byte) error {
	hdr := &tar.Header{Typeflag: tar.TypeReg, Name: name, Size: int64(len(body)), Mode: 0o644, ModTime: time.Unix(0, 0)}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(body)
	return err
}

func hashHarborFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// RunHarborDatasetPublish packs a dataset folder and uploads that one archive.
func RunHarborDatasetPublish(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	nameOverride, err := c.Doit.GetString(c.NS, doctl.ArgHarborName)
	if err != nil {
		return err
	}
	wait, err := harborWaitDurations(c)
	if err != nil {
		return err
	}

	plan, err := planHarborDataset(c.Args[0], nameOverride)
	if err != nil {
		return err
	}
	for _, w := range plan.Warnings {
		warn("%s", w)
	}
	arch, err := packHarborDataset(plan)
	if err != nil {
		return err
	}
	defer arch.Close()
	harborProgress("Packed %d files from %s (%s)", arch.Files, c.Args[0], displayers.BytesToHumanReadableUnitBinary(uint64(arch.Size)))

	svc := c.Harbor()
	start, err := svc.StartDatasetUpload(plan.Name, &do.HarborDatasetUploadRequest{Digest: arch.Digest, SizeBytes: arch.Size})
	if err != nil {
		return err
	}
	if start.Unchanged {
		if start.Dataset == nil {
			start.Dataset = &do.HarborDataset{Name: plan.Name, Digest: arch.Digest, Status: do.HarborDatasetReady}
		}
		notice("Dataset %s is already published with this archive.", plan.Name)
		return c.Display(&displayers.HarborDatasets{Datasets: []do.HarborDataset{*start.Dataset}, Single: true})
	}
	if start.Upload == nil || start.Upload.UUID == "" || start.UploadURL == nil || start.UploadURL.URL == "" {
		return fmt.Errorf("the upload response for %s has no upload link", plan.Name)
	}
	if err := putHarborArchive(start.UploadURL.URL, arch.Path, arch.Size); err != nil {
		return err
	}
	upload, err := svc.CompleteDatasetUpload(plan.Name, start.Upload.UUID)
	if err != nil {
		return fmt.Errorf("completing the upload of %s: %w", plan.Name, err)
	}
	if upload.UUID == "" {
		upload.UUID = start.Upload.UUID
	}
	harborProgress("Upload %s. Check it later with `doctl genai harbor dataset upload get %s %s`.", upload.UUID, plan.Name, upload.UUID)
	upload, err = waitForHarborDatasetUpload(c, plan.Name, upload, wait)
	if err != nil {
		return err
	}
	if err := displayHarborDatasetFindings(c, plan.Name, upload.Findings); err != nil {
		return err
	}
	if upload.Status == do.HarborDatasetFailed {
		msg := fmt.Sprintf("the server rejected dataset %s", plan.Name)
		if upload.Error != "" {
			msg += ": " + upload.Error
		}
		return errors.New(msg)
	}

	ds, err := svc.GetDataset(plan.Name)
	if err != nil {
		ds = &do.HarborDataset{Name: plan.Name, Digest: upload.Digest, Status: upload.Status, SizeBytes: upload.SizeBytes}
	}
	if ds.Status == "" {
		ds.Status = upload.Status
	}
	return c.Display(&displayers.HarborDatasets{Datasets: []do.HarborDataset{*ds}, Single: true})
}

func waitForHarborDatasetUpload(c *CmdConfig, name string, upload *do.HarborDatasetUpload, wait harborWait) (*do.HarborDatasetUpload, error) {
	started := time.Now()
	deadline := started.Add(wait.Timeout)
	var line harborStatusLine
	defer line.finish()
	for !do.HarborDatasetTerminal(upload.Status) {
		line.show(harborDatasetWaitText(name, upload, time.Since(started)))
		if wait.Timeout > 0 && !time.Now().Before(deadline) {
			return nil, harborUploadTimeout(name, upload, wait.Timeout)
		}
		harborWaitPause(wait.Interval, func() {
			line.show(harborDatasetWaitText(name, upload, time.Since(started)))
		})
		if wait.Timeout > 0 && !time.Now().Before(deadline) {
			return nil, harborUploadTimeout(name, upload, wait.Timeout)
		}
		next, err := c.Harbor().GetDatasetUpload(name, upload.UUID)
		if err != nil {
			return nil, err
		}
		upload = next
	}
	return upload, nil
}

func harborUploadTimeout(name string, upload *do.HarborDatasetUpload, timeout time.Duration) error {
	return fmt.Errorf("publish of %s did not finish within %s; upload %s is still %s. Check it with `doctl genai harbor dataset upload get %s %s`", name, timeout, upload.UUID, upload.Status, name, upload.UUID)
}

func harborDatasetWaitText(name string, upload *do.HarborDatasetUpload, elapsed time.Duration) string {
	return fmt.Sprintf("%s    %s elapsed    upload %s", harborDatasetProgress(name, upload.Status), elapsed.Truncate(time.Second), upload.UUID)
}

// harborStatusLine is the single progress line under a dataset publish. On a
// terminal it rewrites itself so the elapsed time moves. Anywhere else each
// update is a new line, which is what logs and tests capture.
type harborStatusLine struct {
	rewriting bool
	last      string
}

func (l *harborStatusLine) show(text string) {
	w := uiEnv().ErrWriter()
	if uiEnv().ErrTTY {
		padded := text
		if extra := len(l.last) - len(text); extra > 0 {
			padded += strings.Repeat(" ", extra)
		}
		fmt.Fprint(w, "\r"+padded)
		l.rewriting = true
		l.last = text
		return
	}
	fmt.Fprintln(w, text)
}

func (l *harborStatusLine) finish() {
	if l.rewriting {
		fmt.Fprintln(uiEnv().ErrWriter())
		l.rewriting = false
		l.last = ""
	}
}

// harborWaitPause waits for total, and on a terminal refreshes tick about once
// a second. Tests replace harborSleep with a no-op, so a non-terminal wait is
// one sleep: looping until the clock moves would spin.
func harborWaitPause(total time.Duration, tick func()) {
	if total <= 0 {
		return
	}
	if !uiEnv().ErrTTY || total <= time.Second {
		harborSleep(total)
		return
	}
	left := total
	for left > 0 {
		step := time.Second
		if step > left {
			step = left
		}
		harborSleep(step)
		left -= step
		if left > 0 {
			tick()
		}
	}
}

func harborDatasetProgress(name, status string) string {
	switch status {
	case do.HarborDatasetUploading:
		return fmt.Sprintf("Uploading %s", name)
	case do.HarborDatasetValidating:
		return fmt.Sprintf("Validating %s", name)
	default:
		return fmt.Sprintf("Waiting for %s (%s)", name, status)
	}
}

func displayHarborDatasetFindings(c *CmdConfig, name string, findings []do.HarborLintFinding) error {
	if len(findings) == 0 || Output == "json" {
		return nil
	}
	fmt.Fprintf(c.Out, "\nFindings for %s:\n", name)
	return c.Display(&displayers.HarborLintFindings{Findings: findings})
}

// RunHarborDatasetList lists datasets.
func RunHarborDatasetList(c *CmdConfig) error {
	query, err := c.Doit.GetString(c.NS, doctl.ArgHarborQuery)
	if err != nil {
		return err
	}
	datasets, err := c.Harbor().ListDatasets(strings.TrimSpace(query))
	if err != nil {
		return err
	}
	return c.Display(&displayers.HarborDatasets{Datasets: datasets})
}

// RunHarborDatasetPull downloads a dataset archive and unpacks it. The digest
// is the SHA-256 of the .tar.gz bytes.
func RunHarborDatasetPull(c *CmdConfig) error {
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
	ds, err := svc.GetDataset(name)
	if err != nil {
		return err
	}
	archive, err := svc.GetDatasetArchive(name)
	if err != nil {
		return err
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
	if _, err := downloadHarborFile(archive.URL, archivePath, ds.SizeBytes, ds.Digest); err != nil {
		return err
	}
	unpacked := filepath.Join(staging, "dataset")
	sums, err := extractHarborArchive(archivePath, unpacked)
	if err != nil {
		return err
	}
	if force {
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
	}
	if err := moveHarborDir(unpacked, dest); err != nil {
		return err
	}
	return c.Display(&displayers.HarborPull{Result: &do.HarborPullResult{Name: name, Digest: ds.Digest, Path: dest, Files: len(sums)}})
}

// RunHarborDatasetUploadGet shows one in-progress or finished upload. dataset
// get cannot see it until the dataset is ready.
func RunHarborDatasetUploadGet(c *CmdConfig) error {
	switch len(c.Args) {
	case 0, 1:
		return doctl.NewMissingArgsErr(c.NS)
	case 2:
	default:
		return doctl.NewTooManyArgsErr(c.NS)
	}
	name, uploadUUID := c.Args[0], c.Args[1]
	upload, err := c.Harbor().GetDatasetUpload(name, uploadUUID)
	if err != nil {
		return err
	}
	if upload.Dataset == "" {
		upload.Dataset = name
	}
	if upload.UUID == "" {
		upload.UUID = uploadUUID
	}
	return c.Display(&displayers.HarborDatasetUploads{Uploads: []do.HarborDatasetUpload{*upload}})
}

// RunHarborDatasetGet retrieves a dataset.
func RunHarborDatasetGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	ds, err := c.Harbor().GetDataset(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.HarborDatasets{Datasets: []do.HarborDataset{*ds}, Single: true})
}

// RunHarborDatasetDelete deletes a dataset.
func RunHarborDatasetDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if !force {
		if err := AskForConfirmDelete("Harbor dataset", 1); err != nil {
			return err
		}
	}
	return c.Harbor().DeleteDataset(c.Args[0])
}
