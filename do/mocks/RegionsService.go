package mocks

import "github.com/bryanl/doit/do"
import "github.com/stretchr/testify/mock"

type RegionsService struct {
	mock.Mock
}

// List provides a mock function with given fields:
func (_m *RegionsService) List() (do.Regions, error) {
	ret := _m.Called()

	var r0 do.Regions
	if rf, ok := ret.Get(0).(func() do.Regions); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(do.Regions)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
