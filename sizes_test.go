package godo

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestSizes_SizesServiceOpImplementsSizesService(t *testing.T) {
	if !Implements((*SizesService)(nil), new(SizesServiceOp)) {
		t.Error("SizesServiceOp does not implement SizesService")
	}
}

func TestSizes_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/sizes", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"sizes":[{"slug":"1"},{"slug":"2"}]}`)
	})

	sizes, _, err := client.Sizes.List()
	if err != nil {
		t.Errorf("Sizes.List returned error: %v", err)
	}

	expected := []Size{{Slug: "1"}, {Slug: "2"}}
	if !reflect.DeepEqual(sizes, expected) {
		t.Errorf("Sizes.List returned %+v, expected %+v", sizes, expected)
	}
}

func TestSize_String(t *testing.T) {
	size := &Size{
		Slug:         "slize",
		Memory:       123,
		Vcpus:        456,
		Disk:         789,
		PriceMonthly: 123,
		PriceHourly:  456,
		Regions:      []string{"1", "2"},
	}

	stringified := size.String()
	expected := `godo.Size{Slug:"slize", Memory:123, Vcpus:456, Disk:789, PriceMonthly:123, PriceHourly:456, Regions:["1" "2"]}`
	if expected != stringified {
		t.Errorf("Size.String returned %+v, expected %+v", stringified, expected)
	}
}
