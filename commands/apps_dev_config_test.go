package commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/apps/workspace"
	"github.com/stretchr/testify/require"
)

func TestRunAppsDevConfigSet(t *testing.T) {
	withTestClient(t, func(cmdConfig *CmdConfig, tm *tcMocks) {
		tcs := []struct {
			name      string
			args      []string
			expectErr error
			expect    func(*testing.T, *workspace.AppDev)
		}{
			{
				name:      "no args",
				args:      []string{},
				expectErr: errors.New("you must provide at least one argument"),
			},
			{
				name:      "unexpected format",
				args:      []string{"only-key"},
				expectErr: errors.New("unexpected arg: only-key"),
			},
			{
				name: "single key",
				args: []string{"registry=docker-registry"},
				expect: func(t *testing.T, ws *workspace.AppDev) {
					require.Equal(t, "docker-registry", ws.Config.Registry, "registry")
				},
			},
			{
				name: "multiple keys",
				args: []string{"registry=docker-registry", "timeout=5m"},
				expect: func(t *testing.T, ws *workspace.AppDev) {
					require.Equal(t, "docker-registry", ws.Config.Registry, "registry")
					require.Equal(t, 5*time.Minute, ws.Config.Timeout, "timeout")
				},
			},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				file := tempFile(t, "dev-config.yaml")
				cmdConfig.Doit.Set(cmdConfig.NS, doctl.ArgAppDevConfig, file)
				cmdConfig.Args = tc.args
				err := RunAppsDevConfigSet(cmdConfig)
				if tc.expectErr != nil {
					require.EqualError(t, err, tc.expectErr.Error())
					return
				}
				require.NoError(t, err, "running command")

				ws, err := appDevWorkspace(cmdConfig)
				require.NoError(t, err, "getting workspace")
				if tc.expect != nil {
					tc.expect(t, ws)
				}
			})
		}
	})
}

func TestRunAppsDevConfigUnset(t *testing.T) {
	withTestClient(t, func(cmdConfig *CmdConfig, tm *tcMocks) {
		tcs := []struct {
			name      string
			args      []string
			pre       func(*testing.T, *workspace.AppDev)
			expectErr error
			expect    func(*testing.T, *workspace.AppDev)
		}{
			{
				name:      "no args",
				args:      []string{},
				expectErr: errors.New("you must provide at least one argument"),
			},
			{
				name: "single key",
				args: []string{"registry"},
				pre: func(t *testing.T, ws *workspace.AppDev) {
					ws.Config.Set("registry", "docker-registry")
					err := ws.Config.Write()
					require.NoError(t, err, "setting up default values")
				},
				expect: func(t *testing.T, ws *workspace.AppDev) {
					require.Equal(t, "", ws.Config.Registry, "registry")
				},
			},
			{
				name: "multiple keys",
				args: []string{"registry", "timeout"},
				pre: func(t *testing.T, ws *workspace.AppDev) {
					ws.Config.Set("registry", "docker-registry")
					ws.Config.Set("timeout", 5*time.Minute)
					err := ws.Config.Write()
					require.NoError(t, err, "setting up default values")
				},
				expect: func(t *testing.T, ws *workspace.AppDev) {
					require.Equal(t, "", ws.Config.Registry, "registry")
					require.Equal(t, time.Duration(0), ws.Config.Timeout, "timeout")
				},
			},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				file := tempFile(t, "dev-config.yaml")
				cmdConfig.Doit.Set(cmdConfig.NS, doctl.ArgAppDevConfig, file)

				ws, err := appDevWorkspace(cmdConfig)
				require.NoError(t, err, "getting workspace")
				if tc.pre != nil {
					tc.pre(t, ws)
				}

				cmdConfig.Args = tc.args
				err = RunAppsDevConfigUnset(cmdConfig)
				if tc.expectErr != nil {
					require.EqualError(t, err, tc.expectErr.Error())
					return
				}
				require.NoError(t, err, "running command")

				ws, err = appDevWorkspace(cmdConfig)
				require.NoError(t, err, "getting workspace")
				if tc.expect != nil {
					tc.expect(t, ws)
				}
			})
		}
	})
}

func tempFile(t *testing.T, name string) (path string) {
	file := filepath.Join(t.TempDir(), name)
	f, err := os.Create(file)
	require.NoError(t, err)
	f.Close()
	return file
}
