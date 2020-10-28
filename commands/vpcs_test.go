package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"

	"github.com/stretchr/testify/assert"
)

var (
	testVPC = do.VPC{
		VPC: &godo.VPC{
			Name:        "vpc-name",
			RegionSlug:  "nyc1",
			Description: "vpc description",
			IPRange:     "10.116.0.0/20",
		}}

	testVPCList = do.VPCs{
		testVPC,
	}
)

func TestVPCsCommand(t *testing.T) {
	cmd := VPCs()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "list", "create", "update", "delete")
}

func TestVPCGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		vpcUUID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.vpcs.EXPECT().Get(vpcUUID).Return(&testVPC, nil)

		config.Args = append(config.Args, vpcUUID)

		err := RunVPCGet(config)
		assert.NoError(t, err)
	})
}

func TestVPCGetNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVPCGet(config)
		assert.Error(t, err)
	})
}

func TestVPCList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.vpcs.EXPECT().List().Return(testVPCList, nil)

		err := RunVPCList(config)
		assert.NoError(t, err)
	})
}

func TestVPCCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.VPCCreateRequest{
			Name:        "vpc-name",
			RegionSlug:  "nyc1",
			Description: "vpc description",
			IPRange:     "10.116.0.0/20",
		}
		tm.vpcs.EXPECT().Create(&r).Return(&testVPC, nil)

		config.Doit.Set(config.NS, doctl.ArgVPCName, "vpc-name")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, doctl.ArgVPCDescription, "vpc description")
		config.Doit.Set(config.NS, doctl.ArgVPCIPRange, "10.116.0.0/20")

		err := RunVPCCreate(config)
		assert.NoError(t, err)
	})
}

func TestVPCUpdate(t *testing.T) {
	tests := []struct {
		desc            string
		setup           func(*CmdConfig)
		expectedVPCId   string
		expectedRequest []godo.VPCSetField
	}{
		{
			desc: "update vpc name",
			setup: func(in *CmdConfig) {
				in.Args = append(in.Args, "vpc-uuid")
				in.Doit.Set(in.NS, doctl.ArgVPCName, "update-vpc-name-test")

			},
			expectedVPCId: "vpc-uuid",
			expectedRequest: []godo.VPCSetField{
				godo.VPCSetName("update-vpc-name-test"),
			},
		},

		{
			desc: "update vpc name and description",
			setup: func(in *CmdConfig) {
				in.Args = append(in.Args, "vpc-uuid")
				in.Doit.Set(in.NS, doctl.ArgVPCName, "update-vpc-name-test")
				in.Doit.Set(in.NS, doctl.ArgVPCDescription, "i am a new desc")

			},
			expectedVPCId: "vpc-uuid",
			expectedRequest: []godo.VPCSetField{
				godo.VPCSetName("update-vpc-name-test"),
				godo.VPCSetDescription("i am a new desc"),
			},
		},

		{
			desc: "update vpc name and description and set to default",
			setup: func(in *CmdConfig) {
				in.Args = append(in.Args, "vpc-uuid")
				in.Doit.Set(in.NS, doctl.ArgVPCName, "update-vpc-name-test")
				in.Doit.Set(in.NS, doctl.ArgVPCDescription, "i am a new desc")
				in.Doit.Set(in.NS, doctl.ArgVPCDefault, true)
			},
			expectedVPCId: "vpc-uuid",
			expectedRequest: []godo.VPCSetField{
				godo.VPCSetName("update-vpc-name-test"),
				godo.VPCSetDescription("i am a new desc"),
				godo.VPCSetDefault(),
			},
		},

		{
			desc: "update only default",
			setup: func(in *CmdConfig) {
				in.Args = append(in.Args, "vpc-uuid")
				in.Doit.Set(in.NS, doctl.ArgVPCDefault, true)
			},
			expectedVPCId: "vpc-uuid",
			expectedRequest: []godo.VPCSetField{
				godo.VPCSetDefault(),
			},
		},
	}

	for _, tt := range tests {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			if tt.setup != nil {
				tt.setup(config)
			}

			tm.vpcs.EXPECT().PartialUpdate(tt.expectedVPCId, tt.expectedRequest).Return(&testVPC, nil)
			err := RunVPCUpdate(config)

			assert.NoError(t, err)
		})
	}
}

func TestVPCUpdatefNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVPCUpdate(config)
		assert.Error(t, err)
	})
}

func TestVPCDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		vpcUUID := "e819b321-a9a1-4078-b437-8e6b8bf13530"
		tm.vpcs.EXPECT().Delete(vpcUUID).Return(nil)

		config.Args = append(config.Args, vpcUUID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunVPCDelete(config)
		assert.NoError(t, err)
	})
}

func TestVPCDeleteNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunVPCDelete(config)
		assert.Error(t, err)
	})
}
