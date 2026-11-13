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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const (
	harborJobNS = harborNS + ".job"

	harborDefaultJobTimeout  = 24 * time.Hour
	harborDefaultJobInterval = 15 * time.Second

	harborMaxAttempts    = 10
	harborMaxConcurrency = 50
	harborDownloadPool   = 4
)

// HarborRunCmd adds the run command, which submits a job, to parent.
func HarborRunCmd(parent *Command) *Command {
	cmd := CmdBuilder(
		parent,
		RunHarborRun,
		"run",
		"Run agents against Harbor tasks",
		`Submits a job that runs each agent and model pair against tasks, `+"`--attempts`"+` times each. Every attempt is a trial that runs in its own DO Sandbox and is scored by the task's tests. Prints the job; follow it with `+"`doctl genai harbor job get`"+`, or pass `+"`--wait`"+`.

Choose the tasks with `+"`--task`"+` (repeatable) or `+"`--dataset`"+`. Pair agents and models with `+"`--agent`"+` and `+"`--model`"+`: the first agent uses the first model, and so on. A single `+"`--model`"+` applies to every agent, and a single `+"`--agent`"+` runs against every model. `+"`--model`"+` takes a DigitalOcean model slug, or `+"`dedicated:<endpoint>:<model>`"+` for a Dedicated Inference endpoint. List agents with `+"`doctl genai harbor agents list`"+`.

The job runs each task's current version; the digests it pinned are shown in its details. Use `+"`--dry-run`"+` to validate a job without starting it.

A retried submission with the same `+"`--idempotency-key`"+` returns the original job instead of starting another one, which makes the command safe to retry in CI. A new key is generated for every run by default.

With `+"`--wait`"+`, the command exits with an error if the job fails or is cancelled.`,
		Writer,
		displayerType(&displayers.HarborJobs{}),
		overrideCmdNS(harborNS),
	)
	AddStringArrayFlag(cmd, doctl.ArgHarborTask, doctl.ArgHarborTaskShort, nil, "A task to run; repeat the flag to run more tasks")
	AddStringFlag(cmd, doctl.ArgHarborDataset, "d", "", "A dataset whose tasks to run, instead of --task")
	AddStringArrayFlag(cmd, doctl.ArgHarborAgent, "a", nil, "An agent to run, such as claude-code; repeat the flag to run more agents")
	AddStringArrayFlag(cmd, doctl.ArgHarborModel, "m", nil, "The model for the matching --agent; repeat the flag to pair more models")
	AddIntFlag(cmd, doctl.ArgHarborAttempts, "", 1, "How many times each agent attempts each task (1-10)")
	AddIntFlag(cmd, doctl.ArgHarborConcurrency, "", 4, "How many trials run at once (1-50)")
	AddIntFlag(cmd, doctl.ArgHarborMaxRetries, "", 1, "How many times an errored trial is retried")
	AddStringFlag(cmd, doctl.ArgHarborName, "", "", "A name for the job")
	AddStringFlag(cmd, doctl.ArgHarborIdempotencyKey, "", "", "A key that makes a retried submission return the original job (default: a new random key)")
	AddBoolFlag(cmd, doctl.ArgHarborDryRun, "", false, "Validates the job and shows the trials it would run, without starting it")
	addHarborWaitFlags(cmd, "Waits for the job to finish", harborDefaultJobTimeout, harborDefaultJobInterval)
	cmd.Example = "The following example runs claude-code against a task three times and waits for the result: " +
		"`doctl genai harbor run -T fix-login-bug -a claude-code -m anthropic-claude-sonnet-4 --attempts 3 --wait`"
	return cmd
}

// HarborJobCmd inspects jobs.
func HarborJobCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "job",
			Aliases: []string{"jobs"},
			Short:   "Display commands that inspect Harbor jobs",
			Long:    "The subcommands of `doctl genai harbor job` follow, inspect, cancel, and download the results of jobs started with `doctl genai harbor run`.",
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
		"Lists your team's jobs, where each job contains:"+jobDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HarborJobs{}),
		overrideCmdNS(harborJobNS),
	)
	cmdList.Example = "The following example lists jobs: `doctl genai harbor job list`"

	cmdGet := CmdBuilder(
		cmd,
		RunHarborJobGet,
		"get <job-uuid>",
		"Retrieve a Harbor job",
		"Retrieves a job:"+jobDetails+"\n\n`-o json` also includes the pinned task digests, agent versions, and per agent and model metrics. Use `--wait` to follow the job until it finishes.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.HarborJobs{}),
		overrideCmdNS(harborJobNS),
	)
	addHarborWaitFlags(cmdGet, "Follows the job until it finishes", harborDefaultJobTimeout, harborDefaultJobInterval)
	cmdGet.Example = "The following example follows a job until it finishes: " +
		"`doctl genai harbor job get 3f2a5c1e-9b7d-4c6a-8e1f-2d4b6a8c0e12 --wait`"

	cmdTrials := CmdBuilder(
		cmd,
		RunHarborJobTrials,
		"trials <job-uuid>",
		"List the trials of a Harbor job",
		"Lists each trial of a job with its task, agent, model, attempt, status, reward, token usage, and error.",
		Writer,
		displayerType(&displayers.HarborTrials{}),
		overrideCmdNS(harborJobNS),
	)
	cmdTrials.Example = "The following example lists a job's trials: " +
		"`doctl genai harbor job trials 3f2a5c1e-9b7d-4c6a-8e1f-2d4b6a8c0e12`"

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

	cmdDownload := CmdBuilder(
		cmd,
		RunHarborJobDownload,
		"download <job-uuid>",
		"Download the results of a Harbor job",
		`Downloads a job's results into `+"`<out>/<job-uuid>`"+`, in Harbor's jobs folder layout: the job's config.json, result.json, and job.log, and a folder per trial with its result, logs, agent trajectory, and verifier output. Open them with `+"`harbor view <out>`"+`.

Files that are already downloaded are skipped, so an interrupted download can be resumed by running the command again.`,
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
	wait, err := harborWaitSettings(c)
	if err != nil {
		return err
	}

	job, err := c.Harbor().CreateJob(req, key)
	if err != nil {
		return err
	}
	if req.DryRun {
		return displayHarborJob(c, job)
	}
	return followHarborJob(c, job, wait)
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

	tasks = harborTrimAll(tasks)
	dataset = strings.TrimSpace(dataset)
	switch {
	case len(tasks) == 0 && dataset == "":
		return nil, "", fmt.Errorf("choose what to run with --%s or --%s", doctl.ArgHarborTask, doctl.ArgHarborDataset)
	case len(tasks) > 0 && dataset != "":
		return nil, "", fmt.Errorf("use either --%s or --%s, not both", doctl.ArgHarborTask, doctl.ArgHarborDataset)
	}
	for _, t := range tasks {
		if !harborNameRE.MatchString(t) {
			return nil, "", fmt.Errorf("%q is not a valid task name", t)
		}
	}

	pairs, err := pairHarborAgents(harborTrimAll(agents), harborTrimAll(models))
	if err != nil {
		return nil, "", err
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
	if key = strings.TrimSpace(key); key == "" {
		key = uuid.NewString()
	}

	return &do.HarborJobRequest{
		Name:              name,
		Tasks:             tasks,
		Dataset:           dataset,
		Agents:            pairs,
		NAttempts:         attempts,
		NConcurrentTrials: concurrency,
		Retry:             do.HarborJobRetry{MaxRetries: maxRetries},
		DryRun:            dryRun,
		TriggerSource:     "cli",
	}, key, nil
}

// pairHarborAgents matches --agent and --model values by position. One value
// on either side is paired with every value on the other.
func pairHarborAgents(agents, models []string) ([]do.HarborJobAgent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("choose at least one agent with --%s; list them with `doctl genai harbor agents list`", doctl.ArgHarborAgent)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("choose the model for each agent with --%s", doctl.ArgHarborModel)
	}

	var pairs []do.HarborJobAgent
	switch {
	case len(agents) == len(models):
		for i := range agents {
			pairs = append(pairs, do.HarborJobAgent{Name: agents[i], Model: models[i]})
		}
	case len(models) == 1:
		for _, a := range agents {
			pairs = append(pairs, do.HarborJobAgent{Name: a, Model: models[0]})
		}
	case len(agents) == 1:
		for _, m := range models {
			pairs = append(pairs, do.HarborJobAgent{Name: agents[0], Model: m})
		}
	default:
		return nil, fmt.Errorf("got %d --%s and %d --%s values; pass one model per agent, or a single model for all agents", len(agents), doctl.ArgHarborAgent, len(models), doctl.ArgHarborModel)
	}

	seen := map[string]bool{}
	for _, p := range pairs {
		k := p.Name + "\x00" + p.Model
		if seen[k] {
			return nil, fmt.Errorf("the agent %s with the model %s is listed more than once", p.Name, p.Model)
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
	jobs, err := c.Harbor().ListJobs()
	if err != nil {
		return err
	}
	return c.Display(&displayers.HarborJobs{Jobs: jobs})
}

// RunHarborJobGet retrieves a job, optionally following it.
func RunHarborJobGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	wait, err := harborWaitSettings(c)
	if err != nil {
		return err
	}
	job, err := c.Harbor().GetJob(c.Args[0])
	if err != nil {
		return err
	}
	return followHarborJob(c, job, wait)
}

// followHarborJob polls job until it finishes when wait is enabled, then
// displays it. A failed or cancelled job is an error once waited for, so CI
// can rely on the exit status.
func followHarborJob(c *CmdConfig, job *do.HarborJob, wait harborWait) error {
	if !wait.Enabled {
		return displayHarborJob(c, job)
	}

	deadline := time.Now().Add(wait.Timeout)
	last := ""
	for {
		if line := harborJobProgressLine(job); line != last {
			harborProgress("%s", line)
			last = line
		}
		if do.HarborJobTerminal(job.Status) {
			break
		}
		if wait.Timeout > 0 && time.Now().After(deadline) {
			return fmt.Errorf("job %s did not finish within %s; it is still %s (follow it with `doctl genai harbor job get %s --wait`)", job.JobUUID, wait.Timeout, job.Status, job.JobUUID)
		}
		harborSleep(wait.Interval)
		next, err := c.Harbor().GetJob(job.JobUUID)
		if err != nil {
			return err
		}
		job = next
	}

	if err := displayHarborJob(c, job); err != nil {
		return err
	}
	switch job.Status {
	case do.HarborJobFailed, do.HarborJobCancelled:
		msg := fmt.Sprintf("job %s %s", job.JobUUID, job.Status)
		if job.ErrorDescription != "" {
			msg += ": " + job.ErrorDescription
		}
		return errors.New(msg)
	case do.HarborJobPartiallySuccessful:
		warn("%d of %d trials of job %s errored; see `doctl genai harbor job trials %s`.", job.NErrored, job.NTrials, job.JobUUID, job.JobUUID)
	}
	return nil
}

func harborJobProgressLine(job *do.HarborJob) string {
	line := fmt.Sprintf("Job %s is %s: %d/%d trials complete", job.JobUUID, job.Status, job.NCompleted, job.NTrials)
	if job.NErrored > 0 {
		line += fmt.Sprintf(", %d errored", job.NErrored)
	}
	return line
}

func displayHarborJob(c *CmdConfig, job *do.HarborJob) error {
	return c.Display(&displayers.HarborJobs{Jobs: []do.HarborJob{*job}, Single: true})
}

// RunHarborJobTrials lists a job's trials.
func RunHarborJobTrials(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	trials, err := c.Harbor().ListTrials(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.HarborTrials{Trials: trials})
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

// RunHarborJobDownload downloads a job's results in Harbor's jobs layout.
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

	artifacts, err := c.Harbor().ListArtifacts(jobUUID)
	if err != nil {
		return err
	}
	if len(artifacts) == 0 {
		warn("Job %s has no results to download yet.", jobUUID)
	}

	// Resolve every path before downloading anything, so one bad entry
	// cannot leave a half-written folder behind.
	dests := make([]string, len(artifacts))
	for i, a := range artifacts {
		if dests[i], err = harborSafePath(dir, a.Path); err != nil {
			return err
		}
	}

	result := &do.HarborDownloadResult{JobUUID: jobUUID, Path: dir}
	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
		next = make(chan int)
	)
	for range min(harborDownloadPool, max(len(artifacts), 1)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				a := artifacts[i]
				if harborFileMatches(dests[i], a.SizeBytes, a.SHA256) {
					mu.Lock()
					result.Skipped++
					mu.Unlock()
					continue
				}
				n, err := downloadHarborFile(a.URL, dests[i], a.SizeBytes, a.SHA256)
				mu.Lock()
				if err != nil {
					errs = append(errs, fmt.Errorf("%s: %w", a.Path, err))
				} else {
					result.Downloaded++
					result.Bytes += n
				}
				mu.Unlock()
			}
		}()
	}
	for i := range artifacts {
		next <- i
	}
	close(next)
	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("%d of %d files failed to download; run the command again to retry them:\n%w", len(errs), len(artifacts), errors.Join(errs...))
	}
	if abs, err := filepath.Abs(dir); err == nil {
		result.Path = abs
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return c.Display(&displayers.HarborDownload{Result: result})
}
