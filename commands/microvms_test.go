/*
Copyright 2025 The Doctl Authors All rights reserved.
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
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testMicroVMID = "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2"

	testMicroVM = do.MicroVM{
		MicroVM: &godo.MicroVM{
			ID:         testMicroVMID,
			Name:       "sammy-microvm",
			Region:     "nyc1",
			State:      godo.MicroVMStateRunning,
			Size:       &godo.MicroVMSize{CPU: 2, Memory: 4096, Disk: 80},
			Networking: godo.MicroVMNetworkingPublic,
			Source:     &godo.MicroVMSource{OCIRef: "docker.io/library/nginx:1.27"},
			URLs: []godo.MicroVMURL{
				{Hostname: "sammy.example.com", Port: 8080, Default: true, Status: godo.MicroVMURLStatusActive},
			},
			Ports:   []uint32{8080},
			Created: "2026-07-16T10:00:00Z",
		},
	}

	testMicroVMList = do.MicroVMs{testMicroVM}

	testMicroVMCheckpoint = do.MicroVMCheckpoint{
		MicroVMCheckpoint: &godo.MicroVMCheckpoint{
			ID:          "8f7a9a3f-5555-4444-9999-000000000001",
			MicroVMID:   testMicroVMID,
			MicroVMName: "sammy-microvm",
			Name:        "sammy-checkpoint",
			Region:      "nyc1",
			Status:      godo.MicroVMCheckpointStatusAvailable,
			MemoryBytes: 512 * 1024 * 1024,
			DiskBytes:   1024 * 1024 * 1024,
			Created:     "2026-07-16T10:05:00Z",
		},
	}
)

func TestMicroVMCommand(t *testing.T) {
	cmd := MicroVM()
	assert.NotNil(t, cmd)
	assert.False(t, cmd.Hidden)
	assertCommandNames(t, cmd,
		"access-token", "checkpoint", "console", "create", "delete", "download", "exec", "fork", "get", "list", "options", "pause", "port-forward", "resume", "update", "upload", "url",
	)
}

func TestMicroVMAccessTokenCommand(t *testing.T) {
	cmd := microVMAccessTokens()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "revoke")
}

func TestMicroVMURLCommand(t *testing.T) {
	cmd := microVMURLs()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "add", "list", "remove")
}

func TestMicroVMCheckpointCommand(t *testing.T) {
	cmd := microVMCheckpoints()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list")
}

func TestMicroVMsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().List(do.MicroVMListFilter{}).Return(testMicroVMList, nil)
		err := RunMicroVMList(config)
		require.NoError(t, err)
	})
}

func TestMicroVMsListByRegion(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().List(do.MicroVMListFilter{Region: "nyc1"}).Return(testMicroVMList, nil)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		err := RunMicroVMList(config)
		require.NoError(t, err)
	})
}

func TestMicroVMsListByNameAndTag(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().List(do.MicroVMListFilter{Name: "sammy-microvm", TagName: "prod"}).Return(testMicroVMList, nil)
		config.Doit.Set(config.NS, "name", "sammy-microvm")
		config.Doit.Set(config.NS, doctl.ArgTagName, "prod")
		err := RunMicroVMList(config)
		require.NoError(t, err)
	})
}

func TestMicroVMsListByFork(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().List(do.MicroVMListFilter{ForkID: testMicroVMForkID, ForkedFrom: testMicroVMID}).Return(testMicroVMList, nil)
		config.Doit.Set(config.NS, "fork-id", testMicroVMForkID)
		config.Doit.Set(config.NS, "forked-from", testMicroVMID)
		require.NoError(t, RunMicroVMList(config))
	})
}

func TestMicroVMGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().Get(testMicroVMID).Return(&testMicroVM, nil)
		config.Args = append(config.Args, testMicroVMID)
		err := RunMicroVMGet(config)
		require.NoError(t, err)
	})
}

func TestMicroVMGet_missingArg(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunMicroVMGet(config)
		require.Error(t, err)
	})
}

func TestMicroVMCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		enabled := true
		expected := &godo.MicroVMCreateRequest{
			Name:         "sammy-microvm",
			Region:       "nyc1",
			Size:         &godo.MicroVMSizeRequest{CPU: 2, Memory: 4096},
			Source:       &godo.MicroVMSource{OCIRef: "docker.io/library/nginx:1.27"},
			Networking:   godo.MicroVMNetworkingVPC,
			VPCUUID:      "vpc-uuid-1234",
			AutoPause:    &godo.AutoPauseConfig{Enabled: &enabled, IdleTimeout: "5m"},
			AutoResume:   &enabled,
			HTTPPort:     8080,
			HTTPProtocol: godo.MicroVMHTTPProtocolHTTP2,
			Ports:        []uint32{80, 8080},
			Environment:  map[string]string{"FOO": "bar", "BAZ": "qux"},
			Tags:         []string{"prod", "web"},
		}
		tm.microVMs.EXPECT().Create(expected).Return(&testMicroVM, nil)

		config.Args = append(config.Args, "sammy-microvm")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, "cpu", 2)
		config.Doit.Set(config.NS, "memory", 4096)
		config.Doit.Set(config.NS, "oci-ref", "docker.io/library/nginx:1.27")
		config.Doit.Set(config.NS, "networking", "vpc")
		config.Doit.Set(config.NS, doctl.ArgVPCUUID, "vpc-uuid-1234")
		config.Doit.Set(config.NS, "auto-pause", true)
		config.Doit.Set(config.NS, "auto-pause-idle-timeout", "5m")
		config.Doit.Set(config.NS, "auto-resume", true)
		config.Doit.Set(config.NS, "http-port", 8080)
		config.Doit.Set(config.NS, "http-protocol", "http2")
		config.Doit.Set(config.NS, "ports", []string{"80", "8080"})
		config.Doit.Set(config.NS, "env", []string{"FOO=bar", "BAZ=qux"})
		config.Doit.Set(config.NS, doctl.ArgTag, []string{"prod", "web"})

		err := RunMicroVMCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroVMCreate_egressPolicy(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expected := &godo.MicroVMCreateRequest{
			Name:   "sammy-microvm",
			Region: "nyc1",
			Size:   &godo.MicroVMSizeRequest{CPU: 2, Memory: 4096},
			Source: &godo.MicroVMSource{OCIRef: "docker.io/library/nginx:1.27"},
			EgressPolicy: &godo.MicroVMEgressPolicy{
				DefaultAction: godo.MicroVMEgressActionDeny,
				Rules: []godo.MicroVMEgressRule{
					{Domain: "*.example.com"},
					{IP: "203.0.113.5"},
				},
			},
		}
		tm.microVMs.EXPECT().Create(expected).Return(&testMicroVM, nil)

		config.Args = append(config.Args, "sammy-microvm")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, "cpu", 2)
		config.Doit.Set(config.NS, "memory", 4096)
		config.Doit.Set(config.NS, "oci-ref", "docker.io/library/nginx:1.27")
		config.Doit.Set(config.NS, "egress-allow", []string{"*.example.com", "203.0.113.5"})
		config.Doit.Set(config.NS, "egress-default-action", "deny")

		err := RunMicroVMCreate(config)
		require.NoError(t, err)
	})
}

func TestBuildEgressPolicy(t *testing.T) {
	tests := []struct {
		name    string
		allow   []string
		action  string
		want    *godo.MicroVMEgressPolicy
		wantErr string
	}{
		{
			name:   "default action alone is a policy with no rules",
			action: "deny",
			want:   &godo.MicroVMEgressPolicy{DefaultAction: godo.MicroVMEgressActionDeny},
		},
		{
			name:  "domain, wildcard and IPv4/IPv6 literals",
			allow: []string{"api.example.com", "*.example.com", "203.0.113.5", "2001:db8::1"},
			want: &godo.MicroVMEgressPolicy{Rules: []godo.MicroVMEgressRule{
				{Domain: "api.example.com"},
				{Domain: "*.example.com"},
				{IP: "203.0.113.5"},
				{IP: "2001:db8::1"},
			}},
		},
		{name: "CIDR", allow: []string{"10.0.0.0/8"}, wantErr: "CIDR"},
		{name: "host and port", allow: []string{"example.com:443"}, wantErr: "ports"},
		{name: "empty value", allow: []string{""}, wantErr: "must not be empty"},
		{name: "bad action", action: "block", wantErr: "expected deny or allow"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildEgressPolicy(tt.allow, tt.action)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMicroVMCreate_fromCheckpoint(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expected := &godo.MicroVMCreateRequest{
			Name:   "sammy-clone",
			Source: &godo.MicroVMSource{CheckpointID: testMicroVMCheckpoint.ID},
		}
		tm.microVMs.EXPECT().Create(expected).Return(&testMicroVM, nil)

		config.Args = append(config.Args, "sammy-clone")
		config.Doit.Set(config.NS, "checkpoint-id", testMicroVMCheckpoint.ID)

		err := RunMicroVMCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroVMCreate_requiresSource(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "sammy-microvm")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, "cpu", 2)
		config.Doit.Set(config.NS, "memory", 4096)

		err := RunMicroVMCreate(config)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exactly one of --oci-ref or --checkpoint-id")
	})
}

func TestMicroVMCreate_badEnv(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "sammy-microvm")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, "cpu", 2)
		config.Doit.Set(config.NS, "memory", 4096)
		config.Doit.Set(config.NS, "oci-ref", "docker.io/library/nginx:1.27")
		config.Doit.Set(config.NS, "env", []string{"MALFORMED"})

		err := RunMicroVMCreate(config)
		require.Error(t, err)
	})
}

func TestMicroVMCreate_idleTimeoutWithoutAutoPause(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expected := &godo.MicroVMCreateRequest{
			Name:      "sammy-microvm",
			Region:    "nyc1",
			Size:      &godo.MicroVMSizeRequest{CPU: 2, Memory: 4096},
			Source:    &godo.MicroVMSource{OCIRef: "docker.io/library/nginx:1.27"},
			AutoPause: &godo.AutoPauseConfig{IdleTimeout: "5m"},
		}
		tm.microVMs.EXPECT().Create(expected).Return(&testMicroVM, nil)

		config.Args = append(config.Args, "sammy-microvm")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, "cpu", 2)
		config.Doit.Set(config.NS, "memory", 4096)
		config.Doit.Set(config.NS, "oci-ref", "docker.io/library/nginx:1.27")
		config.Doit.Set(config.NS, "auto-pause-idle-timeout", "5m")

		err := RunMicroVMCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroVMCreate_disablesAutoPauseAndResume(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		disabled := false
		expected := &godo.MicroVMCreateRequest{
			Name:       "sammy-microvm",
			Region:     "nyc1",
			Size:       &godo.MicroVMSizeRequest{CPU: 2, Memory: 4096},
			Source:     &godo.MicroVMSource{OCIRef: "docker.io/library/nginx:1.27"},
			AutoPause:  &godo.AutoPauseConfig{Enabled: &disabled},
			AutoResume: &disabled,
		}
		tm.microVMs.EXPECT().Create(expected).Return(&testMicroVM, nil)

		config.Args = append(config.Args, "sammy-microvm")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, "cpu", 2)
		config.Doit.Set(config.NS, "memory", 4096)
		config.Doit.Set(config.NS, "oci-ref", "docker.io/library/nginx:1.27")
		config.Doit.Set(config.NS, "auto-pause", false)
		config.Doit.Set(config.NS, "auto-resume", false)

		err := RunMicroVMCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroVMCreate_noIngress(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ingress := false
		expected := &godo.MicroVMCreateRequest{
			Name:    "sammy-microvm",
			Region:  "nyc1",
			Size:    &godo.MicroVMSizeRequest{CPU: 2, Memory: 4096},
			Source:  &godo.MicroVMSource{OCIRef: "docker.io/library/nginx:1.27"},
			Ingress: &ingress,
		}
		tm.microVMs.EXPECT().Create(expected).Return(&testMicroVM, nil)

		config.Args = append(config.Args, "sammy-microvm")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, "cpu", 2)
		config.Doit.Set(config.NS, "memory", 4096)
		config.Doit.Set(config.NS, "oci-ref", "docker.io/library/nginx:1.27")
		config.Doit.Set(config.NS, "ingress", false)

		err := RunMicroVMCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroVMURLList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().Get(testMicroVMID).Return(&testMicroVM, nil)
		config.Args = append(config.Args, testMicroVMID)
		err := RunMicroVMURLList(config)
		require.NoError(t, err)
	})
}

func TestMicroVMURLAdd(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().AddURL(testMicroVMID, &godo.MicroVMURLCreateRequest{Port: 9090}).Return(
			&godo.MicroVMURL{ID: "url-1", Hostname: "added.example.com", Port: 9090, Status: godo.MicroVMURLStatusPending}, nil,
		)
		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "port", 9090)
		err := RunMicroVMURLAdd(config)
		require.NoError(t, err)
	})
}

func TestMicroVMURLAdd_defaultPort(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().AddURL(testMicroVMID, &godo.MicroVMURLCreateRequest{}).Return(
			&godo.MicroVMURL{ID: "url-2", Status: godo.MicroVMURLStatusActive}, nil,
		)
		config.Args = append(config.Args, testMicroVMID)
		err := RunMicroVMURLAdd(config)
		require.NoError(t, err)
	})
}

func TestMicroVMURLAdd_badPort(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "port", 70000)
		err := RunMicroVMURLAdd(config)
		require.Error(t, err)
	})
}

func TestMicroVMURLRemove(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().RemoveURL(testMicroVMID, "url-1").Return(nil)
		config.Args = append(config.Args, testMicroVMID, "url-1")
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		err := RunMicroVMURLRemove(config)
		require.NoError(t, err)
	})
}

func TestMicroVMURLRemove_missingArg(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		err := RunMicroVMURLRemove(config)
		require.Error(t, err)
	})
}

func TestMicroVMAccessTokenCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().CreateAccessToken(testMicroVMID, &godo.MicroVMAccessTokenCreateRequest{Name: "alice", ExpiresInSeconds: 3600}).Return(
			&godo.MicroVMAccessTokenCreated{
				MicroVMAccessToken: godo.MicroVMAccessToken{ID: "tok-1", MicroVMID: testMicroVMID, Name: "alice"},
				Token:              "dop_v1_test",
			}, nil,
		)
		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "name", "alice")
		config.Doit.Set(config.NS, "expires-in-seconds", 3600)
		err := RunMicroVMAccessTokenCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroVMAccessTokenCreate_invalidFlags(t *testing.T) {
	tests := []struct {
		name      string
		tokenName string
		expiresIn int
	}{
		{name: "missing name", tokenName: ""},
		{name: "name too long", tokenName: strings.Repeat("a", 65)},
		{name: "negative expiry", tokenName: "alice", expiresIn: -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Args = append(config.Args, testMicroVMID)
				config.Doit.Set(config.NS, "name", tc.tokenName)
				config.Doit.Set(config.NS, "expires-in-seconds", tc.expiresIn)
				err := RunMicroVMAccessTokenCreate(config)
				require.Error(t, err)
			})
		})
	}
}

func TestMicroVMAccessTokenList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().ListAccessTokens(testMicroVMID).Return([]godo.MicroVMAccessToken{
			{ID: "tok-1", MicroVMID: testMicroVMID, Name: "alice"},
			{ID: "tok-2", MicroVMID: testMicroVMID, Name: "bob", ExpiresAt: "2026-11-21T14:02:11Z"},
		}, nil)
		config.Args = append(config.Args, testMicroVMID)
		err := RunMicroVMAccessTokenList(config)
		require.NoError(t, err)
	})
}

func TestMicroVMAccessTokenRevoke(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().RevokeAccessToken(testMicroVMID, "tok-1").Return(nil)
		config.Args = append(config.Args, testMicroVMID, "tok-1")
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		err := RunMicroVMAccessTokenRevoke(config)
		require.NoError(t, err)
	})
}

func TestMicroVMAccessTokenRevoke_missingArg(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		err := RunMicroVMAccessTokenRevoke(config)
		require.Error(t, err)
	})
}

func TestMicroVMUpdate(t *testing.T) {
	tests := []struct {
		name     string
		flags    map[string]any
		expected *godo.MicroVMUpdateRequest
	}{
		{
			name: "all fields",
			flags: map[string]any{
				"name":                    "renamed",
				doctl.ArgTag:              []string{"prod", "web"},
				"auto-pause-idle-timeout": "10m",
			},
			expected: &godo.MicroVMUpdateRequest{
				Name:      "renamed",
				Tags:      &[]string{"prod", "web"},
				AutoPause: &godo.AutoPauseConfig{IdleTimeout: "10m"},
			},
		},
		{
			name:     "name only",
			flags:    map[string]any{"name": "renamed"},
			expected: &godo.MicroVMUpdateRequest{Name: "renamed"},
		},
		{
			name:     "clear tags",
			flags:    map[string]any{"clear-tags": true},
			expected: &godo.MicroVMUpdateRequest{Tags: &[]string{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.microVMs.EXPECT().Update(testMicroVMID, tt.expected).Return(&testMicroVM, nil)

				config.Args = append(config.Args, testMicroVMID)
				for k, v := range tt.flags {
					config.Doit.Set(config.NS, k, v)
				}
				require.NoError(t, RunMicroVMUpdate(config))
			})
		})
	}
}

const testMicroVMForkID = "8f7a9a3f-5555-4444-9999-0000000000f0"

func forkCopy(id string, state godo.MicroVMState) do.MicroVM {
	return do.MicroVM{MicroVM: &godo.MicroVM{
		ID:         id,
		Name:       "sammy-fork-" + id,
		State:      state,
		Source:     &godo.MicroVMSource{CheckpointID: testMicroVMCheckpoint.ID},
		ForkID:     testMicroVMForkID,
		ForkedFrom: testMicroVMID,
	}}
}

// keepMicroVMMock holds RetryMax at zero so a fork runs against the mock;
// with retries on, the runner swaps in a live client that does not retry.
func keepMicroVMMock(t *testing.T) {
	t.Helper()
	prev := RetryMax
	RetryMax = 0
	t.Cleanup(func() { RetryMax = prev })
}

func TestMicroVMFork(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		keepMicroVMMock(t)
		tm.microVMs.EXPECT().
			Fork(testMicroVMID, &godo.MicroVMForkRequest{Count: 2, Name: "eval", Tags: []string{"run-42"}}).
			Return(&do.MicroVMFork{
				MicroVMs:     do.MicroVMs{forkCopy("1", godo.MicroVMStateCreating), forkCopy("2", godo.MicroVMStateCreating)},
				ForkID:       testMicroVMForkID,
				CheckpointID: testMicroVMCheckpoint.ID,
			}, nil)

		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "count", 2)
		config.Doit.Set(config.NS, "name", "eval")
		config.Doit.Set(config.NS, doctl.ArgTag, []string{"run-42"})
		require.NoError(t, RunMicroVMFork(config))
	})
}

// A retried fork whose first attempt was accepted would fork twice, so the
// call goes through a client with retries off whenever the shared one has
// them on.
func TestMicroVMForkDoesNotRetry(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		prev := RetryMax
		t.Cleanup(func() { RetryMax = prev })

		RetryMax = 0
		svc, err := microVMsWithoutRetries(config)
		require.NoError(t, err)
		assert.Same(t, config.MicroVMs(), svc, "with retries already off there is nothing to swap")

		RetryMax = 5
		svc, err = microVMsWithoutRetries(config)
		require.NoError(t, err)
		assert.NotSame(t, config.MicroVMs(), svc, "fork must not run against the retrying client")
	})
}

// Only an answer that leaves the outcome open points at the --forked-from
// check: a 4xx says the fork was refused.
func TestMicroVMFork_unknownOutcomeSaysHowToCheck(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantHint bool
	}{
		{name: "server error", err: godoStatusErr(http.StatusBadGateway, "bad gateway"), wantHint: true},
		{name: "transport error", err: io.ErrUnexpectedEOF, wantHint: true},
		{name: "refused", err: godoStatusErr(http.StatusBadRequest, "count must be at most 10")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				keepMicroVMMock(t)
				tm.microVMs.EXPECT().Fork(testMicroVMID, gomock.Any()).Return(nil, tt.err)

				config.Args = append(config.Args, testMicroVMID)
				config.Doit.Set(config.NS, "count", 1)
				err := RunMicroVMFork(config)
				require.ErrorIs(t, err, tt.err)
				var ns NextStepper
				if !tt.wantHint {
					assert.False(t, errors.As(err, &ns), "a refused fork keeps the default next step")
					return
				}
				require.ErrorAs(t, err, &ns)
				assert.Contains(t, ns.NextStep(), "--forked-from "+testMicroVMID)
			})
		})
	}
}

func TestMicroVMFork_waitReportsFailedCopies(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		keepMicroVMMock(t)
		tm.microVMs.EXPECT().Fork(testMicroVMID, &godo.MicroVMForkRequest{Count: 2}).Return(&do.MicroVMFork{
			MicroVMs:     do.MicroVMs{forkCopy("1", godo.MicroVMStateCreating), forkCopy("2", godo.MicroVMStateCreating)},
			ForkID:       testMicroVMForkID,
			CheckpointID: testMicroVMCheckpoint.ID,
		}, nil)
		running, failed := forkCopy("1", godo.MicroVMStateRunning), forkCopy("2", godo.MicroVMStateFailed)
		tm.microVMs.EXPECT().Get("1").Return(&running, nil)
		tm.microVMs.EXPECT().Get("2").Return(&failed, nil)

		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "count", 2)
		config.Doit.Set(config.NS, doctl.ArgCommandWait, true)
		err := RunMicroVMFork(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "1 of 2 copies failed")
	})
}

func TestMicroVMFork_badCount(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "count", 0)
		require.Error(t, RunMicroVMFork(config))
	})
}

func TestWaitForMicroVMCopies(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		copies := do.MicroVMs{
			forkCopy("1", godo.MicroVMStateCreating),
			forkCopy("2", godo.MicroVMStateCreating),
			forkCopy("3", godo.MicroVMStateFailed),
		}
		creating, running := forkCopy("1", godo.MicroVMStateCreating), forkCopy("1", godo.MicroVMStateRunning)
		gomock.InOrder(
			tm.microVMs.EXPECT().Get("1").Return(&creating, nil),
			tm.microVMs.EXPECT().Get("1").Return(&running, nil),
		)
		tm.microVMs.EXPECT().Get("2").Return(nil, godoStatusErr(http.StatusNotFound, "not found"))

		require.NoError(t, waitForMicroVMCopies(newTestWaiter(), tm.microVMs, copies))
		assert.Equal(t, godo.MicroVMStateRunning, copies[0].State)
		assert.Equal(t, godo.MicroVMStateTerminated, copies[1].State)
		assert.Equal(t, godo.MicroVMStateFailed, copies[2].State)
	})
}

func TestMicroVMUpdate_invalidFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags map[string]any
	}{
		{name: "no fields", flags: map[string]any{}},
		{name: "tag and clear-tags", flags: map[string]any{doctl.ArgTag: []string{"prod"}, "clear-tags": true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Args = append(config.Args, testMicroVMID)
				for k, v := range tt.flags {
					config.Doit.Set(config.NS, k, v)
				}
				require.Error(t, RunMicroVMUpdate(config))
			})
		})
	}
}

func TestMicroVMPause(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().Pause(testMicroVMID).Return(&testMicroVM, nil)

		config.Args = append(config.Args, testMicroVMID)
		err := RunMicroVMPause(config)
		require.NoError(t, err)
	})
}

func TestMicroVMResume(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().Resume(testMicroVMID).Return(&testMicroVM, nil)

		config.Args = append(config.Args, testMicroVMID)
		err := RunMicroVMResume(config)
		require.NoError(t, err)
	})
}

func TestMicroVMDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().Delete(testMicroVMID).Return(nil)

		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunMicroVMDelete(config)
		require.NoError(t, err)
	})
}

func TestMicroVMDelete_multiple(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ids := []string{testMicroVMID, "aabbccdd-1111-2222-3333-444455556666"}
		for _, id := range ids {
			tm.microVMs.EXPECT().Delete(id).Return(nil)
		}

		config.Args = append(config.Args, ids...)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunMicroVMDelete(config)
		require.NoError(t, err)
	})
}

func TestMicroVMCheckpointList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().ListCheckpoints(testMicroVMID).Return(
			do.MicroVMCheckpoints{testMicroVMCheckpoint}, nil,
		)

		config.Doit.Set(config.NS, "microvm-id", testMicroVMID)
		err := RunMicroVMCheckpointList(config)
		require.NoError(t, err)
	})
}

func TestMicroVMCheckpointGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().GetCheckpoint(testMicroVMCheckpoint.ID).Return(&testMicroVMCheckpoint, nil)

		config.Args = append(config.Args, testMicroVMCheckpoint.ID)
		err := RunMicroVMCheckpointGet(config)
		require.NoError(t, err)
	})
}

func TestMicroVMCheckpointCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().CreateCheckpoint(testMicroVMID, &godo.MicroVMCheckpointCreateRequest{
			Name: "named",
		}).Return(&testMicroVMCheckpoint, nil)

		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "name", "named")
		err := RunMicroVMCheckpointCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroVMCheckpointDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().DeleteCheckpoint(testMicroVMCheckpoint.ID).Return(nil)

		config.Args = append(config.Args, testMicroVMCheckpoint.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		err := RunMicroVMCheckpointDelete(config)
		require.NoError(t, err)
	})
}

func TestMicroVMOptions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().GetCreateOptions().Return(&godo.MicroVMCreateOptions{
			DefaultRegion: "nyc1",
			Sizes: []godo.MicroVMSizeOption{{
				CPU: 2, Memory: 4096, Disk: 50, Available: true, Regions: []string{"nyc1", "sfo3"},
			}},
		}, nil)
		err := RunMicroVMOptions(config)
		require.NoError(t, err)
	})
}

func TestMicroVMExec(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().Exec(testMicroVMID, &godo.MicroVMExecRequest{
			Argv: []string{"echo", "hi"},
			Cwd:  "/app",
		}).Return(&godo.MicroVMExecResult{
			Stdout:   "hi\n",
			ExitCode: 0,
		}, nil)

		config.Args = append(config.Args, testMicroVMID, "echo", "hi")
		config.Doit.Set(config.NS, "cwd", "/app")
		err := RunMicroVMExec(config)
		require.NoError(t, err)
	})
}

func TestMicroVMUpload(t *testing.T) {
	content := []byte("hello microvm\n")
	wantSHA := fmt.Sprintf("%x", sha256.Sum256(content))
	local := filepath.Join(t.TempDir(), "in.txt")
	require.NoError(t, os.WriteFile(local, content, 0o600))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().UploadFile(testMicroVMID, gomock.Any()).DoAndReturn(
			func(_ string, req *godo.MicroVMFileUploadRequest) (*godo.MicroVMFileInfo, error) {
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				assert.Equal(t, content, body)
				assert.Equal(t, "/app/in.txt", req.Path)
				assert.Equal(t, int64(len(content)), req.Size)
				assert.Equal(t, "0600", req.Mode)
				assert.Equal(t, wantSHA, req.SHA256)
				require.NotNil(t, req.UID)
				require.NotNil(t, req.GID)
				assert.Equal(t, uint32(1000), *req.UID)
				assert.Equal(t, uint32(1001), *req.GID)
				return &godo.MicroVMFileInfo{Path: req.Path, SizeBytes: uint64(req.Size), SHA256: req.SHA256, Mode: req.Mode, UID: 1000, GID: 1001}, nil
			})

		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "local-file", local)
		config.Doit.Set(config.NS, "path", "/app/in.txt")
		config.Doit.Set(config.NS, "mode", "0600")
		config.Doit.Set(config.NS, "uid", 1000)
		config.Doit.Set(config.NS, "gid", 1001)
		require.NoError(t, RunMicroVMUpload(config))
	})
}

func TestMicroVMUploadUIDWithoutGID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "local-file", "/dev/null")
		config.Doit.Set(config.NS, "path", "/app/x")
		config.Doit.Set(config.NS, "uid", 1000)
		err := RunMicroVMUpload(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--uid and --gid")
	})
}

func TestMicroVMDownload(t *testing.T) {
	content := "downloaded bytes\n"
	saveTo := filepath.Join(t.TempDir(), "out.txt")

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().DownloadFile(testMicroVMID, "/app/out.txt").Return(&godo.MicroVMFileDownload{
			Body:      io.NopCloser(strings.NewReader(content)),
			SizeBytes: int64(len(content)),
		}, nil)

		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "path", "/app/out.txt")
		config.Doit.Set(config.NS, "save-to", saveTo)
		require.NoError(t, RunMicroVMDownload(config))

		got, err := os.ReadFile(saveTo)
		require.NoError(t, err)
		assert.Equal(t, content, string(got))
	})
}

func TestMicroVMDownloadBodyErrorLeavesNoFile(t *testing.T) {
	saveTo := filepath.Join(t.TempDir(), "out.txt")

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().DownloadFile(testMicroVMID, "/app/out.txt").Return(&godo.MicroVMFileDownload{
			Body: io.NopCloser(io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(godo.ErrMicroVMFileChecksum))),
		}, nil)

		config.Args = append(config.Args, testMicroVMID)
		config.Doit.Set(config.NS, "path", "/app/out.txt")
		config.Doit.Set(config.NS, "save-to", saveTo)
		err := RunMicroVMDownload(config)
		require.ErrorIs(t, err, godo.ErrMicroVMFileChecksum)

		_, statErr := os.Stat(saveTo)
		assert.True(t, os.IsNotExist(statErr), "partial file must not be left at save-to")
		entries, _ := os.ReadDir(filepath.Dir(saveTo))
		assert.Empty(t, entries, "temp file must be cleaned up")
	})
}

func TestMicroVMExecNonZeroExit(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microVMs.EXPECT().Exec(testMicroVMID, &godo.MicroVMExecRequest{
			Argv: []string{"false"},
		}).Return(&godo.MicroVMExecResult{
			ExitCode: 1,
		}, nil)

		config.Args = append(config.Args, testMicroVMID, "false")
		err := RunMicroVMExec(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exit")
	})
}
