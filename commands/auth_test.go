package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/bryanl/doit-server"
)

func TestAuth_retrieveCredentials(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ac := doitserver.AuthCredentials{ID: "abc", CS: "def"}
		err := json.NewEncoder(w).Encode(&ac)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	ac, err := retrieveAuthCredentials(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	expectedAC := &doitserver.AuthCredentials{ID: "abc", CS: "def"}
	if got, want := ac, expectedAC; !reflect.DeepEqual(got, expectedAC) {
		t.Fatalf("retrieveAuthCredentials() = %#v; got %#v", got, want)
	}
}

func TestAuth_createAuthURL(t *testing.T) {
	serverURL := "http://example.com"
	ac := &doitserver.AuthCredentials{
		ID: "id",
		CS: "cs",
	}
	u, err := createAuthURL(serverURL, ac)
	if err != nil {
		t.Fatal(err)
	}

	newU, err := url.Parse(u)

	if got, want := newU.Host, "example.com"; got != want {
		t.Fatalf("createAuthURL() Host = %q, want = %q", got, want)
	}

	if got, want := newU.Path, "/auth/digitalocean"; got != want {
		t.Fatalf("createAuthURL() Path = %q, want = %q", got, want)
	}

	q := newU.Query()

	if got, want := q.Get("cs"), "cs"; got != want {
		t.Fatalf("createAuthURL() cs param = %q, want = %q", got, want)
	}

	if got, want := q.Get("id"), "id"; got != want {
		t.Fatalf("createAuthURL() id param = %q, want = %q", got, want)
	}
}
