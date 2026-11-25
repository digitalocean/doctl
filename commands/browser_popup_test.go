/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const popupTestURL = "https://cloud.digitalocean.com/account/billing/add-payment-method?i=c949bc"

func fakePopupEnv(goos string, installed ...string) popupEnv {
	set := map[string]bool{}
	for _, p := range installed {
		set[p] = true
	}
	return popupEnv{
		goos: goos,
		home: "/Users/sammy",
		getenv: func(key string) string {
			return map[string]string{
				"ProgramFiles":      `C:\Program Files`,
				"ProgramFiles(x86)": `C:\Program Files (x86)`,
				"LocalAppData":      `C:\Users\sammy\AppData\Local`,
			}[key]
		},
		lookPath: func(name string) (string, error) {
			if set[name] {
				return "/usr/bin/" + name, nil
			}
			return "", errors.New("not found")
		},
		exists: func(path string) bool { return set[path] },
	}
}

func TestPopupLaunchersDarwin(t *testing.T) {
	env := fakePopupEnv("darwin",
		"/Applications/Microsoft Edge.app",
		"/Users/sammy/Applications/Google Chrome.app",
	)

	launchers := popupLaunchers(env, popupTestURL)

	assert.Equal(t, []popupLauncher{
		{
			path: "open",
			args: []string{"-na", "Google Chrome", "--args", "--app=" + popupTestURL, "--window-size=" + popupWindowSize},
			wait: true,
		},
		{
			path: "open",
			args: []string{"-na", "Microsoft Edge", "--args", "--app=" + popupTestURL, "--window-size=" + popupWindowSize},
			wait: true,
		},
	}, launchers)
}

func TestPopupLaunchersWindows(t *testing.T) {
	edge := filepath.Join(`C:\Program Files (x86)`, "Microsoft", "Edge", "Application", "msedge.exe")
	env := fakePopupEnv("windows", edge)

	launchers := popupLaunchers(env, popupTestURL)

	assert.Equal(t, []popupLauncher{
		{path: edge, args: []string{"--app=" + popupTestURL, "--window-size=" + popupWindowSize}},
	}, launchers)
}

func TestPopupLaunchersLinux(t *testing.T) {
	env := fakePopupEnv("linux", "chromium")

	launchers := popupLaunchers(env, popupTestURL)

	assert.Equal(t, []popupLauncher{
		{path: "/usr/bin/chromium", args: []string{"--app=" + popupTestURL, "--window-size=" + popupWindowSize}},
	}, launchers)
}

func TestPopupLaunchersNoChromiumBrowser(t *testing.T) {
	for _, goos := range []string{"darwin", "windows", "linux"} {
		assert.Empty(t, popupLaunchers(fakePopupEnv(goos), popupTestURL), goos)
	}
}
