package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockImagesService struct {
	mock.Mock
}

func (_m *MockImagesService) List(_a0 *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 []godo.Image
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.Image); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Image)
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
func (_m *MockImagesService) ListDistribution(opt *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	ret := _m.Called(opt)

	var r0 []godo.Image
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.Image); ok {
		r0 = rf(opt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Image)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(*godo.ListOptions) *godo.Response); ok {
		r1 = rf(opt)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*godo.ListOptions) error); ok {
		r2 = rf(opt)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockImagesService) ListApplication(opt *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	ret := _m.Called(opt)

	var r0 []godo.Image
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.Image); ok {
		r0 = rf(opt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Image)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(*godo.ListOptions) *godo.Response); ok {
		r1 = rf(opt)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*godo.ListOptions) error); ok {
		r2 = rf(opt)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockImagesService) ListUser(opt *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	ret := _m.Called(opt)

	var r0 []godo.Image
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.Image); ok {
		r0 = rf(opt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Image)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(*godo.ListOptions) *godo.Response); ok {
		r1 = rf(opt)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*godo.ListOptions) error); ok {
		r2 = rf(opt)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockImagesService) GetByID(_a0 int) (*godo.Image, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Image
	if rf, ok := ret.Get(0).(func(int) *godo.Image); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Image)
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
func (_m *MockImagesService) GetBySlug(_a0 string) (*godo.Image, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Image
	if rf, ok := ret.Get(0).(func(string) *godo.Image); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Image)
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
func (_m *MockImagesService) Update(_a0 int, _a1 *godo.ImageUpdateRequest) (*godo.Image, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Image
	if rf, ok := ret.Get(0).(func(int, *godo.ImageUpdateRequest) *godo.Image); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Image)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, *godo.ImageUpdateRequest) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, *godo.ImageUpdateRequest) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockImagesService) Delete(_a0 int) (*godo.Response, error) {
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
