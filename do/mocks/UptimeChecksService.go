// Code generated by MockGen. DO NOT EDIT.
// Source: uptime_checks.go
//
// Generated by this command:
//
//	mockgen -source uptime_checks.go -package=mocks UptimeChecksService
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	do "github.com/digitalocean/doctl/do"
	godo "github.com/digitalocean/godo"
	gomock "go.uber.org/mock/gomock"
)

// MockUptimeChecksService is a mock of UptimeChecksService interface.
type MockUptimeChecksService struct {
	ctrl     *gomock.Controller
	recorder *MockUptimeChecksServiceMockRecorder
	isgomock struct{}
}

// MockUptimeChecksServiceMockRecorder is the mock recorder for MockUptimeChecksService.
type MockUptimeChecksServiceMockRecorder struct {
	mock *MockUptimeChecksService
}

// NewMockUptimeChecksService creates a new mock instance.
func NewMockUptimeChecksService(ctrl *gomock.Controller) *MockUptimeChecksService {
	mock := &MockUptimeChecksService{ctrl: ctrl}
	mock.recorder = &MockUptimeChecksServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUptimeChecksService) EXPECT() *MockUptimeChecksServiceMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockUptimeChecksService) Create(arg0 *godo.CreateUptimeCheckRequest) (*do.UptimeCheck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0)
	ret0, _ := ret[0].(*do.UptimeCheck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockUptimeChecksServiceMockRecorder) Create(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockUptimeChecksService)(nil).Create), arg0)
}

// CreateAlert mocks base method.
func (m *MockUptimeChecksService) CreateAlert(arg0 string, arg1 *godo.CreateUptimeAlertRequest) (*do.UptimeAlert, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAlert", arg0, arg1)
	ret0, _ := ret[0].(*do.UptimeAlert)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAlert indicates an expected call of CreateAlert.
func (mr *MockUptimeChecksServiceMockRecorder) CreateAlert(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAlert", reflect.TypeOf((*MockUptimeChecksService)(nil).CreateAlert), arg0, arg1)
}

// Delete mocks base method.
func (m *MockUptimeChecksService) Delete(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockUptimeChecksServiceMockRecorder) Delete(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockUptimeChecksService)(nil).Delete), arg0)
}

// DeleteAlert mocks base method.
func (m *MockUptimeChecksService) DeleteAlert(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAlert", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAlert indicates an expected call of DeleteAlert.
func (mr *MockUptimeChecksServiceMockRecorder) DeleteAlert(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAlert", reflect.TypeOf((*MockUptimeChecksService)(nil).DeleteAlert), arg0, arg1)
}

// Get mocks base method.
func (m *MockUptimeChecksService) Get(arg0 string) (*do.UptimeCheck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(*do.UptimeCheck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockUptimeChecksServiceMockRecorder) Get(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockUptimeChecksService)(nil).Get), arg0)
}

// GetAlert mocks base method.
func (m *MockUptimeChecksService) GetAlert(arg0, arg1 string) (*do.UptimeAlert, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAlert", arg0, arg1)
	ret0, _ := ret[0].(*do.UptimeAlert)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAlert indicates an expected call of GetAlert.
func (mr *MockUptimeChecksServiceMockRecorder) GetAlert(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAlert", reflect.TypeOf((*MockUptimeChecksService)(nil).GetAlert), arg0, arg1)
}

// GetState mocks base method.
func (m *MockUptimeChecksService) GetState(arg0 string) (*do.UptimeCheckState, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetState", arg0)
	ret0, _ := ret[0].(*do.UptimeCheckState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetState indicates an expected call of GetState.
func (mr *MockUptimeChecksServiceMockRecorder) GetState(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetState", reflect.TypeOf((*MockUptimeChecksService)(nil).GetState), arg0)
}

// List mocks base method.
func (m *MockUptimeChecksService) List() ([]do.UptimeCheck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].([]do.UptimeCheck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockUptimeChecksServiceMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockUptimeChecksService)(nil).List))
}

// ListAlerts mocks base method.
func (m *MockUptimeChecksService) ListAlerts(arg0 string) ([]do.UptimeAlert, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAlerts", arg0)
	ret0, _ := ret[0].([]do.UptimeAlert)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAlerts indicates an expected call of ListAlerts.
func (mr *MockUptimeChecksServiceMockRecorder) ListAlerts(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAlerts", reflect.TypeOf((*MockUptimeChecksService)(nil).ListAlerts), arg0)
}

// Update mocks base method.
func (m *MockUptimeChecksService) Update(arg0 string, arg1 *godo.UpdateUptimeCheckRequest) (*do.UptimeCheck, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1)
	ret0, _ := ret[0].(*do.UptimeCheck)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockUptimeChecksServiceMockRecorder) Update(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockUptimeChecksService)(nil).Update), arg0, arg1)
}

// UpdateAlert mocks base method.
func (m *MockUptimeChecksService) UpdateAlert(arg0, arg1 string, arg2 *godo.UpdateUptimeAlertRequest) (*do.UptimeAlert, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateAlert", arg0, arg1, arg2)
	ret0, _ := ret[0].(*do.UptimeAlert)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateAlert indicates an expected call of UpdateAlert.
func (mr *MockUptimeChecksServiceMockRecorder) UpdateAlert(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAlert", reflect.TypeOf((*MockUptimeChecksService)(nil).UpdateAlert), arg0, arg1, arg2)
}
