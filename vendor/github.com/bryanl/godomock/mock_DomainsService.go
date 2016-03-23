package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockDomainsService struct {
	mock.Mock
}

func (_m *MockDomainsService) List(_a0 *godo.ListOptions) ([]godo.Domain, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 []godo.Domain
	if rf, ok := ret.Get(0).(func(*godo.ListOptions) []godo.Domain); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.Domain)
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
func (_m *MockDomainsService) Get(_a0 string) (*godo.Domain, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Domain
	if rf, ok := ret.Get(0).(func(string) *godo.Domain); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Domain)
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
func (_m *MockDomainsService) Create(_a0 *godo.DomainCreateRequest) (*godo.Domain, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Domain
	if rf, ok := ret.Get(0).(func(*godo.DomainCreateRequest) *godo.Domain); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Domain)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(*godo.DomainCreateRequest) *godo.Response); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*godo.DomainCreateRequest) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDomainsService) Delete(_a0 string) (*godo.Response, error) {
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
func (_m *MockDomainsService) Records(_a0 string, _a1 *godo.ListOptions) ([]godo.DomainRecord, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []godo.DomainRecord
	if rf, ok := ret.Get(0).(func(string, *godo.ListOptions) []godo.DomainRecord); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]godo.DomainRecord)
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
func (_m *MockDomainsService) Record(_a0 string, _a1 int) (*godo.DomainRecord, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.DomainRecord
	if rf, ok := ret.Get(0).(func(string, int) *godo.DomainRecord); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.DomainRecord)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string, int) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, int) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDomainsService) DeleteRecord(_a0 string, _a1 int) (*godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Response
	if rf, ok := ret.Get(0).(func(string, int) *godo.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *MockDomainsService) EditRecord(_a0 string, _a1 int, _a2 *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *godo.DomainRecord
	if rf, ok := ret.Get(0).(func(string, int, *godo.DomainRecordEditRequest) *godo.DomainRecord); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.DomainRecord)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string, int, *godo.DomainRecordEditRequest) *godo.Response); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, int, *godo.DomainRecordEditRequest) error); ok {
		r2 = rf(_a0, _a1, _a2)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDomainsService) CreateRecord(_a0 string, _a1 *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.DomainRecord
	if rf, ok := ret.Get(0).(func(string, *godo.DomainRecordEditRequest) *godo.DomainRecord); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.DomainRecord)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string, *godo.DomainRecordEditRequest) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, *godo.DomainRecordEditRequest) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
