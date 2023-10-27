/*
Copyright 2023 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testUptimeCheck = do.UptimeCheck{
		UptimeCheck: &godo.UptimeCheck{
			ID:      "00000000-0000-4000-8000-000000000000",
			Name:    "Test Check",
			Type:    "https",
			Target:  "https://digitalocean.com",
			Regions: []string{"us_east"},
			Enabled: true,
		},
	}
	testUptimeChecksList = []do.UptimeCheck{
		testUptimeCheck,
	}
)

func TestUptimeCheckCommand(t *testing.T) {
	cmd := UptimeCheck()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "get", "update", "delete", "alert")
}

func TestUptimeChecksCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tuc := godo.CreateUptimeCheckRequest{
			Name:    "Test Check",
			Type:    "https",
			Target:  "https://digitalocean.com",
			Regions: []string{"us_east"},
			Enabled: true,
		}
		tm.uptimeChecks.EXPECT().Create(&tuc).Return(&testUptimeCheck, nil)

		config.Args = append(config.Args, "Test Check")

		config.Doit.Set(config.NS, doctl.ArgUptimeCheckType, "https")
		config.Doit.Set(config.NS, doctl.ArgUptimeCheckTarget, "https://digitalocean.com")
		config.Doit.Set(config.NS, doctl.ArgUptimeCheckRegions, []string{"us_east"})
		config.Doit.Set(config.NS, doctl.ArgUptimeCheckEnabled, true)

		err := RunUptimeChecksCreate(config)
		assert.NoError(t, err)
	})
}

func TestUptimeChecksList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.uptimeChecks.EXPECT().List().Return(testUptimeChecksList, nil)

		err := RunUptimeChecksList(config)
		assert.NoError(t, err)
	})
}

func TestUptimeChecksGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.uptimeChecks.EXPECT().Get("00000000-0000-4000-8000-000000000000").Return(&testUptimeCheck, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		err := RunUptimeChecksGet(config)
		assert.NoError(t, err)
	})
}

func TestUptimeChecksUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tuc := godo.UpdateUptimeCheckRequest{
			Name:    "Test Check",
			Type:    "https",
			Target:  "https://digitalocean.com",
			Regions: []string{"us_east"},
			Enabled: false,
		}
		tm.uptimeChecks.EXPECT().Update("00000000-0000-4000-8000-000000000000", &tuc).Return(&testUptimeCheck, nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		config.Doit.Set(config.NS, doctl.ArgUptimeCheckName, "Test Check")
		config.Doit.Set(config.NS, doctl.ArgUptimeCheckType, "https")
		config.Doit.Set(config.NS, doctl.ArgUptimeCheckTarget, "https://digitalocean.com")
		config.Doit.Set(config.NS, doctl.ArgUptimeCheckRegions, []string{"us_east"})
		config.Doit.Set(config.NS, doctl.ArgUptimeCheckEnabled, false)

		err := RunUptimeChecksUpdate(config)
		assert.NoError(t, err)
	})
}

func TestUptimeChecksDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.uptimeChecks.EXPECT().Delete("00000000-0000-4000-8000-000000000000").Return(nil)

		config.Args = append(config.Args, "00000000-0000-4000-8000-000000000000")

		err := RunUptimeChecksDelete(config)
		assert.NoError(t, err)
	})
}
