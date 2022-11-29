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
	"bytes"
	"context"
	"sort"
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggersCommand(t *testing.T) {
	cmd := Triggers()
	assert.NotNil(t, cmd)
	expected := []string{"get", "list", "enable", "disable"}

	names := []string{}
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}

	sort.Strings(expected)
	sort.Strings(names)
	assert.Equal(t, expected, names)
}

func TestTriggersGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf
		config.Args = append(config.Args, "aTrigger")

		nextRunAt := time.Date(2022, 11, 3, 17, 3, 2, 0, time.UTC)
		theTrigger := do.ServerlessTrigger{
			Name:      "firePoll1",
			Namespace: "123-456",
			Function:  "misc/pollStatus",
			Type:      "SCHEDULED",
			IsEnabled: true,
			CreatedAt: time.Date(2022, 10, 5, 13, 46, 59, 0, time.UTC),
			UpdatedAt: time.Date(2022, 10, 17, 18, 41, 30, 0, time.UTC),
			ScheduledDetails: &do.TriggerScheduledDetails{
				Cron: "5 * * * *",
				Body: map[string]interface{}{
					"foo": "bar",
				},
			},
			ScheduledRuns: &do.TriggerScheduledRuns{
				NextRunAt: &nextRunAt,
			},
		}
		expect := `Name         Cron Expression    Invokes            Enabled    Last Run At
firePoll1    5 * * * *          misc/pollStatus    true       _
`
		tm.serverless.EXPECT().GetTrigger(context.TODO(), "aTrigger").Return(theTrigger, nil)

		err := RunTriggersGet(config)

		require.NoError(t, err)
		assert.Equal(t, expect, buf.String())
	})
}

func TestTriggersList(t *testing.T) {
	lastRunAt0 := time.Date(2022, 11, 3, 17, 3, 2, 0, time.UTC)

	theList := []do.ServerlessTrigger{
		{
			Name:      "fireGC",
			Namespace: "123-456",
			Function:  "misc/garbageCollect",
			Type:      "SCHEDULED",
			ScheduledDetails: &do.TriggerScheduledDetails{
				Cron: "* * * * *",
				Body: map[string]interface{}{},
			},
			IsEnabled: true,
			CreatedAt: time.Date(2022, 10, 5, 13, 46, 59, 0, time.UTC),
			UpdatedAt: time.Date(2022, 10, 17, 18, 41, 30, 0, time.UTC),
			ScheduledRuns: &do.TriggerScheduledRuns{
				LastRunAt: &lastRunAt0,
			},
		},
		{
			Name:     "firePoll1",
			Function: "misc/pollStatus",
			ScheduledDetails: &do.TriggerScheduledDetails{
				Cron: "5 * * * *",
			},
			IsEnabled: true,
			CreatedAt: time.Date(2022, 10, 5, 13, 46, 59, 0, time.UTC),
			UpdatedAt: time.Date(2022, 10, 17, 18, 41, 30, 0, time.UTC),
		},
		{
			Name:     "firePoll2",
			Function: "misc/pollStatus",
			ScheduledDetails: &do.TriggerScheduledDetails{
				Cron: "10 * * * *",
			},
			IsEnabled: false,
		},
	}
	tests := []struct {
		name           string
		doctlFlags     map[string]interface{}
		expectedOutput string
		listArg        string
		listResult     []do.ServerlessTrigger
	}{
		{
			name: "simple list",
			doctlFlags: map[string]interface{}{
				"no-header": "",
			},
			listResult: theList,
			expectedOutput: `fireGC       * * * * *     misc/garbageCollect    true     2022-11-03 17:03:02 +0000 UTC
firePoll1    5 * * * *     misc/pollStatus        true     _
firePoll2    10 * * * *    misc/pollStatus        false    _
`,
		},
		{
			name: "filtered list",
			doctlFlags: map[string]interface{}{
				"function":  "misc/pollStatus",
				"no-header": "",
			},
			listArg:    "misc/pollStatus",
			listResult: theList[1:],
			expectedOutput: `firePoll1    5 * * * *     misc/pollStatus    true     _
firePoll2    10 * * * *    misc/pollStatus    false    _
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				buf := &bytes.Buffer{}
				config.Out = buf
				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}

				tm.serverless.EXPECT().ListTriggers(context.TODO(), tt.listArg).Return(tt.listResult, nil)

				err := RunTriggersList(config)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, buf.String())
			})
		})
	}
}
