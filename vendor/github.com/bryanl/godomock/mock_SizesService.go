package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockSizesService struct {
	mock.Mock
}

func (_m *MockSizesService) List(_a0 *godo.ListOptions) ([]godo.Size, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 []godo.Size
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.Size); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Size)
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
