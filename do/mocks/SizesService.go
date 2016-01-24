package mocks

import "github.com/bryanl/doit/do"
import "github.com/stretchr/testify/mock"

type SizesService struct {
	mock.Mock
}

func (_m *SizesService) List() (do.Sizes, error) {
	ret := _m.Called()

	var r0 do.Sizes
	if rf, ok := ret.Get(0).(func() do.Sizes); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(do.Sizes)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
