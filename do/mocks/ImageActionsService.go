// Code generated by MockGen. DO NOT EDIT.
// Source: image_actions.go
//
// Generated by this command:
//
//	mockgen -source image_actions.go -package=mocks ImageActionsService
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	do "github.com/digitalocean/doctl/do"
	godo "github.com/digitalocean/godo"
	gomock "go.uber.org/mock/gomock"
)

// MockImageActionsService is a mock of ImageActionsService interface.
type MockImageActionsService struct {
	ctrl     *gomock.Controller
	recorder *MockImageActionsServiceMockRecorder
	isgomock struct{}
}

// MockImageActionsServiceMockRecorder is the mock recorder for MockImageActionsService.
type MockImageActionsServiceMockRecorder struct {
	mock *MockImageActionsService
}

// NewMockImageActionsService creates a new mock instance.
func NewMockImageActionsService(ctrl *gomock.Controller) *MockImageActionsService {
	mock := &MockImageActionsService{ctrl: ctrl}
	mock.recorder = &MockImageActionsServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockImageActionsService) EXPECT() *MockImageActionsServiceMockRecorder {
	return m.recorder
}

// Convert mocks base method.
func (m *MockImageActionsService) Convert(arg0 int) (*do.Action, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Convert", arg0)
	ret0, _ := ret[0].(*do.Action)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Convert indicates an expected call of Convert.
func (mr *MockImageActionsServiceMockRecorder) Convert(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Convert", reflect.TypeOf((*MockImageActionsService)(nil).Convert), arg0)
}

// Get mocks base method.
func (m *MockImageActionsService) Get(arg0, arg1 int) (*do.Action, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*do.Action)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockImageActionsServiceMockRecorder) Get(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockImageActionsService)(nil).Get), arg0, arg1)
}

// Transfer mocks base method.
func (m *MockImageActionsService) Transfer(arg0 int, arg1 *godo.ActionRequest) (*do.Action, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transfer", arg0, arg1)
	ret0, _ := ret[0].(*do.Action)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Transfer indicates an expected call of Transfer.
func (mr *MockImageActionsServiceMockRecorder) Transfer(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transfer", reflect.TypeOf((*MockImageActionsService)(nil).Transfer), arg0, arg1)
}
