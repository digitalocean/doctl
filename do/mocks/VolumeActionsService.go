// Code generated by MockGen. DO NOT EDIT.
// Source: volume_actions.go

// Package mocks is a generated GoMock package.
package mocks

import (
	do "github.com/digitalocean/doctl/do"
	godo "github.com/digitalocean/godo"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockVolumeActionsService is a mock of VolumeActionsService interface
type MockVolumeActionsService struct {
	ctrl     *gomock.Controller
	recorder *MockVolumeActionsServiceMockRecorder
}

// MockVolumeActionsServiceMockRecorder is the mock recorder for MockVolumeActionsService
type MockVolumeActionsServiceMockRecorder struct {
	mock *MockVolumeActionsService
}

// NewMockVolumeActionsService creates a new mock instance
func NewMockVolumeActionsService(ctrl *gomock.Controller) *MockVolumeActionsService {
	mock := &MockVolumeActionsService{ctrl: ctrl}
	mock.recorder = &MockVolumeActionsServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockVolumeActionsService) EXPECT() *MockVolumeActionsServiceMockRecorder {
	return m.recorder
}

// Attach mocks base method
func (m *MockVolumeActionsService) Attach(arg0 string, arg1 int) (*do.Action, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Attach", arg0, arg1)
	ret0, _ := ret[0].(*do.Action)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Attach indicates an expected call of Attach
func (mr *MockVolumeActionsServiceMockRecorder) Attach(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Attach", reflect.TypeOf((*MockVolumeActionsService)(nil).Attach), arg0, arg1)
}

// Detach mocks base method
func (m *MockVolumeActionsService) Detach(arg0 string, arg1 int) (*do.Action, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Detach", arg0, arg1)
	ret0, _ := ret[0].(*do.Action)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Detach indicates an expected call of Detach
func (mr *MockVolumeActionsServiceMockRecorder) Detach(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Detach", reflect.TypeOf((*MockVolumeActionsService)(nil).Detach), arg0, arg1)
}

// Get mocks base method
func (m *MockVolumeActionsService) Get(arg0 string, arg1 int) (*do.Action, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*do.Action)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockVolumeActionsServiceMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockVolumeActionsService)(nil).Get), arg0, arg1)
}

// List mocks base method
func (m *MockVolumeActionsService) List(arg0 string, arg1 *godo.ListOptions) ([]do.Action, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0, arg1)
	ret0, _ := ret[0].([]do.Action)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List
func (mr *MockVolumeActionsServiceMockRecorder) List(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockVolumeActionsService)(nil).List), arg0, arg1)
}

// Resize mocks base method
func (m *MockVolumeActionsService) Resize(arg0 string, arg1 int, arg2 string) (*do.Action, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Resize", arg0, arg1, arg2)
	ret0, _ := ret[0].(*do.Action)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Resize indicates an expected call of Resize
func (mr *MockVolumeActionsServiceMockRecorder) Resize(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Resize", reflect.TypeOf((*MockVolumeActionsService)(nil).Resize), arg0, arg1, arg2)
}