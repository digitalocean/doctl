package mocks

import "github.com/bryanl/doit/do"
import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type KeysService struct {
	mock.Mock
}

func (_m *KeysService) List() (do.SSHKeys, error) {
	ret := _m.Called()

	var r0 do.SSHKeys
	if rf, ok := ret.Get(0).(func() do.SSHKeys); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(do.SSHKeys)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *KeysService) Get(id string) (*do.SSHKey, error) {
	ret := _m.Called(id)

	var r0 *do.SSHKey
	if rf, ok := ret.Get(0).(func(string) *do.SSHKey); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.SSHKey)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *KeysService) Create(kcr *godo.KeyCreateRequest) (*do.SSHKey, error) {
	ret := _m.Called(kcr)

	var r0 *do.SSHKey
	if rf, ok := ret.Get(0).(func(*godo.KeyCreateRequest) *do.SSHKey); ok {
		r0 = rf(kcr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.SSHKey)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*godo.KeyCreateRequest) error); ok {
		r1 = rf(kcr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *KeysService) Update(id string, kur *godo.KeyUpdateRequest) (*do.SSHKey, error) {
	ret := _m.Called(id, kur)

	var r0 *do.SSHKey
	if rf, ok := ret.Get(0).(func(string, *godo.KeyUpdateRequest) *do.SSHKey); ok {
		r0 = rf(id, kur)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*do.SSHKey)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, *godo.KeyUpdateRequest) error); ok {
		r1 = rf(id, kur)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *KeysService) Delete(id string) error {
	ret := _m.Called(id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
