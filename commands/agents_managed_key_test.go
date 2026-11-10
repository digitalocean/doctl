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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeedsManagedModelAccessKey(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		want     bool
	}{
		{
			name:     "model only",
			manifest: "harness: openhands\nenv:\n  HARNESS_INFERENCE_MODEL: openai-gpt-5.5\n",
			want:     true,
		},
		{
			name: "model plus harness key value",
			manifest: "harness: openhands\nenv:\n  HARNESS_INFERENCE_MODEL: openai-gpt-5.5\n" +
				"secrets:\n  HARNESS_INFERENCE_API_KEY: sk-do-byok\n",
			want: false,
		},
		{
			name: "model plus empty harness key slot",
			manifest: "harness: openhands\nenv:\n  HARNESS_INFERENCE_MODEL: openai-gpt-5.5\n" +
				"secrets:\n  HARNESS_INFERENCE_API_KEY: \"\"\n",
			want: true,
		},
		{
			name: "model plus openai key",
			manifest: "harness: hermes\nenv:\n  HARNESS_INFERENCE_MODEL: openai-gpt-5.5\n" +
				"secrets:\n  OPENAI_API_KEY: sk-openai\n",
			want: false,
		},
		{
			name: "model plus anthropic key",
			manifest: "harness: openhands\nenv:\n  HARNESS_INFERENCE_MODEL: openai-gpt-5.5\n" +
				"secrets:\n  ANTHROPIC_API_KEY: sk-ant\n",
			want: false,
		},
		{
			name:     "no model",
			manifest: "harness: opencode\n",
			want:     false,
		},
		{
			name: "legacy envelope with model only",
			manifest: "apiVersion: agents.digitalocean.com/v1alpha1\nkind: Agent\n" +
				"spec:\n  runtime:\n    adapter: openhands\n  env:\n    HARNESS_INFERENCE_MODEL: openai-gpt-5.5\n",
			want: true,
		},
		{
			name: "harness key only in env",
			manifest: "harness: openhands\nenv:\n  HARNESS_INFERENCE_MODEL: openai-gpt-5.5\n" +
				"  HARNESS_INFERENCE_API_KEY: sk-do-env\n",
			want: false,
		},
		{
			name: "unresolved key placeholder does not count as provided",
			manifest: "harness: hermes\nenv:\n  HARNESS_INFERENCE_MODEL: openai-gpt-5.5\n" +
				"secrets:\n  OPENAI_API_KEY: ${OPENAI_API_KEY}\n",
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, needsManagedModelAccessKey([]byte(tc.manifest)))
		})
	}
}
