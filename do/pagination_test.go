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
	"sync"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func Test_PaginateResp(t *testing.T) {
	var mu sync.Mutex
	currentPage := 0
	resp := &godo.Response{Links: &godo.Links{Pages: &godo.Pages{Last: "http://example.com/?page=5"}}}

	gen := func(*godo.ListOptions) ([]interface{}, *godo.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		currentPage++
		return []interface{}{currentPage}, resp, nil
	}

	list, err := PaginateResp(gen)
	assert.NoError(t, err)

	assert.Len(t, list, 5)
}

func Test_Pagination_fetchPage(t *testing.T) {
	gen := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		items := []interface{}{}
		resp := &godo.Response{}

		assert.Equal(t, 10, opt.Page)

		return items, resp, nil
	}

	fetchPage(gen, 10)
}

func Test_Pagination_lastPage(t *testing.T) {
	cases := []struct {
		r        *godo.Response
		lastPage int
		isValid  bool
	}{
		{
			r: &godo.Response{
				Links: &godo.Links{
					Pages: &godo.Pages{Last: "http://example.com/?page=1"},
				},
			},
			lastPage: 1,
			isValid:  true,
		},
		{
			r:        &godo.Response{Links: &godo.Links{}},
			lastPage: 1,
			isValid:  true,
		},

		{
			r:        &godo.Response{Links: nil},
			lastPage: 1,
			isValid:  true,
		},
	}

	for _, c := range cases {
		lp, err := lastPage(c.r)
		if c.isValid {
			assert.NoError(t, err)
			assert.Equal(t, c.lastPage, lp)
		} else {
			assert.Error(t, err)
		}
	}
}
