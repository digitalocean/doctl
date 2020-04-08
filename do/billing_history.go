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
	"context"

	"github.com/digitalocean/godo"
)

// BillingHistory is a wrapper for godo.BillingHistory
type BillingHistory struct {
	*godo.BillingHistory
}

// BillingHistoryService is an interface for interacting with DigitalOcean's invoices api.
type BillingHistoryService interface {
	List() (*BillingHistory, error)
}

type billingHistoryService struct {
	client *godo.Client
}

var _ BillingHistoryService = &billingHistoryService{}

// NewBillingHistoryService builds an BillingHistoryService instance.
func NewBillingHistoryService(client *godo.Client) BillingHistoryService {
	return &billingHistoryService{
		client: client,
	}
}

func (is *billingHistoryService) List() (*BillingHistory, error) {
	listFn := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		historyList, resp, err := is.client.BillingHistory.List(context.Background(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(historyList.BillingHistory))
		for i := range historyList.BillingHistory {
			si[i] = historyList.BillingHistory[i]
		}
		return si, resp, err
	}

	paginatedList, err := PaginateResp(listFn)
	if err != nil {
		return nil, err
	}
	list := make([]godo.BillingHistoryEntry, len(paginatedList))
	for i := range paginatedList {
		list[i] = paginatedList[i].(godo.BillingHistoryEntry)
	}

	return &BillingHistory{BillingHistory: &godo.BillingHistory{BillingHistory: list}}, nil
}
