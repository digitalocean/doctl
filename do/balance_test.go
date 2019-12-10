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
	"testing"

	"context"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type GoDoBalanceService struct {
	mock.Mock
}

// Get provides a mock function with given fields: _a0
func (_m *GoDoBalanceService) Get(_a0 context.Context) (*godo.Balance, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Balance
	if rf, ok := ret.Get(0).(func(context.Context) *godo.Balance); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Balance)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(context.Context) *godo.Response); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

func TestBalanceServiceGet(t *testing.T) {

	gBalanceSvc := &GoDoBalanceService{}

	gBalance := &godo.Balance{AccountBalance: "12.34"}
	gBalanceSvc.On("Get", context.TODO()).Return(gBalance, nil, nil)

	client := &godo.Client{
		Balance: gBalanceSvc,
	}
	as := NewBalanceService(client)

	balance, err := as.Get()
	assert.NoError(t, err)
	assert.Equal(t, "12.34", balance.AccountBalance)
}
