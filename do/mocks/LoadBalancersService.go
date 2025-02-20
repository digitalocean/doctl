// Code generated by MockGen. DO NOT EDIT.
// Source: load_balancers.go
//
// Generated by this command:
//
//	mockgen -source load_balancers.go -package=mocks LoadBalancersService
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	do "github.com/digitalocean/doctl/do"
	godo "github.com/digitalocean/godo"
	gomock "go.uber.org/mock/gomock"
)

// MockLoadBalancersService is a mock of LoadBalancersService interface.
type MockLoadBalancersService struct {
	ctrl     *gomock.Controller
	recorder *MockLoadBalancersServiceMockRecorder
	isgomock struct{}
}

// MockLoadBalancersServiceMockRecorder is the mock recorder for MockLoadBalancersService.
type MockLoadBalancersServiceMockRecorder struct {
	mock *MockLoadBalancersService
}

// NewMockLoadBalancersService creates a new mock instance.
func NewMockLoadBalancersService(ctrl *gomock.Controller) *MockLoadBalancersService {
	mock := &MockLoadBalancersService{ctrl: ctrl}
	mock.recorder = &MockLoadBalancersServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLoadBalancersService) EXPECT() *MockLoadBalancersServiceMockRecorder {
	return m.recorder
}

// AddDroplets mocks base method.
func (m *MockLoadBalancersService) AddDroplets(lbID string, dIDs ...int) error {
	m.ctrl.T.Helper()
	varargs := []any{lbID}
	for _, a := range dIDs {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "AddDroplets", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddDroplets indicates an expected call of AddDroplets.
func (mr *MockLoadBalancersServiceMockRecorder) AddDroplets(lbID any, dIDs ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{lbID}, dIDs...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddDroplets", reflect.TypeOf((*MockLoadBalancersService)(nil).AddDroplets), varargs...)
}

// AddForwardingRules mocks base method.
func (m *MockLoadBalancersService) AddForwardingRules(lbID string, rules ...godo.ForwardingRule) error {
	m.ctrl.T.Helper()
	varargs := []any{lbID}
	for _, a := range rules {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "AddForwardingRules", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddForwardingRules indicates an expected call of AddForwardingRules.
func (mr *MockLoadBalancersServiceMockRecorder) AddForwardingRules(lbID any, rules ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{lbID}, rules...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddForwardingRules", reflect.TypeOf((*MockLoadBalancersService)(nil).AddForwardingRules), varargs...)
}

// Create mocks base method.
func (m *MockLoadBalancersService) Create(lbr *godo.LoadBalancerRequest) (*do.LoadBalancer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", lbr)
	ret0, _ := ret[0].(*do.LoadBalancer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockLoadBalancersServiceMockRecorder) Create(lbr any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockLoadBalancersService)(nil).Create), lbr)
}

// Delete mocks base method.
func (m *MockLoadBalancersService) Delete(lbID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", lbID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockLoadBalancersServiceMockRecorder) Delete(lbID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockLoadBalancersService)(nil).Delete), lbID)
}

// Get mocks base method.
func (m *MockLoadBalancersService) Get(lbID string) (*do.LoadBalancer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", lbID)
	ret0, _ := ret[0].(*do.LoadBalancer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockLoadBalancersServiceMockRecorder) Get(lbID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockLoadBalancersService)(nil).Get), lbID)
}

// List mocks base method.
func (m *MockLoadBalancersService) List() (do.LoadBalancers, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].(do.LoadBalancers)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockLoadBalancersServiceMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockLoadBalancersService)(nil).List))
}

// PurgeCache mocks base method.
func (m *MockLoadBalancersService) PurgeCache(lbID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PurgeCache", lbID)
	ret0, _ := ret[0].(error)
	return ret0
}

// PurgeCache indicates an expected call of PurgeCache.
func (mr *MockLoadBalancersServiceMockRecorder) PurgeCache(lbID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PurgeCache", reflect.TypeOf((*MockLoadBalancersService)(nil).PurgeCache), lbID)
}

// RemoveDroplets mocks base method.
func (m *MockLoadBalancersService) RemoveDroplets(lbID string, dIDs ...int) error {
	m.ctrl.T.Helper()
	varargs := []any{lbID}
	for _, a := range dIDs {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "RemoveDroplets", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveDroplets indicates an expected call of RemoveDroplets.
func (mr *MockLoadBalancersServiceMockRecorder) RemoveDroplets(lbID any, dIDs ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{lbID}, dIDs...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveDroplets", reflect.TypeOf((*MockLoadBalancersService)(nil).RemoveDroplets), varargs...)
}

// RemoveForwardingRules mocks base method.
func (m *MockLoadBalancersService) RemoveForwardingRules(lbID string, rules ...godo.ForwardingRule) error {
	m.ctrl.T.Helper()
	varargs := []any{lbID}
	for _, a := range rules {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "RemoveForwardingRules", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveForwardingRules indicates an expected call of RemoveForwardingRules.
func (mr *MockLoadBalancersServiceMockRecorder) RemoveForwardingRules(lbID any, rules ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{lbID}, rules...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveForwardingRules", reflect.TypeOf((*MockLoadBalancersService)(nil).RemoveForwardingRules), varargs...)
}

// Update mocks base method.
func (m *MockLoadBalancersService) Update(lbID string, lbr *godo.LoadBalancerRequest) (*do.LoadBalancer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", lbID, lbr)
	ret0, _ := ret[0].(*do.LoadBalancer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockLoadBalancersServiceMockRecorder) Update(lbID, lbr any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockLoadBalancersService)(nil).Update), lbID, lbr)
}
