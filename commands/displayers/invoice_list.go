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

	"github.com/digitalocean/doctl/do"
)

type InvoiceList struct {
	*do.InvoiceList
}

var _ Displayable = &InvoiceList{}

func (i *InvoiceList) JSON(out io.Writer) error {
	return writeJSON(i.InvoiceList, out)
}

func (i *InvoiceList) Cols() []string {
	return []string{
		"InvoiceUUID", "Amount", "InvoicePeriod",
	}
}

func (i *InvoiceList) ColMap() map[string]string {
	return map[string]string{
		"InvoiceUUID":   "Invoice UUID",
		"Amount":        "Amount",
		"InvoicePeriod": "Invoice Period",
	}
}

func (i *InvoiceList) KV() []map[string]interface{} {
	invoices := i.InvoiceList.Invoices
	out := make([]map[string]interface{}, 0, len(invoices)+1)
	x := map[string]interface{}{
		"InvoiceUUID":   "preview",
		"Amount":        i.InvoicePreview.Amount,
		"InvoicePeriod": i.InvoicePreview.InvoicePeriod,
	}
	out = append(out, x)
	for _, ii := range invoices {
		x := map[string]interface{}{
			"InvoiceUUID":   ii.InvoiceUUID,
			"Amount":        ii.Amount,
			"InvoicePeriod": ii.InvoicePeriod,
		}
		out = append(out, x)
	}

	return out
}
