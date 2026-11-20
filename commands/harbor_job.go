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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const (
	harborJobNS = harborNS + ".job"

	harborMaxAttempts    = 10
	harborMaxConcurrency = 50
	harborMaxAgents      = 20
)

// HarborRunCmd adds the run command, which submits a job, to parent.
func HarborRunCmd(parent *Command) *Command {
	cmd := CmdBuilder(
		parent,
		RunHarborRun,
		"run",
		"Run agents against a Harbor dataset",
		`Submits a job that runs each agent and model pair against the tasks of a dataset, `+"`--attempts`"+` times each. Every attempt is a trial that runs in its own DO Sandbox and is scored by the task's tests. Prints the job that was submitted.

A job runs one dataset, named with `+"`-d`"+` or `+"`--dataset`"+`. To run only some of its tasks, add `+"`-T`"+` or `+"`--task`"+` (repeatable). `+"`-t`"+` is doctl's access token and does not select a task. Pair agents and models with `+"`--agent`"+` and `+"`--model`"+`: the first agent uses the first model, and so on. A single `+"`--model`"+` applies to every agent, and a single `+"`--agent`"+` runs against every model. `+"`--model`"+` takes a DigitalOcean model slug, or `+"`dedicated:<endpoint>:<model>`"+` for a Dedicated Inference endpoint.

The same job can come from a Harbor job file. `+"`-c`"+` is doctl's own config file, so pass the job file with `+"`--job-config`"+`. A flag replaces the matching field in the file. The file's dataset is a published name (`+"`datasets[].name`"+`), not a local folder. Local task paths and an `+"`environment`"+` block are not sent.

Use `+"`--dry-run`"+` to validate a job without starting it.

A retried submission with the same `+"`--idempotency-key`"+` returns the original job instead of starting another one, which makes the command safe to retry in CI. A new key is generated for every run by default.`,
		Writer,
		displayerType(&displayers.HarborJobs{}),
		overrideCmdNS(harborNS),
	)
	AddStringArrayFlag(cmd, doctl.ArgHarborTask, doctl.ArgHarborTaskShort, nil, "A task of the dataset to run, as -T or --task; repeat the flag for more tasks (default: every task)")
	AddStringFlag(cmd, doctl.ArgHarborDataset, "d", "", "The dataset to run")
	AddStringArrayFlag(cmd, doctl.ArgHarborAgent, "a", nil, "An agent to run, such as claude-code; repeat the flag to run more agents")
	AddStringArrayFlag(cmd, doctl.ArgHarborModel, "m", nil, "The model for the matching --agent; repeat the flag to pair more models")
	AddIntFlag(cmd, doctl.ArgHarborAttempts, "", 1, "How many times each agent attempts each task (1-10)")
	AddIntFlag(cmd, doctl.ArgHarborConcurrency, "", 4, "How many trials run at once (1-50)")
	AddIntFlag(cmd, doctl.ArgHarborMaxRetries, "", 1, "How many times an errored trial is retried")
	AddStringFlag(cmd, doctl.ArgHarborName, "", "", "A name for the job")
	AddStringFlag(cmd, doctl.ArgHarborIdempotencyKey, "", "", "A key that makes a retried submission return the original job (default: a new random key)")
	AddBoolFlag(cmd, doctl.ArgHarborDryRun, "", false, "Validates the job and shows the trials it would run, without starting it")
	AddStringFlag(cmd, doctl.ArgHarborJobConfig, "", "", "A Harbor job file in YAML or JSON. Flags replace the matching fields. -c is doctl's own config file and does not select this file")
	cmd.Example = `The following example runs claude-code against one task of a dataset three times: ` +
		"`doctl genai harbor run -d my-dataset -T fix-login-bug -a claude-code -m anthropic-claude-sonnet-4 --attempts 3`" + `

The same job from a file: ` + "`doctl genai harbor run --job-config job.yaml`"
	return cmd
}

// HarborJobCmd inspects jobs.
func HarborJobCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "job",
			Aliases: []string{"jobs"},
			Short:   "Display commands that inspect Harbor jobs",
			Long:    "The subcommands of `doctl genai harbor job` list, cancel, delete, and download the results of jobs started with `doctl genai harbor run`.",
		},
	}

	jobDetails := `
  - The job UUID
  - The job name
  - The status: queued, running, cancelling, cancelled, successful, partially_successful, or failed
  - Progress, as completed trials out of all trials
  - The number of errored trials
  - The creation time`

	cmdList := CmdBuilder(
		cmd,
		RunHarborJobList,
		"list",
		"List Harbor jobs",
		"Lists your team's jobs, newest first. `--status` limits the list to one status. Every page is returned. Each job contains:"+jobDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HarborJobs{}),
		overrideCmdNS(harborJobNS),
	)
	AddStringFlag(cmdList, doctl.ArgHarborStatus, "", "", "Only jobs in this status: queued, running, cancelling, cancelled, successful, partially_successful, or failed")
	cmdList.Example = "The following example lists running jobs: `doctl genai harbor job list --status running`"

	cmdCancel := CmdBuilder(
		cmd,
		RunHarborJobCancel,
		"cancel <job-uuid>",
		"Cancel a Harbor job",
		"Cancels a queued or running job. Trials that already finished keep their results.",
		Writer,
		displayerType(&displayers.HarborJobs{}),
		overrideCmdNS(harborJobNS),
	)
	AddBoolFlag(cmdCancel, doctl.ArgForce, doctl.ArgShortForce, false, "Cancels the job without a confirmation prompt")
	cmdCancel.Example = "The following example cancels a job: " +
		"`doctl genai harbor job cancel 3f2a5c1e-9b7d-4c6a-8e1f-2d4b6a8c0e12 --force`"

	cmdDelete := CmdBuilder(
		cmd,
		RunHarborJobDelete,
		"delete <job-uuid>",
		"Delete a Harbor job",
		"Deletes a finished job and its stored results. A queued or running job must be cancelled first.",
		Writer, aliasOpt("rm"),
		overrideCmdNS(harborJobNS),
	)
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the job without a confirmation prompt")
	cmdDelete.Example = "The following example deletes a finished job: " +
		"`doctl genai harbor job delete 3f2a5c1e-9b7d-4c6a-8e1f-2d4b6a8c0e12 --force`"

	cmdDownload := CmdBuilder(
		cmd,
		RunHarborJobDownload,
		"download <job-uuid>",
		"Download the results of a Harbor job",
		`Downloads a job's results into `+"`<out>/<job-uuid>`"+`, in Harbor's jobs folder layout: the job's config.json, result.json, and job.log, and a folder per trial with its result, logs, agent trajectory, and verifier output. Open them with `+"`harbor view <out>`"+`.

The API returns one link to the job's results archive, job.zip. doctl downloads that archive and unpacks it. Files already on disk with the same contents are left in place.`,
		Writer,
		displayerType(&displayers.HarborDownload{}),
		overrideCmdNS(harborJobNS),
	)
	AddStringFlag(cmdDownload, doctl.ArgHarborOut, "", "./jobs", "The folder to download into")
	cmdDownload.Example = "The following example downloads a job's results and opens them: " +
		"`doctl genai harbor job download 3f2a5c1e-9b7d-4c6a-8e1f-2d4b6a8c0e12 --out ./jobs && harbor view ./jobs`"

	return cmd
}

// RunHarborRun submits a job.
func RunHarborRun(c *CmdConfig) error {
	req, key, err := buildHarborJobRequest(c)
	if err != nil {
		return err
	}
	job, err := c.Harbor().CreateJob(req, key)
	if err != nil {
		return err
	}
	return displayHarborJob(c, job)
}

func buildHarborJobRequest(c *CmdConfig) (*do.HarborJobRequest, string, error) {
	tasks, err := c.Doit.GetStringSlice(c.NS, doctl.ArgHarborTask)
	if err != nil {
		return nil, "", err
	}
	dataset, err := c.Doit.GetString(c.NS, doctl.ArgHarborDataset)
	if err != nil {
		return nil, "", err
	}
	agents, err := c.Doit.GetStringSlice(c.NS, doctl.ArgHarborAgent)
	if err != nil {
		return nil, "", err
	}
	models, err := c.Doit.GetStringSlice(c.NS, doctl.ArgHarborModel)
	if err != nil {
		return nil, "", err
	}
	attempts, err := c.Doit.GetInt(c.NS, doctl.ArgHarborAttempts)
	if err != nil {
		return nil, "", err
	}
	concurrency, err := c.Doit.GetInt(c.NS, doctl.ArgHarborConcurrency)
	if err != nil {
		return nil, "", err
	}
	maxRetries, err := c.Doit.GetInt(c.NS, doctl.ArgHarborMaxRetries)
	if err != nil {
		return nil, "", err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgHarborName)
	if err != nil {
		return nil, "", err
	}
	key, err := c.Doit.GetString(c.NS, doctl.ArgHarborIdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	dryRun, err := c.Doit.GetBool(c.NS, doctl.ArgHarborDryRun)
	if err != nil {
		return nil, "", err
	}
	configPath, err := c.Doit.GetString(c.NS, doctl.ArgHarborJobConfig)
	if err != nil {
		return nil, "", err
	}

	var file *harborJobFile
	if strings.TrimSpace(configPath) != "" {
		file, err = loadHarborJobFile(configPath)
		if err != nil {
			return nil, "", err
		}
	}

	var fileDataset harborJobFileDataset
	if file != nil && len(file.Datasets) == 1 {
		fileDataset = file.Datasets[0]
	}
	dataset = harborChosenString(c, doctl.ArgHarborDataset, dataset, fileDataset.Name)
	tasks = harborChosenStrings(c, doctl.ArgHarborTask, tasks, fileDataset.TaskNames)
	name = harborChosenString(c, doctl.ArgHarborName, name, fileJobName(file))
	attempts = harborChosenInt(c, doctl.ArgHarborAttempts, attempts, fileInt(file, func(f *harborJobFile) *int { return f.NAttempts }), 1)
	concurrency = harborChosenInt(c, doctl.ArgHarborConcurrency, concurrency, fileInt(file, func(f *harborJobFile) *int { return f.NConcurrentTrials }), 4)
	maxRetries = harborChosenInt(c, doctl.ArgHarborMaxRetries, maxRetries, fileRetry(file), 1)
	dryRun = harborChosenBool(c, doctl.ArgHarborDryRun, dryRun, fileBool(file, func(f *harborJobFile) *bool { return f.DryRun }))

	if dataset == "" {
		if len(harborTrimAll(tasks)) > 0 {
			return nil, "", fmt.Errorf("--%s picks tasks inside a dataset; name the dataset with --%s or datasets[].name in --%s", doctl.ArgHarborTask, doctl.ArgHarborDataset, doctl.ArgHarborJobConfig)
		}
		return nil, "", fmt.Errorf("choose the dataset to run with --%s or datasets[].name in --%s", doctl.ArgHarborDataset, doctl.ArgHarborJobConfig)
	}
	if !harborNameRE.MatchString(dataset) {
		return nil, "", fmt.Errorf("%q is not a valid dataset name", dataset)
	}
	tasks = harborTrimAll(tasks)
	for _, t := range tasks {
		if !harborNameRE.MatchString(t) {
			return nil, "", fmt.Errorf("%q is not a valid task name", t)
		}
	}
	for _, t := range fileDataset.ExcludeTaskNames {
		if !harborNameRE.MatchString(t) {
			return nil, "", fmt.Errorf("%q is not a valid task name", t)
		}
	}

	pairs, err := harborJobAgents(c, agents, models, file)
	if err != nil {
		return nil, "", err
	}
	if len(pairs) > harborMaxAgents {
		return nil, "", fmt.Errorf("a job runs at most %d agent and model pairs; got %d", harborMaxAgents, len(pairs))
	}
	if attempts < 1 || attempts > harborMaxAttempts {
		return nil, "", fmt.Errorf("--%s must be between 1 and %d", doctl.ArgHarborAttempts, harborMaxAttempts)
	}
	if concurrency < 1 || concurrency > harborMaxConcurrency {
		return nil, "", fmt.Errorf("--%s must be between 1 and %d", doctl.ArgHarborConcurrency, harborMaxConcurrency)
	}
	if maxRetries < 0 {
		return nil, "", fmt.Errorf("--%s must not be negative", doctl.ArgHarborMaxRetries)
	}
	nTasks := 0
	if fileDataset.NTasks != nil {
		nTasks = *fileDataset.NTasks
		if nTasks < 1 {
			return nil, "", fmt.Errorf("datasets[].n_tasks must be at least 1")
		}
	}
	if key = strings.TrimSpace(key); key == "" {
		key = uuid.NewString()
	}

	if len(tasks) == 0 {
		tasks = nil
	}
	exclude := fileDataset.ExcludeTaskNames
	if len(exclude) == 0 {
		exclude = nil
	}

	return &do.HarborJobRequest{
		JobName:           name,
		Datasets:          []do.HarborJobDataset{{Name: dataset, TaskNames: tasks, ExcludeTaskNames: exclude, NTasks: nTasks}},
		Agents:            pairs,
		NAttempts:         attempts,
		NConcurrentTrials: concurrency,
		Retry:             do.HarborJobRetry{MaxRetries: maxRetries},
		DryRun:            dryRun,
	}, key, nil
}

// harborJobAgents uses --agent and --model when either was passed. Otherwise
// it uses the agents already paired in the job file.
func harborJobAgents(c *CmdConfig, agents, models []string, file *harborJobFile) ([]do.HarborJobAgent, error) {
	agents, models = harborTrimAll(agents), harborTrimAll(models)
	fromFlags := (harborFlagProvided(c, doctl.ArgHarborAgent) && len(agents) > 0) || (harborFlagProvided(c, doctl.ArgHarborModel) && len(models) > 0)
	if fromFlags {
		if file != nil && len(file.Agents) > 0 && (len(agents) == 0 || len(models) == 0) {
			return nil, fmt.Errorf("pass both --%s and --%s to replace the agents in the job file", doctl.ArgHarborAgent, doctl.ArgHarborModel)
		}
		return pairHarborAgents(agents, models)
	}
	if file == nil || len(file.Agents) == 0 {
		return pairHarborAgents(agents, models)
	}
	pairs := make([]do.HarborJobAgent, 0, len(file.Agents))
	seen := map[string]bool{}
	for _, a := range file.Agents {
		name, model := strings.TrimSpace(a.Name), strings.TrimSpace(a.ModelName)
		if name == "" {
			return nil, fmt.Errorf("an agent in the job file has no name")
		}
		if model == "" {
			return nil, fmt.Errorf("agent %s in the job file has no model_name", name)
		}
		k := name + "\x00" + model
		if seen[k] {
			return nil, fmt.Errorf("the agent %s with the model %s is listed more than once", name, model)
		}
		seen[k] = true
		kwargs := a.Kwargs
		if len(kwargs) == 0 {
			kwargs = nil
		}
		pairs = append(pairs, do.HarborJobAgent{Name: name, ModelName: model, Kwargs: kwargs})
	}
	return pairs, nil
}

func harborFlagProvided(c *CmdConfig, name string) bool {
	if c.Command != nil {
		return c.Command.Flags().Changed(name)
	}
	return c.Doit != nil && c.Doit.IsSet(name)
}

func harborChosenString(c *CmdConfig, key, flagVal, fileVal string) string {
	flagVal = strings.TrimSpace(flagVal)
	if harborFlagProvided(c, key) && flagVal != "" {
		return flagVal
	}
	if strings.TrimSpace(fileVal) != "" {
		return strings.TrimSpace(fileVal)
	}
	return flagVal
}

func harborChosenStrings(c *CmdConfig, key string, flagVal, fileVal []string) []string {
	flagVal = harborTrimAll(flagVal)
	if harborFlagProvided(c, key) && len(flagVal) > 0 {
		return flagVal
	}
	if len(harborTrimAll(fileVal)) > 0 {
		return harborTrimAll(fileVal)
	}
	return flagVal
}

func harborChosenInt(c *CmdConfig, key string, flagVal int, fileVal *int, fallback int) int {
	if harborFlagProvided(c, key) {
		return flagVal
	}
	if fileVal != nil {
		return *fileVal
	}
	if c.Command != nil || flagVal != 0 {
		return flagVal
	}
	return fallback
}

func harborChosenBool(c *CmdConfig, key string, flagVal bool, fileVal *bool) bool {
	if harborFlagProvided(c, key) {
		return flagVal
	}
	if fileVal != nil {
		return *fileVal
	}
	return flagVal
}

func fileJobName(file *harborJobFile) string {
	if file == nil {
		return ""
	}
	return file.JobName
}

func fileInt(file *harborJobFile, pick func(*harborJobFile) *int) *int {
	if file == nil {
		return nil
	}
	return pick(file)
}

func fileBool(file *harborJobFile, pick func(*harborJobFile) *bool) *bool {
	if file == nil {
		return nil
	}
	return pick(file)
}

func fileRetry(file *harborJobFile) *int {
	if file == nil || file.Retry == nil {
		return nil
	}
	return file.Retry.MaxRetries
}

// pairHarborAgents matches --agent and --model values by position. One value
// on either side is paired with every value on the other.
func pairHarborAgents(agents, models []string) ([]do.HarborJobAgent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("choose at least one agent with --%s", doctl.ArgHarborAgent)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("choose the model for each agent with --%s", doctl.ArgHarborModel)
	}

	var pairs []do.HarborJobAgent
	switch {
	case len(agents) == len(models):
		for i := range agents {
			pairs = append(pairs, do.HarborJobAgent{Name: agents[i], ModelName: models[i]})
		}
	case len(models) == 1:
		for _, a := range agents {
			pairs = append(pairs, do.HarborJobAgent{Name: a, ModelName: models[0]})
		}
	case len(agents) == 1:
		for _, m := range models {
			pairs = append(pairs, do.HarborJobAgent{Name: agents[0], ModelName: m})
		}
	default:
		return nil, fmt.Errorf("got %d --%s and %d --%s values; pass one model per agent, or a single model for all agents", len(agents), doctl.ArgHarborAgent, len(models), doctl.ArgHarborModel)
	}

	seen := map[string]bool{}
	for _, p := range pairs {
		k := p.Name + "\x00" + p.ModelName
		if seen[k] {
			return nil, fmt.Errorf("the agent %s with the model %s is listed more than once", p.Name, p.ModelName)
		}
		seen[k] = true
	}
	return pairs, nil
}

func harborTrimAll(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// RunHarborJobList lists jobs.
func RunHarborJobList(c *CmdConfig) error {
	status, err := c.Doit.GetString(c.NS, doctl.ArgHarborStatus)
	if err != nil {
		return err
	}
	status = strings.TrimSpace(status)
	if !harborKnownJobStatus(status) {
		return fmt.Errorf("unknown job status %q; choose queued, running, cancelling, cancelled, successful, partially_successful, or failed", status)
	}
	jobs, err := c.Harbor().ListJobs(status)
	if err != nil {
		return err
	}
	return c.Display(&displayers.HarborJobs{Jobs: jobs})
}

func harborKnownJobStatus(status string) bool {
	switch status {
	case "", do.HarborJobQueued, do.HarborJobRunning, do.HarborJobCancelling, do.HarborJobCancelled,
		do.HarborJobSuccessful, do.HarborJobPartiallySuccessful, do.HarborJobFailed:
		return true
	}
	return false
}

func displayHarborJob(c *CmdConfig, job *do.HarborJob) error {
	return c.Display(&displayers.HarborJobs{Jobs: []do.HarborJob{*job}, Single: true})
}

// RunHarborJobCancel cancels a job.
func RunHarborJobCancel(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if !force {
		if err := AskForConfirm("cancel this Harbor job"); err != nil {
			return err
		}
	}
	job, err := c.Harbor().CancelJob(c.Args[0])
	if err != nil {
		return err
	}
	return displayHarborJob(c, job)
}

// RunHarborJobDelete deletes a finished job and its stored results.
func RunHarborJobDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if !force {
		if err := AskForConfirmDelete("Harbor job", 1); err != nil {
			return err
		}
	}
	return c.Harbor().DeleteJob(c.Args[0])
}

// RunHarborJobDownload downloads a job's results archive and unpacks it into
// Harbor's jobs layout.
func RunHarborJobDownload(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	jobUUID := c.Args[0]
	out, err := c.Doit.GetString(c.NS, doctl.ArgHarborOut)
	if err != nil {
		return err
	}
	dir, err := harborSafePath(out, jobUUID)
	if err != nil {
		return err
	}

	link, err := c.Harbor().GetJobArtifacts(jobUUID)
	if err != nil {
		return err
	}
	if link == nil || link.URL == "" {
		return fmt.Errorf("the API returned no download link for job %s", jobUUID)
	}

	tmp, err := os.CreateTemp("", "doctl-harbor-job-*.zip")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	defer os.Remove(tmpName)

	n, err := downloadHarborFile(link.URL, tmpName, link.SizeBytes, "")
	if err != nil {
		return err
	}
	written, skipped, err := extractHarborZip(tmpName, dir)
	if err != nil {
		return err
	}
	if written+skipped == 0 {
		warn("Job %s has no results to download yet.", jobUUID)
	}
	result := &do.HarborDownloadResult{
		JobUUID:    jobUUID,
		Path:       dir,
		Downloaded: written,
		Skipped:    skipped,
		Bytes:      n,
	}
	if abs, err := filepath.Abs(dir); err == nil {
		result.Path = abs
	}
	return c.Display(&displayers.HarborDownload{Result: result})
}
