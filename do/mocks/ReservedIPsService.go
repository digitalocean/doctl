// Code generated by MockGen. DO NOT EDIT.
// Source: reserved_ips.go
//
// Generated by this command:
//
//	mockgen -source reserved_ips.go -package=mocks ReservedIPsService
//
// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	do "github.com/digitalocean/doctl/do"
	godo "github.com/digitalocean/godo"
	gomock "go.uber.org/mock/gomock"
)

// MockReservedIPsService is a mock of ReservedIPsService interface.
type MockReservedIPsService struct {
	ctrl     *gomock.Controller
	recorder *MockReservedIPsServiceMockRecorder
}

// MockReservedIPsServiceMockRecorder is the mock recorder for MockReservedIPsService.
type MockReservedIPsServiceMockRecorder struct {
	mock *MockReservedIPsService
}

// NewMockReservedIPsService creates a new mock instance.
func NewMockReservedIPsService(ctrl *gomock.Controller) *MockReservedIPsService {
	mock := &MockReservedIPsService{ctrl: ctrl}
	mock.recorder = &MockReservedIPsServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockReservedIPsService) EXPECT() *MockReservedIPsServiceMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockReservedIPsService) Create(ficr *godo.ReservedIPCreateRequest) (*do.ReservedIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ficr)
	ret0, _ := ret[0].(*do.ReservedIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockReservedIPsServiceMockRecorder) Create(ficr any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockReservedIPsService)(nil).Create), ficr)
}

// Delete mocks base method.
func (m *MockReservedIPsService) Delete(ip string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ip)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockReservedIPsServiceMockRecorder) Delete(ip any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockReservedIPsService)(nil).Delete), ip)
}

// Get mocks base method.
func (m *MockReservedIPsService) Get(ip string) (*do.ReservedIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ip)
	ret0, _ := ret[0].(*do.ReservedIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockReservedIPsServiceMockRecorder) Get(ip any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockReservedIPsService)(nil).Get), ip)
}

// List mocks base method.
func (m *MockReservedIPsService) List() (do.ReservedIPs, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].(do.ReservedIPs)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockReservedIPsServiceMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockReservedIPsService)(nil).List))
}
