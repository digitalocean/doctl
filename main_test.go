package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/codegangsta/cli"
)

var (
	mux *http.ServeMux

	server *httptest.Server

	app *cli.App

	buf bytes.Buffer

	env string
)

func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	BaseURL, _ = url.Parse(server.URL)

	// Capture output
	Writer = &buf
	log.SetOutput(&buf)

	app = buildApp()
	app.Writer = ioutil.Discard

	// Ensure that we can test not having a key
	env = os.Getenv("DIGITALOCEAN_API_KEY")
	os.Setenv("DIGITALOCEAN_API_KEY", "")
	APIKey = ""
}

func teardown() {
	server.Close()
	buf.Reset()
	log.SetOutput(os.Stdout)
	os.Setenv("DIGITALOCEAN_API_KEY", env)
}

func TestAppWithoutApiKey(t *testing.T) {
	setup()
	defer teardown()

	// test with other global flags
	tests := []struct {
		args    []string
		wantErr error
	}{
		{
			args:    []string{"doctl"},
			wantErr: errors.New("Must provide API Key via DIGITALOCEAN_API_KEY environment variable or via CLI argument."),
		},
		{
			args:    []string{"doctl", "--version"},
			wantErr: nil,
		},
		{
			args:    []string{"doctl", "--help"},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		err := app.Run(tt.args)
		if tt.wantErr == nil {
			if err != nil {
				t.Error("Expected nil error")
			}
		} else if err.Error() != tt.wantErr.Error() {
			t.Errorf("app.Run(%v) = %#v, want %#v", tt.args, err, tt.wantErr)
		}
	}
}

func TestGlobalFormatFlag(t *testing.T) {
	setup()
	defer teardown()

	args := []string{"doctl", "-k", "key", "-f", "invalid"}
	err := app.Run(args)

	expected := `invalid output format: "invalid", available output options: json, yaml.`
	if err.Error() != expected {
		t.Errorf("app.Run(%v) = %#v, want %#v", err, err.Error(), expected)
	}
}
