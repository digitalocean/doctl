package godo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestStoragesActions_Attach(t *testing.T) {
	setup()
	defer teardown()
	const (
		volumeID  = "98d414c6-295e-4e3a-ac58-eb9456c1e1d1"
		dropletID = 12345
	)

	attachRequest := &ActionRequest{
		"type":       "attach",
		"droplet_id": float64(dropletID), // encoding/json decodes numbers as floats
	}

	mux.HandleFunc("/v2/volumes/"+volumeID+"/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")
		if !reflect.DeepEqual(v, attachRequest) {
			t.Errorf("want=%#v", attachRequest)
			t.Errorf("got=%#v", v)
		}
		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)
	})

	_, _, err := client.StorageActions.Attach(volumeID, dropletID)
	if err != nil {
		t.Errorf("StoragesActions.Attach returned error: %v", err)
	}
}

func TestStoragesActions_Detach(t *testing.T) {
	setup()
	defer teardown()
	volumeID := "98d414c6-295e-4e3a-ac58-eb9456c1e1d1"

	detachRequest := &ActionRequest{
		"type": "detach",
	}

	mux.HandleFunc("/v2/volumes/"+volumeID+"/actions", func(w http.ResponseWriter, r *http.Request) {
		v := new(ActionRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		testMethod(t, r, "POST")
		if !reflect.DeepEqual(v, detachRequest) {
			t.Errorf("want=%#v", detachRequest)
			t.Errorf("got=%#v", v)
		}
		fmt.Fprintf(w, `{"action":{"status":"in-progress"}}`)
	})

	_, _, err := client.StorageActions.Detach(volumeID)
	if err != nil {
		t.Errorf("StoragesActions.Detach returned error: %v", err)
	}
}
