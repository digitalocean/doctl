package commands

import (
	"fmt"
	"sort"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
			HA:          true,
		},
	}

	testClusterList = do.KubernetesClusters{
		testCluster,
	}

	testNodePool = do.KubernetesNodePool{
		KubernetesNodePool: &godo.KubernetesNodePool{
			ID:     "ede2c0d6-41e3-479e-ba60-ad9712272324",
			Name:   "antoine_s_pool",
			Size:   "c8",
			Count:  3,
			Tags:   []string{"hello", "bye"},
			Labels: map[string]string{},
			Taints: []godo.Taint{},
			Nodes:  testNodes,
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

	testKubeconfig = clientcmdapi.Config{
		APIVersion:     "v1",
		CurrentContext: "test-context",
		Contexts: map[string]*clientcmdapi.Context{
			"test-context": {
				Cluster: "test-cluster",
			},
		},
		Clusters: map[string]*clientcmdapi.Cluster{
			"test-cluster": clientcmdapi.NewCluster(),
		},
		AuthInfos: make(map[string]*clientcmdapi.AuthInfo),
	}

	testK8sOneClickList = do.OneClicks{
		testOneClick,
	}

	testAssociatedResources = do.KubernetesAssociatedResources{
		KubernetesAssociatedResources: &godo.KubernetesAssociatedResources{
			Volumes: []*godo.AssociatedResource{
				{
					ID:   "1422",
					Name: "vol-1",
				},
			},
			VolumeSnapshots: []*godo.AssociatedResource{
				{
					ID:   "3536",
					Name: "snap-1",
				},
			},
			LoadBalancers: []*godo.AssociatedResource{
				{
					ID:   "7574",
					Name: "lb-1",
				},
			},
		},
	}
	volumeID   = uuid.New()
	snapshotID = uuid.New()
	lbID       = uuid.New()

	testVolumes = []do.Volume{
		{
			Volume: &godo.Volume{
				ID:            volumeID.String(),
				Name:          "vol-1",
				SizeGigaBytes: 4,
				Region: &godo.Region{
					Slug: testCluster.RegionSlug,
				},
			},
		},
	}

	testSnapshots = do.Snapshots{
		{
			Snapshot: &godo.Snapshot{
				ID:            snapshotID.String(),
				Name:          "snap-1",
				SizeGigaBytes: 3,
				ResourceType:  "volume",
				ResourceID:    volumeID.String(),
				Regions:       []string{testCluster.RegionSlug},
			},
		},
	}

	testLoadBalancers = do.LoadBalancers{
		{
			LoadBalancer: &godo.LoadBalancer{
				ID:   lbID.String(),
				Name: "lb-1",
				IP:   "10.12.1.3",
			},
		},
	}
)

type mockKubeconfigProvider struct {
	local, remote, written clientcmdapi.Config
}

func (m *mockKubeconfigProvider) Remote(_ do.KubernetesService, _ string, _ int) (*clientcmdapi.Config, error) {
	return &m.local, nil
}

func (m *mockKubeconfigProvider) Local() (*clientcmdapi.Config, error) {
	return &m.remote, nil
}

func (m *mockKubeconfigProvider) Write(config *clientcmdapi.Config) error {
	m.written = *config
	return nil
}

func (m *mockKubeconfigProvider) ConfigPath() string {
	return "/some/kube/path"
}

func testK8sCmdService() *KubernetesCommandService {
	return &KubernetesCommandService{
		KubeconfigProvider: &mockKubeconfigProvider{
			local:  testKubeconfig,
			remote: testKubeconfig,
		},
	}
}

func TestKubernetesCommand(t *testing.T) {
	cmd := Kubernetes()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"cluster",
		"options",
		"1-click",
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
		"registry",
		"delete-selective",
		"list-associated-resources",
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
		"delete-node",
		"replace-node",
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

func TestKubernetesOneClickCommand(t *testing.T) {
	cmd := kubernetesOneClicks()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"install",
		"list",
	)
}

func TestKubernetesGet(t *testing.T) {
	// by ID
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().Get(testCluster.ID).Return(&testCluster, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := testK8sCmdService().RunKubernetesClusterGet(config)
		assert.NoError(t, err)
	})

	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		config.Args = append(config.Args, testCluster.Name)
		err := testK8sCmdService().RunKubernetesClusterGet(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		name := "not a cluster"
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		config.Args = append(config.Args, name)
		err := testK8sCmdService().RunKubernetesClusterGet(config)
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
		tm.kubernetes.EXPECT().List().Return(clustersWithDups, nil)
		config.Args = append(config.Args, name)
		err := testK8sCmdService().RunKubernetesClusterGet(config)
		assert.EqualError(t, err, errAmbiguousClusterName(name, []string{testCluster.ID, testClusterWithSameName.ID}).Error())
	})
}

func TestKubernetesGetUpgrades(t *testing.T) {
	// by ID
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().GetUpgrades(testCluster.ID).Return(testClusterUpgrades, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := testK8sCmdService().RunKubernetesClusterGetUpgrades(config)
		assert.NoError(t, err)
	})

	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		// then call GetUpgrades
		tm.kubernetes.EXPECT().GetUpgrades(testCluster.ID).Return(testClusterUpgrades, nil)
		config.Args = append(config.Args, testCluster.Name)
		err := testK8sCmdService().RunKubernetesClusterGetUpgrades(config)
		assert.NoError(t, err)
	})

	// cluster does not exist
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		name := "not a cluster"
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		config.Args = append(config.Args, name)
		err := testK8sCmdService().RunKubernetesClusterGetUpgrades(config)
		assert.EqualError(t, err, errNoClusterByName(name).Error())
	})

	// no upgrades available
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().GetUpgrades(testCluster.ID).Return(nil, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := testK8sCmdService().RunKubernetesClusterGetUpgrades(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesKubeconfigSave(t *testing.T) {
	// save the remote kubeconfig locally
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testCluster.ID)
		k8sCmdService := testK8sCmdService()
		err := k8sCmdService.RunKubernetesKubeconfigSave(config)
		assert.NoError(t, err)

		provider := k8sCmdService.KubeconfigProvider.(*mockKubeconfigProvider)
		assert.Equal(t, provider.remote, provider.written)
		assert.Contains(t, provider.written.AuthInfos[""].Exec.Command, "commands.test")
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expirySeconds := int64(60)
		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgKubeConfigExpirySeconds, expirySeconds)

		k8sCmdService := testK8sCmdService()
		err := k8sCmdService.RunKubernetesKubeconfigSave(config)
		assert.NoError(t, err)

		provider := k8sCmdService.KubeconfigProvider.(*mockKubeconfigProvider)
		assert.Equal(t, provider.remote, provider.written)
		assert.Equal(t, provider.remote.AuthInfos[""].Token, provider.written.AuthInfos[""].Token)
		assert.Nil(t, provider.written.AuthInfos[""].Exec)
	})

	// save the remote kubeconfig locally, verifying that the provided auth
	// context is successfully set
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		authContext := "not-default"

		getCurrentAuthContextFn = func() string {
			authContext, err := config.Doit.GetString(config.NS, doctl.ArgContext)
			assert.NoError(t, err)
			return authContext
		}
		defer func() {
			getCurrentAuthContextFn = defaultGetCurrentAuthContextFn
		}()

		config.Args = append(config.Args, testCluster.ID)

		config.Doit.Set(config.NS, doctl.ArgContext, authContext)

		k8sCmdService := testK8sCmdService()
		err := k8sCmdService.RunKubernetesKubeconfigSave(config)
		assert.NoError(t, err)

		provider := k8sCmdService.KubeconfigProvider.(*mockKubeconfigProvider)
		assert.NoError(t, err)
		assert.Equal(t, provider.remote, provider.written)

		expectedExecContextArg := "--" + doctl.ArgContext + "=" + authContext
		assert.Contains(t, provider.written.AuthInfos[""].Exec.Args, expectedExecContextArg)
	})
}

func TestKubernetesKubeconfigShow(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		kubeconfig := []byte(`i'm some yaml`)
		tm.kubernetes.EXPECT().GetKubeConfig(testCluster.ID).Return(kubeconfig, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := testK8sCmdService().RunKubernetesKubeconfigShow(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		kubeconfig := []byte(`i'm some yaml`)
		expirySeconds := int64(60)
		tm.kubernetes.EXPECT().GetKubeConfigWithExpiry(testCluster.ID, expirySeconds).Return(kubeconfig, nil)
		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgKubeConfigExpirySeconds, expirySeconds)
		err := testK8sCmdService().RunKubernetesKubeconfigShow(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		kubeconfig := []byte(`i'm some yaml`)
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		tm.kubernetes.EXPECT().GetKubeConfig(testCluster.ID).Return(kubeconfig, nil)
		config.Args = append(config.Args, testCluster.Name)
		err := testK8sCmdService().RunKubernetesKubeconfigShow(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		err := testK8sCmdService().RunKubernetesClusterList(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesCreate(t *testing.T) {
	testNodePool := testNodePool

	testNodePool.Labels = map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	inputLabels := make([]string, 0, len(testNodePool.Labels))
	for key, val := range testNodePool.Labels {
		inputLabels = append(inputLabels, fmt.Sprintf("%s=%s", key, val))
	}
	sort.Strings(inputLabels)

	testNodePool.Taints = []godo.Taint{
		{
			Key:    "key1",
			Value:  "value1",
			Effect: "NoSchedule",
		},
		{
			Key:    "key2",
			Value:  "value2",
			Effect: "NoExecute",
		},
	}
	inputTaints := make([]string, 0, len(testNodePool.Taints))
	for _, taint := range testNodePool.Taints {
		inputTaints = append(inputTaints, taint.String())
	}

	testNodePool.AutoScale = true
	testNodePool.MinNodes = 1
	testNodePool.MaxNodes = 10

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesClusterCreateRequest{
			Name:        testCluster.Name,
			RegionSlug:  testCluster.RegionSlug,
			VersionSlug: testCluster.VersionSlug,
			Tags:        testCluster.Tags,
			NodePools: []*godo.KubernetesNodePoolCreateRequest{
				{
					Name:      testNodePool.Name + "1",
					Size:      testNodePool.Size,
					Count:     testNodePool.Count,
					Tags:      testNodePool.Tags,
					Labels:    testNodePool.Labels,
					Taints:    testNodePool.Taints,
					AutoScale: testNodePool.AutoScale,
					MinNodes:  testNodePool.MinNodes,
					MaxNodes:  testNodePool.MaxNodes,
				},
				{
					Name:   testNodePool.Name + "2",
					Size:   testNodePool.Size,
					Count:  testNodePool.Count,
					Tags:   testNodePool.Tags,
					Labels: map[string]string{},
					Taints: []godo.Taint{},
				},
			},
			MaintenancePolicy: &godo.KubernetesMaintenancePolicy{
				StartTime: "00:00",
				Day:       godo.KubernetesMaintenanceDayAny,
			},
			AutoUpgrade: true,
			HA:          true,
		}
		tm.kubernetes.EXPECT().Create(&r).Return(&testCluster, nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testCluster.RegionSlug)
		config.Doit.Set(config.NS, doctl.ArgClusterVersionSlug, testCluster.VersionSlug)
		config.Doit.Set(config.NS, doctl.ArgTag, testCluster.Tags)
		config.Doit.Set(config.NS, doctl.ArgMaintenanceWindow, "any=00:00")
		config.Doit.Set(config.NS, doctl.ArgClusterNodePool, []string{
			fmt.Sprintf("name=%s;size=%s;count=%d;tag=%s;tag=%s;label=%s;label=%s;taint=%s;taint=%s;auto-scale=%v;min-nodes=%d;max-nodes=%d",
				testNodePool.Name+"1", testNodePool.Size, testNodePool.Count, testNodePool.Tags[0], testNodePool.Tags[1],
				inputLabels[0], inputLabels[1], inputTaints[0], inputTaints[1], testNodePool.AutoScale, testNodePool.MinNodes, testNodePool.MaxNodes,
			),
			fmt.Sprintf("name=%s;size=%s;count=%d;tag=%s;tag=%s",
				testNodePool.Name+"2", testNodePool.Size, testNodePool.Count, testNodePool.Tags[0], testNodePool.Tags[1],
			),
		})
		config.Doit.Set(config.NS, doctl.ArgAutoUpgrade, testCluster.AutoUpgrade)
		config.Doit.Set(config.NS, doctl.ArgHA, testCluster.HA)

		// Test with no vpc-uuid specified
		err := testK8sCmdService().RunKubernetesClusterCreate("c-8", 3)(config)
		assert.NoError(t, err)

		// Test with vpc-uuid specified
		config.Doit.Set(config.NS, doctl.ArgClusterVPCUUID, "vpc-uuid")
		r.VPCUUID = "vpc-uuid"
		testCluster.VPCUUID = "vpc-uuid"
		tm.kubernetes.EXPECT().Create(&r).Return(&testCluster, nil)
		err = testK8sCmdService().RunKubernetesClusterCreate("c-8", 3)(config)
		assert.NoError(t, err)

		// Test with 1-clicks specified
		config.Doit.Set(config.NS, doctl.ArgOneClicks, []string{"slug1", "slug2"})
		tm.kubernetes.EXPECT().Create(&r).Return(&testCluster, nil)
		tm.oneClick.EXPECT().InstallKubernetes(testCluster.ID, []string{"slug1", "slug2"})
		err = testK8sCmdService().RunKubernetesClusterCreate("c-8", 3)(config)
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
			AutoUpgrade: boolPtr(false),
		}
		tm.kubernetes.EXPECT().Update(testCluster.ID, &r).Return(&testCluster, nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgClusterName, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgTag, testCluster.Tags)
		config.Doit.Set(config.NS, doctl.ArgMaintenanceWindow, "any=00:00")
		config.Doit.Set(config.NS, doctl.ArgAutoUpgrade, false)

		err := testK8sCmdService().RunKubernetesClusterUpdate(config)
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
			AutoUpgrade: boolPtr(false),
		}
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		tm.kubernetes.EXPECT().Update(testCluster.ID, &r).Return(&testCluster, nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgClusterName, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgTag, testCluster.Tags)
		config.Doit.Set(config.NS, doctl.ArgMaintenanceWindow, "any=00:00")
		config.Doit.Set(config.NS, doctl.ArgAutoUpgrade, false)

		err := testK8sCmdService().RunKubernetesClusterUpdate(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesUpgrade(t *testing.T) {
	testUpgradeVersion := testClusterUpgrades[0].Slug

	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().Upgrade(testCluster.ID, testUpgradeVersion).Return(nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgVersion, testUpgradeVersion)

		err := testK8sCmdService().RunKubernetesClusterUpgrade(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		tm.kubernetes.EXPECT().Upgrade(testCluster.ID, testUpgradeVersion).Return(nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgVersion, testUpgradeVersion)

		err := testK8sCmdService().RunKubernetesClusterUpgrade(config)
		assert.NoError(t, err)
	})

	// using "latest" version
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().Get(testCluster.ID).Return(&testCluster, nil)
		tm.kubernetes.EXPECT().GetUpgrades(testCluster.ID).Return(testClusterUpgrades, nil)
		tm.kubernetes.EXPECT().Upgrade(testCluster.ID, testUpgradeVersion).Return(nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgVersion, defaultKubernetesLatestVersion)

		err := testK8sCmdService().RunKubernetesClusterUpgrade(config)
		assert.NoError(t, err)
	})

	// without version flag set (defaults to latest)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().Get(testCluster.ID).Return(&testCluster, nil)
		tm.kubernetes.EXPECT().GetUpgrades(testCluster.ID).Return(testClusterUpgrades, nil)
		tm.kubernetes.EXPECT().Upgrade(testCluster.ID, testUpgradeVersion).Return(nil)

		config.Args = append(config.Args, testCluster.ID)

		err := testK8sCmdService().RunKubernetesClusterUpgrade(config)
		assert.NoError(t, err)
	})

	// for cluster that is up-to-date
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().Get(testCluster.ID).Return(&testCluster, nil)
		tm.kubernetes.EXPECT().GetUpgrades(testCluster.ID).Return(nil, nil)

		config.Args = append(config.Args, testCluster.ID)

		err := testK8sCmdService().RunKubernetesClusterUpgrade(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// shouldn't call `DeleteNodePool` so we don't set any expectations
		config.Doit.Set(config.NS, doctl.ArgForce, "false")
		config.Args = append(config.Args, testCluster.ID)

		err := testK8sCmdService().RunKubernetesClusterDelete(config)
		assert.Error(t, err, "should have been challenged before deletion")
	})
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().Delete(testCluster.ID).Return(nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := testK8sCmdService().RunKubernetesClusterDelete(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		tm.kubernetes.EXPECT().Delete(testCluster.ID).Return(nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := testK8sCmdService().RunKubernetesClusterDelete(config)
		assert.NoError(t, err)
	})
	// multiple clusters
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		id2 := "ce69d914-ae08-4c91-8a4b-383f58b47e6f"

		tm.kubernetes.EXPECT().Delete(testCluster.ID).Return(nil)
		tm.kubernetes.EXPECT().Delete(id2).Return(nil)

		config.Args = append(config.Args, testCluster.ID, id2)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := testK8sCmdService().RunKubernetesClusterDelete(config)
		assert.NoError(t, err)
	})
	// dangerous delete
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		id2 := "ce69d914-ae08-4c91-8a4b-383f58b47e6f"

		tm.kubernetes.EXPECT().DeleteDangerous(testCluster.ID).Return(nil)
		tm.kubernetes.EXPECT().DeleteDangerous(id2).Return(nil)

		config.Args = append(config.Args, testCluster.ID, id2)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")
		config.Doit.Set(config.NS, doctl.ArgDangerous, "true")

		err := testK8sCmdService().RunKubernetesClusterDelete(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesDeleteSelective(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// shouldn't call `DeleteNodePool` so we don't set any expectations
		config.Doit.Set(config.NS, doctl.ArgForce, "false")
		config.Args = append(config.Args, testCluster.ID)

		err := testK8sCmdService().RunKubernetesClusterDeleteSelective(config)
		assert.Error(t, err, "should have been challenged before deletion")
	})
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := &godo.KubernetesClusterDeleteSelectiveRequest{
			Volumes:         []string{volumeID.String()},
			VolumeSnapshots: []string{snapshotID.String()},
			LoadBalancers:   []string{lbID.String()},
		}
		tm.kubernetes.EXPECT().Get(testCluster.ID).Return(&testCluster, nil)
		tm.kubernetes.EXPECT().DeleteSelective(testCluster.ID, r).Return(nil)

		config.Args = append(config.Args, testCluster.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")
		config.Doit.Set(config.NS, doctl.ArgVolumeList, []string{volumeID.String()})
		config.Doit.Set(config.NS, doctl.ArgVolumeSnapshotList, []string{snapshotID.String()})
		config.Doit.Set(config.NS, doctl.ArgLoadBalancerList, []string{lbID.String()})

		err := testK8sCmdService().RunKubernetesClusterDeleteSelective(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := &godo.KubernetesClusterDeleteSelectiveRequest{
			Volumes:         []string{volumeID.String()},
			VolumeSnapshots: []string{snapshotID.String()},
			LoadBalancers:   []string{lbID.String()},
		}
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		tm.kubernetes.EXPECT().Get(testCluster.ID).Return(&testCluster, nil)
		tm.volumes.EXPECT().List().Return(testVolumes, nil)
		tm.snapshots.EXPECT().ListVolume().Return(testSnapshots, nil)
		tm.loadBalancers.EXPECT().List().Return(testLoadBalancers, nil)
		tm.kubernetes.EXPECT().DeleteSelective(testCluster.ID, r).Return(nil)

		config.Args = append(config.Args, testCluster.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")
		config.Doit.Set(config.NS, doctl.ArgVolumeList, []string{"vol-1"})
		config.Doit.Set(config.NS, doctl.ArgVolumeSnapshotList, []string{"snap-1"})
		config.Doit.Set(config.NS, doctl.ArgLoadBalancerList, []string{"lb-1"})

		err := testK8sCmdService().RunKubernetesClusterDeleteSelective(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesListAssociatedResources(t *testing.T) {
	// by ID
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().ListAssociatedResourcesForDeletion(testCluster.ID).Return(&testAssociatedResources, nil)
		config.Args = append(config.Args, testCluster.ID)
		err := testK8sCmdService().RunKubernetesClusterListAssociatedResources(config)
		assert.NoError(t, err)
	})

	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		tm.kubernetes.EXPECT().ListAssociatedResourcesForDeletion(testCluster.ID).Return(&testAssociatedResources, nil)
		config.Args = append(config.Args, testCluster.Name)
		err := testK8sCmdService().RunKubernetesClusterListAssociatedResources(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		name := "not a cluster"
		// it'll see that no UUID is given and do a List call to find the cluster
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		config.Args = append(config.Args, name)
		err := testK8sCmdService().RunKubernetesClusterListAssociatedResources(config)
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
		tm.kubernetes.EXPECT().List().Return(clustersWithDups, nil)
		config.Args = append(config.Args, name)
		err := testK8sCmdService().RunKubernetesClusterListAssociatedResources(config)
		assert.EqualError(t, err, errAmbiguousClusterName(name, []string{testCluster.ID, testClusterWithSameName.ID}).Error())
	})
}

func TestKubernetesNodePool_Get(t *testing.T) {
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().GetNodePool(testCluster.ID, testNodePool.ID).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		err := testK8sCmdService().RunKubernetesNodePoolGet(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().ListNodePools(testCluster.ID).Return(testNodePools, nil)

		// cluster ID but pool name
		config.Args = append(config.Args, testCluster.ID, testNodePool.Name)

		err := testK8sCmdService().RunKubernetesNodePoolGet(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		tm.kubernetes.EXPECT().GetNodePool(testCluster.ID, testNodePool.ID).Return(&testNodePool, nil)

		// cluster name and pool ID
		config.Args = append(config.Args, testCluster.Name, testNodePool.ID)

		err := testK8sCmdService().RunKubernetesNodePoolGet(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		tm.kubernetes.EXPECT().ListNodePools(testCluster.ID).Return(testNodePools, nil)

		// cluster name and pool name
		config.Args = append(config.Args, testCluster.Name, testNodePool.Name)

		err := testK8sCmdService().RunKubernetesNodePoolGet(config)
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
		tm.kubernetes.EXPECT().ListNodePools(testCluster.ID).Return(nodePoolsWithDups, nil)
		config.Args = append(config.Args, testCluster.ID, name)
		err := testK8sCmdService().RunKubernetesNodePoolGet(config)
		assert.EqualError(t, err, errAmbiguousPoolName(name, []string{testNodePool.ID, testNodePoolWithSameName.ID}).Error())
	})
}

func TestKubernetesNodePool_List(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().ListNodePools(testCluster.ID).Return(testNodePools, nil)

		config.Args = append(config.Args, testCluster.ID)

		err := testK8sCmdService().RunKubernetesNodePoolList(config)
		assert.NoError(t, err)
	})
	// by name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		tm.kubernetes.EXPECT().ListNodePools(testCluster.ID).Return(testNodePools, nil)

		config.Args = append(config.Args, testCluster.Name)

		err := testK8sCmdService().RunKubernetesNodePoolList(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesNodePool_Create(t *testing.T) {
	testNodePool := testNodePool
	testNodePool.Labels = map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	testNodePool.Taints = []godo.Taint{
		{
			Key:    "key1",
			Value:  "value1",
			Effect: "NoSchedule",
		},
		{
			Key:    "key2",
			Value:  "value2",
			Effect: "NoExecute",
		},
	}
	testNodePool.AutoScale = true
	testNodePool.MinNodes = 1
	testNodePool.MaxNodes = 10

	// by cluster ID
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolCreateRequest{
			Name:      testNodePool.Name,
			Size:      testNodePool.Size,
			Count:     testNodePool.Count,
			Tags:      testNodePool.Tags,
			Labels:    testNodePool.Labels,
			Taints:    testNodePool.Taints,
			AutoScale: testNodePool.AutoScale,
			MinNodes:  testNodePool.MinNodes,
			MaxNodes:  testNodePool.MaxNodes,
		}
		tm.kubernetes.EXPECT().CreateNodePool(testCluster.ID, &r).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.ID)

		config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, testNodePool.Size)
		config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
		config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)
		config.Doit.Set(config.NS, doctl.ArgKubernetesLabel, testNodePool.Labels)
		config.Doit.Set(config.NS, doctl.ArgKubernetesTaint, taintsToSlice(testNodePool.Taints))
		config.Doit.Set(config.NS, doctl.ArgNodePoolAutoScale, testNodePool.AutoScale)
		config.Doit.Set(config.NS, doctl.ArgNodePoolMinNodes, testNodePool.MinNodes)
		config.Doit.Set(config.NS, doctl.ArgNodePoolMaxNodes, testNodePool.MaxNodes)

		err := testK8sCmdService().RunKubernetesNodePoolCreate(config)
		assert.NoError(t, err)
	})
	// by cluster name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolCreateRequest{
			Name:      testNodePool.Name,
			Size:      testNodePool.Size,
			Count:     testNodePool.Count,
			Tags:      testNodePool.Tags,
			Labels:    testNodePool.Labels,
			Taints:    testNodePool.Taints,
			AutoScale: testNodePool.AutoScale,
			MinNodes:  testNodePool.MinNodes,
			MaxNodes:  testNodePool.MaxNodes,
		}
		tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
		tm.kubernetes.EXPECT().CreateNodePool(testCluster.ID, &r).Return(&testNodePool, nil)

		config.Args = append(config.Args, testCluster.Name)

		config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, testNodePool.Size)
		config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
		config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)
		config.Doit.Set(config.NS, doctl.ArgKubernetesLabel, testNodePool.Labels)
		config.Doit.Set(config.NS, doctl.ArgKubernetesTaint, taintsToSlice(testNodePool.Taints))
		config.Doit.Set(config.NS, doctl.ArgNodePoolAutoScale, testNodePool.AutoScale)
		config.Doit.Set(config.NS, doctl.ArgNodePoolMinNodes, testNodePool.MinNodes)
		config.Doit.Set(config.NS, doctl.ArgNodePoolMaxNodes, testNodePool.MaxNodes)

		err := testK8sCmdService().RunKubernetesNodePoolCreate(config)
		assert.NoError(t, err)
	})
}

func createTestNodePoolUpdateRequest() godo.KubernetesNodePoolUpdateRequest {
	return godo.KubernetesNodePoolUpdateRequest{
		Name:   testNodePool.Name,
		Count:  &testNodePool.Count,
		Tags:   testNodePool.Tags,
		Labels: map[string]string{},
		Taints: nil,
	}
}

func TestKubernetesNodePool_Update(t *testing.T) {
	t.Run("by cluster ID", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			r := createTestNodePoolUpdateRequest()

			tm.kubernetes.EXPECT().UpdateNodePool(testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

			config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

			config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
			config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
			config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)

			err := testK8sCmdService().RunKubernetesNodePoolUpdate(config)
			assert.NoError(t, err)
		})
	})
	t.Run("by cluster name and pool ID", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			r := createTestNodePoolUpdateRequest()

			tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
			tm.kubernetes.EXPECT().UpdateNodePool(testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

			config.Args = append(config.Args, testCluster.Name, testNodePool.ID)

			config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
			config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
			config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)

			err := testK8sCmdService().RunKubernetesNodePoolUpdate(config)
			assert.NoError(t, err)
		})
	})

	t.Run("by cluster ID and pool name", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			r := createTestNodePoolUpdateRequest()

			tm.kubernetes.EXPECT().ListNodePools(testCluster.ID).Return(testNodePools, nil)
			tm.kubernetes.EXPECT().UpdateNodePool(testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

			config.Args = append(config.Args, testCluster.ID, testNodePool.Name)

			config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
			config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
			config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)

			err := testK8sCmdService().RunKubernetesNodePoolUpdate(config)
			assert.NoError(t, err)
		})
	})

	t.Run("by cluster name and pool name", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			r := createTestNodePoolUpdateRequest()

			tm.kubernetes.EXPECT().List().Return(testClusterList, nil)
			tm.kubernetes.EXPECT().ListNodePools(testCluster.ID).Return(testNodePools, nil)
			tm.kubernetes.EXPECT().UpdateNodePool(testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

			config.Args = append(config.Args, testCluster.Name, testNodePool.Name)

			config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
			config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
			config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)

			err := testK8sCmdService().RunKubernetesNodePoolUpdate(config)
			assert.NoError(t, err)
		})
	})

	t.Run("with autoscale config", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			testNodePool := testNodePool
			testNodePool.AutoScale = true
			testNodePool.MinNodes = 1
			testNodePool.MaxNodes = 10

			r := createTestNodePoolUpdateRequest()
			r.AutoScale = &testNodePool.AutoScale
			r.MinNodes = &testNodePool.MinNodes
			r.MaxNodes = &testNodePool.MaxNodes

			tm.kubernetes.EXPECT().UpdateNodePool(testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

			config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

			config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
			config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
			config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)
			config.Doit.Set(config.NS, doctl.ArgNodePoolAutoScale, testNodePool.AutoScale)
			config.Doit.Set(config.NS, doctl.ArgNodePoolMinNodes, testNodePool.MinNodes)
			config.Doit.Set(config.NS, doctl.ArgNodePoolMaxNodes, testNodePool.MaxNodes)

			err := testK8sCmdService().RunKubernetesNodePoolUpdate(config)
			assert.NoError(t, err)
		})
	})
	t.Run("with labels", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			testNodePool := testNodePool
			testNodePool.Labels = map[string]string{
				"key1": "value1",
				"key2": "value2",
			}

			r := createTestNodePoolUpdateRequest()
			r.Labels = testNodePool.Labels

			tm.kubernetes.EXPECT().UpdateNodePool(testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

			config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

			config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
			config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
			config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)
			config.Doit.Set(config.NS, doctl.ArgKubernetesLabel, testNodePool.Labels)

			err := testK8sCmdService().RunKubernetesNodePoolUpdate(config)
			assert.NoError(t, err)
		})
	})
	t.Run("with taints", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			testNodePool := testNodePool
			testNodePool.Taints = []godo.Taint{
				{
					Key:    "key1",
					Value:  "value1",
					Effect: "NoSchedule",
				},
				{
					Key:    "key2",
					Value:  "value2",
					Effect: "NoExecute",
				},
			}

			r := createTestNodePoolUpdateRequest()
			r.Taints = &testNodePool.Taints

			tm.kubernetes.EXPECT().UpdateNodePool(testCluster.ID, testNodePool.ID, &r).Return(&testNodePool, nil)

			config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

			config.Doit.Set(config.NS, doctl.ArgNodePoolName, testNodePool.Name)
			config.Doit.Set(config.NS, doctl.ArgNodePoolCount, testNodePool.Count)
			config.Doit.Set(config.NS, doctl.ArgTag, testNodePool.Tags)
			config.Doit.Set(config.NS, doctl.ArgKubernetesTaint, taintsToSlice(testNodePool.Taints))

			err := testK8sCmdService().RunKubernetesNodePoolUpdate(config)
			assert.NoError(t, err)
		})
	})
}

func TestKubernetesNodePool_Recycle(t *testing.T) {
	// by node IDs
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolRecycleNodesRequest{
			Nodes: []string{testNode.ID},
		}

		tm.kubernetes.EXPECT().RecycleNodePoolNodes(testCluster.ID, testNodePool.ID, &r).Return(nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		config.Doit.Set(config.NS, doctl.ArgNodePoolNodeIDs, testNode.ID)

		err := testK8sCmdService().RunKubernetesNodePoolRecycle(config)
		assert.NoError(t, err)
	})
	// by node names
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.KubernetesNodePoolRecycleNodesRequest{
			Nodes: []string{testNode.ID},
		}

		tm.kubernetes.EXPECT().GetNodePool(testCluster.ID, testNodePool.ID).Return(&testNodePool, nil)
		tm.kubernetes.EXPECT().RecycleNodePoolNodes(testCluster.ID, testNodePool.ID, &r).Return(nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		config.Doit.Set(config.NS, doctl.ArgNodePoolNodeIDs, testNode.Name)

		err := testK8sCmdService().RunKubernetesNodePoolRecycle(config)
		assert.NoError(t, err)
	})
}

func TestKubernetesNodePool_Delete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// shouldn't call `DeleteNodePool` so we don't set any expectations
		config.Doit.Set(config.NS, doctl.ArgForce, "false")
		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)

		err := testK8sCmdService().RunKubernetesNodePoolDelete(config)
		assert.Error(t, err, "should have been challenged before deletion")
	})
	// by id
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().DeleteNodePool(testCluster.ID, testNodePool.ID).Return(nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := testK8sCmdService().RunKubernetesNodePoolDelete(config)
		assert.NoError(t, err)
	})
	// by cluster ID and pool name
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.kubernetes.EXPECT().ListNodePools(testCluster.ID).Return(testNodePools, nil)
		tm.kubernetes.EXPECT().DeleteNodePool(testCluster.ID, testNodePool.ID).Return(nil)

		config.Args = append(config.Args, testCluster.ID, testNodePool.Name)
		config.Doit.Set(config.NS, doctl.ArgForce, "true")

		err := testK8sCmdService().RunKubernetesNodePoolDelete(config)
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
		tm.kubernetes.EXPECT().GetVersions().Return(testVersions, nil)

		err := testK8sCmdService().RunKubeOptionsListVersion(config)
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

func TestKubernetes_DOCRIntegration(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := &godo.KubernetesClusterRegistryRequest{ClusterUUIDs: []string{testCluster.ID}}
		tm.kubernetes.EXPECT().AddRegistry(r).Return(nil)
		// we use testCluster.ID because that represents the uuid of the cluster
		config.Args = append(config.Args, testCluster.ID)

		err := testK8sCmdService().RunKubernetesRegistryAdd(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := &godo.KubernetesClusterRegistryRequest{ClusterUUIDs: []string{testCluster.ID}}
		tm.kubernetes.EXPECT().RemoveRegistry(r).Return(nil)
		// we use testCluster.ID because that represents the uuid of the cluster
		config.Args = append(config.Args, testCluster.ID)

		err := testK8sCmdService().RunKubernetesRegistryRemove(config)
		assert.NoError(t, err)
	})
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

func TestLatestVersionForUpgrade(t *testing.T) {
	tests := []struct {
		name            string
		clusterVersion  string
		upgradeVersions []do.KubernetesVersion
		want            string
	}{
		{
			name:           "only one patch version",
			clusterVersion: "1.12.1-do.1",
			upgradeVersions: []do.KubernetesVersion{
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.1-do.1", KubernetesVersion: "1.12.1"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.1-do.2", KubernetesVersion: "1.12.1"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.1-do.3", KubernetesVersion: "1.12.1"}},
			},
			want: "1.12.1-do.3",
		},
		{
			name:           "only one minor version",
			clusterVersion: "1.12.1-do.1",
			upgradeVersions: []do.KubernetesVersion{
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.3-do.1", KubernetesVersion: "1.12.3"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.1-do.1", KubernetesVersion: "1.12.1"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.2-do.1", KubernetesVersion: "1.12.2"}},
			},
			want: "1.12.3-do.1",
		},
		{
			name:           "multiple minor versions",
			clusterVersion: "1.12.1-do.1",
			upgradeVersions: []do.KubernetesVersion{
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.14.3-do.1", KubernetesVersion: "1.14.3"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.13.2-do.1", KubernetesVersion: "1.13.2"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.1-do.3", KubernetesVersion: "1.12.1"}},
			},
			want: "1.12.1-do.3",
		},
		{
			name:           "multiple major versions",
			clusterVersion: "1.12.1-do.1",
			upgradeVersions: []do.KubernetesVersion{
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.12.3-do.3", KubernetesVersion: "1.12.3"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "2.13.2-do.1", KubernetesVersion: "2.13.2"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.15.1-do.1", KubernetesVersion: "1.15.1"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "2.14.3-do.1", KubernetesVersion: "2.14.3"}},
			},
			want: "1.12.3-do.3",
		},
		{
			name:           "no patch upgrades available",
			clusterVersion: "1.12.1-do.1",
			upgradeVersions: []do.KubernetesVersion{
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "2.13.2-do.1", KubernetesVersion: "2.13.2"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "1.15.1-do.1", KubernetesVersion: "1.15.1"}},
				{KubernetesVersion: &godo.KubernetesVersion{Slug: "2.14.3-do.1", KubernetesVersion: "2.14.3"}},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, found, err := latestVersionForUpgrade(tt.clusterVersion, tt.upgradeVersions)
			require.NoError(t, err)
			if tt.want == "" {
				require.False(t, found)
			} else {
				require.True(t, found)
				require.Equal(t, tt.want, slug)
			}
		})
	}
}

func Test_looksLikeUUID(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want bool
	}{
		{
			name: "UUIDv1",
			arg:  "a235f190-01a4-11ea-af17-4989e8155574",
			want: true,
		},
		{
			name: "UUIDv4",
			arg:  "21464f77-42fc-4a32-9aa4-a101843e94c0",
			want: true,
		},
		{
			name: "my-cluster",
			arg:  "my-cluster",
			want: false,
		},
		{
			name: "f8291060b73f4fa7b60586fe51a1d862",
			arg:  "f8291060b73f4fa7b60586fe51a1d862",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeUUID(tt.arg); got != tt.want {
				t.Errorf("looksLikeUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestK8sOneClickListNoType(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oneClick.EXPECT().List("kubernetes").Return(testK8sOneClickList, nil)
		err := RunKubernetesOneClickList(config)
		assert.NoError(t, err)
	})
}

func TestParseTaints(t *testing.T) {
	const colonSepErrSubstring = "does not have a single colon separator"
	const TooManyEqualSignsErrSubstring = "must not consist of more than one equal sign"

	tests := []struct {
		name             string
		taints           []string
		wantErrSubstring string
		wantTaints       []godo.Taint
	}{
		{
			name:             "empty taint",
			taints:           []string{""},
			wantErrSubstring: colonSepErrSubstring,
		},
		{
			name:             "no colon separator",
			taints:           []string{"key=value"},
			wantErrSubstring: colonSepErrSubstring,
		},
		{
			name:             "too many equal signs",
			taints:           []string{"key=value=foo:NoSchedule"},
			wantErrSubstring: TooManyEqualSignsErrSubstring,
		},
		{
			name:   "taint with value",
			taints: []string{"key=value:NoSchedule"},
			wantTaints: []godo.Taint{
				{
					Key:    "key",
					Value:  "value",
					Effect: "NoSchedule",
				},
			},
		},
		{
			name:   "taint without value",
			taints: []string{"key:NoSchedule"},
			wantTaints: []godo.Taint{
				{
					Key:    "key",
					Value:  "",
					Effect: "NoSchedule",
				},
			},
		},
		{
			name:   "multiple taints",
			taints: []string{"key:NoSchedule", "key=value:NoExecute"},
			wantTaints: []godo.Taint{
				{
					Key:    "key",
					Value:  "",
					Effect: "NoSchedule",
				},
				{
					Key:    "key",
					Value:  "value",
					Effect: "NoExecute",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taints, err := parseTaints(tt.taints)
			if tt.wantErrSubstring != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSubstring)
			} else {
				assert.Equal(t, tt.wantTaints, taints)
			}
		})
	}
}

func taintsToSlice(taints []godo.Taint) []string {
	res := make([]string, 0, len(taints))
	for _, taint := range taints {
		res = append(res, taint.String())
	}
	return res
}
