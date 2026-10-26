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
	"fmt"
	"io"
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
		"checkpoint", "console", "create", "delete", "download", "exec", "get", "list", "options", "pause", "resume", "upload", "url",
	)
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
