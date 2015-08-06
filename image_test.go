package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestMissingArgs(t *testing.T) {
	setup()
	defer teardown()

	tests := []struct {
		args    []string
		wantErr error
	}{
		{
			args:    []string{"doctl", "-k", "TOKEN", "image", "delete"},
			wantErr: errors.New("Error: Must provide ID or name for an image."),
		},
		{
			args:    []string{"doctl", "-k", "TOKEN", "image", "rename"},
			wantErr: errors.New("Error: Must provide ID or name for an image and its new name."),
		},
		{
			args:    []string{"doctl", "-k", "TOKEN", "image", "rename", "--id=123456"},
			wantErr: errors.New("Error: Must provide a new name for the image."),
		},
		{
			args:    []string{"doctl", "-k", "TOKEN", "image", "show"},
			wantErr: errors.New("Error: Must provide ID or name for an image."),
		},
		{
			args:    []string{"doctl", "-k", "TOKEN", "image", "list", "--apps", "--distros"},
			wantErr: errors.New("You can only use one of '--applications', '--distributions', or '--private'."),
		},
		{
			args:    []string{"doctl", "-k", "TOKEN", "image", "list", "--apps", "--private"},
			wantErr: errors.New("You can only use one of '--applications', '--distributions', or '--private'."),
		},
		{
			args:    []string{"doctl", "-k", "TOKEN", "image", "list", "--distros", "--private"},
			wantErr: errors.New("You can only use one of '--applications', '--distributions', or '--private'."),
		},
	}

	for _, tt := range tests {
		err := app.Run(tt.args)
		if err.Error() != tt.wantErr.Error() {
			t.Errorf("app.Run(%v) = %#v, expected %#v", tt.args, err, tt.wantErr)
		}
	}
}

func TestImageDelete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/images", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"images":[{"id":1, "name":"name"}]}`)
	})

	mux.HandleFunc("/v2/images/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Request method = %v, expected %v", r.Method, "DELETE")
		}
	})

	args := []string{"doctl", "-k", "TOKEN", "image", "delete", "name"}
	err := app.Run(args)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	expected := "doctl> Image successfully deleted.\n"
	if output != expected {
		t.Errorf("app.Run(%v) = %#v, expected %#v", args, output, expected)
	}
}

func TestImageList(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/images", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"images":[{"id":1, "name":"name", "distribution": "Ubuntu", "regions":["nyc1"]}]}`)
	})

	args := []string{"doctl", "-k", "TOKEN", "image", "list"}
	err := app.Run(args)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	expected := "OS\tName\tID\tSlug\tRegions\nUbuntu\tname\t1\t\t[nyc1]\n"
	if output != expected {
		t.Errorf("app.Run(%v) = %#v, expected %#v", args, output, expected)
	}
}

func TestImageRename(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/images", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"images":[{"id":1, "name":"name", "distribution": "Ubuntu", "regions":["nyc1"],"min_disk_size":20}]}`)
	})

	mux.HandleFunc("/v2/images/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Request method = %v, expected %v", r.Method, "PUT")
		}
		expected := map[string]interface{}{
			"name": "new_name",
		}

		var v map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&v)
		if err != nil {
			t.Fatalf("decode json: %v", err)
		}

		if !reflect.DeepEqual(v, expected) {
			t.Errorf("Request body = %#v, expected %#v", v, expected)
		}

		fmt.Fprintf(w, `{"image":{"id":1, "name":"new_name", "type":"snapshot", "distribution": "Ubuntu", "regions":["nyc1"],"min_disk_size":20}}`)
	})

	args := []string{"doctl", "-k", "TOKEN", "image", "rename", "name", "new_name"}
	err := app.Run(args)
	if err != nil {
		t.Fatalf("Error %s", err)
	}

	output := buf.String()

	expected := "id: 1\nname: new_name\ntype: snapshot\ndistribution: Ubuntu\nslug: \"\"\npublic: false\nregions:\n- nyc1\nmindisksize: 20\n\n"
	if output != expected {
		t.Errorf("app.Run(%v) = %#v, expected %#v", args, output, expected)
	}
}

func TestImageShow(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/images", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"images":[{"id":1, "name":"name", "distribution": "Ubuntu", "regions":["nyc1"],"min_disk_size":20}]}`)
	})

	mux.HandleFunc("/v2/images/1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"image":{"id":1, "name":"name", "type":"snapshot", "distribution": "Ubuntu", "regions":["nyc1"],"min_disk_size":20}}`)
	})

	args := []string{"doctl", "-k", "TOKEN", "image", "show", "name"}
	err := app.Run(args)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	expected := "id: 1\nname: name\ntype: snapshot\ndistribution: Ubuntu\nslug: \"\"\npublic: false\nregions:\n- nyc1\nmindisksize: 20\n\n"
	if output != expected {
		t.Errorf("app.Run(%v) = %#v, expected %#v", args, output, expected)
	}
}