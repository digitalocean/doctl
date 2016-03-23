package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockImageActionsService struct {
	mock.Mock
}

func (_m *MockImageActionsService) Get(_a0 int, _a1 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int, int) *godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, int) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, int) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockImageActionsService) Transfer(_a0 int, _a1 *godo.ActionRequest) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int, *godo.ActionRequest) *godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, *godo.ActionRequest) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, *godo.ActionRequest) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
