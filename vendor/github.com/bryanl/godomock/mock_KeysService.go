package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockKeysService struct {
	mock.Mock
}

func (_m *MockKeysService) List(_a0 *godo.ListOptions) ([]godo.Key, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 []godo.Key
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.Key); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Key)
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
func (_m *MockKeysService) GetByID(_a0 int) (*godo.Key, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Key
	if rf, ok := ret.Get(0).(func(int) *godo.Key); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Key)
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
func (_m *MockKeysService) GetByFingerprint(_a0 string) (*godo.Key, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Key
	if rf, ok := ret.Get(0).(func(string) *godo.Key); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Key)
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
func (_m *MockKeysService) Create(_a0 *godo.KeyCreateRequest) (*godo.Key, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Key
	if rf, ok := ret.Get(0).(func(*godo.KeyCreateRequest) *godo.Key); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Key)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(*godo.KeyCreateRequest) *godo.Response); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*godo.KeyCreateRequest) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockKeysService) UpdateByID(_a0 int, _a1 *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Key
	if rf, ok := ret.Get(0).(func(int, *godo.KeyUpdateRequest) *godo.Key); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Key)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, *godo.KeyUpdateRequest) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, *godo.KeyUpdateRequest) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockKeysService) UpdateByFingerprint(_a0 string, _a1 *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Key
	if rf, ok := ret.Get(0).(func(string, *godo.KeyUpdateRequest) *godo.Key); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Key)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string, *godo.KeyUpdateRequest) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, *godo.KeyUpdateRequest) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockKeysService) DeleteByID(_a0 int) (*godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Response
	if rf, ok := ret.Get(0).(func(int) *godo.Response); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *MockKeysService) DeleteByFingerprint(_a0 string) (*godo.Response, error) {
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
