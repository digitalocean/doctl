package doitserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/bryanl/doit-server"
)

func TestTokenGenerator(t *testing.T) {
	id := "id"

	key := "key"
	tg := doitserver.NewTokenGenerator(key)
	ts := httptest.NewServer(tg)
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	v := url.Values{}
	v.Set("id", id)
	u.RawQuery = v.Encode()
	ts.URL = u.String()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var ac doitserver.AuthCredentials
	err = json.NewDecoder(res.Body).Decode(&ac)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := ac.ID, id; got != want {
		t.Fatalf("TokenGenerator id = %q; want = %q", got, want)
	}

	if ac.CS == "" {
		t.Fatalf("TokenGenerator returned no cs")
	}
}
