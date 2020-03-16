package godo

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestBillingHistory_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/customers/my/billing_history", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{
			"billing_history": [
				{
					"description": "Invoice for May 2018",
					"amount": "12.34",
					"invoice_id": "123",
					"invoice_uuid": "example-uuid",
					"date": "2018-06-01T08:44:38Z",
					"type": "Invoice"
				},
				{
					"description": "Payment (MC 2018)",
					"amount": "-12.34",
					"date": "2018-06-02T08:44:38Z",
					"type": "Payment"
				}
			],
			"meta": {
				"total": 2
			}
		}`)
	})

	history, resp, err := client.BillingHistory.List(ctx, nil)
	if err != nil {
		t.Errorf("BillingHistory.List returned error: %v", err)
	}

	expectedBillingHistory := []BillingHistoryEntry{
		{
			Description: "Invoice for May 2018",
			Amount:      "12.34",
			InvoiceID:   String("123"),
			InvoiceUUID: String("example-uuid"),
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
	}
	entries := history.BillingHistory
	if !reflect.DeepEqual(entries, expectedBillingHistory) {
		t.Errorf("BillingHistory.List\nBillingHistory: got=%#v\nwant=%#v", entries, expectedBillingHistory)
	}
	expectedMeta := &Meta{Total: 2}
	if !reflect.DeepEqual(resp.Meta, expectedMeta) {
		t.Errorf("BillingHistory.List\nMeta: got=%#v\nwant=%#v", resp.Meta, expectedMeta)
	}
}
