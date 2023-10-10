// Code generated by MockGen. DO NOT EDIT.
// Source: tags.go
//
// Generated by this command:
//
//	mockgen -source tags.go -package=mocks TagsService
//
// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	do "github.com/digitalocean/doctl/do"
	godo "github.com/digitalocean/godo"
	gomock "go.uber.org/mock/gomock"
)

// MockTagsService is a mock of TagsService interface.
type MockTagsService struct {
	ctrl     *gomock.Controller
	recorder *MockTagsServiceMockRecorder
}

// MockTagsServiceMockRecorder is the mock recorder for MockTagsService.
type MockTagsServiceMockRecorder struct {
	mock *MockTagsService
}

// NewMockTagsService creates a new mock instance.
func NewMockTagsService(ctrl *gomock.Controller) *MockTagsService {
	mock := &MockTagsService{ctrl: ctrl}
	mock.recorder = &MockTagsServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTagsService) EXPECT() *MockTagsServiceMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockTagsService) Create(arg0 *godo.TagCreateRequest) (*do.Tag, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0)
	ret0, _ := ret[0].(*do.Tag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockTagsServiceMockRecorder) Create(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockTagsService)(nil).Create), arg0)
}

// Delete mocks base method.
func (m *MockTagsService) Delete(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockTagsServiceMockRecorder) Delete(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockTagsService)(nil).Delete), arg0)
}

// Get mocks base method.
func (m *MockTagsService) Get(arg0 string) (*do.Tag, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(*do.Tag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockTagsServiceMockRecorder) Get(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockTagsService)(nil).Get), arg0)
}

// List mocks base method.
func (m *MockTagsService) List() (do.Tags, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].(do.Tags)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockTagsServiceMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockTagsService)(nil).List))
}

// TagResources mocks base method.
func (m *MockTagsService) TagResources(arg0 string, arg1 *godo.TagResourcesRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TagResources", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// TagResources indicates an expected call of TagResources.
func (mr *MockTagsServiceMockRecorder) TagResources(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TagResources", reflect.TypeOf((*MockTagsService)(nil).TagResources), arg0, arg1)
}

// UntagResources mocks base method.
func (m *MockTagsService) UntagResources(arg0 string, arg1 *godo.UntagResourcesRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UntagResources", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UntagResources indicates an expected call of UntagResources.
func (mr *MockTagsServiceMockRecorder) UntagResources(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UntagResources", reflect.TypeOf((*MockTagsService)(nil).UntagResources), arg0, arg1)
}
