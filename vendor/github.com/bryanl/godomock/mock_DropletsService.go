package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockDropletsService struct {
	mock.Mock
}

func (_m *MockDropletsService) List(_a0 *godo.ListOptions) ([]godo.Droplet, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 []godo.Droplet
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.Droplet); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Droplet)
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
func (_m *MockDropletsService) ListByTag(_a0 string, _a1 *godo.ListOptions) ([]godo.Droplet, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []godo.Droplet
	if rf, ok := ret.Get(0).(func(string, *godo.ListOptions) []godo.Droplet); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Droplet)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string, *godo.ListOptions) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, *godo.ListOptions) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletsService) Get(_a0 int) (*godo.Droplet, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Droplet
	if rf, ok := ret.Get(0).(func(int) *godo.Droplet); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Droplet)
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
func (_m *MockDropletsService) Create(_a0 *godo.DropletCreateRequest) (*godo.Droplet, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Droplet
	if rf, ok := ret.Get(0).(func(*godo.DropletCreateRequest) *godo.Droplet); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Droplet)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(*godo.DropletCreateRequest) *godo.Response); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*godo.DropletCreateRequest) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletsService) CreateMultiple(_a0 *godo.DropletMultiCreateRequest) ([]godo.Droplet, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 []godo.Droplet
	if rf, ok := ret.Get(0).(func(*godo.DropletMultiCreateRequest) []godo.Droplet); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Droplet)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(*godo.DropletMultiCreateRequest) *godo.Response); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*godo.DropletMultiCreateRequest) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletsService) Delete(_a0 int) (*godo.Response, error) {
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
func (_m *MockDropletsService) DeleteByTag(_a0 string) (*godo.Response, error) {
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
func (_m *MockDropletsService) Kernels(_a0 int, _a1 *godo.ListOptions) ([]godo.Kernel, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []godo.Kernel
	if rf, ok := ret.Get(0).(func(int, *godo.ListOptions) []godo.Kernel); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Kernel)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, *godo.ListOptions) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, *godo.ListOptions) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletsService) Snapshots(_a0 int, _a1 *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []godo.Image
	if rf, ok := ret.Get(0).(func(int, *godo.ListOptions) []godo.Image); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Image)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, *godo.ListOptions) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, *godo.ListOptions) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletsService) Backups(_a0 int, _a1 *godo.ListOptions) ([]godo.Image, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []godo.Image
	if rf, ok := ret.Get(0).(func(int, *godo.ListOptions) []godo.Image); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Image)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, *godo.ListOptions) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, *godo.ListOptions) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletsService) Actions(_a0 int, _a1 *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []godo.Action
	if rf, ok := ret.Get(0).(func(int, *godo.ListOptions) []godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, *godo.ListOptions) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, *godo.ListOptions) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletsService) Neighbors(_a0 int) ([]godo.Droplet, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 []godo.Droplet
	if rf, ok := ret.Get(0).(func(int) []godo.Droplet); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Droplet)
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
