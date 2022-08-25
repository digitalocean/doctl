package commands

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/apps/workspace"
	"github.com/stretchr/testify/require"
)

func TestRunAppsDevConfigSet(t *testing.T) {
	file, err := ioutil.TempFile("", "dev-config.*.yaml")
	require.NoError(t, err, "creating temp file")
	t.Cleanup(func() {
		file.Close()
		os.Remove(file.Name())
	})

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
				args: []string{"app=12345"},
				expect: func(t *testing.T, ws *workspace.AppDev) {
					require.Equal(t, "12345", ws.Config.Global(false).GetString("app"), "app-id")
				},
			},
			{
				name: "multiple keys",
				args: []string{"app=value1", "spec=value2"},
				expect: func(t *testing.T, ws *workspace.AppDev) {
					require.Equal(t, "value1", ws.Config.Global(false).GetString("app"), "app-id")
					require.Equal(t, "value2", ws.Config.Global(false).GetString("spec"), "spec")
				},
			},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				cmdConfig.Doit.Set(cmdConfig.NS, doctl.ArgAppDevConfig, file.Name())

				ws, err := appDevWorkspace(cmdConfig)
				require.NoError(t, err, "getting workspace")
				cmdConfig.Args = tc.args
				err = RunAppsDevConfigSet(cmdConfig)
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

func TestRunAppsDevConfigUnset(t *testing.T) {
	file, err := ioutil.TempFile("", "dev-config.*.yaml")
	require.NoError(t, err, "creating temp file")
	t.Cleanup(func() {
		file.Close()
		os.Remove(file.Name())
	})

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
				args: []string{"app"},
				pre: func(t *testing.T, ws *workspace.AppDev) {
					ws.Config.Set("app", "value")
					err := ws.Config.Write()
					require.NoError(t, err, "setting up default values")
				},
				expect: func(t *testing.T, ws *workspace.AppDev) {
					require.Equal(t, "", ws.Config.Global(false).GetString("app"), "app-id")
				},
			},
			{
				name: "multiple keys",
				args: []string{"app", "spec"},
				pre: func(t *testing.T, ws *workspace.AppDev) {
					ws.Config.Set("app", "value")
					ws.Config.Set("spec", "value")
					err := ws.Config.Write()
					require.NoError(t, err, "setting up default values")
				},
				expect: func(t *testing.T, ws *workspace.AppDev) {
					require.Equal(t, "", ws.Config.Global(false).GetString("app"), "app-id")
					require.Equal(t, "", ws.Config.Global(false).GetString("spec"), "spec")
				},
			},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				cmdConfig.Doit.Set(cmdConfig.NS, doctl.ArgAppDevConfig, file.Name())

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
