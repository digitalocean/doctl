package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockFloatingIPsService struct {
	mock.Mock
}

func (_m *MockFloatingIPsService) List(_a0 *godo.ListOptions) ([]godo.FloatingIP, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 []godo.FloatingIP
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.FloatingIP); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.FloatingIP)
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
func (_m *MockFloatingIPsService) Get(_a0 string) (*godo.FloatingIP, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.FloatingIP
	if rf, ok := ret.Get(0).(func(string) *godo.FloatingIP); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.FloatingIP)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string) *godo.Response); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockFloatingIPsService) Create(_a0 *godo.FloatingIPCreateRequest) (*godo.FloatingIP, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.FloatingIP
	if rf, ok := ret.Get(0).(func(*godo.FloatingIPCreateRequest) *godo.FloatingIP); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.FloatingIP)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(*godo.FloatingIPCreateRequest) *godo.Response); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*godo.FloatingIPCreateRequest) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockFloatingIPsService) Delete(_a0 string) (*godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Response
	if rf, ok := ret.Get(0).(func(string) *godo.Response); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Response)
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
