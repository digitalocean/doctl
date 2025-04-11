package commands

import (
	"github.com/digitalocean/doctl"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testPeering = do.VPCPeering{
		VPCPeering: &godo.VPCPeering{
			Name:   "peering-name",
			VPCIDs: []string{"f81d4fae-7dec-11d0-a765-00a0c91e6bf6", "3f900b61-30d7-40d8-9711-8c5d6264b268"},
			Status: "ACTIVE",
		},
	}

	testPeeringList = do.VPCPeerings{
		testPeering,
	}
)

func TestVPCPeeringsCommand(t *testing.T) {
	cmd := VPCPeerings()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "list", "create", "update", "delete")
}

func TestVPCPeeringGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		peeringID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.vpcs.EXPECT().GetPeering(peeringID).Return(&testPeering, nil)

		config.Args = append(config.Args, peeringID)

		err := RunVPCPeeringGet(config)
		assert.NoError(t, err)
	})
}

func TestVPCPeeringGetNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVPCPeeringGet(config)
		assert.Error(t, err)
	})
}

func TestVPCPeeringList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.vpcs.EXPECT().ListVPCPeerings().Return(testPeeringList, nil)

		err := RunVPCPeeringList(config)
		assert.NoError(t, err)
	})
}

func TestVPCPeeringListByVpcID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		vpcID := "vpc-01"
		tm.vpcs.EXPECT().ListVPCPeeringsByVPCID(vpcID).Return(testPeeringList, nil)

		config.Doit.Set(config.NS, doctl.ArgVPCPeeringVPCID, vpcID)
		err := RunVPCPeeringList(config)
		assert.NoError(t, err)
	})
}

func TestVPCPeeringCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.VPCPeeringCreateRequest{
			Name:   "peering-name",
			VPCIDs: []string{"f81d4fae-7dec-11d0-a765-00a0c91e6bf6", "3f900b61-30d7-40d8-9711-8c5d6264b268"},
		}
		tm.vpcs.EXPECT().CreateVPCPeering(&r).Return(&testPeering, nil)

		config.Args = append(config.Args, "peering-name")
		config.Doit.Set(config.NS, doctl.ArgVPCPeeringVPCIDs, "f81d4fae-7dec-11d0-a765-00a0c91e6bf6,3f900b61-30d7-40d8-9711-8c5d6264b268")

		err := RunVPCPeeringCreate(config)
		assert.NoError(t, err)
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "peering-name")

		err := RunVPCPeeringCreate(config)
		assert.EqualError(t, err, "VPC ID is empty")
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "peering-name")
		config.Doit.Set(config.NS, doctl.ArgVPCPeeringVPCIDs, "f81d4fae-7dec-11d0-a765-00a0c91e6bf6")

		err := RunVPCPeeringCreate(config)
		assert.EqualError(t, err, "VPC IDs length should be 2")
	})
}

func TestVPCPeeringUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		peeringID := "peering-uuid1"
		peeringName := "updated-peering-name"
		r := godo.VPCPeeringUpdateRequest{Name: peeringName}
		tm.vpcs.EXPECT().UpdateVPCPeering(peeringID, &r).Return(&testPeering, nil)

		config.Args = append(config.Args, peeringID)
		config.Doit.Set(config.NS, doctl.ArgVPCPeeringName, "updated-peering-name")

		err := RunVPCPeeringUpdate(config)
		assert.NoError(t, err)
	})
}

func TestVPCPeeringUpdateNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVPCPeeringUpdate(config)
		assert.Error(t, err)
	})
}

func TestVPCPeeringDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		peeringID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.vpcs.EXPECT().DeleteVPCPeering(peeringID).Return(nil)

		config.Args = append(config.Args, peeringID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunVPCPeeringDelete(config)
		assert.NoError(t, err)
	})
}

func TestVPCPeeringDeleteNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVPCPeeringDelete(config)
		assert.Error(t, err)
	})
}
