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
	"errors"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRunAgentsBalance(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		cfg := &do.PrepaymentConfig{
			SpendLimit:      "100.00",
			PrepayAmount:    "25.00",
			PrepayThreshold: "5.00",
		}
		status := &do.PrepaymentStatus{
			Balance:            "12.50",
			Eligible:           true,
			MonthToDateBalance: "3.25",
		}
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(cfg, status, nil)

		require.NoError(t, RunAgentsBalance(config))
	})
}

// A token without billing:read is a state to report, not a command failure:
// the user can act on it by asking a Team Owner, but not by retrying.
func TestRunAgentsBalance_NoBillingPermission(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(nil, nil, do.ErrNoBillingPermission)

		assert.NoError(t, RunAgentsBalance(config))
	})
}

func TestRunAgentsBalance_OtherErrorsPropagate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.prepayment.EXPECT().Get(gomock.Any()).Return(nil, nil, errors.New("billing is down"))

		assert.EqualError(t, RunAgentsBalance(config), "billing is down")
	})
}
