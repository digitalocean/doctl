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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var testInvoicesList = &do.InvoiceList{
	InvoiceList: &godo.InvoiceList{
		Invoices: []godo.InvoiceListItem{
			{
				InvoiceUUID:   "example-invoice-uuid-1",
				Amount:        "12.34",
				InvoicePeriod: "2020-01",
			},
			{
				InvoiceUUID:   "example-invoice-uuid-2",
				Amount:        "23.45",
				InvoicePeriod: "2019-12",
			},
		},
		InvoicePreview: godo.InvoiceListItem{
			InvoiceUUID:   "example-invoice-uuid-preview",
			Amount:        "34.56",
			InvoicePeriod: "2020-02",
			UpdatedAt:     time.Date(2020, 2, 5, 5, 43, 10, 0, time.UTC),
		},
	},
}

var testInvoicesGet = &do.Invoice{
	Invoice: &godo.Invoice{
		InvoiceItems: []godo.InvoiceItem{
			{
				Product:          "Droplets",
				ResourceID:       "1234",
				ResourceUUID:     "droplet-1234-uuid",
				GroupDescription: "",
				Description:      "My Example Droplet",
				Amount:           "12.34",
				Duration:         "672",
				DurationUnit:     "Hours",
				StartTime:        time.Date(2018, 6, 20, 8, 44, 38, 0, time.UTC),
				EndTime:          time.Date(2018, 6, 21, 8, 44, 38, 0, time.UTC),
				ProjectName:      "My project",
				Category:         "iaas",
			},
			{
				Product:          "Load Balancers",
				ResourceID:       "2345",
				ResourceUUID:     "load-balancer-2345-uuid",
				GroupDescription: "",
				Description:      "My Example Load Balancer",
				Amount:           "23.45",
				Duration:         "744",
				DurationUnit:     "Hours",
				StartTime:        time.Date(2018, 6, 20, 8, 44, 38, 0, time.UTC),
				EndTime:          time.Date(2018, 6, 21, 8, 44, 38, 0, time.UTC),
				ProjectName:      "My Second Project",
				Category:         "paas",
			},
		},
	},
}

var testInvoiceSummary = &do.InvoiceSummary{
	InvoiceSummary: &godo.InvoiceSummary{
		InvoiceUUID:   "example-invoice-uuid",
		BillingPeriod: "2020-01",
		Amount:        "27.13",
		UserName:      "Frodo Baggins",
		UserBillingAddress: godo.Address{
			AddressLine1:    "101 Bagshot Row",
			AddressLine2:    "#2",
			City:            "Hobbiton",
			Region:          "Shire",
			PostalCode:      "12345",
			CountryISO2Code: "ME",
			CreatedAt:       time.Date(2018, 6, 20, 8, 44, 38, 0, time.UTC),
			UpdatedAt:       time.Date(2018, 6, 21, 8, 44, 38, 0, time.UTC),
		},
		UserCompany: "DigitalOcean",
		UserEmail:   "fbaggins@example.com",
		ProductCharges: godo.InvoiceSummaryBreakdown{
			Name:   "Product usage charges",
			Amount: "12.34",
			Items: []godo.InvoiceSummaryBreakdownItem{
				{
					Name:   "Spaces Subscription",
					Amount: "10.00",
					Count:  "1",
				},
				{
					Name:   "Database Clusters",
					Amount: "2.34",
					Count:  "1",
				},
			},
		},
		Overages: godo.InvoiceSummaryBreakdown{
			Name:   "Overages",
			Amount: "3.45",
		},
		Taxes: godo.InvoiceSummaryBreakdown{
			Name:   "Taxes",
			Amount: "4.56",
		},
		CreditsAndAdjustments: godo.InvoiceSummaryBreakdown{
			Name:   "Credits & adjustments",
			Amount: "6.78",
		},
	},
}

func TestInvoicesCommand(t *testing.T) {
	invoicesCmd := Invoices()
	assert.NotNil(t, invoicesCmd)
	assertCommandNames(t, invoicesCmd, "get", "list", "summary", "csv", "pdf")
}

func TestInvoicesGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.invoices.EXPECT().Get("example-invoice-uuid").Return(testInvoicesGet, nil)

		config.Args = append(config.Args, "example-invoice-uuid")

		err := RunInvoicesGet(config)
		assert.NoError(t, err)
	})
}

func TestInvoicesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.invoices.EXPECT().List().Return(testInvoicesList, nil)

		err := RunInvoicesList(config)
		assert.NoError(t, err)
	})
}

func TestInvoicesSummary(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.invoices.EXPECT().GetSummary("example-invoice-uuid").Return(testInvoiceSummary, nil)

		config.Args = append(config.Args, "example-invoice-uuid")

		err := RunInvoicesSummary(config)
		assert.NoError(t, err)
	})
}

func TestInvoicesGetPDF(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		path := os.TempDir()
		content := []byte("pdf response")
		fileUUID := uuid.New().String()

		tm.invoices.EXPECT().GetPDF("example-invoice-uuid").Return(content, nil)
		fpath := filepath.Join(path, fileUUID)

		config.Args = append(config.Args, "example-invoice-uuid", fpath)

		err := RunInvoicesGetPDF(config)
		assert.NoError(t, err)

		// Assert the file exists
		result, err := ioutil.ReadFile(fpath)
		assert.NoError(t, err)
		assert.Equal(t, content, result)

		os.Remove(fpath)
	})
}

func TestInvoicesGetCSV(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		path := os.TempDir()
		content := []byte("csv response")
		fileUUID := uuid.New().String()

		tm.invoices.EXPECT().GetCSV("example-invoice-uuid").Return(content, nil)
		fpath := filepath.Join(path, fileUUID)

		config.Args = append(config.Args, "example-invoice-uuid", fpath)

		err := RunInvoicesGetCSV(config)
		assert.NoError(t, err)

		// Assert the file exists
		result, err := ioutil.ReadFile(fpath)
		assert.NoError(t, err)
		assert.Equal(t, content, result)

		os.Remove(fpath)
	})
}

func TestOutputFileArg(t *testing.T) {
	result := getOutputFileArg("pdf", []string{"invoice-uuid"})
	assert.Equal(t, "invoice.pdf", result)

	result = getOutputFileArg("pdf", []string{"invoice-uuid", "target.any"})
	assert.Equal(t, "target.any", result)
}
