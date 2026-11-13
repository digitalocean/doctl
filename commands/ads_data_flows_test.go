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

package commands

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testADSDataFlow = "3f2c9a1e-0b7d-4c1e-9a55-2d8e6f1b7c40"

// setADSDataFlowLocation sets the --region and --workspace flags and the data flow argument.
func setADSDataFlowLocation(config *CmdConfig, args ...string) {
	config.Args = args
	config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
	config.Doit.Set(config.NS, doctl.ArgADSWorkspace, testADSWorkspace)
}

func TestADSDataFlowCommand(t *testing.T) {
	assertCommandNames(t, adsDataFlow(),
		"create", "list", "get", "update", "delete",
		"pause", "resume", "restart", "rewind", "upload-url", "attach")
	assert.Equal(t, "metrics", adsMetrics().Name())
}

func TestRunADSDataFlowCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		manifest := filepath.Join(t.TempDir(), "orders.yaml")
		require.NoError(t, os.WriteFile(manifest, []byte("source:\n  kind: postgres\n"), 0o600))

		tm.ads.EXPECT().CreateDataFlow(&godo.ADSDataFlowCreateRequest{
			Region:          testADSRegion,
			Workspace:       testADSWorkspace,
			Name:            "orders",
			DesiredReplicas: 1,
			MaxReplicas:     3,
			ImageRef:        "registry.example.com/orders:1",
			Manifest:        json.RawMessage(`{"source":{"kind":"postgres"}}`),
		}).Return(&godo.ADSDataFlowCreateResponse{
			UUID:   testADSDataFlow,
			State:  "pending",
			Upload: &godo.ADSTarballUpload{URL: "https://upload.example.com/signed", ObjectKey: "fchain/key"},
		}, nil)
		setADSDataFlowLocation(config)
		config.Doit.Set(config.NS, doctl.ArgADSName, "orders")
		config.Doit.Set(config.NS, doctl.ArgADSDataFlowManifestFile, manifest)
		config.Doit.Set(config.NS, doctl.ArgADSDataFlowImageRef, "registry.example.com/orders:1")
		config.Doit.Set(config.NS, doctl.ArgADSDataFlowDesiredReplicas, 1)
		config.Doit.Set(config.NS, doctl.ArgADSDataFlowMaxReplicas, 3)
		buf := withInsightsOutput(t, config, "text")

		require.NoError(t, RunADSDataFlowCreate(config))
		assert.Contains(t, buf.String(), "fchain/key")
		assert.NotContains(t, buf.String(), "upload.example.com", "the presigned URL is not a default column")
	})
}

func TestRunADSDataFlowCreateRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	array := filepath.Join(dir, "array.json")
	require.NoError(t, os.WriteFile(array, []byte(`[1,2]`), 0o600))

	tests := []struct {
		name    string
		set     map[string]any
		wantErr string
	}{
		{name: "manifest not an object", set: map[string]any{doctl.ArgADSDataFlowManifestFile: array}, wantErr: "single JSON or YAML object"},
		{name: "desired above max", set: map[string]any{doctl.ArgADSDataFlowDesiredReplicas: 4, doctl.ArgADSDataFlowMaxReplicas: 2}, wantErr: "must not exceed"},
		{name: "negative replicas", set: map[string]any{doctl.ArgADSDataFlowDesiredReplicas: -1}, wantErr: "must not be negative"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				setADSDataFlowLocation(config)
				for key, value := range tt.set {
					config.Doit.Set(config.NS, key, value)
				}
				assert.ErrorContains(t, RunADSDataFlowCreate(config), tt.wantErr)
			})
		})
	}
}

func TestRunADSDataFlowListFollowsPages(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().ListDataFlows(&godo.ADSDataFlowListRequest{
			Region: testADSRegion, Workspace: testADSWorkspace,
		}).Return(&godo.ADSListDataFlowsResponse{
			DataFlows:     []godo.ADSDataFlow{{UUID: "flow-1", Name: "orders"}},
			NextPageToken: "page-2",
		}, nil)
		tm.ads.EXPECT().ListDataFlows(&godo.ADSDataFlowListRequest{
			Region: testADSRegion, Workspace: testADSWorkspace, PageToken: "page-2",
		}).Return(&godo.ADSListDataFlowsResponse{
			DataFlows: []godo.ADSDataFlow{{UUID: "flow-2", Name: "customers"}},
		}, nil)
		setADSDataFlowLocation(config)
		buf := withInsightsOutput(t, config, "text")

		require.NoError(t, RunADSDataFlowList(config))
		assert.Contains(t, buf.String(), "orders")
		assert.Contains(t, buf.String(), "customers")
	})
}

func TestRunADSDataFlowGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().GetDataFlow(testADSRegion, testADSWorkspace, testADSDataFlow).Return(&godo.ADSDataFlow{
			UUID: testADSDataFlow, Name: "orders", State: "active",
		}, nil)
		setADSDataFlowLocation(config, testADSDataFlow)

		require.NoError(t, RunADSDataFlowGet(config))
	})
}

func TestRunADSDataFlowUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		prev := adsStdin
		adsStdin = strings.NewReader(`{"source":{"kind":"mysql"}}`)
		t.Cleanup(func() { adsStdin = prev })

		tm.ads.EXPECT().UpdateDataFlow(&godo.ADSDataFlowUpdateRequest{
			Region:          testADSRegion,
			ID:              testADSDataFlow,
			Workspace:       testADSWorkspace,
			DesiredReplicas: 2,
			SecretRefs:      []string{"warehouse-db"},
			Manifest:        json.RawMessage(`{"source":{"kind":"mysql"}}`),
		}).Return(&godo.ADSDataFlowUpdateResponse{UUID: testADSDataFlow, State: "updating"}, nil)
		setADSDataFlowLocation(config, testADSDataFlow)
		config.Doit.Set(config.NS, doctl.ArgADSDataFlowDesiredReplicas, 2)
		config.Doit.Set(config.NS, doctl.ArgADSDataFlowSecretRefs, []string{"warehouse-db"})
		config.Doit.Set(config.NS, doctl.ArgADSDataFlowManifestFile, "-")

		require.NoError(t, RunADSDataFlowUpdate(config))
	})
}

func TestRunADSDataFlowUpdateRejectsBadInput(t *testing.T) {
	tests := []struct {
		name    string
		set     map[string]any
		wantErr string
	}{
		{name: "nothing to update", wantErr: "nothing to update"},
		{name: "set and clear secret refs", set: map[string]any{
			doctl.ArgADSDataFlowSecretRefs:      []string{"warehouse-db"},
			doctl.ArgADSDataFlowClearSecretRefs: true,
		}, wantErr: "cannot be combined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				setADSDataFlowLocation(config, testADSDataFlow)
				for key, value := range tt.set {
					config.Doit.Set(config.NS, key, value)
				}
				assert.ErrorContains(t, RunADSDataFlowUpdate(config), tt.wantErr)
			})
		})
	}
}

func TestRunADSDataFlowDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().DeleteDataFlow(testADSRegion, testADSWorkspace, testADSDataFlow).Return(nil)
		setADSDataFlowLocation(config, testADSDataFlow)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		require.NoError(t, RunADSDataFlowDelete(config))
	})
}

func TestRunADSDataFlowDeleteRequiresConfirmation(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		setADSDataFlowLocation(config, testADSDataFlow)

		require.ErrorIs(t, RunADSDataFlowDelete(config), errConfirmationRequired)
	})
}

func TestRunADSDataFlowLifecycle(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().PauseDataFlow(testADSRegion, testADSWorkspace, testADSDataFlow).Return(nil)
		tm.ads.EXPECT().ResumeDataFlow(testADSRegion, testADSWorkspace, testADSDataFlow).Return(nil)
		tm.ads.EXPECT().RestartDataFlow(testADSRegion, testADSWorkspace, testADSDataFlow).Return(nil)
		setADSDataFlowLocation(config, testADSDataFlow)

		require.NoError(t, RunADSDataFlowPause(config))
		require.NoError(t, RunADSDataFlowResume(config))
		require.NoError(t, RunADSDataFlowRestart(config))
	})
}

func TestRunADSDataFlowRewindSurfacesUnsupported(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		unsupported := errors.New("rewind is not supported")
		tm.ads.EXPECT().RewindDataFlow(&godo.ADSDataFlowRewindRequest{
			Region:    testADSRegion,
			ID:        testADSDataFlow,
			Workspace: testADSWorkspace,
			RewindTo:  "2026-11-01T00:00:00Z",
		}).Return(unsupported)
		setADSDataFlowLocation(config, testADSDataFlow)
		config.Doit.Set(config.NS, doctl.ArgADSDataFlowRewindTo, "2026-11-01T00:00:00Z")

		require.ErrorIs(t, RunADSDataFlowRewind(config), unsupported)
	})
}

func TestRunADSDataFlowUploadURL(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().CreateFchainUploadURL(&godo.ADSFchainUploadURLRequest{
			Region:    testADSRegion,
			Workspace: testADSWorkspace,
			SHA256:    "abc123",
		}).Return(&godo.ADSFchainUploadURLResponse{UploadURL: "https://upload.example.com/signed", ObjectKey: "fchain/key"}, nil)
		setADSDataFlowLocation(config)
		config.Doit.Set(config.NS, doctl.ArgADSFchainSHA256, "abc123")
		buf := withInsightsOutput(t, config, "text")

		require.NoError(t, RunADSDataFlowUploadURL(config))
		assert.Contains(t, buf.String(), "https://upload.example.com/signed")
	})
}

func TestRunADSDataFlowAttach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().AttachDataFlowFchain(&godo.ADSAttachDataFlowFchainRequest{
			Region:    testADSRegion,
			ID:        testADSDataFlow,
			Workspace: testADSWorkspace,
			ObjectKey: "fchain/key",
		}).Return(nil)
		setADSDataFlowLocation(config, testADSDataFlow)
		config.Doit.Set(config.NS, doctl.ArgADSFchainObjectKey, "fchain/key")

		require.NoError(t, RunADSDataFlowAttach(config))
	})
}

func TestRunADSDataFlowAttachRequiresObjectKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		setADSDataFlowLocation(config, testADSDataFlow)

		var missing *doctl.MissingArgsErr
		require.ErrorAs(t, RunADSDataFlowAttach(config), &missing)
	})
}

func TestRunADSMetrics(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().GetWorkspaceMetrics(&godo.ADSWorkspaceMetricsRequest{
			Region:      testADSRegion,
			Workspace:   testADSWorkspace,
			MetricNames: []string{"rows_written"},
			DataFlowID:  testADSDataFlow,
			From:        "2026-11-01T00:00:00Z",
			To:          "2026-11-02T00:00:00Z",
			GroupBy:     []string{"data_flow_id"},
			Limit:       50,
		}).Return(&godo.ADSWorkspaceMetricsResponse{
			Count:   1,
			Columns: []string{"data_flow_id", "rows_written"},
			Rows:    []map[string]any{{"data_flow_id": testADSDataFlow, "rows_written": 1200}},
		}, nil)
		setADSDataFlowLocation(config)
		config.Doit.Set(config.NS, doctl.ArgADSMetricNames, []string{"rows_written"})
		config.Doit.Set(config.NS, doctl.ArgADSMetricDataFlow, testADSDataFlow)
		config.Doit.Set(config.NS, doctl.ArgADSMetricFrom, "2026-11-01T00:00:00Z")
		config.Doit.Set(config.NS, doctl.ArgADSMetricTo, "2026-11-02T00:00:00Z")
		config.Doit.Set(config.NS, doctl.ArgADSMetricGroupBy, []string{"data_flow_id"})
		config.Doit.Set(config.NS, doctl.ArgADSMetricLimit, 50)
		buf := withInsightsOutput(t, config, "text")

		require.NoError(t, RunADSMetrics(config))
		assert.Contains(t, buf.String(), "rows_written")
		assert.Contains(t, buf.String(), "1200")
	})
}
