/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package do

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPagedRegistriesService returns a registriesService backed by a server
// that serves 201 items under key for any path, split into two pages.
func newPagedRegistriesService(t *testing.T, key string, item func(i int) map[string]any) RegistriesService {
	t.Helper()

	var base string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := []map[string]any{}
		if r.URL.Query().Get("page") == "2" {
			items = append(items, item(201))
		} else {
			for i := 1; i <= 200; i++ {
				items = append(items, item(i))
			}
		}
		last := base + r.URL.Path + "?page=2&per_page=200"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			key:     items,
			"links": map[string]any{"pages": map[string]string{"last": last, "next": last}},
			"meta":  map[string]int{"total": 201},
		})
	}))
	t.Cleanup(server.Close)
	base = server.URL

	client := godo.NewClient(server.Client())
	u, err := url.Parse(base + "/")
	require.NoError(t, err)
	client.BaseURL = u

	return NewRegistriesService(client)
}

func TestRegistriesService_ListFollowsPagination(t *testing.T) {
	tests := []struct {
		name string
		key  string
		item func(i int) map[string]any
		list func(RegistriesService) (int, error)
	}{
		{
			name: "ListRepositories",
			key:  "repositories",
			item: func(i int) map[string]any { return map[string]any{"name": fmt.Sprintf("repo-%d", i)} },
			list: func(s RegistriesService) (int, error) { l, err := s.ListRepositories("r"); return len(l), err },
		},
		{
			name: "ListRepositoriesV2",
			key:  "repositories",
			item: func(i int) map[string]any { return map[string]any{"name": fmt.Sprintf("repo-%d", i)} },
			list: func(s RegistriesService) (int, error) { l, err := s.ListRepositoriesV2("r"); return len(l), err },
		},
		{
			name: "ListRepositoryTags",
			key:  "tags",
			item: func(i int) map[string]any { return map[string]any{"tag": fmt.Sprintf("tag-%d", i)} },
			list: func(s RegistriesService) (int, error) {
				l, err := s.ListRepositoryTags("r", "repo")
				return len(l), err
			},
		},
		{
			name: "ListRepositoryManifests",
			key:  "manifests",
			item: func(i int) map[string]any { return map[string]any{"digest": fmt.Sprintf("sha256:%d", i)} },
			list: func(s RegistriesService) (int, error) {
				l, err := s.ListRepositoryManifests("r", "repo")
				return len(l), err
			},
		},
		{
			name: "ListGarbageCollections",
			key:  "garbage_collections",
			item: func(i int) map[string]any { return map[string]any{"uuid": strings.Repeat("a", 3) + fmt.Sprint(i)} },
			list: func(s RegistriesService) (int, error) { l, err := s.ListGarbageCollections("r"); return len(l), err },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newPagedRegistriesService(t, tc.key, tc.item)
			n, err := tc.list(svc)
			assert.NoError(t, err)
			assert.Equal(t, 201, n)
		})
	}
}
