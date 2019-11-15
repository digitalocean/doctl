// Code generated by MockGen. DO NOT EDIT.
// Source: registry.go

// Package mocks is a generated GoMock package.
package mocks

import (
	do "github.com/digitalocean/doctl/do"
	godo "github.com/digitalocean/godo"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockRegistryService is a mock of RegistryService interface
type MockRegistryService struct {
	ctrl     *gomock.Controller
	recorder *MockRegistryServiceMockRecorder
}

// MockRegistryServiceMockRecorder is the mock recorder for MockRegistryService
type MockRegistryServiceMockRecorder struct {
	mock *MockRegistryService
}

// NewMockRegistryService creates a new mock instance
func NewMockRegistryService(ctrl *gomock.Controller) *MockRegistryService {
	mock := &MockRegistryService{ctrl: ctrl}
	mock.recorder = &MockRegistryServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockRegistryService) EXPECT() *MockRegistryServiceMockRecorder {
	return m.recorder
}

// Get mocks base method
func (m *MockRegistryService) Get() (*do.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get")
	ret0, _ := ret[0].(*do.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockRegistryServiceMockRecorder) Get() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockRegistryService)(nil).Get))
}

// Create mocks base method
func (m *MockRegistryService) Create(arg0 *godo.RegistryCreateRequest) (*do.Registry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0)
	ret0, _ := ret[0].(*do.Registry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create
func (mr *MockRegistryServiceMockRecorder) Create(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockRegistryService)(nil).Create), arg0)
}

// Delete mocks base method
func (m *MockRegistryService) Delete() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete")
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockRegistryServiceMockRecorder) Delete() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockRegistryService)(nil).Delete))
}

// DockerCredentials mocks base method
func (m *MockRegistryService) DockerCredentials(arg0 *godo.RegistryDockerCredentialsRequest) (*godo.DockerCredentials, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DockerCredentials", arg0)
	ret0, _ := ret[0].(*godo.DockerCredentials)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DockerCredentials indicates an expected call of DockerCredentials
func (mr *MockRegistryServiceMockRecorder) DockerCredentials(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DockerCredentials", reflect.TypeOf((*MockRegistryService)(nil).DockerCredentials), arg0)
}

// Endpoint mocks base method
func (m *MockRegistryService) Endpoint() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Endpoint")
	ret0, _ := ret[0].(string)
	return ret0
}

// Endpoint indicates an expected call of Endpoint
func (mr *MockRegistryServiceMockRecorder) Endpoint() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Endpoint", reflect.TypeOf((*MockRegistryService)(nil).Endpoint))
}
