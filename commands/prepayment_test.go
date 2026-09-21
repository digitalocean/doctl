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

var testPrepaymentConfig = &do.PrepaymentConfigResponse{
	PrepaymentConfigResponse: &godo.PrepaymentConfigResponse{
		Config: &godo.PrepaymentConfig{
			SpendLimit:          "100.00",
			IsAutoPrepayEnabled: true,
			PrepayAmount:        "50.00",
			PrepayThreshold:     "10.00",
			CreatedAt:           time.Date(2026, 6, 22, 19, 31, 51, 0, time.UTC),
			UpdatedAt:           time.Date(2026, 6, 25, 18, 40, 7, 0, time.UTC),
		},
		Status: &godo.PrepaymentStatus{
			Balance:             "25.00",
			IsAutoPrepayEnabled: true,
			Blocked:             true,
			Eligible:            true,
			MonthToDateBalance:  "75.00",
		},
	},
}

var testPrepaymentStatus = &do.PrepaymentStatusResponse{
	PrepaymentStatusResponse: &godo.PrepaymentStatusResponse{
		Status: &godo.PrepaymentStatus{
			Balance:             "25.00",
			IsAutoPrepayEnabled: true,
			Blocked:             true,
			Eligible:            true,
			MonthToDateBalance:  "75.00",
		},
	},
}

func TestPrepaymentCommand(t *testing.T) {
	cmd := Prepayment()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "config", "status")
}

func TestPrepaymentConfigGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.prepayment.EXPECT().GetConfig().Return(testPrepaymentConfig, nil)

		err := RunPrepaymentConfigGet(config)
		assert.NoError(t, err)
	})
}

func TestPrepaymentStatusGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.prepayment.EXPECT().GetStatus().Return(testPrepaymentStatus, nil)

		err := RunPrepaymentStatusGet(config)
		assert.NoError(t, err)
	})
}
