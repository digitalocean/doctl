package commands

import (
	"fmt"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testCluster = do.KubernetesCluster{
		KubernetesCluster: &godo.KubernetesCluster{
			ID:          "cde2c0d6-41e3-479e-ba60-ad971227232b",
			Name:        "antoine_s_cluster",
			RegionSlug:  "sfo2",
			VersionSlug: "",
			NodePools: []*godo.KubernetesNodePool{
				testNodePool.KubernetesNodePool,
			},
		},
	}

	testClusterList = do.KubernetesClusters{
		testCluster,
	}

	testNodePool = do.KubernetesNodePool{
		KubernetesNodePool: &godo.KubernetesNodePool{
			ID:    "ede2c0d6-41e3-479e-ba60-ad9712272324",
			Name:  "antoine_s_pool",
			Size:  "c8",
			Count: 3,
			Tags:  []string{"hello", "bye"},
			Nodes: testNodes,
		},
	}

	testNodePools = do.KubernetesNodePools{
		testNodePool,
	}

	testNode = &godo.KubernetesNode{
		ID:   "ede2c0d6-41e3-479e-ba60-ad9712272324",
		Name: "antoine_s_node",
	}

	testNodes = []*godo.KubernetesNode{
		testNode,
	}
)

func TestKubernetesCommand(t *testing.T) {
	cmd := Kubernetes()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"get",
		"kubeconfig",
		"list",
		"create",
		"update",
		"delete",
		"node-pool",
		"options",
	)
}

func TestKubernetesNodePoolCommand(t *testing.T) {
	cmd := kubernetesNodePools()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"get",
		"list",
		"create",
		"update",
		"recycle",
		"delete",
	)
}

func TestKubernetesOptionsCommand(t *testing.T) {
	cmd := kubernetesOptions()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"versions",
	)
}

func TestKubernetesGet(t *testing.T) {
	// by ID
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("Get", testCluster.ID).Return(&testCluster, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := RunKubernetesGet(config)
		assert.NoError(t, err)
	})

	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.On("List").Return(testClusterList, nil)
		config.Args = append(config.Args, testCluster.Name)
		err := RunKubernetesGet(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		name := "not a cluster"
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.On("List").Return(testClusterList, nil)
		config.Args = append(config.Args, name)
		err := RunKubernetesGet(config)
		assert.EqualError(t, err, errNoClusterByName(name).Error())
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		name := testCluster.Name
		testClusterWithSameName := do.KubernetesCluster{
			KubernetesCluster: &godo.KubernetesCluster{
				ID:          "cde2c0d6-41e3-479e-ba60-ad9712272322",
				Name:        name,
				RegionSlug:  "sfo2",
				VersionSlug: "",
				NodePools: []*godo.KubernetesNodePool{
					testNodePool.KubernetesNodePool,
				},
			},
		}

		clustersWithDups := append(testClusterList, testClusterWithSameName)
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.On("List").Return(clustersWithDups, nil)
		config.Args = append(config.Args, name)
		err := RunKubernetesGet(config)
		assert.EqualError(t, err, errAmbigousClusterName(name, []string{testCluster.ID, testClusterWithSameName.ID}).Error())
	})
}

func TestKubernetesKubeconfig(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		kubeconfig := []byte(`i'm some yaml`)
		tm.kubernetes.On("GetKubeConfig", testCluster.ID).Return(kubeconfig, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := RunKubernetesGetKubeconfig(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		kubeconfig := []byte(`i'm some yaml`)
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("GetKubeConfig", testCluster.ID).Return(kubeconfig, nil)
		config.Args = append(config.Args, testCluster.Name)
		err := RunKubernetesGetKubeconfig(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		err := RunKubernetesList(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesClusterCreateRequest{
			Name:        testCluster.Name,
			RegionSlug:  testCluster.RegionSlug,
			VersionSlug: testCluster.VersionSlug,
			Tags:        testCluster.Tags,
			NodePools: []*godo.KubernetesNodePoolCreateRequest{
				{
					Name:  testNodePool.Name + "1",
					Size:  testNodePool.Size,
					Count: testNodePool.Count,
					Tags:  testNodePool.Tags,
				},
				{
					Name:  testNodePool.Name + "2",
					Size:  testNodePool.Size,
					Count: testNodePool.Count,
					Tags:  testNodePool.Tags,
				},
			},
		}
		tm.kubernetes.On("Create", &r).Return(&testCluster, nil)

		config.Doit.Set(config.NS, doctl.ArgClusterName, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testCluster.RegionSlug)
		config.Doit.Set(config.NS, doctl.ArgClusterVersionSlug, testCluster.VersionSlug)
		config.Doit.Set(config.NS, doctl.ArgTagNames, testCluster.Tags)
		config.Doit.Set(config.NS, doctl.ArgClusterNodePools, []string{
			fmt.Sprintf("name=%s;size=%s;count=%d;tag=%s;tag=%s",
				testNodePool.Name+"1", testNodePool.Size, testNodePool.Count, testNodePool.Tags[0], testNodePool.Tags[1],
			),
			fmt.Sprintf("name=%s;size=%s;count=%d;tag=%s;tag=%s",
				testNodePool.Name+"2", testNodePool.Size, testNodePool.Count, testNodePool.Tags[0], testNodePool.Tags[1],
			),
		})

		err := RunKubernetesCreate(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesUpdate(t *testing.T) {
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesClusterUpdateRequest{
			Name: testCluster.Name,
			Tags: testCluster.Tags,
		}
		tm.kubernetes.On("Update", testCluster.ID, &r).Return(&testCluster, nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgClusterName, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgTagNames, testCluster.Tags)

		err := RunKubernetesUpdate(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesClusterUpdateRequest{
			Name: testCluster.Name,
			Tags: testCluster.Tags,
		}
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("Update", testCluster.ID, &r).Return(&testCluster, nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgClusterName, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgTagNames, testCluster.Tags)

		err := RunKubernetesUpdate(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// should'nt call `DeleteNodePool` so we don't set any expectations
		config.Doit.Set(config.NS, doctl.ArgForce, "false")
		config.Args = append(config.Args, testCluster.ID)

		err := RunKubernetesDelete(config)
		assert.Error(t, err, "should have been challenged before deletion")
	})
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("Delete", testCluster.ID).Return(nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunKubernetesDelete(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("Delete", testCluster.ID).Return(nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunKubernetesDelete(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesNodePool_Get(t *testing.T) {
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("GetNodePool", testCluster.ID, testNodePool.ID).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		err := RunClusterNodePoolGet(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)

		// cluster ID but pool name
		config.Args = append(config.Args, testCluster.ID, testNodePool.Name)

		err := RunClusterNodePoolGet(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("GetNodePool", testCluster.ID, testNodePool.ID).Return(&testNodePool, nil)

		// cluster name and pool ID
		config.Args = append(config.Args, testCluster.Name, testNodePool.ID)

		err := RunClusterNodePoolGet(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)

		// cluster name and pool name
		config.Args = append(config.Args, testCluster.Name, testNodePool.Name)

		err := RunClusterNodePoolGet(config)
		assert.NoError(t, err)
	})
	// ambiguous names
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		name := testNodePool.Name
		testNodePoolWithSameName := do.KubernetesNodePool{
			KubernetesNodePool: &godo.KubernetesNodePool{
				ID:   "cde2c0d6-41e3-479e-ba60-ad9712272322",
				Name: name,
			},
		}

		nodePoolsWithDups := append(testNodePools, testNodePoolWithSameName)
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(nodePoolsWithDups, nil)
		config.Args = append(config.Args, testCluster.ID, name)
		err := RunClusterNodePoolGet(config)
		assert.EqualError(t, err, errAmbigousPoolName(name, []string{testNodePool.ID, testNodePoolWithSameName.ID}).Error())
	})
}

func TestKubernetesNodePool_List(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)

		config.Args = append(config.Args, testCluster.ID)

		err := RunClusterNodePoolList(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)

		config.Args = append(config.Args, testCluster.Name)

		err := RunClusterNodePoolList(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesNodePool_Create(t *testing.T) {
	// by cluster ID
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolCreateRequest{
			Name:  testNodePool.Name,
			Size:  testNodePool.Size,
			Count: testNodePool.Count,
			Tags:  testNodePool.Tags,
		}
		tm.kubernetes.On("CreateNodePool", testCluster.ID, &r).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.ID)

		config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, testNodePool.Size)
		config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
		config.Doit.Set(config.NS, doctl.ArgTagNames, testNodePool.Tags)

		err := RunClusterNodePoolCreate(config)
		assert.NoError(t, err)
	})
	// by cluster name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolCreateRequest{
			Name:  testNodePool.Name,
			Size:  testNodePool.Size,
			Count: testNodePool.Count,
			Tags:  testNodePool.Tags,
		}
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("CreateNodePool", testCluster.ID, &r).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.Name)

		config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, testNodePool.Size)
		config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
		config.Doit.Set(config.NS, doctl.ArgTagNames, testNodePool.Tags)

		err := RunClusterNodePoolCreate(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesNodePool_Update(t *testing.T) {
	// by cluster ID
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolUpdateRequest{
			Name:  testNodePool.Name,
			Count: testNodePool.Count,
			Tags:  testNodePool.Tags,
		}
		tm.kubernetes.On("UpdateNodePool", testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
		config.Doit.Set(config.NS, doctl.ArgTagNames, testNodePool.Tags)

		err := RunClusterNodePoolUpdate(config)
		assert.NoError(t, err)
	})
	// by cluster name, pool ID
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolUpdateRequest{
			Name:  testNodePool.Name,
			Count: testNodePool.Count,
			Tags:  testNodePool.Tags,
		}
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("UpdateNodePool", testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.Name, testNodePool.ID)

		config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
		config.Doit.Set(config.NS, doctl.ArgTagNames, testNodePool.Tags)

		err := RunClusterNodePoolUpdate(config)
		assert.NoError(t, err)
	})
	// by cluster ID, pool name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolUpdateRequest{
			Name:  testNodePool.Name,
			Count: testNodePool.Count,
			Tags:  testNodePool.Tags,
		}
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)
		tm.kubernetes.On("UpdateNodePool", testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.Name)

		config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
		config.Doit.Set(config.NS, doctl.ArgTagNames, testNodePool.Tags)

		err := RunClusterNodePoolUpdate(config)
		assert.NoError(t, err)
	})
	// by cluster name, pool name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolUpdateRequest{
			Name:  testNodePool.Name,
			Count: testNodePool.Count,
			Tags:  testNodePool.Tags,
		}
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)
		tm.kubernetes.On("UpdateNodePool", testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.Name, testNodePool.Name)

		config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
		config.Doit.Set(config.NS, doctl.ArgTagNames, testNodePool.Tags)

		err := RunClusterNodePoolUpdate(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesNodePool_Recycle(t *testing.T) {
	// by node IDs
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolRecycleNodesRequest{
			Nodes: []string{testNode.ID},
		}

		tm.kubernetes.On("RecycleNodePoolNodes", testCluster.ID, testNodePool.ID, &r).Return(nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		config.Doit.Set(config.NS, doctl.ArgNodePoolNodeIDs, testNode.ID)

		err := RunClusterNodePoolRecycle(config)
		assert.NoError(t, err)
	})
	// by node names
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolRecycleNodesRequest{
			Nodes: []string{testNode.ID},
		}

		tm.kubernetes.On("GetNodePool", testCluster.ID, testNodePool.ID).Return(&testNodePool, nil)
		tm.kubernetes.On("RecycleNodePoolNodes", testCluster.ID, testNodePool.ID, &r).Return(nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		config.Doit.Set(config.NS, doctl.ArgNodePoolNodeIDs, testNode.Name)

		err := RunClusterNodePoolRecycle(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesNodePool_Delete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// should'nt call `DeleteNodePool` so we don't set any expectations
		config.Doit.Set(config.NS, doctl.ArgForce, "false")
		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		err := RunClusterNodePoolDelete(config)
		assert.Error(t, err, "should have been challenged before deletion")
	})
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("DeleteNodePool", testCluster.ID, testNodePool.ID).Return(nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunClusterNodePoolDelete(config)
		assert.NoError(t, err)
	})
	// by cluster ID and pool name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)
		tm.kubernetes.On("DeleteNodePool", testCluster.ID, testNodePool.ID).Return(nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunClusterNodePoolDelete(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesOptions_Versions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		testVersions := do.KubernetesVersions{
			do.KubernetesVersion{
				KubernetesVersion: &godo.KubernetesVersion{Slug: "1.10gen3", KubernetesVersion: "1.10"},
			},
		}
		tm.kubernetes.On("GetVersions").Return(testVersions, nil)

		err := RunKubeOptionsListVersion(config)
		assert.NoError(t, err)
	})
}
