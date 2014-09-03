package godo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestDomains_AllRecordsForDomainName(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/domains/example.com/records", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"domain_records":[{"id":1},{"id":2}]}`)
	})

	records, _, err := client.Domains.Records("example.com", nil)
	if err != nil {
		t.Errorf("Domains.List returned error: %v", err)
	}

	expected := []DomainRecord{{ID: 1}, {ID: 2}}
	if !reflect.DeepEqual(records, expected) {
		t.Errorf("Domains.List returned %+v, expected %+v", records, expected)
	}
}

func TestDomains_AllRecordsForDomainName_PerPage(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/domains/example.com/records", func(w http.ResponseWriter, r *http.Request) {
		perPage := r.URL.Query().Get("per_page")
		if perPage != "2" {
			t.Fatalf("expected '2', got '%s'", perPage)
		}

		fmt.Fprint(w, `{"domain_records":[{"id":1},{"id":2}]}`)
	})

	dro := &DomainRecordsOptions{ListOptions{PerPage: 2}}
	records, _, err := client.Domains.Records("example.com", dro)
	if err != nil {
		t.Errorf("Domains.List returned error: %v", err)
	}

	expected := []DomainRecord{{ID: 1}, {ID: 2}}
	if !reflect.DeepEqual(records, expected) {
		t.Errorf("Domains.List returned %+v, expected %+v", records, expected)
	}
}

func TestDomains_GetRecordforDomainName(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/domains/example.com/records/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"domain_record":{"id":1}}`)
	})

	record, _, err := client.Domains.Record("example.com", 1)
	if err != nil {
		t.Errorf("Domains.GetRecord returned error: %v", err)
	}

	expected := &DomainRecord{ID: 1}
	if !reflect.DeepEqual(record, expected) {
		t.Errorf("Domains.GetRecord returned %+v, expected %+v", record, expected)
	}
}

func TestDomains_DeleteRecordForDomainName(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/domains/example.com/records/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
	})

	_, err := client.Domains.DeleteRecord("example.com", 1)
	if err != nil {
		t.Errorf("Domains.RecordDelete returned error: %v", err)
	}
}

func TestDomains_CreateRecordForDomainName(t *testing.T) {
	setup()
	defer teardown()

	createRequest := &DomainRecordEditRequest{
		Type:     "CNAME",
		Name:     "example",
		Data:     "@",
		Priority: 10,
		Port:     10,
		Weight:   10,
	}

	mux.HandleFunc("/v2/domains/example.com/records",
		func(w http.ResponseWriter, r *http.Request) {
			v := new(DomainRecordEditRequest)
			json.NewDecoder(r.Body).Decode(v)

			testMethod(t, r, "POST")
			if !reflect.DeepEqual(v, createRequest) {
				t.Errorf("Request body = %+v, expected %+v", v, createRequest)
			}

			fmt.Fprintf(w, `{"domain_record": {"id":1}}`)
		})

	record, _, err := client.Domains.CreateRecord("example.com", createRequest)
	if err != nil {
		t.Errorf("Domains.CreateRecord returned error: %v", err)
	}

	expected := &DomainRecord{ID: 1}
	if !reflect.DeepEqual(record, expected) {
		t.Errorf("Domains.CreateRecord returned %+v, expected %+v", record, expected)
	}
}

func TestDomains_EditRecordForDomainName(t *testing.T) {
	setup()
	defer teardown()

	editRequest := &DomainRecordEditRequest{
		Type:     "CNAME",
		Name:     "example",
		Data:     "@",
		Priority: 10,
		Port:     10,
		Weight:   10,
	}

	mux.HandleFunc("/v2/domains/example.com/records/1", func(w http.ResponseWriter, r *http.Request) {
		v := new(DomainRecordEditRequest)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "PUT")
		if !reflect.DeepEqual(v, editRequest) {
			t.Errorf("Request body = %+v, expected %+v", v, editRequest)
		}

		fmt.Fprintf(w, `{"id":1}`)
	})

	record, _, err := client.Domains.EditRecord("example.com", 1, editRequest)
	if err != nil {
		t.Errorf("Domains.EditRecord returned error: %v", err)
	}

	expected := &DomainRecord{ID: 1}
	if !reflect.DeepEqual(record, expected) {
		t.Errorf("Domains.EditRecord returned %+v, expected %+v", record, expected)
	}
}

func TestDomainRecord_String(t *testing.T) {
	record := &DomainRecord{
		ID:       1,
		Type:     "CNAME",
		Name:     "example",
		Data:     "@",
		Priority: 10,
		Port:     10,
		Weight:   10,
	}

	stringified := record.String()
	expected := `godo.DomainRecord{ID:1, Type:"CNAME", Name:"example", Data:"@", Priority:10, Port:10, Weight:10}`
	if expected != stringified {
		t.Errorf("DomainRecord.String returned %+v, expected %+v", stringified, expected)
	}
}

func TestDomainRecordEditRequest_String(t *testing.T) {
	record := &DomainRecordEditRequest{
		Type:     "CNAME",
		Name:     "example",
		Data:     "@",
		Priority: 10,
		Port:     10,
		Weight:   10,
	}

	stringified := record.String()
	expected := `godo.DomainRecordEditRequest{Type:"CNAME", Name:"example", Data:"@", Priority:10, Port:10, Weight:10}`
	if expected != stringified {
		t.Errorf("DomainRecordEditRequest.String returned %+v, expected %+v", stringified, expected)
	}
}
