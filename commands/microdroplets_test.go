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
			Size:       "md-1vcpu-512mb",
			Networking: godo.MicroDropletNetworkingPublic,
			Image:      "do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000",
			Endpoint:   "https://sammy.microdroplets.digitalocean.app",
			Created:    "2026-07-16T10:00:00Z",
		},
	}

	testMicroDropletList = do.MicroDroplets{testMicroDroplet}

	testMicroDropletSnapshot = do.MicroDropletSnapshot{
		MicroDropletSnapshot: &godo.MicroDropletSnapshot{
			ID:             "8f7a9a3f-5555-4444-9999-000000000001",
			MicroDropletID: testMicroDropletID,
			Name:           "sammy-snap",
			Status:         godo.MicroDropletSnapshotStatusAvailable,
			MemoryBytes:    512 * 1024 * 1024,
			DiskBytes:      1024 * 1024 * 1024,
			Created:        "2026-07-16T10:05:00Z",
		},
	}

	testMicroDropletImage = do.MicroDropletImage{
		MicroDropletImage: &godo.MicroDropletImage{
			ID:      "aa11bb22-cc33-dd44-ee55-ff6600000000",
			Name:    "hello-world",
			Source:  "docker.io/library/hello-world:latest",
			Status:  godo.MicroDropletImageStatusAvailable,
			Created: "2026-07-16T09:00:00Z",
		},
	}
)

func TestMicroDropletCommand(t *testing.T) {
	cmd := MicroDroplet()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"create", "delete", "get", "image", "list", "pause", "resume", "snapshots",
	)
}

func TestMicroDropletImageCommand(t *testing.T) {
	cmd := microDropletImages()
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
			Name:         "sammy-microdroplet",
			Region:       "nyc1",
			Size:         "md-1vcpu-512mb",
			Image:        "do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000",
			Networking:   godo.MicroDropletNetworkingVPC,
			VPCUUID:      "vpc-uuid-1234",
			AutoPause:    &godo.AutoPauseConfig{Enabled: &enabled, IdleTimeout: "5m"},
			AutoResume:   &enabled,
			HTTPPort:     8080,
			HTTPProtocol: godo.MicroDropletHTTPProtocolHTTPS,
			Environment:  map[string]string{"FOO": "bar", "BAZ": "qux"},
			Tags:         []string{"prod", "web"},
		}
		tm.microDroplets.EXPECT().Create(expected).Return(&testMicroDroplet, nil)

		config.Args = append(config.Args, "sammy-microdroplet")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "md-1vcpu-512mb")
		config.Doit.Set(config.NS, doctl.ArgImage, "do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000")
		config.Doit.Set(config.NS, "networking", "vpc")
		config.Doit.Set(config.NS, doctl.ArgVPCUUID, "vpc-uuid-1234")
		config.Doit.Set(config.NS, "auto-pause", true)
		config.Doit.Set(config.NS, "auto-pause-idle-timeout", "5m")
		config.Doit.Set(config.NS, "auto-resume", true)
		config.Doit.Set(config.NS, "http-port", 8080)
		config.Doit.Set(config.NS, "http-protocol", "https")
		config.Doit.Set(config.NS, "env", []string{"FOO=bar", "BAZ=qux"})
		config.Doit.Set(config.NS, doctl.ArgTag, []string{"prod", "web"})

		err := RunMicroDropletCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletCreate_minimal(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expected := &godo.MicroDropletCreateRequest{
			Name:   "sammy-microdroplet",
			Region: "nyc1",
			Size:   "md-1vcpu-512mb",
			Image:  "do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000",
		}
		tm.microDroplets.EXPECT().Create(expected).Return(&testMicroDroplet, nil)

		config.Args = append(config.Args, "sammy-microdroplet")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "md-1vcpu-512mb")
		config.Doit.Set(config.NS, doctl.ArgImage, "do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000")

		err := RunMicroDropletCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletCreate_badEnv(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "sammy-microdroplet")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc1")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "md-1vcpu-512mb")
		config.Doit.Set(config.NS, doctl.ArgImage, "do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000")
		config.Doit.Set(config.NS, "env", []string{"MALFORMED"})

		err := RunMicroDropletCreate(config)
		require.Error(t, err)
	})
}

func TestMicroDropletPause(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expected := &godo.MicroDropletUpdateRequest{State: godo.MicroDropletStatePaused}
		tm.microDroplets.EXPECT().Update(testMicroDropletID, expected).Return(&testMicroDroplet, nil)

		config.Args = append(config.Args, testMicroDropletID)
		err := RunMicroDropletPause(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletResume(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expected := &godo.MicroDropletUpdateRequest{State: godo.MicroDropletStateRunning}
		tm.microDroplets.EXPECT().Update(testMicroDropletID, expected).Return(&testMicroDroplet, nil)

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

func TestMicroDropletSnapshots(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDroplets.EXPECT().ListSnapshots(testMicroDropletID).Return(
			do.MicroDropletSnapshots{testMicroDropletSnapshot}, nil,
		)

		config.Args = append(config.Args, testMicroDropletID)
		err := RunMicroDropletSnapshots(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletImageList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDropletImages.EXPECT().List().Return(
			do.MicroDropletImages{testMicroDropletImage}, nil,
		)
		err := RunMicroDropletImageList(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletImageGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDropletImages.EXPECT().Get(testMicroDropletImage.ID).Return(&testMicroDropletImage, nil)

		config.Args = append(config.Args, testMicroDropletImage.ID)
		err := RunMicroDropletImageGet(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletImageCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		expected := &godo.MicroDropletImageCreateRequest{
			Name:   "hello-world",
			Source: "docker.io/library/hello-world:latest",
		}
		tm.microDropletImages.EXPECT().Create(expected).Return(&testMicroDropletImage, nil)

		config.Args = append(config.Args, "hello-world")
		config.Doit.Set(config.NS, "source", "docker.io/library/hello-world:latest")

		err := RunMicroDropletImageCreate(config)
		require.NoError(t, err)
	})
}

func TestMicroDropletImageDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.microDropletImages.EXPECT().Delete(testMicroDropletImage.ID).Return(nil)

		config.Args = append(config.Args, testMicroDropletImage.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunMicroDropletImageDelete(config)
		require.NoError(t, err)
	})
}
