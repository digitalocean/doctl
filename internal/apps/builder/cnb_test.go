package builder

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/golang/mock/gomock"
	"github.com/kballard/go-shellquote"
	"github.com/stretchr/testify/require"
)

func TestCNBComponentBuild(t *testing.T) {
	ctx := context.Background()
	dockerSocketPath = filepath.Join(t.TempDir(), "docker.sock")
	require.NoError(t, ioutil.WriteFile(dockerSocketPath, nil, 0644))

	t.Run("no component", func(t *testing.T) {
		builder := &CNBComponentBuilder{}
		_, err := builder.Build(ctx)
		require.ErrorContains(t, err, "no component")
	})

	t.Run("happy path - service", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		service := &godo.AppServiceSpec{
			SourceDir: "./subdir",
			Name:      "web",
			Envs: []*godo.AppVariableDefinition{
				{
					Key:   "build-arg-1",
					Value: "build-val-1",
					Type:  godo.AppVariableType_General,
					Scope: godo.AppVariableScope_BuildTime,
				},
				{
					Key:   "override-1",
					Value: "newval",
				},
				{
					Key:   "useroverride-1",
					Value: "newval",
				},
				{
					Key:   "run-build-arg-1",
					Value: "run-build-val-1",
					Type:  godo.AppVariableType_General,
					Scope: godo.AppVariableScope_RunAndBuildTime,
				},
				{
					Key:   "run-arg-1",
					Value: "run-val-1",
					Type:  godo.AppVariableType_General,
					Scope: godo.AppVariableScope_RunTime,
				},
				{
					Key:   "secret-arg-1",
					Value: "secret-val-1",
					Type:  godo.AppVariableType_Secret,
				},
			},
		}
		spec := &godo.AppSpec{
			Services: []*godo.AppServiceSpec{service},
			Envs: []*godo.AppVariableDefinition{
				{
					Key:   "override-1",
					Value: "override-1",
				},
			},
		}

		mockClient := NewMockDockerEngineClient(ctrl)
		builder := &CNBComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				cli:       mockClient,
				spec:      spec,
				component: service,
				envOverrides: map[string]string{
					"useroverride-1": "newval",
				},
				buildCommandOverride: "custom build command",
				contextDir:           t.TempDir(),
			},
			cnbComponent:  service,
			localCacheDir: "/cache",
			versioning: CNBVersioning{
				Buildpacks: []*Buildpack{{ID: "do/bp", Version: "1.0.0"}},
			},
		}

		buildID := "build-id"
		mockClient.EXPECT().ContainerCreate(ctx, gomock.Any(), gomock.Any(), nil, nil, "").Return(container.ContainerCreateCreatedBody{
			ID: buildID,
		}, nil)

		mockClient.EXPECT().ContainerRemove(gomock.Any(), buildID, types.ContainerRemoveOptions{
			Force: true,
		}).Return(nil)
		mockClient.EXPECT().ContainerStart(ctx, buildID, types.ContainerStartOptions{}).Return(nil)
		mockClient.EXPECT().ImageList(ctx, types.ImageListOptions{
			Filters: filters.NewArgs(filters.Arg("reference", builder.AppImageOutputName())),
		}).Return([]types.ImageSummary{{ /*single entry*/ }}, nil)

		execID := "exec-id"
		mockClient.EXPECT().ContainerExecCreate(ctx, buildID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			Env: []string{
				"APP_CACHE_DIR=" + cnbCacheDir,
				"APP_IMAGE_URL=" + builder.AppImageOutputName(),
				"APP_PLATFORM_COMPONENT_TYPE=" + string(service.GetType()),
				appVarAllowListKey + "=build-arg-1,override-1,run-build-arg-1,useroverride-1",
				appVarPrefix + "build-arg-1=build-val-1",
				appVarPrefix + "override-1=newval",
				appVarPrefix + "run-build-arg-1=run-build-val-1",
				appVarPrefix + "useroverride-1=newval",
				"BUILD_COMMAND=" + builder.buildCommandOverride,
				"CNB_UPLOAD_RETRY=1",
				"PREVIOUS_APP_IMAGE_URL=" + builder.AppImageOutputName(),
				"SOURCE_DIR=" + service.GetSourceDir(),
				`VERSION_PINNING_LIST=[{"id":"do/bp","version":"1.0.0"}]`,
			},
			Cmd: []string{"sh", "-c", "/.app_platform/build.sh"},
		}).Return(types.IDResponse{
			ID: execID,
		}, nil)

		// NOTE: we use net.Pipe as a simple way to create an in-memory
		// net.Conn resource so we can safely validate the HijackedResponse.
		c1, c2 := net.Pipe()
		defer c2.Close()
		mockClient.EXPECT().ContainerExecAttach(ctx, execID, types.ExecStartCheck{}).Return(types.HijackedResponse{
			Reader: bufio.NewReader(strings.NewReader("")),
			Conn:   c1,
		}, nil)

		mockClient.EXPECT().ContainerExecInspect(gomock.Any(), execID).Return(types.ContainerExecInspect{
			ExitCode: 0,
		}, nil)

		_, err := builder.Build(ctx)
		require.NoError(t, err)
	})

	t.Run("copy on write", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		service := &godo.AppServiceSpec{
			SourceDir: "./subdir",
			Name:      "web",
		}
		spec := &godo.AppSpec{
			Services: []*godo.AppServiceSpec{service},
		}

		mockClient := NewMockDockerEngineClient(ctrl)
		builder := &CNBComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				cli:                  mockClient,
				spec:                 spec,
				component:            service,
				contextDir:           t.TempDir(),
				copyOnWriteSemantics: true,
			},
			cnbComponent: service,
		}

		buildID := "build-id"
		mockClient.EXPECT().ContainerCreate(ctx, gomock.Any(), gomock.Any(), nil, nil, "").Return(container.ContainerCreateCreatedBody{
			ID: buildID,
		}, nil)

		mockClient.EXPECT().ContainerRemove(gomock.Any(), buildID, types.ContainerRemoveOptions{
			Force: true,
		}).Return(nil)
		mockClient.EXPECT().ContainerStart(ctx, buildID, types.ContainerStartOptions{}).Return(nil)

		mockClient.EXPECT().CopyToContainer(ctx, buildID, filepath.Clean("/"), gomock.Any(), gomock.Any()).Return(nil)

		mockClient.EXPECT().ImageList(ctx, types.ImageListOptions{
			Filters: filters.NewArgs(filters.Arg("reference", builder.AppImageOutputName())),
		}).Return([]types.ImageSummary{ /*no entries*/ }, nil)

		execID := "exec-id"
		mockClient.EXPECT().ContainerExecCreate(ctx, buildID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			Env: []string{
				"APP_IMAGE_URL=" + builder.AppImageOutputName(),
				"APP_PLATFORM_COMPONENT_TYPE=" + string(service.GetType()),
				"CNB_UPLOAD_RETRY=1",
				"SOURCE_DIR=" + service.GetSourceDir(),
			},
			Cmd: []string{"sh", "-c", "/.app_platform/build.sh"},
		}).Return(types.IDResponse{
			ID: execID,
		}, nil)

		// NOTE: we use net.Pipe as a simple way to create an in-memory
		// net.Conn resource so we can safely validate the HijackedResponse.
		c1, c2 := net.Pipe()
		defer c2.Close()
		mockClient.EXPECT().ContainerExecAttach(ctx, execID, types.ExecStartCheck{}).Return(types.HijackedResponse{
			Reader: bufio.NewReader(strings.NewReader("")),
			Conn:   c1,
		}, nil)

		mockClient.EXPECT().ContainerExecInspect(gomock.Any(), execID).Return(types.ContainerExecInspect{
			ExitCode: 0,
		}, nil)

		_, err := builder.Build(ctx)
		require.NoError(t, err)
	})

	t.Run("override unrecognized env", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		service := &godo.AppServiceSpec{
			SourceDir: "./subdir",
			Name:      "web",
			Envs: []*godo.AppVariableDefinition{
				{
					Key:   "build-arg-1",
					Value: "build-val-1",
					Type:  godo.AppVariableType_General,
					Scope: godo.AppVariableScope_BuildTime,
				},
			},
		}
		spec := &godo.AppSpec{
			Services: []*godo.AppServiceSpec{service},
		}

		mockClient := NewMockDockerEngineClient(ctrl)
		builder := &CNBComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				cli:       mockClient,
				spec:      spec,
				component: service,
				envOverrides: map[string]string{
					"useroverride-1": "newval",
				},
			},
			cnbComponent: service,
		}

		_, err := builder.Build(ctx)
		require.EqualError(t, err, "configuring environment variables: variable not in found in app spec: useroverride-1")
	})

	t.Run("happy path - static site", func(t *testing.T) {
		var wg sync.WaitGroup
		ctrl := gomock.NewController(t)
		site := &godo.AppStaticSiteSpec{
			SourceDir:    "./subdir",
			Name:         "web",
			OutputDir:    "public",
			BuildCommand: "npm run build",
			Envs: []*godo.AppVariableDefinition{
				{
					Key:   "build-arg-1",
					Value: "build-val-1",
					Type:  godo.AppVariableType_General,
					Scope: godo.AppVariableScope_BuildTime,
				},
			},
		}
		spec := &godo.AppSpec{
			StaticSites: []*godo.AppStaticSiteSpec{site},
		}

		mockClient := NewMockDockerEngineClient(ctrl)
		builder := &CNBComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				cli:        mockClient,
				spec:       spec,
				component:  site,
				contextDir: t.TempDir(),
			},
			cnbComponent:  site,
			localCacheDir: "/cache",
		}

		buildID := "build-id"
		mockClient.EXPECT().ContainerCreate(ctx, gomock.Any(), gomock.Any(), nil, nil, "").Return(container.ContainerCreateCreatedBody{
			ID: buildID,
		}, nil)

		mockClient.EXPECT().ContainerRemove(gomock.Any(), buildID, types.ContainerRemoveOptions{
			Force: true,
		}).Return(nil)
		mockClient.EXPECT().ContainerStart(ctx, buildID, types.ContainerStartOptions{}).Return(nil)
		mockClient.EXPECT().ImageList(ctx, types.ImageListOptions{
			Filters: filters.NewArgs(filters.Arg("reference", builder.AppImageOutputName())),
		}).Return([]types.ImageSummary{{ /*single entry*/ }}, nil)

		// app image build

		mockClient.EXPECT().ContainerExecCreate(ctx, buildID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			Env: []string{
				"APP_CACHE_DIR=" + cnbCacheDir,
				"APP_IMAGE_URL=" + builder.AppImageOutputName(),
				"APP_PLATFORM_COMPONENT_TYPE=" + string(site.GetType()),
				appVarAllowListKey + "=build-arg-1",
				appVarPrefix + "build-arg-1=build-val-1",
				"BUILD_COMMAND=npm run build",
				"CNB_UPLOAD_RETRY=1",
				"PREVIOUS_APP_IMAGE_URL=" + builder.AppImageOutputName(),
				"SOURCE_DIR=" + site.GetSourceDir(),
			},
			Cmd: []string{"sh", "-c", "/.app_platform/build.sh"},
		}).Return(types.IDResponse{
			ID: "build-exec-id",
		}, nil)

		// NOTE: we use net.Pipe as a simple way to create an in-memory
		// net.Conn resource so we can safely validate the HijackedResponse.
		c1, c2 := net.Pipe()
		defer c2.Close()
		mockClient.EXPECT().ContainerExecAttach(ctx, "build-exec-id", types.ExecStartCheck{}).Return(types.HijackedResponse{
			Reader: bufio.NewReader(strings.NewReader("")),
			Conn:   c1,
		}, nil)
		mockClient.EXPECT().ContainerExecInspect(gomock.Any(), "build-exec-id").Return(types.ContainerExecInspect{
			ExitCode: 0,
		}, nil)

		// static site image build

		// cat workspace path
		mockClient.EXPECT().ContainerExecCreate(ctx, buildID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			Cmd:          []string{"cat", "/.app_platform/local/WORKSPACE_PATH"},
		}).Return(types.IDResponse{
			ID: "workspace-path-exec-id",
		}, nil)
		mockClient.EXPECT().ContainerExecAttach(ctx, "workspace-path-exec-id", types.ExecStartCheck{}).Return(types.HijackedResponse{
			Reader: testExecSimpleBody("/workspace"),
			Conn:   c1,
		}, nil)
		mockClient.EXPECT().ContainerExecInspect(gomock.Any(), "workspace-path-exec-id").Return(types.ContainerExecInspect{
			ExitCode: 0,
		}, nil)

		// cat assets path
		mockClient.EXPECT().ContainerExecCreate(ctx, buildID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			Cmd:          []string{"cat", "/.app_platform/local/ASSETS_PATH"},
		}).Return(types.IDResponse{
			ID: "assets-path-exec-id",
		}, nil)
		mockClient.EXPECT().ContainerExecAttach(ctx, "assets-path-exec-id", types.ExecStartCheck{}).Return(types.HijackedResponse{
			Reader: testExecSimpleBody("/workspace/public"),
			Conn:   c1,
		}, nil)
		mockClient.EXPECT().ContainerExecInspect(gomock.Any(), "assets-path-exec-id").Return(types.ContainerExecInspect{
			ExitCode: 0,
		}, nil)

		// write nginx conf
		mockClient.EXPECT().ContainerExecCreate(ctx, buildID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			AttachStdin:  true,
			Cmd:          []string{"sh", "-c", "cat >/workspace/nginx.conf"},
		}).Return(types.IDResponse{
			ID: "nginx-conf-exec-id",
		}, nil)
		nginxConfR, nginxConfW := net.Pipe()
		wg.Add(1)
		go func() {
			defer wg.Done()
			nginxConf, err := ioutil.ReadAll(nginxConfR)
			require.NoError(t, err)
			require.Equal(t, `
server {
	listen 8080;
	listen [::]:8080;

	resolver 127.0.0.11;
	autoindex off;

	server_name _;
	server_tokens off;

	root /www;
	gzip_static on;
}
`, string(nginxConf))
		}()
		mockClient.EXPECT().ContainerExecAttach(ctx, "nginx-conf-exec-id", types.ExecStartCheck{}).Return(types.HijackedResponse{
			Reader: testExecSimpleBody("/workspace/public"),
			Conn:   nginxConfW,
		}, nil)
		mockClient.EXPECT().ContainerExecInspect(gomock.Any(), "nginx-conf-exec-id").Return(types.ContainerExecInspect{
			ExitCode: 0,
		}, nil)

		// write static dockerfile
		mockClient.EXPECT().ContainerExecCreate(ctx, buildID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			AttachStdin:  true,
			Cmd:          []string{"sh", "-c", "cat >/workspace/Dockerfile.static"},
		}).Return(types.IDResponse{
			ID: "dockerfile.static-exec-id",
		}, nil)
		dockerfileR, dockerfileW := net.Pipe()
		wg.Add(1)
		go func() {
			defer wg.Done()
			dockerfile, err := ioutil.ReadAll(dockerfileR)
			require.NoError(t, err)
			require.Equal(t, `
ARG nginx_image
ARG assets_path
FROM ${nginx_image}

COPY ${assets_path} /www
RUN test -f /www/nginx.conf && rm -f /www/nginx.conf

COPY ./nginx.conf /etc/nginx/conf.d/default.conf
`, string(dockerfile))
		}()
		mockClient.EXPECT().ContainerExecAttach(ctx, "dockerfile.static-exec-id", types.ExecStartCheck{}).Return(types.HijackedResponse{
			Reader: testExecSimpleBody("/workspace/public"),
			Conn:   dockerfileW,
		}, nil)
		mockClient.EXPECT().ContainerExecInspect(gomock.Any(), "dockerfile.static-exec-id").Return(types.ContainerExecInspect{
			ExitCode: 0,
		}, nil)

		// run static image build
		mockClient.EXPECT().ContainerExecCreate(ctx, buildID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			Cmd: []string{"sh", "-c", shellquote.Join(
				"docker", "build",
				"-t", builder.StaticSiteImageOutputName(),
				"-f", "/workspace/Dockerfile.static",
				"--build-arg", "assets_path=./public/",
				"--build-arg", "nginx_image="+StaticSiteNginxImage,
				"/workspace",
			)},
		}).Return(types.IDResponse{
			ID: "assets-path-exec-id",
		}, nil)
		mockClient.EXPECT().ContainerExecAttach(ctx, "assets-path-exec-id", types.ExecStartCheck{}).Return(types.HijackedResponse{
			Reader: testExecSimpleBody("/workspace/public"),
			Conn:   c1,
		}, nil)
		mockClient.EXPECT().ContainerExecInspect(gomock.Any(), "assets-path-exec-id").Return(types.ContainerExecInspect{
			ExitCode: 0,
		}, nil)

		_, err := builder.Build(ctx)
		require.NoError(t, err)
		wg.Wait()
	})
}

func testExecBody(fn func(stdout, stderr io.Writer)) *bufio.Reader {
	var buf bytes.Buffer
	stdout := stdcopy.NewStdWriter(&buf, stdcopy.Stdout)
	stderr := stdcopy.NewStdWriter(&buf, stdcopy.Stderr)
	fn(stdout, stderr)
	return bufio.NewReader(&buf)
}

func testExecSimpleBody(s string) *bufio.Reader {
	return testExecBody(func(stdout, stderr io.Writer) {
		stdout.Write([]byte(s))
	})
}
