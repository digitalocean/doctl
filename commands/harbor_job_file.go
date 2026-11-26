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
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// harborJobFile is the part of a Harbor job config the managed API accepts.
// Harbor's own schema is larger; fields that would run a different job are
// rejected, and fields that only affect the local CLI are reported and dropped.
type harborJobFile struct {
	JobName           string                 `yaml:"job_name"`
	NAttempts         *int                   `yaml:"n_attempts"`
	NConcurrentTrials *int                   `yaml:"n_concurrent_trials"`
	DryRun            *bool                  `yaml:"dry_run"`
	Retry             *harborJobFileRetry    `yaml:"retry"`
	Agents            []harborJobFileAgent   `yaml:"agents"`
	Datasets          []harborJobFileDataset `yaml:"datasets"`
	ignored           []string
}

type harborJobFileRetry struct {
	MaxRetries *int `yaml:"max_retries"`
}

type harborJobFileAgent struct {
	Name      string         `yaml:"name"`
	ModelName string         `yaml:"model_name"`
	Kwargs    map[string]any `yaml:"kwargs"`
}

type harborJobFileDataset struct {
	Name             string   `yaml:"name"`
	Path             string   `yaml:"path"`
	Repo             string   `yaml:"repo"`
	TaskNames        []string `yaml:"task_names"`
	ExcludeTaskNames []string `yaml:"exclude_task_names"`
	NTasks           *int     `yaml:"n_tasks"`
}

// loadHarborJobFile reads a Harbor job config. YAML and JSON both decode.
func loadHarborJobFile(path string) (*harborJobFile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading job file: %w", err)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("job file %s is empty", path)
	}

	var top map[string]any
	if err := yaml.Unmarshal(body, &top); err != nil {
		return nil, fmt.Errorf("parsing job file: %w", err)
	}
	if len(top) == 0 {
		return nil, fmt.Errorf("job file %s is empty", path)
	}
	if problems := harborJobFileProblems(top); len(problems) > 0 {
		return nil, fmt.Errorf("this job file is not one the managed Harbor API can run:\n- %s", strings.Join(problems, "\n- "))
	}

	var file harborJobFile
	if err := yaml.Unmarshal(body, &file); err != nil {
		return nil, fmt.Errorf("parsing job file: %w", err)
	}
	if err := file.checkDatasets(); err != nil {
		return nil, err
	}
	file.noteIgnored(top)
	if len(file.ignored) > 0 {
		warn("The job file sets %s. Those fields apply to the local harbor CLI and are not sent.", strings.Join(file.ignored, ", "))
	}
	return &file, nil
}

func (f *harborJobFile) checkDatasets() error {
	switch len(f.Datasets) {
	case 0:
		return nil
	case 1:
		d := f.Datasets[0]
		if strings.TrimSpace(d.Path) != "" || strings.TrimSpace(d.Repo) != "" {
			return fmt.Errorf("the job file selects a local dataset. Publish it first and set datasets[].name to that published name")
		}
		if strings.TrimSpace(d.Name) == "" {
			return fmt.Errorf("datasets[].name is required in the job file")
		}
		return nil
	default:
		return fmt.Errorf("a job runs one dataset; the file lists %d", len(f.Datasets))
	}
}

func harborJobFileProblems(top map[string]any) []string {
	var problems []string
	if !harborYAMLBlank(top["tasks"]) {
		problems = append(problems, "tasks: publish the dataset and set datasets[].name instead of a local task path")
	}
	if !harborYAMLBlank(top["environment"]) {
		problems = append(problems, "environment: each trial runs in its own sandbox, so this block is not sent")
	}
	if !harborYAMLBlank(top["user_agent"]) {
		problems = append(problems, "user_agent: a simulated user is not part of a managed job")
	}
	if !harborYAMLBlank(top["source_jobs"]) {
		problems = append(problems, "source_jobs: regrading a previous job is not part of a managed job")
	}
	if on, ok := top["install_only"].(bool); ok && on {
		problems = append(problems, "install_only: a managed job always runs the agent and the verifier")
	}
	return problems
}

func (f *harborJobFile) noteIgnored(top map[string]any) {
	accepted := map[string]bool{
		"job_name": true, "n_attempts": true, "n_concurrent_trials": true, "dry_run": true,
		"retry": true, "agents": true, "datasets": true,
	}
	for key, val := range top {
		if accepted[key] || harborYAMLBlank(val) {
			continue
		}
		f.ignored = append(f.ignored, key)
	}
	if retry, ok := top["retry"].(map[string]any); ok {
		for key, val := range retry {
			if key == "max_retries" || harborYAMLBlank(val) {
				continue
			}
			f.ignored = append(f.ignored, "retry."+key)
		}
	}
	sort.Strings(f.ignored)
}

func harborYAMLBlank(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t) == ""
	case bool:
		return !t
	case int:
		return t == 0
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	default:
		return false
	}
}
