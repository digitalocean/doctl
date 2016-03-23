package godomock

import "github.com/stretchr/testify/mock"

import "github.com/digitalocean/godo"

type MockDropletActionsService struct {
	mock.Mock
}

func (_m *MockDropletActionsService) Shutdown(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) ShutdownByTag(_a0 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) PowerOff(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) PowerOffByTag(_a0 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) PowerOn(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) PowerOnByTag(_a0 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) PowerCycle(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) PowerCycleByTag(_a0 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) Reboot(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) Restore(_a0 int, _a1 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int, int) *godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, int) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, int) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletActionsService) Resize(_a0 int, _a1 string, _a2 bool) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int, string, bool) *godo.Action); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, string, bool) *godo.Response); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, string, bool) error); ok {
		r2 = rf(_a0, _a1, _a2)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletActionsService) Rename(_a0 int, _a1 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int, string) *godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, string) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, string) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletActionsService) Snapshot(_a0 int, _a1 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int, string) *godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, string) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, string) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletActionsService) SnapshotByTag(_a0 string, _a1 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string, string) *godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(string, string) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string, string) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletActionsService) EnableBackups(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) EnableBackupsByTag(_a0 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) DisableBackups(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) DisableBackupsByTag(_a0 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) PasswordReset(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) RebuildByImageID(_a0 int, _a1 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int, int) *godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, int) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, int) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletActionsService) RebuildByImageSlug(_a0 int, _a1 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int, string) *godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, string) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, string) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletActionsService) ChangeKernel(_a0 int, _a1 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int, int) *godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, int) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, int) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletActionsService) EnableIPv6(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) EnableIPv6ByTag(_a0 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) EnablePrivateNetworking(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) EnablePrivateNetworkingByTag(_a0 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) Upgrade(_a0 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
func (_m *MockDropletActionsService) Get(_a0 int, _a1 int) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(int, int) *godo.Action); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
		}
	}

	var r1 *godo.Response
	if rf, ok := ret.Get(1).(func(int, int) *godo.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*godo.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, int) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *MockDropletActionsService) GetByURI(_a0 string) (*godo.Action, *godo.Response, error) {
	ret := _m.Called(_a0)

	var r0 *godo.Action
	if rf, ok := ret.Get(0).(func(string) *godo.Action); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*godo.Action)
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
