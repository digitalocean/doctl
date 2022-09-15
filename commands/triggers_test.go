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

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggersCommand(t *testing.T) {
	cmd := Triggers()
	assert.NotNil(t, cmd)
	expected := []string{"disable", "enable", "fire", "get", "list"}

	names := []string{}
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}

	sort.Strings(expected)
	sort.Strings(names)
	assert.Equal(t, expected, names)
}

func TestTriggersDisable(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf
		config.Args = append(config.Args, "aTrigger")

		disabledTrigger := do.ServerlessTrigger{Enabled: false}
		tm.serverless.EXPECT().SetTriggerEnablement(context.TODO(), "aTrigger", false).Return(disabledTrigger, nil)

		err := RunTriggersDisable(config)

		require.NoError(t, err)
		assert.Equal(t, "", buf.String())
	})
}

func TestTriggersEnable(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf
		config.Args = append(config.Args, "aTrigger")

		enabledTrigger := do.ServerlessTrigger{Enabled: true}
		tm.serverless.EXPECT().SetTriggerEnablement(context.TODO(), "aTrigger", true).Return(enabledTrigger, nil)

		err := RunTriggersEnable(config)

		require.NoError(t, err)
		assert.Equal(t, "", buf.String())
	})
}

func TestTriggersFire(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf
		config.Args = append(config.Args, "aTrigger")

		tm.serverless.EXPECT().FireTrigger(context.TODO(), "aTrigger")

		err := RunTriggersFire(config)

		require.NoError(t, err)
		assert.Equal(t, "", buf.String())
	})
}

func TestTriggersList(t *testing.T) {
	theList := []do.ServerlessTrigger{
		{
			Name:     "fireGC",
			Function: "misc/garbageCollect",
			Cron:     "* * * * *",
			Enabled:  true,
		},
		{
			Name:     "firePoll1",
			Function: "misc/pollStatus",
			Cron:     "5 * * * *",
			Enabled:  true,
		},
		{
			Name:     "firePoll2",
			Function: "misc/pollStatus",
			Cron:     "10 * * * *",
			Enabled:  false,
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
			expectedOutput: `fireGC       * * * * *     misc/garbageCollect    true     
firePoll1    5 * * * *     misc/pollStatus        true     
firePoll2    10 * * * *    misc/pollStatus        false    
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
			expectedOutput: `firePoll1    5 * * * *     misc/pollStatus    true     
firePoll2    10 * * * *    misc/pollStatus    false    
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
