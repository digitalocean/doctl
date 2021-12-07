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
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/digitalocean/godo"
)

const maxFetchPages = 5

var perPage = 200

var fetchFn = fetchPage

type paginatedList struct {
	list  [][]interface{}
	total int
	mu    sync.Mutex
}

func (pl *paginatedList) set(page int, items []interface{}) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.total += len(items)
	pl.list[page-1] = items
}

// Generator is a function that generates the list to be paginated.
type Generator func(*godo.ListOptions) ([]interface{}, *godo.Response, error)

// PaginateResp paginates a Response.
func PaginateResp(gen Generator) ([]interface{}, error) {
	opt := &godo.ListOptions{Page: 1, PerPage: perPage}

	// fetch first page to get page count (x)
	firstPage, resp, err := gen(opt)
	if err != nil {
		return nil, err
	}

	// find last page
	lp, err := lastPage(resp)
	if err != nil {
		return nil, err
	}

	l := paginatedList{
		list: make([][]interface{}, lp),
	}

	// set results from the first page
	l.set(1, firstPage)

	fetchChan := make(chan int, maxFetchPages)

	var wg sync.WaitGroup
	for i := 0; i < maxFetchPages-1; i++ {
		wg.Add(1)
		go func() {
			for page := range fetchChan {
				items, err := fetchFn(gen, page)
				if err == nil {
					l.set(page, items)
				}
			}
			wg.Done()
		}()
	}

	// start with second page
	opt.Page++
	for ; opt.Page <= lp; opt.Page++ {
		fetchChan <- opt.Page
	}
	close(fetchChan)

	wg.Wait()

	// flatten paginated list
	items := make([]interface{}, l.total)[:0]
	for _, page := range l.list {
		if page == nil {
			// must have been an error getting page results
			continue
		}
		for _, item := range page {
			items = append(items, item)
		}
	}

	return items, nil
}

func fetchPage(gen Generator, page int) ([]interface{}, error) {
	opt := &godo.ListOptions{Page: page, PerPage: perPage}
	items, _, err := gen(opt)
	return items, err
}

func lastPage(resp *godo.Response) (int, error) {
	if resp.Links == nil || resp.Links.Pages == nil {
		// no other pages
		return 1, nil
	}

	uStr := resp.Links.Pages.Last
	if uStr == "" {
		return 1, nil
	}

	u, err := url.Parse(uStr)
	if err != nil {
		return 0, fmt.Errorf("could not parse last page: %v", err)
	}

	pageStr := u.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return 0, fmt.Errorf("could not find page param: %v", err)
	}

	return page, err
}
