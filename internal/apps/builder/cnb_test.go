package builder

import (
	"bufio"
	"context"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCNBComponentBuild(t *testing.T) {
	ctx := context.Background()

	t.Run("no component", func(t *testing.T) {
		builder := &CNBComponentBuilder{}
		_, err := builder.Build(ctx)
		require.ErrorContains(t, err, "no component")
	})

	t.Run("happy path", func(t *testing.T) {
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
			},
		}

		buildID := "build-id"
		mockClient.EXPECT().ContainerCreate(ctx, gomock.Any(), gomock.Any(), nil, nil, "").Return(container.ContainerCreateCreatedBody{
			ID: buildID,
		}, nil)

		mockClient.EXPECT().ContainerRemove(ctx, buildID, types.ContainerRemoveOptions{
			Force: true,
		}).Return(nil)
		mockClient.EXPECT().ContainerStart(ctx, buildID, types.ContainerStartOptions{}).Return(nil)

		execID := "exec-id"
		mockClient.EXPECT().ContainerExecCreate(ctx, buildID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			Env: []string{
				"APP_IMAGE_URL=" + builder.ImageOutputName(),
				"APP_PLATFORM_COMPONENT_TYPE=" + string(service.GetType()),
				appVarAllowListKey + "=build-arg-1,override-1,run-build-arg-1,useroverride-1",
				appVarPrefix + "build-arg-1=build-val-1",
				appVarPrefix + "override-1=newval",
				appVarPrefix + "run-build-arg-1=run-build-val-1",
				appVarPrefix + "useroverride-1=newval",
				"BUILD_COMMAND=" + builder.buildCommandOverride,
				"CNB_UPLOAD_RETRY=1",
				"SOURCE_DIR=" + service.GetSourceDir(),
			},
			Cmd: []string{"sh", "-c", "/.app_platform/build.sh"},
		}).Return(types.IDResponse{
			ID: execID,
		}, nil)

		// NOTE: we use net.Pipe as a simple way to create an in-memory
		// net.Conn resource so we can safley validate the HijackedResponse.
		c1, c2 := net.Pipe()
		defer c2.Close()
		mockClient.EXPECT().ContainerExecAttach(ctx, execID, types.ExecStartCheck{}).Return(types.HijackedResponse{
			Reader: bufio.NewReader(strings.NewReader("")),
			Conn:   c1,
		}, nil)

		mockClient.EXPECT().ContainerExecInspect(ctx, execID).Return(types.ContainerExecInspect{
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

		file, err := ioutil.TempFile("", "dev-config.*.yaml")
		require.NoError(t, err, "creating temp file")
		t.Cleanup(func() {
			file.Close()
			os.Remove(file.Name())
		})

		mockClient := NewMockDockerEngineClient(ctrl)
		builder := &CNBComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				cli:                  mockClient,
				spec:                 spec,
				component:            service,
				contextDir:           filepath.Dir(file.Name()),
				copyOnWriteSemantics: true,
			},
		}

		buildID := "build-id"
		mockClient.EXPECT().ContainerCreate(ctx, gomock.Any(), gomock.Any(), nil, nil, "").Return(container.ContainerCreateCreatedBody{
			ID: buildID,
		}, nil)

		mockClient.EXPECT().ContainerRemove(ctx, buildID, types.ContainerRemoveOptions{
			Force: true,
		}).Return(nil)
		mockClient.EXPECT().ContainerStart(ctx, buildID, types.ContainerStartOptions{}).Return(nil)

		mockClient.EXPECT().CopyToContainer(ctx, buildID, filepath.Clean("/"), gomock.Any(), gomock.Any()).Return(nil)

		execID := "exec-id"
		mockClient.EXPECT().ContainerExecCreate(ctx, buildID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			Env: []string{
				"APP_IMAGE_URL=" + builder.ImageOutputName(),
				"APP_PLATFORM_COMPONENT_TYPE=" + string(service.GetType()),
				"CNB_UPLOAD_RETRY=1",
				"SOURCE_DIR=" + service.GetSourceDir(),
			},
			Cmd: []string{"sh", "-c", "/.app_platform/build.sh"},
		}).Return(types.IDResponse{
			ID: execID,
		}, nil)

		// NOTE: we use net.Pipe as a simple way to create an in-memory
		// net.Conn resource so we can safley validate the HijackedResponse.
		c1, c2 := net.Pipe()
		defer c2.Close()
		mockClient.EXPECT().ContainerExecAttach(ctx, execID, types.ExecStartCheck{}).Return(types.HijackedResponse{
			Reader: bufio.NewReader(strings.NewReader("")),
			Conn:   c1,
		}, nil)

		mockClient.EXPECT().ContainerExecInspect(ctx, execID).Return(types.ContainerExecInspect{
			ExitCode: 0,
		}, nil)

		_, err = builder.Build(ctx)
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
		}

		_, err := builder.Build(ctx)
		require.EqualError(t, err, "configuring environment variables: variable not in found in app spec: useroverride-1")
	})
}
