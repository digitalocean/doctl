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
	"os"
	"path/filepath"
	"sort"
	"strings"

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
		`Reads dataset.toml in a Harbor dataset folder and creates or replaces the dataset.

Each `+"`[[tasks]]`"+` entry becomes a task name, with any org prefix (`+"`my-org/`"+`) dropped because tasks belong to your team. Tasks are not versioned, so `+"`digest`"+` fields are ignored; a job always runs each task's current version.

Task folders found inside the dataset folder (any folder with a task.toml, up to two levels deep) are published first when the dataset lists them, and the command waits for each to pass lint. Listed tasks with no local folder must already be published.

The dataset's name comes from `+"`--name`"+`, then from `+"`[dataset] name`"+` without its org prefix, then from the folder name.`,
		Writer, aliasOpt("p"),
		displayerType(&displayers.HarborDatasets{}),
		overrideCmdNS(harborDatasetNS),
	)
	AddStringFlag(cmdPublish, doctl.ArgHarborName, "", "", "The name to publish the dataset under (default: the name in dataset.toml, then the folder name)")
	AddDurationFlag(cmdPublish, doctl.ArgWaitTimeout, "", harborDefaultLintTimeout, waitTimeoutDesc)
	AddDurationFlag(cmdPublish, doctl.ArgHarborPollInterval, "", harborDefaultLintInterval, "How often to check lint progress of the dataset's tasks")
	cmdPublish.Example = "The following example publishes the dataset in `./backend-bugs`: " +
		"`doctl genai harbor dataset publish ./backend-bugs`"

	cmdList := CmdBuilder(
		cmd,
		RunHarborDatasetList,
		"list",
		"List Harbor datasets",
		"Lists your team's datasets with the number of tasks in each.",
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HarborDatasets{}),
		overrideCmdNS(harborDatasetNS),
	)
	cmdList.Example = "The following example lists datasets: `doctl genai harbor dataset list`"

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

// RunHarborDatasetPublish publishes the dataset's local tasks, then the
// dataset itself.
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

	names := make([]string, 0, len(plan.LocalTasks))
	for n := range plan.LocalTasks {
		names = append(names, n)
	}
	sort.Strings(names)
	var rejected []string
	for _, n := range names {
		result, err := publishHarborTask(c, plan.LocalTasks[n], n, wait)
		if err != nil {
			return fmt.Errorf("publishing task %s: %w", n, err)
		}
		harborProgress("Task %s: %s (%s)", n, result.State, result.Digest)
		if result.LintReport != nil {
			for _, f := range result.LintReport.Findings {
				harborProgress("  %s %s %s: %s", f.Severity, f.Rule, f.Path, f.Message)
			}
		}
		if result.State == do.HarborUploadRejected {
			rejected = append(rejected, n)
		}
	}
	if len(rejected) > 0 {
		return fmt.Errorf("the server rejected %s, so the dataset was not published", strings.Join(rejected, ", "))
	}

	ds, err := c.Harbor().PutDataset(plan.Name, &do.HarborDatasetRequest{
		Description: plan.Description,
		Tasks:       plan.Tasks,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.HarborDatasets{Datasets: []do.HarborDataset{*ds}, Single: true})
}

// RunHarborDatasetList lists datasets.
func RunHarborDatasetList(c *CmdConfig) error {
	datasets, err := c.Harbor().ListDatasets()
	if err != nil {
		return err
	}
	return c.Display(&displayers.HarborDatasets{Datasets: datasets})
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
