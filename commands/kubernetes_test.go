package commands

import (
	"fmt"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testCluster = do.KubernetesCluster{
		KubernetesCluster: &godo.KubernetesCluster{
			ID:          "cde2c0d6-41e3-479e-ba60-ad971227232b",
			Name:        "antoine_s_cluster",
			RegionSlug:  "sfo2",
			VersionSlug: "1.13.0",
			NodePools: []*godo.KubernetesNodePool{
				testNodePool.KubernetesNodePool,
			},
			MaintenancePolicy: &godo.KubernetesMaintenancePolicy{
				StartTime: "00:00",
				Day:       godo.KubernetesMaintenanceDayAny,
			},
			AutoUpgrade: true,
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

	testClusterUpgrades = do.KubernetesVersions{{
		KubernetesVersion: &godo.KubernetesVersion{
			Slug:              "1.13.1-do.1",
			KubernetesVersion: "1.13.1",
		},
	}}
)

func TestKubernetesCommand(t *testing.T) {
	cmd := Kubernetes()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"cluster",
		"options",
	)
}

func TestKubernetesClusterCommand(t *testing.T) {
	cmd := kubernetesCluster()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"get",
		"kubeconfig",
		"get-upgrades",
		"list",
		"create",
		"update",
		"upgrade",
		"delete",
		"node-pool",
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
		"regions",
		"sizes",
	)
}

func TestKubernetesGet(t *testing.T) {
	// by ID
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("Get", testCluster.ID).Return(&testCluster, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := RunKubernetesClusterGet(config)
		assert.NoError(t, err)
	})

	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.On("List").Return(testClusterList, nil)
		config.Args = append(config.Args, testCluster.Name)
		err := RunKubernetesClusterGet(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		name := "not a cluster"
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.On("List").Return(testClusterList, nil)
		config.Args = append(config.Args, name)
		err := RunKubernetesClusterGet(config)
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
		err := RunKubernetesClusterGet(config)
		assert.EqualError(t, err, errAmbigousClusterName(name, []string{testCluster.ID, testClusterWithSameName.ID}).Error())
	})
}

func TestKubernetesGetUpgrades(t *testing.T) {
	// by ID
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("GetUpgrades", testCluster.ID).Return(testClusterUpgrades, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := RunKubernetesClusterGetUpgrades(config)
		assert.NoError(t, err)
	})

	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.On("List").Return(testClusterList, nil)
		// then call GetUpgrades
		tm.kubernetes.On("GetUpgrades", testCluster.ID).Return(testClusterUpgrades, nil)
		config.Args = append(config.Args, testCluster.Name)
		err := RunKubernetesClusterGetUpgrades(config)
		assert.NoError(t, err)
	})

	// cluster does not exist
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		name := "not a cluster"
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.On("List").Return(testClusterList, nil)
		config.Args = append(config.Args, name)
		err := RunKubernetesClusterGetUpgrades(config)
		assert.EqualError(t, err, errNoClusterByName(name).Error())
	})

	// no upgrades available
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("GetUpgrades", testCluster.ID).Return(nil, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := RunKubernetesClusterGetUpgrades(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesKubeconfig(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		kubeconfig := []byte(`i'm some yaml`)
		tm.kubernetes.On("GetKubeConfig", testCluster.ID).Return(kubeconfig, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := RunKubernetesKubeconfigShow(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		kubeconfig := []byte(`i'm some yaml`)
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("GetKubeConfig", testCluster.ID).Return(kubeconfig, nil)
		config.Args = append(config.Args, testCluster.Name)
		err := RunKubernetesKubeconfigShow(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		err := RunKubernetesClusterList(config)
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
			MaintenancePolicy: &godo.KubernetesMaintenancePolicy{
				StartTime: "00:00",
				Day:       godo.KubernetesMaintenanceDayAny,
			},
			AutoUpgrade: true,
		}
		tm.kubernetes.On("Create", &r).Return(&testCluster, nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testCluster.RegionSlug)
		config.Doit.Set(config.NS, doctl.ArgClusterVersionSlug, testCluster.VersionSlug)
		config.Doit.Set(config.NS, doctl.ArgTag, testCluster.Tags)
		config.Doit.Set(config.NS, doctl.ArgMaintenanceWindow, "any=00:00")
		config.Doit.Set(config.NS, doctl.ArgClusterNodePool, []string{
			fmt.Sprintf("name=%s;size=%s;count=%d;tag=%s;tag=%s",
				testNodePool.Name+"1", testNodePool.Size, testNodePool.Count, testNodePool.Tags[0], testNodePool.Tags[1],
			),
			fmt.Sprintf("name=%s;size=%s;count=%d;tag=%s;tag=%s",
				testNodePool.Name+"2", testNodePool.Size, testNodePool.Count, testNodePool.Tags[0], testNodePool.Tags[1],
			),
		})
		config.Doit.Set(config.NS, doctl.ArgAutoUpgrade, testCluster.AutoUpgrade)

		err := RunKubernetesClusterCreate("c-8", 3)(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesUpdate(t *testing.T) {
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesClusterUpdateRequest{
			Name: testCluster.Name,
			Tags: testCluster.Tags,
			MaintenancePolicy: &godo.KubernetesMaintenancePolicy{
				StartTime: "00:00",
				Day:       godo.KubernetesMaintenanceDayAny,
			},
			AutoUpgrade: false,
		}
		tm.kubernetes.On("Update", testCluster.ID, &r).Return(&testCluster, nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgClusterName, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgTag, testCluster.Tags)
		config.Doit.Set(config.NS, doctl.ArgMaintenanceWindow, "any=00:00")
		config.Doit.Set(config.NS, doctl.ArgAutoUpgrade, false)

		err := RunKubernetesClusterUpdate(config)
		assert.NoError(t, err)
	})

	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesClusterUpdateRequest{
			Name: testCluster.Name,
			Tags: testCluster.Tags,
			MaintenancePolicy: &godo.KubernetesMaintenancePolicy{
				StartTime: "00:00",
				Day:       godo.KubernetesMaintenanceDayAny,
			},
			AutoUpgrade: false,
		}
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("Update", testCluster.ID, &r).Return(&testCluster, nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgClusterName, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgTag, testCluster.Tags)
		config.Doit.Set(config.NS, doctl.ArgMaintenanceWindow, "any=00:00")
		config.Doit.Set(config.NS, doctl.ArgAutoUpgrade, false)

		err := RunKubernetesClusterUpdate(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesUpgrade(t *testing.T) {
	testUpgradeVersion := testClusterUpgrades[0].Slug

	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("Upgrade", testCluster.ID, testUpgradeVersion).Return(nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgVersion, testUpgradeVersion)

		err := RunKubernetesClusterUpgrade(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("Upgrade", testCluster.ID, testUpgradeVersion).Return(nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgVersion, testUpgradeVersion)

		err := RunKubernetesClusterUpgrade(config)
		assert.NoError(t, err)
	})

	// using "latest" version
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("GetUpgrades", testCluster.ID).Return(testClusterUpgrades, nil)
		tm.kubernetes.On("Upgrade", testCluster.ID, testUpgradeVersion).Return(nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgVersion, defaultKubernetesLatestVersion)

		err := RunKubernetesClusterUpgrade(config)
		assert.NoError(t, err)
	})

	// without version flag set (defaults to latest)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("GetUpgrades", testCluster.ID).Return(testClusterUpgrades, nil)
		tm.kubernetes.On("Upgrade", testCluster.ID, testUpgradeVersion).Return(nil)

		config.Args = append(config.Args, testCluster.ID)

		err := RunKubernetesClusterUpgrade(config)
		assert.NoError(t, err)
	})

	// for cluster that is up-to-date
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("GetUpgrades", testCluster.ID).Return(nil, nil)

		config.Args = append(config.Args, testCluster.ID)

		err := RunKubernetesClusterUpgrade(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// should'nt call `DeleteNodePool` so we don't set any expectations
		config.Doit.Set(config.NS, doctl.ArgForce, "false")
		config.Args = append(config.Args, testCluster.ID)

		err := RunKubernetesClusterDelete(config)
		assert.Error(t, err, "should have been challenged before deletion")
	})
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("Delete", testCluster.ID).Return(nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunKubernetesClusterDelete(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("Delete", testCluster.ID).Return(nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunKubernetesClusterDelete(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesNodePool_Get(t *testing.T) {
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("GetNodePool", testCluster.ID, testNodePool.ID).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		err := RunKubernetesNodePoolGet(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)

		// cluster ID but pool name
		config.Args = append(config.Args, testCluster.ID, testNodePool.Name)

		err := RunKubernetesNodePoolGet(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("GetNodePool", testCluster.ID, testNodePool.ID).Return(&testNodePool, nil)

		// cluster name and pool ID
		config.Args = append(config.Args, testCluster.Name, testNodePool.ID)

		err := RunKubernetesNodePoolGet(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)

		// cluster name and pool name
		config.Args = append(config.Args, testCluster.Name, testNodePool.Name)

		err := RunKubernetesNodePoolGet(config)
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
		err := RunKubernetesNodePoolGet(config)
		assert.EqualError(t, err, errAmbigousPoolName(name, []string{testNodePool.ID, testNodePoolWithSameName.ID}).Error())
	})
}

func TestKubernetesNodePool_List(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)

		config.Args = append(config.Args, testCluster.ID)

		err := RunKubernetesNodePoolList(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("List").Return(testClusterList, nil)
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)

		config.Args = append(config.Args, testCluster.Name)

		err := RunKubernetesNodePoolList(config)
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
		config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)

		err := RunKubernetesNodePoolCreate(config)
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
		config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)

		err := RunKubernetesNodePoolCreate(config)
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
		config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)

		err := RunKubernetesNodePoolUpdate(config)
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
		config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)

		err := RunKubernetesNodePoolUpdate(config)
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
		config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)

		err := RunKubernetesNodePoolUpdate(config)
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
		config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)

		err := RunKubernetesNodePoolUpdate(config)
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

		err := RunKubernetesNodePoolRecycle(config)
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

		err := RunKubernetesNodePoolRecycle(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesNodePool_Delete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// should'nt call `DeleteNodePool` so we don't set any expectations
		config.Doit.Set(config.NS, doctl.ArgForce, "false")
		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		err := RunKubernetesNodePoolDelete(config)
		assert.Error(t, err, "should have been challenged before deletion")
	})
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("DeleteNodePool", testCluster.ID, testNodePool.ID).Return(nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunKubernetesNodePoolDelete(config)
		assert.NoError(t, err)
	})
	// by cluster ID and pool name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.On("ListNodePools", testCluster.ID).Return(testNodePools, nil)
		tm.kubernetes.On("DeleteNodePool", testCluster.ID, testNodePool.ID).Return(nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := RunKubernetesNodePoolDelete(config)
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

func TestKubernetesLatestVersions(t *testing.T) {
	tests := []struct {
		name  string
		input []do.KubernetesVersion
		want  []do.KubernetesVersion
	}{
		{
			name: "base case",
			input: []do.KubernetesVersion{
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.13.0-do.not.use", KubernetesVersion: "1.13.0"},
				},
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.1-do.3", KubernetesVersion: "1.12.1"},
				},
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.1-do.2", KubernetesVersion: "1.12.1"},
				},
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.11.1-do.2", KubernetesVersion: "1.11.1"},
				},
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.11.1-do.1", KubernetesVersion: "1.11.1"},
				},
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.10.7-gen2", KubernetesVersion: "1.10.7"},
				},
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.10.7-gen1", KubernetesVersion: "1.10.7"},
				},
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.10.7-gen0", KubernetesVersion: "1.10.7"},
				},
			},
			want: []do.KubernetesVersion{
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.10.7-gen2", KubernetesVersion: "1.10.7"},
				},
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.11.1-do.2", KubernetesVersion: "1.11.1"},
				},
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.1-do.3", KubernetesVersion: "1.12.1"},
				},
				{
					KubernetesVersion: &godo.KubernetesVersion{Slug: "1.13.0-do.not.use", KubernetesVersion: "1.13.0"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := latestReleases(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

type nilCluster struct {
	do.KubernetesService
}

func (n *nilCluster) Get(clusterID string) (*do.KubernetesCluster, error) {
	return nil, fmt.Errorf("can't find %s", clusterID)
}

func Test_waitForClusterRunningDoesntPanicWithNilGet(t *testing.T) {
	cluster, err := waitForClusterRunning(&nilCluster{}, "123")
	require.Nil(t, cluster)
	require.EqualError(t, err, "can't find 123")
}
