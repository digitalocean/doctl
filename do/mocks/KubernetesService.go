// Code generated by MockGen. DO NOT EDIT.
// Source: kubernetes.go
//
// Generated by this command:
//
//	mockgen -source kubernetes.go -package=mocks KubernetesService
//
// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	do "github.com/digitalocean/doctl/do"
	godo "github.com/digitalocean/godo"
	gomock "go.uber.org/mock/gomock"
)

// MockKubernetesService is a mock of KubernetesService interface.
type MockKubernetesService struct {
	ctrl     *gomock.Controller
	recorder *MockKubernetesServiceMockRecorder
}

// MockKubernetesServiceMockRecorder is the mock recorder for MockKubernetesService.
type MockKubernetesServiceMockRecorder struct {
	mock *MockKubernetesService
}

// NewMockKubernetesService creates a new mock instance.
func NewMockKubernetesService(ctrl *gomock.Controller) *MockKubernetesService {
	mock := &MockKubernetesService{ctrl: ctrl}
	mock.recorder = &MockKubernetesServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKubernetesService) EXPECT() *MockKubernetesServiceMockRecorder {
	return m.recorder
}

// AddRegistry mocks base method.
func (m *MockKubernetesService) AddRegistry(req *godo.KubernetesClusterRegistryRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddRegistry", req)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddRegistry indicates an expected call of AddRegistry.
func (mr *MockKubernetesServiceMockRecorder) AddRegistry(req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddRegistry", reflect.TypeOf((*MockKubernetesService)(nil).AddRegistry), req)
}

// Create mocks base method.
func (m *MockKubernetesService) Create(create *godo.KubernetesClusterCreateRequest) (*do.KubernetesCluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", create)
	ret0, _ := ret[0].(*do.KubernetesCluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockKubernetesServiceMockRecorder) Create(create any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockKubernetesService)(nil).Create), create)
}

// CreateNodePool mocks base method.
func (m *MockKubernetesService) CreateNodePool(clusterID string, req *godo.KubernetesNodePoolCreateRequest) (*do.KubernetesNodePool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNodePool", clusterID, req)
	ret0, _ := ret[0].(*do.KubernetesNodePool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNodePool indicates an expected call of CreateNodePool.
func (mr *MockKubernetesServiceMockRecorder) CreateNodePool(clusterID, req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNodePool", reflect.TypeOf((*MockKubernetesService)(nil).CreateNodePool), clusterID, req)
}

// Delete mocks base method.
func (m *MockKubernetesService) Delete(clusterID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", clusterID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockKubernetesServiceMockRecorder) Delete(clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockKubernetesService)(nil).Delete), clusterID)
}

// DeleteDangerous mocks base method.
func (m *MockKubernetesService) DeleteDangerous(clusterID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDangerous", clusterID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteDangerous indicates an expected call of DeleteDangerous.
func (mr *MockKubernetesServiceMockRecorder) DeleteDangerous(clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDangerous", reflect.TypeOf((*MockKubernetesService)(nil).DeleteDangerous), clusterID)
}

// DeleteNode mocks base method.
func (m *MockKubernetesService) DeleteNode(clusterID, poolID, nodeID string, req *godo.KubernetesNodeDeleteRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNode", clusterID, poolID, nodeID, req)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNode indicates an expected call of DeleteNode.
func (mr *MockKubernetesServiceMockRecorder) DeleteNode(clusterID, poolID, nodeID, req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNode", reflect.TypeOf((*MockKubernetesService)(nil).DeleteNode), clusterID, poolID, nodeID, req)
}

// DeleteNodePool mocks base method.
func (m *MockKubernetesService) DeleteNodePool(clusterID, poolID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNodePool", clusterID, poolID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNodePool indicates an expected call of DeleteNodePool.
func (mr *MockKubernetesServiceMockRecorder) DeleteNodePool(clusterID, poolID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNodePool", reflect.TypeOf((*MockKubernetesService)(nil).DeleteNodePool), clusterID, poolID)
}

// DeleteSelective mocks base method.
func (m *MockKubernetesService) DeleteSelective(clusterID string, deleteReq *godo.KubernetesClusterDeleteSelectiveRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSelective", clusterID, deleteReq)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSelective indicates an expected call of DeleteSelective.
func (mr *MockKubernetesServiceMockRecorder) DeleteSelective(clusterID, deleteReq any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSelective", reflect.TypeOf((*MockKubernetesService)(nil).DeleteSelective), clusterID, deleteReq)
}

// Get mocks base method.
func (m *MockKubernetesService) Get(clusterID string) (*do.KubernetesCluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", clusterID)
	ret0, _ := ret[0].(*do.KubernetesCluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockKubernetesServiceMockRecorder) Get(clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockKubernetesService)(nil).Get), clusterID)
}

// GetCredentials mocks base method.
func (m *MockKubernetesService) GetCredentials(clusterID string) (*do.KubernetesClusterCredentials, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCredentials", clusterID)
	ret0, _ := ret[0].(*do.KubernetesClusterCredentials)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCredentials indicates an expected call of GetCredentials.
func (mr *MockKubernetesServiceMockRecorder) GetCredentials(clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCredentials", reflect.TypeOf((*MockKubernetesService)(nil).GetCredentials), clusterID)
}

// GetKubeConfig mocks base method.
func (m *MockKubernetesService) GetKubeConfig(clusterID string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKubeConfig", clusterID)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetKubeConfig indicates an expected call of GetKubeConfig.
func (mr *MockKubernetesServiceMockRecorder) GetKubeConfig(clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKubeConfig", reflect.TypeOf((*MockKubernetesService)(nil).GetKubeConfig), clusterID)
}

// GetKubeConfigWithExpiry mocks base method.
func (m *MockKubernetesService) GetKubeConfigWithExpiry(clusterID string, expirySeconds int64) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKubeConfigWithExpiry", clusterID, expirySeconds)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetKubeConfigWithExpiry indicates an expected call of GetKubeConfigWithExpiry.
func (mr *MockKubernetesServiceMockRecorder) GetKubeConfigWithExpiry(clusterID, expirySeconds any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKubeConfigWithExpiry", reflect.TypeOf((*MockKubernetesService)(nil).GetKubeConfigWithExpiry), clusterID, expirySeconds)
}

// GetNodePool mocks base method.
func (m *MockKubernetesService) GetNodePool(clusterID, poolID string) (*do.KubernetesNodePool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodePool", clusterID, poolID)
	ret0, _ := ret[0].(*do.KubernetesNodePool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodePool indicates an expected call of GetNodePool.
func (mr *MockKubernetesServiceMockRecorder) GetNodePool(clusterID, poolID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodePool", reflect.TypeOf((*MockKubernetesService)(nil).GetNodePool), clusterID, poolID)
}

// GetNodeSizes mocks base method.
func (m *MockKubernetesService) GetNodeSizes() (do.KubernetesNodeSizes, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodeSizes")
	ret0, _ := ret[0].(do.KubernetesNodeSizes)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodeSizes indicates an expected call of GetNodeSizes.
func (mr *MockKubernetesServiceMockRecorder) GetNodeSizes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeSizes", reflect.TypeOf((*MockKubernetesService)(nil).GetNodeSizes))
}

// GetRegions mocks base method.
func (m *MockKubernetesService) GetRegions() (do.KubernetesRegions, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegions")
	ret0, _ := ret[0].(do.KubernetesRegions)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegions indicates an expected call of GetRegions.
func (mr *MockKubernetesServiceMockRecorder) GetRegions() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegions", reflect.TypeOf((*MockKubernetesService)(nil).GetRegions))
}

// GetUpgrades mocks base method.
func (m *MockKubernetesService) GetUpgrades(clusterID string) (do.KubernetesVersions, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpgrades", clusterID)
	ret0, _ := ret[0].(do.KubernetesVersions)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUpgrades indicates an expected call of GetUpgrades.
func (mr *MockKubernetesServiceMockRecorder) GetUpgrades(clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpgrades", reflect.TypeOf((*MockKubernetesService)(nil).GetUpgrades), clusterID)
}

// GetVersions mocks base method.
func (m *MockKubernetesService) GetVersions() (do.KubernetesVersions, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVersions")
	ret0, _ := ret[0].(do.KubernetesVersions)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVersions indicates an expected call of GetVersions.
func (mr *MockKubernetesServiceMockRecorder) GetVersions() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVersions", reflect.TypeOf((*MockKubernetesService)(nil).GetVersions))
}

// List mocks base method.
func (m *MockKubernetesService) List() (do.KubernetesClusters, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].(do.KubernetesClusters)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockKubernetesServiceMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockKubernetesService)(nil).List))
}

// ListAssociatedResourcesForDeletion mocks base method.
func (m *MockKubernetesService) ListAssociatedResourcesForDeletion(clusterID string) (*do.KubernetesAssociatedResources, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAssociatedResourcesForDeletion", clusterID)
	ret0, _ := ret[0].(*do.KubernetesAssociatedResources)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAssociatedResourcesForDeletion indicates an expected call of ListAssociatedResourcesForDeletion.
func (mr *MockKubernetesServiceMockRecorder) ListAssociatedResourcesForDeletion(clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAssociatedResourcesForDeletion", reflect.TypeOf((*MockKubernetesService)(nil).ListAssociatedResourcesForDeletion), clusterID)
}

// ListNodePools mocks base method.
func (m *MockKubernetesService) ListNodePools(clusterID string) (do.KubernetesNodePools, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNodePools", clusterID)
	ret0, _ := ret[0].(do.KubernetesNodePools)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNodePools indicates an expected call of ListNodePools.
func (mr *MockKubernetesServiceMockRecorder) ListNodePools(clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNodePools", reflect.TypeOf((*MockKubernetesService)(nil).ListNodePools), clusterID)
}

// RecycleNodePoolNodes mocks base method.
func (m *MockKubernetesService) RecycleNodePoolNodes(clusterID, poolID string, req *godo.KubernetesNodePoolRecycleNodesRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecycleNodePoolNodes", clusterID, poolID, req)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecycleNodePoolNodes indicates an expected call of RecycleNodePoolNodes.
func (mr *MockKubernetesServiceMockRecorder) RecycleNodePoolNodes(clusterID, poolID, req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecycleNodePoolNodes", reflect.TypeOf((*MockKubernetesService)(nil).RecycleNodePoolNodes), clusterID, poolID, req)
}

// RemoveRegistry mocks base method.
func (m *MockKubernetesService) RemoveRegistry(req *godo.KubernetesClusterRegistryRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveRegistry", req)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveRegistry indicates an expected call of RemoveRegistry.
func (mr *MockKubernetesServiceMockRecorder) RemoveRegistry(req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveRegistry", reflect.TypeOf((*MockKubernetesService)(nil).RemoveRegistry), req)
}

// Update mocks base method.
func (m *MockKubernetesService) Update(clusterID string, update *godo.KubernetesClusterUpdateRequest) (*do.KubernetesCluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", clusterID, update)
	ret0, _ := ret[0].(*do.KubernetesCluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockKubernetesServiceMockRecorder) Update(clusterID, update any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockKubernetesService)(nil).Update), clusterID, update)
}

// UpdateNodePool mocks base method.
func (m *MockKubernetesService) UpdateNodePool(clusterID, poolID string, req *godo.KubernetesNodePoolUpdateRequest) (*do.KubernetesNodePool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateNodePool", clusterID, poolID, req)
	ret0, _ := ret[0].(*do.KubernetesNodePool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateNodePool indicates an expected call of UpdateNodePool.
func (mr *MockKubernetesServiceMockRecorder) UpdateNodePool(clusterID, poolID, req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateNodePool", reflect.TypeOf((*MockKubernetesService)(nil).UpdateNodePool), clusterID, poolID, req)
}

// Upgrade mocks base method.
func (m *MockKubernetesService) Upgrade(clusterID, versionSlug string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Upgrade", clusterID, versionSlug)
	ret0, _ := ret[0].(error)
	return ret0
}

// Upgrade indicates an expected call of Upgrade.
func (mr *MockKubernetesServiceMockRecorder) Upgrade(clusterID, versionSlug any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upgrade", reflect.TypeOf((*MockKubernetesService)(nil).Upgrade), clusterID, versionSlug)
}
