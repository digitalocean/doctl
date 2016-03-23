
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

type FloatingIPsService struct {
	mock.Mock
}

// List provides a mock function with given fields:
func (_m *FloatingIPsService) List() (do.FloatingIPs, error) {
	ret := _m.Called()

	var r0 do.FloatingIPs
	if rf, ok := ret.Get(0).(func() do.FloatingIPs); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(do.FloatingIPs)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Get provides a mock function with given fields: ip
func (_m *FloatingIPsService) Get(ip string) (*do.FloatingIP, error) {
	ret := _m.Called(ip)

	var r0 *do.FloatingIP
	if rf, ok := ret.Get(0).(func(string) *do.FloatingIP); ok {
		r0 = rf(ip)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.FloatingIP)
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

// Create provides a mock function with given fields: ficr
func (_m *FloatingIPsService) Create(ficr *godo.FloatingIPCreateRequest) (*do.FloatingIP, error) {
	ret := _m.Called(ficr)

	var r0 *do.FloatingIP
	if rf, ok := ret.Get(0).(func(*godo.FloatingIPCreateRequest) *do.FloatingIP); ok {
		r0 = rf(ficr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.FloatingIP)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*godo.FloatingIPCreateRequest) error); ok {
		r1 = rf(ficr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: ip
func (_m *FloatingIPsService) Delete(ip string) error {
	ret := _m.Called(ip)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(ip)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
