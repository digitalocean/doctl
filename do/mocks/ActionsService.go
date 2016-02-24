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
