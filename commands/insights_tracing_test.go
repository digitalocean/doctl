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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInsightsTracingTime(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	unixTime := time.Unix(1756684800, 0)

	tests := []struct {
		name  string
		value string
		want  time.Time
	}{
		{name: "now", value: "now", want: now},
		{name: "lookback", value: "1h", want: now.Add(-time.Hour)},
		{name: "negative offset", value: "now-15m", want: now.Add(-15 * time.Minute)},
		{name: "positive offset", value: "now+30m", want: now.Add(30 * time.Minute)},
		{name: "RFC3339", value: "2026-09-01T00:00:00.123456789Z", want: time.Date(2026, 9, 1, 0, 0, 0, 123456789, time.UTC)},
		{name: "Unix seconds", value: "1756684800", want: unixTime},
		{name: "Unix milliseconds", value: "1756684800000", want: unixTime},
		{name: "Unix microseconds", value: "1756684800000000", want: unixTime},
		{name: "Unix nanoseconds", value: "1756684800000000000", want: unixTime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInsightsTracingTime(tt.value, now)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseInsightsTracingTimeRejectsInvalidValues(t *testing.T) {
	now := time.Now().UTC()
	for _, value := range []string{"", "now-1x", "yesterday", "999999999999999999999"} {
		t.Run(value, func(t *testing.T) {
			_, err := parseInsightsTracingTime(value, now)
			assert.Error(t, err)
		})
	}
}

func TestInsightsTracingDurationRejectsOverflow(t *testing.T) {
	_, err := insightsTracingDuration("2562048", "h")
	assert.ErrorContains(t, err, "too large")
}

func TestValidateInsightsTracingTimeRange(t *testing.T) {
	assert.NoError(t, validateInsightsTracingTimeRange("7d", "now"))
	assert.ErrorContains(t, validateInsightsTracingTimeRange("8d", "now"), "must not exceed 7 days")
	assert.ErrorContains(t, validateInsightsTracingTimeRange("now", "1h"), "must not be after")
}
