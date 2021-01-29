/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// KubernetesCluster wraps a godo KubernetesCluster.
type KubernetesCluster struct {
	*godo.KubernetesCluster
}

// KubernetesClusterCredentials wraps a godo KubernetesClusterCredentials.
type KubernetesClusterCredentials struct {
	*godo.KubernetesClusterCredentials
}

// KubernetesClusters is a slice of KubernetesCluster.
type KubernetesClusters []KubernetesCluster

// KubernetesNodePool wraps a godo KubernetesNodePool.
type KubernetesNodePool struct {
	*godo.KubernetesNodePool
}

// KubernetesNodePools is a slice of KubernetesNodePool.
type KubernetesNodePools []KubernetesNodePool

// KubernetesVersions is a slice of KubernetesVersions.
type KubernetesVersions []KubernetesVersion

// KubernetesVersion wraps a godo KubernetesVersion.
type KubernetesVersion struct {
	*godo.KubernetesVersion
}

// KubernetesRegions is a slice of KubernetesRegions.
type KubernetesRegions []KubernetesRegion

// KubernetesRegion wraps a godo KubernetesRegion.
type KubernetesRegion struct {
	*godo.KubernetesRegion
}

// KubernetesNodeSizes is a slice of KubernetesNodeSizes.
type KubernetesNodeSizes []KubernetesNodeSize

// KubernetesNodeSize wraps a godo KubernetesNodeSize.
type KubernetesNodeSize struct {
	*godo.KubernetesNodeSize
}

// KubernetesAssociatedResources wraps a godo KubernetesAssociatedResources
type KubernetesAssociatedResources struct {
	*godo.KubernetesAssociatedResources
}

// KubernetesService is the godo KubernetesService interface.
type KubernetesService interface {
	Get(clusterID string) (*KubernetesCluster, error)
	GetKubeConfig(clusterID string) ([]byte, error)
	GetKubeConfigWithExpiry(clusterID string, expirySeconds int64) ([]byte, error)
	GetCredentials(clusterID string) (*KubernetesClusterCredentials, error)
	GetUpgrades(clusterID string) (KubernetesVersions, error)
	List() (KubernetesClusters, error)
	ListAssociatedResourcesForDeletion(clusterID string) (*KubernetesAssociatedResources, error)
	Create(create *godo.KubernetesClusterCreateRequest) (*KubernetesCluster, error)
	Update(clusterID string, update *godo.KubernetesClusterUpdateRequest) (*KubernetesCluster, error)
	Upgrade(clusterID string, versionSlug string) error
	Delete(clusterID string) error
	DeleteDangerous(clusterID string) error
	DeleteSelective(clusterID string, deleteReq *godo.KubernetesClusterDeleteSelectiveRequest) error

	CreateNodePool(clusterID string, req *godo.KubernetesNodePoolCreateRequest) (*KubernetesNodePool, error)
	GetNodePool(clusterID, poolID string) (*KubernetesNodePool, error)
	ListNodePools(clusterID string) (KubernetesNodePools, error)
	UpdateNodePool(clusterID, poolID string, req *godo.KubernetesNodePoolUpdateRequest) (*KubernetesNodePool, error)
	// RecycleNodePoolNodes is DEPRECATED please use DeleteNode
	RecycleNodePoolNodes(clusterID, poolID string, req *godo.KubernetesNodePoolRecycleNodesRequest) error
	DeleteNodePool(clusterID, poolID string) error
	DeleteNode(clusterID, poolID, nodeID string, req *godo.KubernetesNodeDeleteRequest) error

	GetVersions() (KubernetesVersions, error)
	GetRegions() (KubernetesRegions, error)
	GetNodeSizes() (KubernetesNodeSizes, error)
	AddRegistry(req *godo.KubernetesClusterRegistryRequest) error
	RemoveRegistry(req *godo.KubernetesClusterRegistryRequest) error
}

var _ KubernetesService = &kubernetesClusterService{}

type kubernetesClusterService struct {
	client godo.KubernetesService
}

// NewKubernetesService builds an instance of KubernetesService.
func NewKubernetesService(client *godo.Client) KubernetesService {
	return &kubernetesClusterService{
		client: client.Kubernetes,
	}
}

func (k8s *kubernetesClusterService) Get(clusterID string) (*KubernetesCluster, error) {
	cluster, _, err := k8s.client.Get(context.TODO(), clusterID)
	if err != nil {
		return nil, err
	}

	return &KubernetesCluster{KubernetesCluster: cluster}, nil
}

func (k8s *kubernetesClusterService) GetKubeConfig(clusterID string) ([]byte, error) {
	config, _, err := k8s.client.GetKubeConfig(context.TODO(), clusterID)
	if err != nil {
		return nil, err
	}

	return config.KubeconfigYAML, nil
}

func (k8s *kubernetesClusterService) GetKubeConfigWithExpiry(clusterID string, expirySeconds int64) ([]byte, error) {
	config, _, err := k8s.client.GetKubeConfigWithExpiry(context.TODO(), clusterID, expirySeconds)
	if err != nil {
		return nil, err
	}

	return config.KubeconfigYAML, nil
}

func (k8s *kubernetesClusterService) GetCredentials(clusterID string) (*KubernetesClusterCredentials, error) {
	credentials, _, err := k8s.client.GetCredentials(context.TODO(), clusterID, &godo.KubernetesClusterCredentialsGetRequest{})
	if err != nil {
		return nil, err
	}

	return &KubernetesClusterCredentials{
		KubernetesClusterCredentials: credentials,
	}, nil
}

func (k8s *kubernetesClusterService) GetUpgrades(clusterID string) (KubernetesVersions, error) {
	upgrades, _, err := k8s.client.GetUpgrades(context.TODO(), clusterID)
	if err != nil {
		return nil, err
	}

	versions := make([]KubernetesVersion, len(upgrades))
	for i, v := range upgrades {
		versions[i] = KubernetesVersion{
			KubernetesVersion: v,
		}
	}

	return versions, nil
}

func (k8s *kubernetesClusterService) List() (KubernetesClusters, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := k8s.client.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, 0, len(list))
		for _, item := range list {
			si = append(si, item)
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]KubernetesCluster, 0, len(si))
	for _, item := range si {
		a := item.(*godo.KubernetesCluster)
		list = append(list, KubernetesCluster{KubernetesCluster: a})
	}

	return list, nil
}

func (k8s *kubernetesClusterService) ListAssociatedResourcesForDeletion(clusterID string) (*KubernetesAssociatedResources, error) {
	ar, _, err := k8s.client.ListAssociatedResourcesForDeletion(context.TODO(), clusterID)
	if err != nil {
		return nil, err
	}
	return &KubernetesAssociatedResources{ar}, nil
}

func (k8s *kubernetesClusterService) Create(create *godo.KubernetesClusterCreateRequest) (*KubernetesCluster, error) {
	cluster, _, err := k8s.client.Create(context.TODO(), create)
	if err != nil {
		return nil, err
	}
	return &KubernetesCluster{KubernetesCluster: cluster}, nil
}

func (k8s *kubernetesClusterService) Update(clusterID string, update *godo.KubernetesClusterUpdateRequest) (*KubernetesCluster, error) {
	cluster, _, err := k8s.client.Update(context.TODO(), clusterID, update)
	if err != nil {
		return nil, err
	}
	return &KubernetesCluster{KubernetesCluster: cluster}, nil
}

func (k8s *kubernetesClusterService) Upgrade(clusterID string, versionSlug string) error {
	req := &godo.KubernetesClusterUpgradeRequest{
		VersionSlug: versionSlug,
	}

	_, err := k8s.client.Upgrade(context.TODO(), clusterID, req)
	return err
}

func (k8s *kubernetesClusterService) Delete(clusterID string) error {
	_, err := k8s.client.Delete(context.TODO(), clusterID)
	return err
}

func (k8s *kubernetesClusterService) DeleteDangerous(clusterID string) error {
	_, err := k8s.client.DeleteDangerous(context.TODO(), clusterID)
	return err
}

func (k8s *kubernetesClusterService) DeleteSelective(clusterID string, req *godo.KubernetesClusterDeleteSelectiveRequest) error {
	_, err := k8s.client.DeleteSelective(context.TODO(), clusterID, req)
	return err
}

func (k8s *kubernetesClusterService) CreateNodePool(clusterID string, req *godo.KubernetesNodePoolCreateRequest) (*KubernetesNodePool, error) {
	pool, _, err := k8s.client.CreateNodePool(context.TODO(), clusterID, req)
	if err != nil {
		return nil, err
	}
	return &KubernetesNodePool{KubernetesNodePool: pool}, nil
}

func (k8s *kubernetesClusterService) GetNodePool(clusterID, poolID string) (*KubernetesNodePool, error) {
	pool, _, err := k8s.client.GetNodePool(context.TODO(), clusterID, poolID)
	if err != nil {
		return nil, err
	}
	return &KubernetesNodePool{KubernetesNodePool: pool}, nil
}

func (k8s *kubernetesClusterService) ListNodePools(clusterID string) (KubernetesNodePools, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := k8s.client.ListNodePools(context.TODO(), clusterID, opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, 0, len(list))
		for _, item := range list {
			si = append(si, item)
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]KubernetesNodePool, 0, len(si))
	for _, item := range si {
		a := item.(*godo.KubernetesNodePool)
		list = append(list, KubernetesNodePool{KubernetesNodePool: a})
	}

	return list, nil

}

func (k8s *kubernetesClusterService) UpdateNodePool(clusterID, poolID string, req *godo.KubernetesNodePoolUpdateRequest) (*KubernetesNodePool, error) {
	pool, _, err := k8s.client.UpdateNodePool(context.TODO(), clusterID, poolID, req)
	if err != nil {
		return nil, err
	}
	return &KubernetesNodePool{KubernetesNodePool: pool}, nil
}

func (k8s *kubernetesClusterService) RecycleNodePoolNodes(clusterID, poolID string, req *godo.KubernetesNodePoolRecycleNodesRequest) error {
	_, err := k8s.client.RecycleNodePoolNodes(context.TODO(), clusterID, poolID, req)
	return err
}

func (k8s *kubernetesClusterService) DeleteNodePool(clusterID, poolID string) error {
	_, err := k8s.client.DeleteNodePool(context.TODO(), clusterID, poolID)
	return err
}

func (k8s *kubernetesClusterService) DeleteNode(clusterID, poolID, nodeID string, req *godo.KubernetesNodeDeleteRequest) error {
	_, err := k8s.client.DeleteNode(context.TODO(), clusterID, poolID, nodeID, req)
	return err
}

func (k8s *kubernetesClusterService) GetVersions() (KubernetesVersions, error) {
	opts, _, err := k8s.client.GetOptions(context.TODO())
	if err != nil {
		return nil, err
	}
	list := make(KubernetesVersions, 0, len(opts.Versions))
	for _, item := range opts.Versions {
		list = append(list, KubernetesVersion{KubernetesVersion: item})
	}
	return list, err
}

func (k8s *kubernetesClusterService) GetRegions() (KubernetesRegions, error) {
	opts, _, err := k8s.client.GetOptions(context.TODO())
	if err != nil {
		return nil, err
	}
	list := make(KubernetesRegions, 0, len(opts.Regions))
	for _, item := range opts.Regions {
		list = append(list, KubernetesRegion{KubernetesRegion: item})
	}
	return list, err
}

func (k8s *kubernetesClusterService) GetNodeSizes() (KubernetesNodeSizes, error) {
	opts, _, err := k8s.client.GetOptions(context.TODO())
	if err != nil {
		return nil, err
	}
	list := make(KubernetesNodeSizes, 0, len(opts.Sizes))
	for _, item := range opts.Sizes {
		list = append(list, KubernetesNodeSize{KubernetesNodeSize: item})
	}
	return list, err
}

func (k8s *kubernetesClusterService) AddRegistry(req *godo.KubernetesClusterRegistryRequest) error {
	_, err := k8s.client.AddRegistry(context.TODO(), req)
	return err
}

func (k8s *kubernetesClusterService) RemoveRegistry(req *godo.KubernetesClusterRegistryRequest) error {
	_, err := k8s.client.RemoveRegistry(context.TODO(), req)
	return err
}
