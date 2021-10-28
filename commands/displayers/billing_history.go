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

package displayers

import (
	"io"
	"time"

	"github.com/digitalocean/doctl/do"
)

type BillingHistory struct {
	*do.BillingHistory
}

var _ Displayable = &BillingHistory{}

func (i *BillingHistory) JSON(out io.Writer) error {
	return writeJSON(i.BillingHistory, out)
}

func (i *BillingHistory) Cols() []string {
	return []string{
		"Date", "Type", "Description", "Amount", "InvoiceID", "InvoiceUUID",
	}
}

func (i *BillingHistory) ColMap() map[string]string {
	return map[string]string{
		"Date":        "Date",
		"Type":        "Type",
		"Description": "Description",
		"Amount":      "Amount",
		"InvoiceID":   "Invoice ID",
		"InvoiceUUID": "Invoice UUID",
	}
}

func (i *BillingHistory) KV() []map[string]interface{} {
	fromStringP := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	history := i.BillingHistory.BillingHistory.BillingHistory
	out := make([]map[string]interface{}, 0, len(history))
	for _, ii := range history {
		x := map[string]interface{}{
			"Date":        ii.Date.Format(time.RFC3339),
			"Type":        ii.Type,
			"Description": ii.Description,
			"Amount":      ii.Amount,
			"InvoiceID":   fromStringP(ii.InvoiceID),
			"InvoiceUUID": fromStringP(ii.InvoiceUUID),
		}
		out = append(out, x)
	}

	return out
}
