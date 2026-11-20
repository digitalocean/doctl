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

package displayers

import (
	"bytes"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// APICLI-5566: a team on the credit prepay flow has no cash prepayment, so
// its balance is 0.00, and the Control Panel's Inference & Agents balance
// arrives as available_credit. The table must show that figure.
func TestHarnessPrepayBalance_ShowsInferenceAndAgentsBalance(t *testing.T) {
	b := &HarnessPrepayBalance{Status: &do.PrepaymentStatus{
		Balance:            "0.00",
		Eligible:           true,
		MonthToDateBalance: "38.66",
		AvailableCredit:    "9.29",
	}}

	var out bytes.Buffer
	require.NoError(t, DisplayText(b, &out, false, nil, ui.Env{}))

	assert.Equal(t,
		"Balance    Inference & Agents Balance    Month-to-date Balance    Auto Top-off    Status\n"+
			"$0.00      $9.29                         $38.66                   off             OK\n",
		out.String())
}
