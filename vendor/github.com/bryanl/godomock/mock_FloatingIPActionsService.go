package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockFloatingIPActionsService struct {
	mock.Mock
}

func (_m *MockFloatingIPActionsService) Assign(ip string, dropletID int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(ip, dropletID)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string, int) *godo.Action); ok {
		r0 = rf(ip, dropletID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string, int) *godo.Response); ok {
		r1 = rf(ip, dropletID)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, int) error); ok {
		r2 = rf(ip, dropletID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockFloatingIPActionsService) Unassign(ip string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(ip)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string) *godo.Action); ok {
		r0 = rf(ip)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string) *godo.Response); ok {
		r1 = rf(ip)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string) error); ok {
		r2 = rf(ip)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockFloatingIPActionsService) Get(ip string, actionID int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(ip, actionID)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string, int) *godo.Action); ok {
		r0 = rf(ip, actionID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string, int) *godo.Response); ok {
		r1 = rf(ip, actionID)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, int) error); ok {
		r2 = rf(ip, actionID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockFloatingIPActionsService) List(ip string, opt *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
	ret := _m.Called(ip, opt)

	var r0 []godo.Action
	if rf, ok := ret.Get(0).(func(string, *godo.ListOptions) []godo.Action); ok {
		r0 = rf(ip, opt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string, *godo.ListOptions) *godo.Response); ok {
		r1 = rf(ip, opt)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, *godo.ListOptions) error); ok {
		r2 = rf(ip, opt)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
