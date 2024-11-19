// Code generated by MockGen. DO NOT EDIT.
// Source: monitoring.go
//
// Generated by this command:
//
//	mockgen -source monitoring.go -package=mocks MonitoringService
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	do "github.com/digitalocean/doctl/do"
	godo "github.com/digitalocean/godo"
	gomock "go.uber.org/mock/gomock"
)

// MockMonitoringService is a mock of MonitoringService interface.
type MockMonitoringService struct {
	ctrl     *gomock.Controller
	recorder *MockMonitoringServiceMockRecorder
	isgomock struct{}
}

// MockMonitoringServiceMockRecorder is the mock recorder for MockMonitoringService.
type MockMonitoringServiceMockRecorder struct {
	mock *MockMonitoringService
}

// NewMockMonitoringService creates a new mock instance.
func NewMockMonitoringService(ctrl *gomock.Controller) *MockMonitoringService {
	mock := &MockMonitoringService{ctrl: ctrl}
	mock.recorder = &MockMonitoringServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMonitoringService) EXPECT() *MockMonitoringServiceMockRecorder {
	return m.recorder
}

// CreateAlertPolicy mocks base method.
func (m *MockMonitoringService) CreateAlertPolicy(request *godo.AlertPolicyCreateRequest) (*do.AlertPolicy, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAlertPolicy", request)
	ret0, _ := ret[0].(*do.AlertPolicy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAlertPolicy indicates an expected call of CreateAlertPolicy.
func (mr *MockMonitoringServiceMockRecorder) CreateAlertPolicy(request any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAlertPolicy", reflect.TypeOf((*MockMonitoringService)(nil).CreateAlertPolicy), request)
}

// DeleteAlertPolicy mocks base method.
func (m *MockMonitoringService) DeleteAlertPolicy(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAlertPolicy", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAlertPolicy indicates an expected call of DeleteAlertPolicy.
func (mr *MockMonitoringServiceMockRecorder) DeleteAlertPolicy(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAlertPolicy", reflect.TypeOf((*MockMonitoringService)(nil).DeleteAlertPolicy), arg0)
}

// GetAlertPolicy mocks base method.
func (m *MockMonitoringService) GetAlertPolicy(arg0 string) (*do.AlertPolicy, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAlertPolicy", arg0)
	ret0, _ := ret[0].(*do.AlertPolicy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAlertPolicy indicates an expected call of GetAlertPolicy.
func (mr *MockMonitoringServiceMockRecorder) GetAlertPolicy(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAlertPolicy", reflect.TypeOf((*MockMonitoringService)(nil).GetAlertPolicy), arg0)
}

// ListAlertPolicies mocks base method.
func (m *MockMonitoringService) ListAlertPolicies() (do.AlertPolicies, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAlertPolicies")
	ret0, _ := ret[0].(do.AlertPolicies)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAlertPolicies indicates an expected call of ListAlertPolicies.
func (mr *MockMonitoringServiceMockRecorder) ListAlertPolicies() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAlertPolicies", reflect.TypeOf((*MockMonitoringService)(nil).ListAlertPolicies))
}

// UpdateAlertPolicy mocks base method.
func (m *MockMonitoringService) UpdateAlertPolicy(uuid string, request *godo.AlertPolicyUpdateRequest) (*do.AlertPolicy, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateAlertPolicy", uuid, request)
	ret0, _ := ret[0].(*do.AlertPolicy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateAlertPolicy indicates an expected call of UpdateAlertPolicy.
func (mr *MockMonitoringServiceMockRecorder) UpdateAlertPolicy(uuid, request any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAlertPolicy", reflect.TypeOf((*MockMonitoringService)(nil).UpdateAlertPolicy), uuid, request)
}
