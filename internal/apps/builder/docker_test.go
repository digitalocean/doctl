package builder

import (
	"bytes"
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/digitalocean/doctl/commands/charm/text"
	"github.com/digitalocean/godo"
	"github.com/docker/docker/api/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

		mockClient := NewMockDockerEngineClient(ctrl)
		var logBuf bytes.Buffer
		builder := &DockerComponentBuilder{
			baseComponentBuilder: baseComponentBuilder{
				cli:                  mockClient,
				spec:                 spec,
				component:            service,
				buildCommandOverride: "test",
				logWriter:            &logBuf,
				noCache:              true,
			},
		}

		mockClient.EXPECT().ImageBuild(ctx, gomock.Any(), types.ImageBuildOptions{
			Dockerfile: service.DockerfilePath,
			Tags: []string{
				builder.AppImageOutputName(),
			},
			BuildArgs: map[string]*string{
				"build-arg-1":     strPtr("build-val-1"),
				"override-1":      strPtr("newval"),
				"run-build-arg-1": strPtr("run-build-val-1"),
			},
			NoCache: true,
		}).Return(types.ImageBuildResponse{
			Body: ioutil.NopCloser(strings.NewReader("")),
		}, nil)

		_, err := builder.Build(ctx)
		require.NoError(t, err)

		assert.Contains(t, logBuf.String(), text.Crossmark.String()+" build command overrides are ignored for Dockerfile based builds")
	})
}

type logWriter struct {
	t *testing.T
}

func (lw logWriter) Write(data []byte) (int, error) {
	lw.t.Log(string(data))
	return len(data), nil
}
