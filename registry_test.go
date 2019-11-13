package godo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry_Create(t *testing.T) {
	setup()
	defer teardown()

	want := &Registry{
		Name: "foo",
	}

	createRequest := &RegistryCreateRequest{
		Name: want.Name,
	}

	createResponseJSON := `
{
	"registry": {
		"name": "foo"
	}
}`

	mux.HandleFunc("/v2/registry", func(w http.ResponseWriter, r *http.Request) {
		v := new(RegistryCreateRequest)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			t.Fatal(err)
		}

		testMethod(t, r, http.MethodPost)
		require.Equal(t, v, createRequest)
		fmt.Fprint(w, createResponseJSON)
	})

	got, _, err := client.Registry.Create(ctx, createRequest)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestRegistry_Get(t *testing.T) {
	setup()
	defer teardown()

	want := &Registry{
		Name: "foo",
	}

	getResponseJSON := `
{
	"registry": {
		"name": "foo"
	}
}`

	mux.HandleFunc("/v2/registry", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, getResponseJSON)
	})
	got, _, err := client.Registry.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestRegistry_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/registry", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodDelete)
	})

	_, err := client.Registry.Delete(ctx)
	require.NoError(t, err)
}

func TestRegistry_DockerCredentials(t *testing.T) {
	setup()
	defer teardown()

	want := []byte("this is a valid docker config json")

	mux.HandleFunc("/v2/registry/docker-credentials", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write(want)
	})
	got, _, err := client.Registry.DockerCredentials(ctx)

	require.NoError(t, err)
	require.Equal(t, want, got.DockerConfigJSON)
}
