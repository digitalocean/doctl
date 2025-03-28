package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/apps/builder"
	"github.com/digitalocean/godo"
	"github.com/docker/docker/api/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRunAppsDevBuild(t *testing.T) {
	component := "service"
	appID := uuid.New().String()
	registryName := "test-registry"
	sampleSpec := &godo.AppSpec{
		Services: []*godo.AppServiceSpec{{
			Name:           component,
			DockerfilePath: ".",
		}},
	}

	imageList := []types.ImageSummary{{
		ID:       uuid.New().String(),
		RepoTags: []string{builder.CNBBuilderImage_Heroku22},
		Labels: map[string]string{
			"io.buildpacks.builder.metadata": "{\"stack\":{\"runImage\":{\"image\":\"digitaloceanapps/apps-run:heroku-22_3da9f73\"}}}",
		},
	}}

	t.Run("with local app spec", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			setTempWorkingDir(t)

			specJSON, err := json.Marshal(sampleSpec)
			require.NoError(t, err, "marshalling sample spec")
			specFile := testTempFile(t, []byte(specJSON))

			config.Args = append(config.Args, component)
			config.Doit.Set(config.NS, doctl.ArgAppSpec, specFile)
			config.Doit.Set(config.NS, doctl.ArgRegistry, registryName)
			config.Doit.Set(config.NS, doctl.ArgInteractive, false)

			ws, err := appDevWorkspace(config)
			require.NoError(t, err, "getting workspace")

			tm.appBuilder.EXPECT().Build(gomock.Any()).Return(builder.ComponentBuilderResult{}, nil)
			tm.appBuilderFactory.EXPECT().NewComponentBuilder(gomock.Any(), ws.Context(), sampleSpec, gomock.Any()).Return(tm.appBuilder, nil)
			tm.appDockerEngineClient.EXPECT().ImageList(gomock.Any(), gomock.Any()).Return(imageList, nil).Times(2)

			err = RunAppsDevBuild(config)
			require.NoError(t, err)
		})
	})

	t.Run("with appID", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			setTempWorkingDir(t)

			ws, err := appDevWorkspace(config)
			require.NoError(t, err, "getting workspace")
			tm.appBuilderFactory.EXPECT().NewComponentBuilder(gomock.Any(), ws.Context(), sampleSpec, gomock.Any()).Return(tm.appBuilder, nil)
			tm.appBuilder.EXPECT().Build(gomock.Any()).Return(builder.ComponentBuilderResult{}, nil)
			tm.appDockerEngineClient.EXPECT().ImageList(gomock.Any(), gomock.Any()).Return(imageList, nil).Times(2)

			tm.apps.EXPECT().Get(appID).Times(1).Return(&godo.App{
				Spec: sampleSpec,
			}, nil)

			config.Args = append(config.Args, component)
			config.Doit.Set(config.NS, doctl.ArgApp, appID)
			config.Doit.Set(config.NS, doctl.ArgRegistry, registryName)
			config.Doit.Set(config.NS, doctl.ArgInteractive, false)

			err = RunAppsDevBuild(config)
			require.NoError(t, err)
		})
	})
}

func setTempWorkingDir(t *testing.T) {
	tmp := t.TempDir()
	err := os.Mkdir(filepath.Join(tmp, ".git"), os.ModePerm)
	require.NoError(t, err)
	oldCwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Chdir(oldCwd)
	})
	os.Chdir(tmp)
}
