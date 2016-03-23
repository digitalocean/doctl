
/*
Copyright 2016 The Doctl Authors All rights reserved.
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

package mocks

import "github.com/bryanl/doit/do"
import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type FloatingIPActionsService struct {
	mock.Mock
}

// Assign provides a mock function with given fields: ip, dropletID
func (_m *FloatingIPActionsService) Assign(ip string, dropletID int) (*do.Action, error) {
	ret := _m.Called(ip, dropletID)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(string, int) *do.Action); ok {
		r0 = rf(ip, dropletID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int) error); ok {
		r1 = rf(ip, dropletID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Unassign provides a mock function with given fields: ip
func (_m *FloatingIPActionsService) Unassign(ip string) (*do.Action, error) {
	ret := _m.Called(ip)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(string) *do.Action); ok {
		r0 = rf(ip)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(ip)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Get provides a mock function with given fields: ip, actionID
func (_m *FloatingIPActionsService) Get(ip string, actionID int) (*do.Action, error) {
	ret := _m.Called(ip, actionID)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(string, int) *do.Action); ok {
		r0 = rf(ip, actionID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int) error); ok {
		r1 = rf(ip, actionID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: ip, opt
func (_m *FloatingIPActionsService) List(ip string, opt *godo.ListOptions) ([]do.Action, error) {
	ret := _m.Called(ip, opt)

	var r0 []do.Action
	if rf, ok := ret.Get(0).(func(string, *godo.ListOptions) []do.Action); ok {
		r0 = rf(ip, opt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, *godo.ListOptions) error); ok {
		r1 = rf(ip, opt)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
