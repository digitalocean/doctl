package main

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/digitalocean/godo"
)

func TestFindImageByName_PageOne(t *testing.T) {
	setup()
	defer teardown()
	client = godo.NewClient(nil)
	client.BaseURL = BaseURL

	mux.HandleFunc("/v2/images", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"images":[{"id":1, "name":"name"}]}`)
	})

	image, err := FindImageByName(client, "name")
	if err != nil {
		t.Errorf("Images.List returned error: %v", err)
	}
	expected := &godo.Image{ID: 1, Name: "name"}
	if !reflect.DeepEqual(image, expected) {
		t.Errorf("Images.List returned %#v, expected %#v", image, expected)
	}
}

func TestFindImageByName_PageTwo(t *testing.T) {
	setup()
	defer teardown()
	client = godo.NewClient(nil)
	client.BaseURL = BaseURL

	mux.HandleFunc("/v2/images", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `{"images":[{"id":2, "name":"name"}]}`)
		} else {
			fmt.Fprint(w, `{"images": [{"id":1, "name": "foo"}], "links":{"pages":{"next":"http://example.com/v2/images/?page=2", "last":"http://example.com/v2/images/?page=2"}}}`)
		}
	})

	image, err := FindImageByName(client, "name")
	if err != nil {
		t.Errorf("Images.List returned error: %v", err)
	}
	expected := &godo.Image{ID: 2, Name: "name"}
	if !reflect.DeepEqual(image, expected) {
		t.Errorf("Images.List returned %#v, expected %#v", image, expected)
	}
}

func TestFindImageByName_NotFound(t *testing.T) {
	setup()
	defer teardown()
	client = godo.NewClient(nil)
	client.BaseURL = BaseURL

	mux.HandleFunc("/v2/images", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"images": [{"id":1, "name": "foo"}]}`)
	})

	expected := "error: name not found."
	image, err := FindImageByName(client, "name")
	if err == nil {
		t.Errorf("Images.List returned %#v, expected %#v", image, expected)
	}
}
