package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockActionsService struct {
	mock.Mock
}

func (_m *MockActionsService) List(_a0 *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 []godo.Action
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(*godo.ListOptions) *godo.Response); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*godo.ListOptions) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockActionsService) Get(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int) *godo.Response); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
