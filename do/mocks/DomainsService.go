
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

type DomainsService struct {
	mock.Mock
}

// List provides a mock function with given fields:
func (_m *DomainsService) List() (do.Domains, error) {
	ret := _m.Called()

	var r0 do.Domains
	if rf, ok := ret.Get(0).(func() do.Domains); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(do.Domains)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Get provides a mock function with given fields: _a0
func (_m *DomainsService) Get(_a0 string) (*do.Domain, error) {
	ret := _m.Called(_a0)

	var r0 *do.Domain
	if rf, ok := ret.Get(0).(func(string) *do.Domain); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Domain)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Create provides a mock function with given fields: _a0
func (_m *DomainsService) Create(_a0 *godo.DomainCreateRequest) (*do.Domain, error) {
	ret := _m.Called(_a0)

	var r0 *do.Domain
	if rf, ok := ret.Get(0).(func(*godo.DomainCreateRequest) *do.Domain); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Domain)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*godo.DomainCreateRequest) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: _a0
func (_m *DomainsService) Delete(_a0 string) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Records provides a mock function with given fields: _a0
func (_m *DomainsService) Records(_a0 string) (do.DomainRecords, error) {
	ret := _m.Called(_a0)

	var r0 do.DomainRecords
	if rf, ok := ret.Get(0).(func(string) do.DomainRecords); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(do.DomainRecords)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Record provides a mock function with given fields: _a0, _a1
func (_m *DomainsService) Record(_a0 string, _a1 int) (*do.DomainRecord, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *do.DomainRecord
	if rf, ok := ret.Get(0).(func(string, int) *do.DomainRecord); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.DomainRecord)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteRecord provides a mock function with given fields: _a0, _a1
func (_m *DomainsService) DeleteRecord(_a0 string, _a1 int) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// EditRecord provides a mock function with given fields: _a0, _a1, _a2
func (_m *DomainsService) EditRecord(_a0 string, _a1 int, _a2 *godo.DomainRecordEditRequest) (*do.DomainRecord, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *do.DomainRecord
	if rf, ok := ret.Get(0).(func(string, int, *godo.DomainRecordEditRequest) *do.DomainRecord); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.DomainRecord)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int, *godo.DomainRecordEditRequest) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateRecord provides a mock function with given fields: _a0, _a1
func (_m *DomainsService) CreateRecord(_a0 string, _a1 *godo.DomainRecordEditRequest) (*do.DomainRecord, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *do.DomainRecord
	if rf, ok := ret.Get(0).(func(string, *godo.DomainRecordEditRequest) *do.DomainRecord); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.DomainRecord)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, *godo.DomainRecordEditRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
