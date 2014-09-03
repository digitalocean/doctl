package godo

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestRegions_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/regions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"regions":[{"slug":"1"},{"slug":"2"}]}`)
	})

	regions, _, err := client.Regions.List()
	if err != nil {
		t.Errorf("Regions.List returned error: %v", err)
	}

	expected := []Region{{Slug: "1"}, {Slug: "2"}}
	if !reflect.DeepEqual(regions, expected) {
		t.Errorf("Regions.List returned %+v, expected %+v", regions, expected)
	}
}

func TestRegion_String(t *testing.T) {
	region := &Region{
		Slug:      "region",
		Name:      "Region",
		Sizes:     []string{"1", "2"},
		Available: true,
	}

	stringified := region.String()
	expected := `godo.Region{Slug:"region", Name:"Region", Sizes:["1" "2"], Available:true}`
	if expected != stringified {
		t.Errorf("Region.String returned %+v, expected %+v", stringified, expected)
	}
}
