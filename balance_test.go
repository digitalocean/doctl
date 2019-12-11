package godo

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestBalanceGet(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/customers/my/balance", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)

		response := `
		{
			"month_to_date_balance": "23.44",
			"account_balance": "12.23",
			"month_to_date_usage": "11.21",
			"generated_at": "2018-06-21T08:44:38Z"
		}
		`

		fmt.Fprint(w, response)
	})

	bal, _, err := client.Balance.Get(ctx)
	if err != nil {
		t.Errorf("Balance.Get returned error: %v", err)
	}

	expected := &Balance{
		MonthToDateBalance: "23.44",
		AccountBalance:     "12.23",
		MonthToDateUsage:   "11.21",
		GeneratedAt:        time.Date(2018, 6, 21, 8, 44, 38, 0, time.UTC),
	}
	if !reflect.DeepEqual(bal, expected) {
		t.Errorf("Balance.Get returned %+v, expected %+v", bal, expected)
	}
}
