
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

type ActionsService struct {
	mock.Mock
}

// List provides a mock function with given fields:
func (_m *ActionsService) List() (do.Actions, error) {
	ret := _m.Called()

	var r0 do.Actions
	if rf, ok := ret.Get(0).(func() do.Actions); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(do.Actions)
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
func (_m *ActionsService) Get(_a0 int) (*do.Action, error) {
	ret := _m.Called(_a0)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(int) *do.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
