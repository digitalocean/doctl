package mocks

import "github.com/stretchr/testify/mock"

type Runner struct {
	mock.Mock
}

// Run provides a mock function with given fields:
func (_m *Runner) Run() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
