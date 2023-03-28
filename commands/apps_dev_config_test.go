package commands

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/apps/workspace"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestRunAppsDevConfigSet(t *testing.T) {
	withTestClient(t, func(cmdConfig *CmdConfig, tm *tcMocks) {
		tcs := []struct {
			name string
			args []string
			// appSpec is an optional app spec for the workspace
			appSpec   *godo.AppSpec
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
			{
				name: "component setting",
				appSpec: &godo.AppSpec{
					// Note: the service name intentionally includes a dash to ensure that the appsDevFlagConfigCompat
					// mutator works as expected -- i.e. only dashes in config options are mutated but not component names.
					// `www-svc` remains `www-svc` but `build-command` becomes `build_command`.
					Services: []*godo.AppServiceSpec{{Name: "www-svc"}},
				},
				args: []string{"components.www-svc.build_command=npm run start"},
				expect: func(t *testing.T, ws *workspace.AppDev) {
					cc := ws.Config.Components["www-svc"]
					require.NotNil(t, cc, "component config exists")
					require.Equal(t, "npm run start", cc.BuildCommand)
				},
			},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				configFile := tempFile(t, "dev-config.yaml")
				cmdConfig.Doit.Set(cmdConfig.NS, doctl.ArgAppDevConfig, configFile)
				cmdConfig.Args = tc.args

				if tc.appSpec != nil {
					appSpecFile := tempAppSpec(t, tc.appSpec)
					cmdConfig.Doit.Set(cmdConfig.NS, doctl.ArgAppSpec, appSpecFile)
				}

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
			name string
			args []string
			// appSpec is an optional app spec for the workspace
			appSpec   *godo.AppSpec
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
			{
				name: "component setting",
				args: []string{"components.www-svc.build_command"},
				appSpec: &godo.AppSpec{
					Services: []*godo.AppServiceSpec{{Name: "www-svc"}},
				},
				pre: func(t *testing.T, ws *workspace.AppDev) {
					ws.Config.Set("components.www-svc.build_command", "npm run start")
					err := ws.Config.Write()
					require.NoError(t, err, "setting up default values")
				},
				expect: func(t *testing.T, ws *workspace.AppDev) {
					cc := ws.Config.Components["www-svc"]
					require.NotNil(t, cc, "component config exists")
					require.Equal(t, "", cc.BuildCommand)
				},
			},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				file := tempFile(t, "dev-config.yaml")
				cmdConfig.Doit.Set(cmdConfig.NS, doctl.ArgAppDevConfig, file)

				if tc.appSpec != nil {
					appSpecFile := tempAppSpec(t, tc.appSpec)
					cmdConfig.Doit.Set(cmdConfig.NS, doctl.ArgAppSpec, appSpecFile)
				}

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

func tempAppSpec(t *testing.T, spec *godo.AppSpec) (path string) {
	path = tempFile(t, "app.yaml")
	specYaml, err := yaml.Marshal(spec)
	require.NoError(t, err, "marshaling app spec")
	err = ioutil.WriteFile(path, specYaml, 0664)
	require.NoError(t, err, "writing app spec to disk")
	return
}
