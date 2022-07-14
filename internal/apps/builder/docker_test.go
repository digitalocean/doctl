package builder

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/docker/docker/api/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string {
	return &s
}

func TestDockerComponentBuild(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	t.Run("no component", func(t *testing.T) {
		builder := &DockerComponentBuilder{}
		_, err := builder.Build(ctx)
		require.ErrorContains(t, err, "no component")
	})

	t.Run("happy path", func(t *testing.T) {
		service := &godo.AppServiceSpec{
			DockerfilePath: "./Dockerfile",
			SourceDir:      "./subdir",
			Name:           "web",
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

		mockClient := NewMockContainerEngineClient(ctrl)
		builder := &DockerComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				cli:       mockClient,
				spec:      spec,
				component: service,
			},
		}

		mockClient.EXPECT().ImageBuild(ctx, gomock.Any(), types.ImageBuildOptions{
			Dockerfile: service.DockerfilePath,
			Tags: []string{
				builder.ImageOutputName(),
			},
			BuildArgs: map[string]*string{
				"APP_PLATFORM_COMPONENT_TYPE": strPtr(string(service.GetType())),
				"SOURCE_DIR":                  strPtr(service.GetSourceDir()),
				"build-arg-1":                 strPtr("build-val-1"),
				"override-1":                  strPtr("newval"),
				"run-build-arg-1":             strPtr("run-build-val-1"),
			},
		}).Return(types.ImageBuildResponse{
			Body: ioutil.NopCloser(strings.NewReader("")),
		}, nil)

		_, err := builder.Build(ctx)
		require.NoError(t, err)
	})
}
