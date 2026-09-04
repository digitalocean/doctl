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

package ui

import (
	"bytes"
	"testing"

	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

func TestToneFor(t *testing.T) {
	tests := []struct {
		name  string
		value string
		tone  Tone
	}{
		// Lowercase, as compute and databases report it.
		{name: "droplet active", value: "active", tone: ToneSuccess},
		{name: "droplet off", value: "off", tone: ToneMuted},
		{name: "droplet new", value: "new", tone: TonePending},
		{name: "action in-progress", value: "in-progress", tone: TonePending},
		{name: "action errored", value: "errored", tone: ToneError},
		{name: "database online", value: "online", tone: ToneSuccess},
		{name: "kubernetes degraded", value: "degraded", tone: ToneError},
		{name: "certificate verified", value: "verified", tone: ToneSuccess},

		// Upper, as NFS and dedicated inference report it.
		{name: "nfs share CREATING", value: "CREATING", tone: TonePending},
		{name: "nfs share FAILED", value: "FAILED", tone: ToneError},
		{name: "vpc peering ACTIVE", value: "ACTIVE", tone: ToneSuccess},

		// Proto-style, as apps and gradient AI report it. The meaning is not
		// always in the same position, which is why every word is tried.
		{name: "deployment phase leads", value: "PENDING_DEPLOY", tone: TonePending},
		{name: "deployment phase active", value: "ACTIVE", tone: ToneSuccess},
		{name: "deployment phase superseded", value: "SUPERSEDED", tone: ToneMuted},
		{name: "scenario set trails", value: "SCENARIO_SET_STATUS_READY", tone: ToneSuccess},
		{name: "indexing job trails", value: "BATCH_JOB_PHASE_SUCCEEDED", tone: ToneSuccess},
		{name: "access point trails", value: "ACCESS_POINT_FAILED", tone: ToneError},
		{name: "journey verdict", value: "SIMULATION_JOURNEY_VERDICT_INCONCLUSIVE", tone: TonePending},

		// Multi-word, as activations report it.
		{name: "activation success", value: "success", tone: ToneSuccess},
		{name: "activation developer error", value: "developer error", tone: ToneError},

		// Word matching does mean prose containing a state word is classified,
		// which is why callers gate on the column carrying state at all.
		{name: "prose containing a state word", value: "waiting on a human", tone: TonePending},

		// Unrecognized values are left alone rather than guessed at.
		{name: "identifier", value: "f4e37431-a0f4-458f", tone: ToneNone},
		{name: "message", value: "could not reach the region", tone: ToneNone},
		{name: "empty", value: "", tone: ToneNone},
		{name: "whitespace", value: "   ", tone: ToneNone},
		{name: "boolean", value: "true", tone: ToneNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tone, ok := ToneFor(tt.value)
			assert.Equal(t, tt.tone, tone)
			assert.Equal(t, tt.tone != ToneNone, ok, "known should track whether a tone was found")
		})
	}
}

func TestSprintTone(t *testing.T) {
	var out, errOut bytes.Buffer

	// The green slot, not a fixed green: the profile is TrueColor and the
	// sequence is still the plain SGR 32, because the palette names a slot and
	// leaves the value to whatever theme the user is running.
	styled := Detect(&out, &errOut, WithProfile(termenv.TrueColor))
	assert.Equal(t, "\x1b[32mactive\x1b[0m", styled.SprintTone(ToneSuccess, "active"),
		"success renders in ColorSuccess")
	assert.Equal(t, "active", styled.SprintTone(ToneNone, "active"),
		"an unclassified value must not be painted")

	plain := Plain(&out, &errOut)
	assert.Equal(t, "active", plain.SprintTone(ToneSuccess, "active"),
		"styling must stay off when the env forbids it")
}
