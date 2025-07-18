package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testParentAgentID = "12345678-1234-1234-1234-123456789012"
	testChildAgentID  = "12345678-1234-1234-1234-123456789013"
	testRouteUUID     = "12345678-1234-1234-1234-123456789014"

	testAgentRouteResponse = &do.AgentRouteResponse{
		AgentRouteResponse: &godo.AgentRouteResponse{
			ParentAgentUuid: testParentAgentID,
			ChildAgentUuid:  testChildAgentID,
			UUID:            testRouteUUID,
			Rollback:        false,
		},
	}
)

func TestAgentRouteAdd(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.genAI.EXPECT().AddAgentRoute(testParentAgentID, testChildAgentID).Return(testAgentRouteResponse, nil)

		config.Args = []string{}
		config.Doit.Set(config.NS, doctl.ArgParentAgentId, testParentAgentID)
		config.Doit.Set(config.NS, doctl.ArgChildAgentId, testChildAgentID)
		config.Doit.Set(config.NS, doctl.ArgAgentRouteId, testRouteUUID)
		config.Doit.Set(config.NS, doctl.ArgAgentRouteName, "test_route")
		config.Doit.Set(config.NS, doctl.ArgAgentRouteIfCase, "test if case")

		err := RunAgentRouteAdd(config)
		assert.NoError(t, err)
	})
}

func TestAgentRouteUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expectedReq := &godo.AgentRouteUpdateRequest{
			ParentAgentUuid: testParentAgentID,
			ChildAgentUuid:  testChildAgentID,
			UUID:            testRouteUUID,
			RouteName:       "test_route",
			IfCase:          "test if case",
		}

		tm.genAI.EXPECT().UpdateAgentRoute(testParentAgentID, testChildAgentID, expectedReq).Return(testAgentRouteResponse, nil)

		config.Args = []string{}
		config.Doit.Set(config.NS, doctl.ArgParentAgentId, testParentAgentID)
		config.Doit.Set(config.NS, doctl.ArgChildAgentId, testChildAgentID)
		config.Doit.Set(config.NS, doctl.ArgAgentRouteId, testRouteUUID)
		config.Doit.Set(config.NS, doctl.ArgAgentRouteName, "test_route")
		config.Doit.Set(config.NS, doctl.ArgAgentRouteIfCase, "test if case")

		err := RunAgentRouteUpdate(config)
		assert.NoError(t, err)
	})
}

func TestAgentRouteDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.genAI.EXPECT().DeleteAgentRoute(testParentAgentID, testChildAgentID).Return(nil)

		config.Args = []string{}
		config.Doit.Set(config.NS, doctl.ArgParentAgentId, testParentAgentID)
		config.Doit.Set(config.NS, doctl.ArgChildAgentId, testChildAgentID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunAgentRouteDelete(config)
		assert.NoError(t, err)
	})
}
