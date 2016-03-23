package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockTagsService struct {
	mock.Mock
}

func (_m *MockTagsService) List(_a0 *godo.ListOptions) ([]godo.Tag, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 []godo.Tag
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.Tag); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Tag)
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
func (_m *MockTagsService) Get(_a0 string) (*godo.Tag, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Tag
	if rf, ok := ret.Get(0).(func(string) *godo.Tag); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Tag)
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
func (_m *MockTagsService) Create(_a0 *godo.TagCreateRequest) (*godo.Tag, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Tag
	if rf, ok := ret.Get(0).(func(*godo.TagCreateRequest) *godo.Tag); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Tag)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(*godo.TagCreateRequest) *godo.Response); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*godo.TagCreateRequest) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockTagsService) Update(_a0 string, _a1 *godo.TagUpdateRequest) (*godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Response
	if rf, ok := ret.Get(0).(func(string, *godo.TagUpdateRequest) *godo.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, *godo.TagUpdateRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *MockTagsService) Delete(_a0 string) (*godo.Response, error) {
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
func (_m *MockTagsService) TagResources(_a0 string, _a1 *godo.TagResourcesRequest) (*godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Response
	if rf, ok := ret.Get(0).(func(string, *godo.TagResourcesRequest) *godo.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, *godo.TagResourcesRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *MockTagsService) UntagResources(_a0 string, _a1 *godo.UntagResourcesRequest) (*godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Response
	if rf, ok := ret.Get(0).(func(string, *godo.UntagResourcesRequest) *godo.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, *godo.UntagResourcesRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
