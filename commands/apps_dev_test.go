package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/apps/builder"
	"github.com/digitalocean/godo"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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
