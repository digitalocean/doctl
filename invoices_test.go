package godo

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestInvoices_GetInvoices(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/customers/my/invoices/example-invoice-uuid", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{
			"invoice_items": [
				{
					"product": "Droplets",
					"resource_id": "1234",
					"resource_uuid": "droplet-1234-uuid",
					"group_description": "",
					"description": "My Example Droplet",
					"amount": "12.34",
					"duration": "672",
					"duration_unit": "Hours",
					"start_time": "2018-06-20T08:44:38Z",
					"end_time": "2018-06-21T08:44:38Z",
					"project_name": "My project"
				},
				{
					"product": "Load Balancers",
					"resource_id": "2345",
					"resource_uuid": "load-balancer-2345-uuid",
					"group_description": "",
					"description": "My Example Load Balancer",
					"amount": "23.45",
					"duration": "744",
					"duration_unit": "Hours",
					"start_time": "2018-06-20T08:44:38Z",
					"end_time": "2018-06-21T08:44:38Z",
					"project_name": "My Second Project"
				}
			],
			"meta": {
				"total": 2
			}
		}`)
	})

	invoice, resp, err := client.Invoices.Get(ctx, "example-invoice-uuid", nil)
	if err != nil {
		t.Errorf("Invoices.Get returned error: %v", err)
	}

	expectedInvoiceItems := []InvoiceItem{
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
		},
	}
	actualItems := invoice.InvoiceItems
	if !reflect.DeepEqual(actualItems, expectedInvoiceItems) {
		t.Errorf("Invoices.Get\nInvoiceItems: got=%#v\nwant=%#v", actualItems, expectedInvoiceItems)
	}
	expectedMeta := &Meta{Total: 2}
	if !reflect.DeepEqual(resp.Meta, expectedMeta) {
		t.Errorf("Invoices.Get\nMeta: got=%#v\nwant=%#v", resp.Meta, expectedMeta)
	}
}

func TestInvoices_ListInvoices(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/customers/my/invoices", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{
			"invoices": [
				{
				"invoice_uuid": "example-invoice-uuid-1",
				"amount": "12.34",
				"invoice_period": "2020-01"
				},
				{
				"invoice_uuid": "example-invoice-uuid-2",
				"amount": "23.45",
				"invoice_period": "2019-12"
				}
			],
			"invoice_preview": {
				"invoice_uuid": "example-invoice-uuid-preview",
				"amount": "34.56",
				"invoice_period": "2020-02",
				"updated_at": "2020-02-05T05:43:10Z"
			},
			"meta": {
				"total": 2
			}
			}`)
	})

	invoiceListResponse, resp, err := client.Invoices.List(ctx, nil)
	if err != nil {
		t.Errorf("Invoices.List returned error: %v", err)
	}

	expectedInvoiceListItems := []InvoiceListItem{
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
	}
	actualItems := invoiceListResponse.Invoices
	if !reflect.DeepEqual(actualItems, expectedInvoiceListItems) {
		t.Errorf("Invoices.List\nInvoiceListItems: got=%#v\nwant=%#v", actualItems, expectedInvoiceListItems)
	}

	expectedPreview := InvoiceListItem{
		InvoiceUUID:   "example-invoice-uuid-preview",
		Amount:        "34.56",
		InvoicePeriod: "2020-02",
		UpdatedAt:     time.Date(2020, 2, 5, 5, 43, 10, 0, time.UTC),
	}
	if !reflect.DeepEqual(invoiceListResponse.InvoicePreview, expectedPreview) {
		t.Errorf("Invoices.List\nInvoicePreview: got=%#v\nwant=%#v", invoiceListResponse.InvoicePreview, expectedPreview)
	}
	expectedMeta := &Meta{Total: 2}
	if !reflect.DeepEqual(resp.Meta, expectedMeta) {
		t.Errorf("Invoices.List\nMeta: got=%#v\nwant=%#v", resp.Meta, expectedMeta)
	}
}

func TestInvoices_GetSummary(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/customers/my/invoices/example-invoice-uuid/summary", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{
			"invoice_uuid": "example-invoice-uuid",
			"billing_period": "2020-01",
			"amount": "27.13",
			"user_name": "Frodo Baggins",
			"user_billing_address": {
				"address_line1": "101 Bagshot Row",
				"address_line2": "#2",
				"city": "Hobbiton",
				"region": "Shire",
				"postal_code": "12345",
				"country_iso2_code": "ME",
				"created_at": "2018-06-20T08:44:38Z",
				"updated_at": "2018-06-21T08:44:38Z"
			},
			"user_company": "DigitalOcean",
			"user_email": "fbaggins@example.com",
			"product_charges": {
				"name": "Product usage charges",
				"amount": "12.34",
				"items": [
				{
					"amount": "10.00",
					"name": "Spaces Subscription",
					"count": "1"
				},
				{
					"amount": "2.34",
					"name": "Database Clusters",
					"count": "1"
				}
				]
			},
			"overages": {
				"name": "Overages",
				"amount": "3.45"
			},
			"taxes": {
				"name": "Taxes",
				"amount": "4.56"
			},
			"credits_and_adjustments": {
				"name": "Credits & adjustments",
				"amount": "6.78"
			}
			}`)
	})

	invoiceSummaryResponse, _, err := client.Invoices.GetSummary(ctx, "example-invoice-uuid")
	if err != nil {
		t.Errorf("Invoices.GetSummary returned error: %v", err)
	}

	expectedSummary := InvoiceSummary{
		InvoiceUUID:   "example-invoice-uuid",
		BillingPeriod: "2020-01",
		Amount:        "27.13",
		UserName:      "Frodo Baggins",
		UserBillingAddress: Address{
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
		ProductCharges: InvoiceSummaryBreakdown{
			Name:   "Product usage charges",
			Amount: "12.34",
			Items: []InvoiceSummaryBreakdownItem{
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
		Overages: InvoiceSummaryBreakdown{
			Name:   "Overages",
			Amount: "3.45",
		},
		Taxes: InvoiceSummaryBreakdown{
			Name:   "Taxes",
			Amount: "4.56",
		},
		CreditsAndAdjustments: InvoiceSummaryBreakdown{
			Name:   "Credits & adjustments",
			Amount: "6.78",
		},
	}
	if !reflect.DeepEqual(invoiceSummaryResponse, &expectedSummary) {
		t.Errorf("Invoices.GetSummary\nInvoiceSummary: got=%#v\nwant=%#v", invoiceSummaryResponse, &expectedSummary)
	}
}

func TestInvoices_GetPDF(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/customers/my/invoices/example-invoice-uuid/pdf", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `some pdf content`)
	})

	invoicePDFResponse, _, err := client.Invoices.GetPDF(ctx, "example-invoice-uuid")
	if err != nil {
		t.Errorf("Invoices.GetPDF returned error: %v", err)
	}

	expected := []byte("some pdf content")
	if !bytes.Equal(invoicePDFResponse, expected) {
		t.Errorf("Invoices.GetPDF\n got=%#v\nwant=%#v", invoicePDFResponse, expected)
	}
}

func TestInvoices_GetCSV(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/customers/my/invoices/example-invoice-uuid/csv", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `some csv content`)
	})

	invoiceCSVResponse, _, err := client.Invoices.GetCSV(ctx, "example-invoice-uuid")
	if err != nil {
		t.Errorf("Invoices.GetCSV returned error: %v", err)
	}

	expected := []byte("some csv content")
	if !bytes.Equal(invoiceCSVResponse, expected) {
		t.Errorf("Invoices.GetCSV\n got=%#v\nwant=%#v", invoiceCSVResponse, expected)
	}
}
