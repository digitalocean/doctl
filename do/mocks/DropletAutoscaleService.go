// Code generated by MockGen. DO NOT EDIT.
// Source: droplet_autoscale.go
//
// Generated by this command:
//
//	mockgen -source droplet_autoscale.go -package=mocks DropletAutoscaleService
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	godo "github.com/digitalocean/godo"
	gomock "go.uber.org/mock/gomock"
)

// MockDropletAutoscaleService is a mock of DropletAutoscaleService interface.
type MockDropletAutoscaleService struct {
	ctrl     *gomock.Controller
	recorder *MockDropletAutoscaleServiceMockRecorder
	isgomock struct{}
}

// MockDropletAutoscaleServiceMockRecorder is the mock recorder for MockDropletAutoscaleService.
type MockDropletAutoscaleServiceMockRecorder struct {
	mock *MockDropletAutoscaleService
}

// NewMockDropletAutoscaleService creates a new mock instance.
func NewMockDropletAutoscaleService(ctrl *gomock.Controller) *MockDropletAutoscaleService {
	mock := &MockDropletAutoscaleService{ctrl: ctrl}
	mock.recorder = &MockDropletAutoscaleServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDropletAutoscaleService) EXPECT() *MockDropletAutoscaleServiceMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockDropletAutoscaleService) Create(arg0 *godo.DropletAutoscalePoolRequest) (*godo.DropletAutoscalePool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0)
	ret0, _ := ret[0].(*godo.DropletAutoscalePool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockDropletAutoscaleServiceMockRecorder) Create(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockDropletAutoscaleService)(nil).Create), arg0)
}

// Delete mocks base method.
func (m *MockDropletAutoscaleService) Delete(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockDropletAutoscaleServiceMockRecorder) Delete(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockDropletAutoscaleService)(nil).Delete), arg0)
}

// DeleteDangerous mocks base method.
func (m *MockDropletAutoscaleService) DeleteDangerous(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDangerous", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteDangerous indicates an expected call of DeleteDangerous.
func (mr *MockDropletAutoscaleServiceMockRecorder) DeleteDangerous(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDangerous", reflect.TypeOf((*MockDropletAutoscaleService)(nil).DeleteDangerous), arg0)
}

// Get mocks base method.
func (m *MockDropletAutoscaleService) Get(arg0 string) (*godo.DropletAutoscalePool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(*godo.DropletAutoscalePool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockDropletAutoscaleServiceMockRecorder) Get(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockDropletAutoscaleService)(nil).Get), arg0)
}

// List mocks base method.
func (m *MockDropletAutoscaleService) List() ([]*godo.DropletAutoscalePool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].([]*godo.DropletAutoscalePool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockDropletAutoscaleServiceMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockDropletAutoscaleService)(nil).List))
}

// ListHistory mocks base method.
func (m *MockDropletAutoscaleService) ListHistory(arg0 string) ([]*godo.DropletAutoscaleHistoryEvent, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListHistory", arg0)
	ret0, _ := ret[0].([]*godo.DropletAutoscaleHistoryEvent)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListHistory indicates an expected call of ListHistory.
func (mr *MockDropletAutoscaleServiceMockRecorder) ListHistory(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListHistory", reflect.TypeOf((*MockDropletAutoscaleService)(nil).ListHistory), arg0)
}

// ListMembers mocks base method.
func (m *MockDropletAutoscaleService) ListMembers(arg0 string) ([]*godo.DropletAutoscaleResource, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListMembers", arg0)
	ret0, _ := ret[0].([]*godo.DropletAutoscaleResource)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListMembers indicates an expected call of ListMembers.
func (mr *MockDropletAutoscaleServiceMockRecorder) ListMembers(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListMembers", reflect.TypeOf((*MockDropletAutoscaleService)(nil).ListMembers), arg0)
}

// Update mocks base method.
func (m *MockDropletAutoscaleService) Update(arg0 string, arg1 *godo.DropletAutoscalePoolRequest) (*godo.DropletAutoscalePool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1)
	ret0, _ := ret[0].(*godo.DropletAutoscalePool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockDropletAutoscaleServiceMockRecorder) Update(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockDropletAutoscaleService)(nil).Update), arg0, arg1)
}
