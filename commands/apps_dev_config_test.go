package commands

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/apps/config"
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
			expect    func(*testing.T, *config.AppDev)
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
				expect: func(t *testing.T, c *config.AppDev) {
					require.Equal(t, "12345", c.GetString("app"), "app-id")
				},
			},
			{
				name: "multiple keys",
				args: []string{"app=value1", "spec=value2"},
				expect: func(t *testing.T, c *config.AppDev) {
					require.Equal(t, "value1", c.GetString("app"), "app-id")
					require.Equal(t, "value2", c.GetString("spec"), "spec")
				},
			},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				cmdConfig.Args = tc.args
				cmdConfig.Doit.Set(cmdConfig.NS, doctl.ArgAppDevConfig, file.Name())
				err = RunAppsDevConfigSet(cmdConfig)
				if tc.expectErr != nil {
					require.EqualError(t, err, tc.expectErr.Error())
					return
				}
				require.NoError(t, err, "running command")
				devConf, err := newAppDevConfig(cmdConfig)
				require.NoError(t, err, "getting dev config")
				if tc.expect != nil {
					tc.expect(t, devConf)
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
			pre       func(*testing.T, *config.AppDev)
			expectErr error
			expect    func(*testing.T, *config.AppDev)
		}{
			{
				name:      "no args",
				args:      []string{},
				expectErr: errors.New("you must provide at least one argument"),
			},
			{
				name: "single key",
				args: []string{"app"},
				pre: func(t *testing.T, c *config.AppDev) {
					c.Set("app", "value")
					err := c.WriteConfig()
					require.NoError(t, err, "setting up default values")
				},
				expect: func(t *testing.T, c *config.AppDev) {
					require.Equal(t, "", c.GetString("app"), "app-id")
				},
			},
			{
				name: "multiple keys",
				args: []string{"app", "spec"},
				pre: func(t *testing.T, c *config.AppDev) {
					c.Set("app", "value")
					c.Set("spec", "value")
					err := c.WriteConfig()
					require.NoError(t, err, "setting up default values")
				},
				expect: func(t *testing.T, c *config.AppDev) {
					require.Equal(t, "", c.GetString("app"), "app-id")
					require.Equal(t, "", c.GetString("spec"), "spec")
				},
			},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				devConf, err := newAppDevConfig(cmdConfig)
				require.NoError(t, err, "getting dev config")
				if tc.pre != nil {
					tc.pre(t, devConf)
				}

				cmdConfig.Args = tc.args
				cmdConfig.Doit.Set(cmdConfig.NS, doctl.ArgAppDevConfig, file.Name())
				err = RunAppsDevConfigUnset(cmdConfig)
				if tc.expectErr != nil {
					require.EqualError(t, err, tc.expectErr.Error())
					return
				}
				require.NoError(t, err, "running command")

				if tc.expect != nil {
					devConf, err = newAppDevConfig(cmdConfig)
					require.NoError(t, err, "getting dev config")
					tc.expect(t, devConf)
				}
			})
		}
	})
}

func Test_ensureStringInFile(t *testing.T) {
	ensureValue := "newvalue"

	tcs := []struct {
		name   string
		pre    func(t *testing.T, fname string)
		expect []byte
	}{
		{
			name:   "no pre-existing file",
			pre:    func(t *testing.T, fname string) {},
			expect: []byte(ensureValue),
		},
		{
			name: "pre-existing file with value",
			pre: func(t *testing.T, fname string) {
				f, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				require.NoError(t, err)
				f.WriteString("line1\n" + ensureValue)
				f.Close()
			},
			expect: []byte("line1\n" + ensureValue),
		},
		{
			name: "pre-existing file without value",
			pre: func(t *testing.T, fname string) {
				f, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				require.NoError(t, err)
				f.WriteString("line1\n")
				f.Close()
			},
			expect: []byte("line1\n" + ensureValue),
		},
		{
			name: "pre-existing file without value or newline",
			pre: func(t *testing.T, fname string) {
				f, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				require.NoError(t, err)
				f.WriteString("line1")
				f.Close()
			},
			expect: []byte("line1\n" + ensureValue),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			file, err := ioutil.TempFile("", "dev-config.*.yaml")
			require.NoError(t, err, "creating temp file")
			file.Close()

			// allow the test to dictate existence; we just use this
			// to get a valid temporary filename that is unique
			err = os.Remove(file.Name())
			require.NoError(t, err, "deleting temp file")

			if tc.pre != nil {
				tc.pre(t, file.Name())
			}

			err = ensureStringInFile(file.Name(), ensureValue)
			require.NoError(t, err, "ensuring string in file")

			b, err := ioutil.ReadFile(file.Name())
			require.NoError(t, err)
			require.Equal(t, string(tc.expect), string(b))
		})
	}
}
