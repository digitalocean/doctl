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
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testMicroDropletID = "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2"

	testMicroDroplet = do.MicroDroplet{
		MicroDroplet: &godo.MicroDroplet{
			ID:         testMicroDropletID,
			Name:       "sammy-microdroplet",
			Region:     "nyc1",
			State:      godo.MicroDropletStateRunning,
			Size:       &godo.MicroDropletSize{CPU: 2, Memory: 4096, Disk: 80},
			Networking: godo.MicroDropletNetworkingPublic,
			Source:     &godo.MicroDropletSource{OCIRef: "docker.io/library/nginx:1.27"},
			URLs: []godo.MicroDropletURL{
				{Hostname: "sammy.example.com", Port: 8080, Default: true, Status: godo.MicroDropletURLStatusActive},
			},
			Ports:   []uint32{8080},
			Created: "2026-07-16T10:00:00Z",
		},
	}

	testMicroDropletList = do.MicroDroplets{testMicroDroplet}

	testMicroDropletCheckpoint = do.MicroDropletCheckpoint{
		MicroDropletCheckpoint: &godo.MicroDropletCheckpoint{
			ID:               "8f7a9a3f-5555-4444-9999-000000000001",
			MicroDropletID:   testMicroDropletID,
			MicroDropletName: "sammy-microdroplet",
			Name:             "sammy-checkpoint",
			Region:           "nyc1",
			Status:           godo.MicroDropletCheckpointStatusAvailable,
			MemoryBytes:      512 * 1024 * 1024,
			DiskBytes:        1024 * 1024 * 1024,
			Created:          "2026-07-16T10:05:00Z",
		},
	}
)

func TestMicroDropletCommand(t *testing.T) {
	cmd := MicroDroplet()
	assert.NotNil(t, cmd)
	assert.True(t, cmd.Hidden)
	assertCommandNames(t, cmd,
		"checkpoint", "create", "delete", "get", "list", "options", "pause", "resume",
	)
}

func TestMicroDropletCheckpointCommand(t *testing.T) {
	cmd := microDropletCheckpoints()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "delete", "get", "list")
}

func TestMicroDropletsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().List().Return(testMicroDropletList, nil)
		err := RunMicroDropletList(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletsListByRegion(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().ListByRegion("nyc1").Return(testMicroDropletList, nil)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		err := RunMicroDropletList(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().Get(testMicroDropletID).Return(&testMicroDroplet, nil)
		config.Args = append(config.Args, testMicroDropletID)
		err := RunMicroDropletGet(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletGet_missingArg(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunMicroDropletGet(config)
		require.Error(t, err)
	})
}

func TestMicroDropletCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		enabled := true
		expected := &godo.MicroDropletCreateRequest{
			Name:   "sammy-microdroplet",
			Region: "nyc1",
			Size:   &godo.MicroDropletSizeRequest{CPU: 2, Memory: 4096},
			Source: &godo.MicroDropletSource{OCIRef: "docker.io/library/nginx:1.27"},
			Networking:   godo.MicroDropletNetworkingVPC,
			VPCUUID:      "vpc-uuid-1234",
			AutoPause:    &godo.AutoPauseConfig{Enabled: &enabled, IdleTimeout: "5m"},
			AutoResume:   &enabled,
			HTTPPort:     8080,
			HTTPProtocol: godo.MicroDropletHTTPProtocolHTTP2,
			Ports:        []uint32{80, 8080},
			Environment:  map[string]string{"FOO": "bar", "BAZ": "qux"},
			Tags:         []string{"prod", "web"},
		}
		tm.microDroplets.EXPECT().Create(expected).Return(&testMicroDroplet, nil)

		config.Args = append(config.Args, "sammy-microdroplet")
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

		err := RunMicroDropletCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletCreate_fromCheckpoint(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expected := &godo.MicroDropletCreateRequest{
			Name:   "sammy-clone",
			Source: &godo.MicroDropletSource{CheckpointID: testMicroDropletCheckpoint.ID},
		}
		tm.microDroplets.EXPECT().Create(expected).Return(&testMicroDroplet, nil)

		config.Args = append(config.Args, "sammy-clone")
		config.Doit.Set(config.NS, "checkpoint-id", testMicroDropletCheckpoint.ID)

		err := RunMicroDropletCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletCreate_requiresSource(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "sammy-microdroplet")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, "cpu", 2)
		config.Doit.Set(config.NS, "memory", 4096)

		err := RunMicroDropletCreate(config)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exactly one of --oci-ref or --checkpoint-id")
	})
}

func TestMicroDropletCreate_badEnv(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "sammy-microdroplet")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, "cpu", 2)
		config.Doit.Set(config.NS, "memory", 4096)
		config.Doit.Set(config.NS, "oci-ref", "docker.io/library/nginx:1.27")
		config.Doit.Set(config.NS, "env", []string{"MALFORMED"})

		err := RunMicroDropletCreate(config)
		require.Error(t, err)
	})
}

func TestMicroDropletCreate_idleTimeoutRequiresAutoPause(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "sammy-microdroplet")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, "cpu", 2)
		config.Doit.Set(config.NS, "memory", 4096)
		config.Doit.Set(config.NS, "oci-ref", "docker.io/library/nginx:1.27")
		config.Doit.Set(config.NS, "auto-pause-idle-timeout", "5m")

		err := RunMicroDropletCreate(config)
		require.Error(t, err)
		require.Contains(t, err.Error(), "--auto-pause-idle-timeout requires --auto-pause")
	})
}

func TestMicroDropletPause(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().Pause(testMicroDropletID).Return(&testMicroDroplet, nil)

		config.Args = append(config.Args, testMicroDropletID)
		err := RunMicroDropletPause(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletResume(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().Resume(testMicroDropletID).Return(&testMicroDroplet, nil)

		config.Args = append(config.Args, testMicroDropletID)
		err := RunMicroDropletResume(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().Delete(testMicroDropletID).Return(nil)

		config.Args = append(config.Args, testMicroDropletID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunMicroDropletDelete(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletDelete_multiple(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ids := []string{testMicroDropletID, "aabbccdd-1111-2222-3333-444455556666"}
		for _, id := range ids {
			tm.microDroplets.EXPECT().Delete(id).Return(nil)
		}

		config.Args = append(config.Args, ids...)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunMicroDropletDelete(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletCheckpointList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().ListCheckpoints(testMicroDropletID).Return(
			do.MicroDropletCheckpoints{testMicroDropletCheckpoint}, nil,
		)

		config.Doit.Set(config.NS, "microdroplet-id", testMicroDropletID)
		err := RunMicroDropletCheckpointList(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletCheckpointGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().GetCheckpoint(testMicroDropletCheckpoint.ID).Return(&testMicroDropletCheckpoint, nil)

		config.Args = append(config.Args, testMicroDropletCheckpoint.ID)
		err := RunMicroDropletCheckpointGet(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletCheckpointCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().CreateCheckpoint(testMicroDropletID, &godo.MicroDropletCheckpointCreateRequest{
			Name: "named",
		}).Return(&testMicroDropletCheckpoint, nil)

		config.Args = append(config.Args, testMicroDropletID)
		config.Doit.Set(config.NS, "name", "named")
		err := RunMicroDropletCheckpointCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletCheckpointDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().DeleteCheckpoint(testMicroDropletCheckpoint.ID).Return(nil)

		config.Args = append(config.Args, testMicroDropletCheckpoint.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		err := RunMicroDropletCheckpointDelete(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletOptions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().GetCreateOptions().Return(&godo.MicroDropletCreateOptions{
			DefaultRegion: "nyc1",
			Regions:       []godo.MicroDropletRegionOption{{Slug: "nyc1", Available: true}},
		}, nil)
		err := RunMicroDropletOptions(config)
		require.NoError(t, err)
	})
}
