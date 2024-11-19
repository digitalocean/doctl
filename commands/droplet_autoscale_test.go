package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testAutoscalePools = []*godo.DropletAutoscalePool{
		{
			ID:   "51154959-e07b-4093-98fb-828590ecc76d",
			Name: "test-droplet-autoscale-pool-01",
			Config: &godo.DropletAutoscaleConfiguration{
				TargetNumberInstances: 3,
			},
			DropletTemplate: &godo.DropletAutoscaleResourceTemplate{
				Size:             "s-1vcpu-512mb-10gb",
				Region:           "s2r1",
				Image:            "547864",
				Tags:             []string{"test-pool-01"},
				SSHKeys:          []string{"key-1", "key-2"},
				VpcUUID:          "05790d02-c7e0-47d6-a917-5b4cf68cf5b7",
				WithDropletAgent: true,
				UserData:         "\n#cloud-config\nruncmd:\n- apt-get update\n- apt-get install -y stress-ng\n",
			},
		},
		{
			ID:   "4d99fb28-b33d-4791-aff5-bf30f8f4f917",
			Name: "test-droplet-autoscale-pool-02",
			Config: &godo.DropletAutoscaleConfiguration{
				TargetNumberInstances: 3,
			},
			DropletTemplate: &godo.DropletAutoscaleResourceTemplate{
				Size:             "s-1vcpu-512mb-10gb",
				Region:           "s2r1",
				Image:            "547864",
				Tags:             []string{"test-pool-02"},
				SSHKeys:          []string{"key-1", "key-2"},
				VpcUUID:          "05790d02-c7e0-47d6-a917-5b4cf68cf5b7",
				WithDropletAgent: true,
				UserData:         "\n#cloud-config\nruncmd:\n- apt-get update\n- apt-get install -y stress-ng\n",
			},
		},
	}
	testAutoscaleMembers = []*godo.DropletAutoscaleResource{
		{
			DropletID:    1,
			HealthStatus: "healthy",
			Status:       "active",
		},
		{
			DropletID:    2,
			HealthStatus: "healthy",
			Status:       "active",
		},
	}
	testAutoscaleHistory = []*godo.DropletAutoscaleHistoryEvent{
		{
			HistoryEventID:       "c4686a63-4996-484d-b269-b329b7e97051",
			CurrentInstanceCount: 0,
			DesiredInstanceCount: 3,
			Reason:               "configuration update",
			Status:               "success",
		},
		{
			HistoryEventID:       "cb676933-73ba-43da-9c43-bb286a5af545",
			CurrentInstanceCount: 3,
			DesiredInstanceCount: 3,
			Reason:               "scale up",
			Status:               "success",
		},
	}
)

func TestDropletAutoscaleCommand(t *testing.T) {
	cmd := DropletAutoscale()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "update", "get", "list", "list-members", "list-history", "delete", "delete-dangerous")
}

func TestDropletAutoscaleCreate(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		createReq := godo.DropletAutoscalePoolRequest{
			Name: "test-droplet-autoscale-pool-01",
			Config: &godo.DropletAutoscaleConfiguration{
				TargetNumberInstances: 3,
			},
			DropletTemplate: &godo.DropletAutoscaleResourceTemplate{
				Size:             "s-1vcpu-512mb-10gb",
				Region:           "s2r1",
				Image:            "547864",
				Tags:             []string{"test-pool-01"},
				SSHKeys:          []string{"key-1", "key-2"},
				VpcUUID:          "05790d02-c7e0-47d6-a917-5b4cf68cf5b7",
				WithDropletAgent: true,
				UserData:         "\n#cloud-config\nruncmd:\n- apt-get update\n- apt-get install -y stress-ng\n",
			},
		}

		tm.dropletAutoscale.EXPECT().Create(&createReq).Return(testAutoscalePools[0], nil)

		c.Doit.Set(c.NS, doctl.ArgAutoscaleName, "test-droplet-autoscale-pool-01")
		c.Doit.Set(c.NS, doctl.ArgAutoscaleTargetInstances, "3")
		c.Doit.Set(c.NS, doctl.ArgSizeSlug, "s-1vcpu-512mb-10gb")
		c.Doit.Set(c.NS, doctl.ArgRegionSlug, "s2r1")
		c.Doit.Set(c.NS, doctl.ArgImage, "547864")
		c.Doit.Set(c.NS, doctl.ArgTag, "test-pool-01")
		c.Doit.Set(c.NS, doctl.ArgSSHKeys, []string{"key-1", "key-2"})
		c.Doit.Set(c.NS, doctl.ArgVPCUUID, "05790d02-c7e0-47d6-a917-5b4cf68cf5b7")
		c.Doit.Set(c.NS, doctl.ArgDropletAgent, "true")
		c.Doit.Set(c.NS, doctl.ArgUserData, "\n#cloud-config\nruncmd:\n- apt-get update\n- apt-get install -y stress-ng\n")

		err := RunDropletAutoscaleCreate(c)
		assert.NoError(t, err)
	})
}

func TestDropletAutoscaleUpdate(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		poolID := "51154959-e07b-4093-98fb-828590ecc76d"
		updateReq := godo.DropletAutoscalePoolRequest{
			Name: "test-droplet-autoscale-pool-01",
			Config: &godo.DropletAutoscaleConfiguration{
				TargetNumberInstances: 3,
			},
			DropletTemplate: &godo.DropletAutoscaleResourceTemplate{
				Size:             "s-1vcpu-512mb-10gb",
				Region:           "s2r1",
				Image:            "547864",
				Tags:             []string{"test-pool-01"},
				SSHKeys:          []string{"key-1", "key-2"},
				VpcUUID:          "05790d02-c7e0-47d6-a917-5b4cf68cf5b7",
				WithDropletAgent: true,
				UserData:         "\n#cloud-config\nruncmd:\n- apt-get update\n- apt-get install -y stress-ng\n",
			},
		}

		tm.dropletAutoscale.EXPECT().Update(poolID, &updateReq).Return(testAutoscalePools[0], nil)
		c.Args = append(c.Args, poolID)

		c.Doit.Set(c.NS, doctl.ArgAutoscaleName, "test-droplet-autoscale-pool-01")
		c.Doit.Set(c.NS, doctl.ArgAutoscaleTargetInstances, "3")
		c.Doit.Set(c.NS, doctl.ArgSizeSlug, "s-1vcpu-512mb-10gb")
		c.Doit.Set(c.NS, doctl.ArgRegionSlug, "s2r1")
		c.Doit.Set(c.NS, doctl.ArgImage, "547864")
		c.Doit.Set(c.NS, doctl.ArgTag, "test-pool-01")
		c.Doit.Set(c.NS, doctl.ArgSSHKeys, []string{"key-1", "key-2"})
		c.Doit.Set(c.NS, doctl.ArgVPCUUID, "05790d02-c7e0-47d6-a917-5b4cf68cf5b7")
		c.Doit.Set(c.NS, doctl.ArgDropletAgent, "true")
		c.Doit.Set(c.NS, doctl.ArgUserData, "\n#cloud-config\nruncmd:\n- apt-get update\n- apt-get install -y stress-ng\n")

		err := RunDropletAutoscaleUpdate(c)
		assert.NoError(t, err)
	})
}

func TestDropletAutoscaleGet(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		poolID := "51154959-e07b-4093-98fb-828590ecc76d"
		tm.dropletAutoscale.EXPECT().Get(poolID).Return(testAutoscalePools[0], nil)
		c.Args = append(c.Args, poolID)

		err := RunDropletAutoscaleGet(c)
		assert.NoError(t, err)
	})
}

func TestDropletAutoscaleList(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		tm.dropletAutoscale.EXPECT().List().Return(testAutoscalePools, nil)

		err := RunDropletAutoscaleList(c)
		assert.NoError(t, err)
	})
}

func TestDropletAutoscaleListMembers(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		poolID := "51154959-e07b-4093-98fb-828590ecc76d"
		tm.dropletAutoscale.EXPECT().ListMembers(poolID).Return(testAutoscaleMembers, nil)
		c.Args = append(c.Args, poolID)

		err := RunDropletAutoscaleListMembers(c)
		assert.NoError(t, err)
	})
}

func TestDropletAutoscaleListHistory(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		poolID := "51154959-e07b-4093-98fb-828590ecc76d"
		tm.dropletAutoscale.EXPECT().ListHistory(poolID).Return(testAutoscaleHistory, nil)
		c.Args = append(c.Args, poolID)

		err := RunDropletAutoscaleListHistory(c)
		assert.NoError(t, err)
	})
}

func TestDropletAutoscaleDelete(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		poolID := "51154959-e07b-4093-98fb-828590ecc76d"
		tm.dropletAutoscale.EXPECT().Delete(poolID).Return(nil)
		c.Args = append(c.Args, poolID)
		c.Doit.Set(c.NS, doctl.ArgForce, "true")

		err := RunDropletAutoscaleDelete(c)
		assert.NoError(t, err)
	})
}

func TestDropletAutoscaleDeleteDangerous(t *testing.T) {
	withTestClient(t, func(c *CmdConfig, tm *tcMocks) {
		poolID := "51154959-e07b-4093-98fb-828590ecc76d"
		tm.dropletAutoscale.EXPECT().DeleteDangerous(poolID).Return(nil)
		c.Args = append(c.Args, poolID)
		c.Doit.Set(c.NS, doctl.ArgForce, "true")

		err := RunDropletAutoscaleDeleteDangerous(c)
		assert.NoError(t, err)
	})
}
