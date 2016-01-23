package mocks

import "github.com/bryanl/doit/do"
import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type ImageActionsService struct {
	mock.Mock
}

func (_m *ImageActionsService) Get(_a0 int, _a1 int) (*do.Action, error) {
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
func (_m *ImageActionsService) Transfer(_a0 int, _a1 *godo.ActionRequest) (*do.Action, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *do.Action
	if rf, ok := ret.Get(0).(func(int, *godo.ActionRequest) *do.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.Action)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, *godo.ActionRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
