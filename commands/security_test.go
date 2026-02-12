/*
Copyright 2026 The Doctl Authors All rights reserved.
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

package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testSecurityScan = do.Scan{
		Scan: &godo.Scan{
			ID:        "497dcba3-ecbf-4587-a2dd-5eb0665e6880",
			Status:    "COMPLETED",
			CreatedAt: "2025-12-04T00:00:00Z",
			Findings: []*godo.ScanFinding{
				{RuleUUID: "rule-1", Name: "test", Severity: "critical", AffectedResourcesCount: 2},
			},
		},
	}

	testSecurityScanList = do.Scans{testSecurityScan}

	testSecurityAffectedResource = do.AffectedResource{
		AffectedResource: &godo.AffectedResource{
			URN:  "do:droplet:1",
			Name: "droplet-1",
			Type: "Droplet",
		},
	}

	testSecurityAffectedResources = do.AffectedResources{testSecurityAffectedResource}
)

func TestSecurityCommand(t *testing.T) {
	cmd := Security()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "scan", "finding")
	assertCommandNames(t, cmd.childCommands[0], "create", "get", "latest", "list")
	assertCommandNames(t, cmd.childCommands[1], "affected-resources")
}

func TestSecurityScanCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		request := &godo.CreateScanRequest{
			Resources: []string{"do:droplet"},
		}
		tm.security.EXPECT().CreateScan(request).Return(&testSecurityScan, nil)

		config.Doit.Set(config.NS, doctl.ArgSecurityScanResources, []string{"do:droplet"})

		err := RunCmdSecurityScanCreate(config)
		assert.NoError(t, err)
	})
}

func TestSecurityScanGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		opts := &godo.ScanFindingsOptions{Severity: "critical", Type: "CSPM"}
		tm.security.EXPECT().GetScan(testSecurityScan.Scan.ID, opts).Return(&testSecurityScan, nil)

		config.Args = append(config.Args, testSecurityScan.Scan.ID)
		config.Doit.Set(config.NS, doctl.ArgSecurityScanFindingSeverity, "critical")
		config.Doit.Set(config.NS, doctl.ArgSecurityScanFindingType, "CSPM")

		err := RunCmdSecurityScanGet(config)
		assert.NoError(t, err)
	})
}

func TestSecurityScanLatest(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		opts := &godo.ScanFindingsOptions{Severity: "high"}
		tm.security.EXPECT().GetLatestScan(opts).Return(&testSecurityScan, nil)

		config.Doit.Set(config.NS, doctl.ArgSecurityScanFindingSeverity, "high")

		err := RunCmdSecurityScanLatest(config)
		assert.NoError(t, err)
	})
}

func TestSecurityScanList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.security.EXPECT().ListScans().Return(testSecurityScanList, nil)

		err := RunCmdSecurityScanList(config)
		assert.NoError(t, err)
	})
}

func TestSecurityFindingAffectedResources(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.security.EXPECT().ListFindingAffectedResources("scan-uuid", "finding-uuid").Return(testSecurityAffectedResources, nil)

		config.Doit.Set(config.NS, doctl.ArgSecurityFindingScanUUID, "scan-uuid")
		config.Doit.Set(config.NS, doctl.ArgSecurityFindingUUID, "finding-uuid")

		err := RunCmdSecurityFindingAffectedResources(config)
		assert.NoError(t, err)
	})
}
