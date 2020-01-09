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

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var testBalance = &do.Balance{
	Balance: &godo.Balance{
		MonthToDateBalance: "23.44",
		AccountBalance:     "12.23",
		MonthToDateUsage:   "11.21",
		GeneratedAt:        time.Date(2018, 6, 21, 8, 44, 38, 0, time.UTC),
	},
}

func TestBalanceCommand(t *testing.T) {
	acctCmd := Balance()
	assert.NotNil(t, acctCmd)
	assertCommandNames(t, acctCmd, "get")
}

func TestBalanceGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.balance.EXPECT().Get().Return(testBalance, nil)

		err := RunBalanceGet(config)
		assert.NoError(t, err)
	})
}
