package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockAccountService struct {
	mock.Mock
}

func (_m *MockAccountService) Get() (*godo.Account, *godo.Response, error) {
	ret := _m.Called()

	var r0 *godo.Account
	if rf, ok := ret.Get(0).(func() *godo.Account); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Account)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func() *godo.Response); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func() error); ok {
		r2 = rf()
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
