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

package doctl

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSnapPackagingWorkflows(t *testing.T) {
	for _, tt := range []struct {
		path, job, channel, trigger, ref string
	}{
		{".github/workflows/release.yml", "snapcraft-stable", "stable", "tags", "v[1-9].[0-9]+.[0-9]+"},
		{".github/workflows/snapcraft-candidate.yml", "build-and-publish", "candidate", "branches", "main"},
	} {
		t.Run(tt.channel, func(t *testing.T) {
			data, err := os.ReadFile(tt.path)
			require.NoError(t, err)
			var workflow struct {
				On struct {
					Push map[string][]string
				}
				Jobs map[string]struct {
					Needs    string
					RunsOn   string `yaml:"runs-on"`
					Strategy struct {
						FailFast *bool `yaml:"fail-fast"`
						Matrix   struct {
							Include []map[string]string
						}
					}
					Steps []struct {
						ID, Uses, Run, Shell, If string
						ContinueOnError          bool `yaml:"continue-on-error"`
						Env, With                map[string]string
					}
				}
			}
			require.NoError(t, yaml.Unmarshal(data, &workflow))
			assert.Equal(t, []string{tt.ref}, workflow.On.Push[tt.trigger])
			job, ok := workflow.Jobs[tt.job]
			require.True(t, ok, "Snap job missing")
			if tt.channel == "stable" {
				assert.Equal(t, "integration-tests", job.Needs)
			}
			assert.Equal(t, "${{ matrix.runner }}", job.RunsOn)
			require.NotNil(t, job.Strategy.FailFast)
			assert.False(t, *job.Strategy.FailFast)
			assert.ElementsMatch(t, []map[string]string{
				{"arch": "amd64", "runner": "ubuntu-24.04", "elf-machine": "Advanced Micro Devices X86-64"},
				{"arch": "arm64", "runner": "ubuntu-24.04-arm", "elf-machine": "AArch64"},
			}, job.Strategy.Matrix.Include)

			checkout, build, verify, publish := -1, -1, -1, -1
			for i, step := range job.Steps {
				switch {
				case strings.HasPrefix(step.Uses, "actions/checkout@"):
					checkout = i
					assert.Equal(t, "0", step.With["fetch-depth"])
				case step.Uses == "snapcore/action-build@v1":
					build = i
					assert.Equal(t, "build", step.ID)
				case step.ID == "verify":
					verify = i
					assert.Equal(t, "bash", step.Shell)
					assert.Empty(t, step.If)
					assert.False(t, step.ContinueOnError)
					assert.Equal(t, "${{ steps.build.outputs.snap }}", step.Env["SNAP_PATH"])
					assert.Equal(t, "${{ matrix.arch }}", step.Env["SNAP_ARCH"])
					assert.Equal(t, "${{ matrix.elf-machine }}", step.Env["ELF_MACHINE"])
					for _, check := range []string{
						"set -euo pipefail",
						`unsquashfs -d "$snap_dir" "$SNAP_PATH"`,
						`yaml.safe_load`,
						`["architectures"] == [os.environ["SNAP_ARCH"]]`,
						`readelf -h "$snap_dir/bin/doctl.real"`,
						`grep -q "Machine:.*${ELF_MACHINE}"`,
						`"$snap_dir/bin/doctl.real" version`,
					} {
						assert.Contains(t, step.Run, check)
					}
				case step.Uses == "snapcore/action-publish@master":
					publish = i
					assert.Empty(t, step.If)
					assert.Equal(t, "${{ steps.build.outputs.snap }}", step.With["snap"])
					assert.Equal(t, tt.channel, step.With["release"])
					assert.Equal(t, "${{ secrets.SNAP_TOKEN }}", step.Env["SNAPCRAFT_STORE_CREDENTIALS"])
				}
			}
			assert.GreaterOrEqual(t, checkout, 0)
			assert.Greater(t, build, checkout)
			assert.Greater(t, verify, build)
			assert.Greater(t, publish, verify)
		})
	}

	data, err := os.ReadFile(".github/workflows/beta-release.yml")
	require.NoError(t, err)
	var beta struct {
		Jobs map[string]struct {
			Steps []struct{ Uses string }
		}
	}
	require.NoError(t, yaml.Unmarshal(data, &beta))
	for _, job := range beta.Jobs {
		for _, step := range job.Steps {
			assert.False(t, strings.HasPrefix(step.Uses, "snapcore/action-publish@"), "beta must not publish Snaps")
		}
	}
}

func TestSnapPackagingSource(t *testing.T) {
	data, err := os.ReadFile("snap/snapcraft.yaml")
	require.NoError(t, err)
	var recipe struct {
		Parts map[string]map[string]interface{}
	}
	require.NoError(t, yaml.Unmarshal(data, &recipe))
	assert.Equal(t, ".", recipe.Parts["doctl"]["source"])
	assert.NotContains(t, recipe.Parts["doctl"], "source-type")
}
