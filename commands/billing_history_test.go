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

var testBillingHistoryList = &do.BillingHistory{
	BillingHistory: &godo.BillingHistory{
		BillingHistory: []godo.BillingHistoryEntry{
			{
				Description: "Invoice for May 2018",
				Amount:      "12.34",
				InvoiceID:   godo.String("123"),
				InvoiceUUID: godo.String("example-uuid"),
				Date:        time.Date(2018, 6, 1, 8, 44, 38, 0, time.UTC),
				Type:        "Invoice",
			},
			{
				Description: "Payment (MC 2018)",
				Amount:      "-12.34",
				InvoiceID:   nil,
				InvoiceUUID: nil,
				Date:        time.Date(2018, 6, 2, 8, 44, 38, 0, time.UTC),
				Type:        "Payment",
			},
		},
	},
}

func TestBillingHistoryCommand(t *testing.T) {
	historyCmd := BillingHistory()
	assert.NotNil(t, historyCmd)
	assertCommandNames(t, historyCmd, "list")
}

func TestBillingHistoryList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.billingHistory.EXPECT().List().Return(testBillingHistoryList, nil)

		err := RunBillingHistoryList(config)
		assert.NoError(t, err)
	})
}
