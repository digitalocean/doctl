package godo

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestImages_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/images", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"images":[{"id":1},{"id":2}]}`)
	})

	images, _, err := client.Images.List()
	if err != nil {
		t.Errorf("Images.List returned error: %v", err)
	}

	expected := []Image{{ID: 1}, {ID: 2}}
	if !reflect.DeepEqual(images, expected) {
		t.Errorf("Images.List returned %+v, expected %+v", images, expected)
	}
}

func TestImage_String(t *testing.T) {
	image := &Image{
		ID:           1,
		Name:         "Image",
		Distribution: "Ubuntu",
		Slug:         "image",
		Public:       true,
		Regions:      []string{"one", "two"},
	}

	stringified := image.String()
	expected := `godo.Image{ID:1, Name:"Image", Distribution:"Ubuntu", Slug:"image", Public:true, Regions:["one" "two"]}`
	if expected != stringified {
		t.Errorf("Image.String returned %+v, expected %+v", stringified, expected)
	}
}
