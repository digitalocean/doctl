package mocks

import "github.com/digitalocean/doctl/do"
import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

// Generated: please do not edit by hand

type FloatingIPsService struct {
	mock.Mock
}

func (_m *FloatingIPsService) List() (do.FloatingIPs, error) {
	ret := _m.Called()

	var r0 do.FloatingIPs
	if rf, ok := ret.Get(0).(func() do.FloatingIPs); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(do.FloatingIPs)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FloatingIPsService) Get(ip string) (*do.FloatingIP, error) {
	ret := _m.Called(ip)

	var r0 *do.FloatingIP
	if rf, ok := ret.Get(0).(func(string) *do.FloatingIP); ok {
		r0 = rf(ip)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.FloatingIP)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(ip)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FloatingIPsService) Create(ficr *godo.FloatingIPCreateRequest) (*do.FloatingIP, error) {
	ret := _m.Called(ficr)

	var r0 *do.FloatingIP
	if rf, ok := ret.Get(0).(func(*godo.FloatingIPCreateRequest) *do.FloatingIP); ok {
		r0 = rf(ficr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.FloatingIP)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*godo.FloatingIPCreateRequest) error); ok {
		r1 = rf(ficr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *FloatingIPsService) Delete(ip string) error {
	ret := _m.Called(ip)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(ip)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
