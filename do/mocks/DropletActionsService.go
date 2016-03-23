
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

type DropletActionsService struct {
	mock.Mock
}

// Shutdown provides a mock function with given fields: _a0
func (_m *DropletActionsService) Shutdown(_a0 int) (*do.Action, error) {
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

// PowerOff provides a mock function with given fields: _a0
func (_m *DropletActionsService) PowerOff(_a0 int) (*do.Action, error) {
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

// PowerOn provides a mock function with given fields: _a0
func (_m *DropletActionsService) PowerOn(_a0 int) (*do.Action, error) {
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

// PowerCycle provides a mock function with given fields: _a0
func (_m *DropletActionsService) PowerCycle(_a0 int) (*do.Action, error) {
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

// Reboot provides a mock function with given fields: _a0
func (_m *DropletActionsService) Reboot(_a0 int) (*do.Action, error) {
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

// Restore provides a mock function with given fields: _a0, _a1
func (_m *DropletActionsService) Restore(_a0 int, _a1 int) (*do.Action, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(int, int) *do.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Resize provides a mock function with given fields: _a0, _a1, _a2
func (_m *DropletActionsService) Resize(_a0 int, _a1 string, _a2 bool) (*do.Action, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(int, string, bool) *do.Action); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, string, bool) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Rename provides a mock function with given fields: _a0, _a1
func (_m *DropletActionsService) Rename(_a0 int, _a1 string) (*do.Action, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(int, string) *do.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Snapshot provides a mock function with given fields: _a0, _a1
func (_m *DropletActionsService) Snapshot(_a0 int, _a1 string) (*do.Action, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(int, string) *do.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EnableBackups provides a mock function with given fields: _a0
func (_m *DropletActionsService) EnableBackups(_a0 int) (*do.Action, error) {
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

// DisableBackups provides a mock function with given fields: _a0
func (_m *DropletActionsService) DisableBackups(_a0 int) (*do.Action, error) {
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

// PasswordReset provides a mock function with given fields: _a0
func (_m *DropletActionsService) PasswordReset(_a0 int) (*do.Action, error) {
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

// RebuildByImageID provides a mock function with given fields: _a0, _a1
func (_m *DropletActionsService) RebuildByImageID(_a0 int, _a1 int) (*do.Action, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(int, int) *do.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RebuildByImageSlug provides a mock function with given fields: _a0, _a1
func (_m *DropletActionsService) RebuildByImageSlug(_a0 int, _a1 string) (*do.Action, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(int, string) *do.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChangeKernel provides a mock function with given fields: _a0, _a1
func (_m *DropletActionsService) ChangeKernel(_a0 int, _a1 int) (*do.Action, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(int, int) *do.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EnableIPv6 provides a mock function with given fields: _a0
func (_m *DropletActionsService) EnableIPv6(_a0 int) (*do.Action, error) {
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

// EnablePrivateNetworking provides a mock function with given fields: _a0
func (_m *DropletActionsService) EnablePrivateNetworking(_a0 int) (*do.Action, error) {
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

// Upgrade provides a mock function with given fields: _a0
func (_m *DropletActionsService) Upgrade(_a0 int) (*do.Action, error) {
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

// Get provides a mock function with given fields: _a0, _a1
func (_m *DropletActionsService) Get(_a0 int, _a1 int) (*do.Action, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(int, int) *do.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByURI provides a mock function with given fields: _a0
func (_m *DropletActionsService) GetByURI(_a0 string) (*do.Action, error) {
	ret := _m.Called(_a0)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(string) *do.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
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
