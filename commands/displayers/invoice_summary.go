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

type InvoiceSummary struct {
	*do.InvoiceSummary
}

var _ Displayable = &InvoiceSummary{}

func (i *InvoiceSummary) JSON(out io.Writer) error {
	return writeJSON(i.InvoiceSummary, out)
}

func (i *InvoiceSummary) Cols() []string {
	return []string{
		"InvoiceUUID",
		"BillingPeriod",
		"Amount",
		"UserName",
		"UserCompany",
		"UserEmail",
		"ProductCharges",
		"Overages",
		"Taxes",
		"CreditsAndAdjustments",
	}
}

func (i *InvoiceSummary) ColMap() map[string]string {
	return map[string]string{
		"InvoiceUUID":           "Invoice UUID",
		"BillingPeriod":         "Billing Period",
		"Amount":                "Amount",
		"UserName":              "User Name",
		"UserCompany":           "Company",
		"UserEmail":             "Email",
		"ProductCharges":        "Product Charges Amount",
		"Overages":              "Overages Amount",
		"Taxes":                 "Taxes Amount",
		"CreditsAndAdjustments": "Credits and Adjustments Amount",
	}
}

func (i *InvoiceSummary) KV() []map[string]interface{} {
	x := map[string]interface{}{
		"InvoiceUUID":           i.InvoiceSummary.InvoiceUUID,
		"BillingPeriod":         i.InvoiceSummary.BillingPeriod,
		"Amount":                i.InvoiceSummary.Amount,
		"UserName":              i.InvoiceSummary.UserName,
		"UserCompany":           i.InvoiceSummary.UserCompany,
		"UserEmail":             i.InvoiceSummary.UserEmail,
		"ProductCharges":        i.InvoiceSummary.ProductCharges.Amount,
		"Overages":              i.InvoiceSummary.Overages.Amount,
		"Taxes":                 i.InvoiceSummary.Taxes.Amount,
		"CreditsAndAdjustments": i.InvoiceSummary.CreditsAndAdjustments.Amount,
	}

	return []map[string]interface{}{x}
}
