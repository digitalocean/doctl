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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/pkg/browser"
)

// popupWindowSize is the initial width,height of the popup window. Chromium
// ignores it when the browser is already running with a remembered size.
const popupWindowSize = "600,800"

// popupLauncher is one command that opens a URL in a Chromium app window.
type popupLauncher struct {
	path string
	args []string
	// wait is set when the command returns once the window is handed off
	// (macOS open), so its exit status reports whether the launch worked.
	wait bool
}

// popupEnv is the slice of the host environment that popup discovery reads.
type popupEnv struct {
	goos     string
	home     string
	getenv   func(string) string
	lookPath func(string) (string, error)
	exists   func(string) bool
}

func hostPopupEnv() popupEnv {
	home, _ := os.UserHomeDir()
	return popupEnv{
		goos:     runtime.GOOS,
		home:     home,
		getenv:   os.Getenv,
		lookPath: exec.LookPath,
		exists: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
	}
}

// openURLInPopupWindow opens url in a small Chromium app window (no tabs or
// address bar). If no Chromium-based browser is installed or every launch
// fails, it falls back to a regular tab in the default browser.
func openURLInPopupWindow(url string) error {
	for _, l := range popupLaunchers(hostPopupEnv(), url) {
		cmd := exec.Command(l.path, l.args...)
		var err error
		if l.wait {
			err = cmd.Run()
		} else {
			err = cmd.Start()
		}
		if err == nil {
			return nil
		}
	}
	return browser.OpenURL(url)
}

// popupLaunchers lists the app-window commands to try, in preference order,
// for the browsers installed on this host.
func popupLaunchers(env popupEnv, url string) []popupLauncher {
	chromiumArgs := []string{"--app=" + url, "--window-size=" + popupWindowSize}
	var launchers []popupLauncher

	switch env.goos {
	case "darwin":
		for _, app := range []string{"Google Chrome", "Microsoft Edge", "Brave Browser", "Chromium"} {
			for _, dir := range []string{"/Applications", filepath.Join(env.home, "Applications")} {
				if env.exists(filepath.Join(dir, app+".app")) {
					// -n is required: without it macOS drops --args when the
					// browser is already running.
					launchers = append(launchers, popupLauncher{
						path: "open",
						args: append([]string{"-na", app, "--args"}, chromiumArgs...),
						wait: true,
					})
					break
				}
			}
		}
	case "windows":
		var candidates []string
		for _, root := range []string{env.getenv("ProgramFiles"), env.getenv("ProgramFiles(x86)"), env.getenv("LocalAppData")} {
			if root == "" {
				continue
			}
			candidates = append(candidates,
				filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"),
				filepath.Join(root, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
			)
		}
		for _, path := range candidates {
			if env.exists(path) {
				launchers = append(launchers, popupLauncher{path: path, args: chromiumArgs})
			}
		}
	default:
		for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge", "brave-browser"} {
			if path, err := env.lookPath(name); err == nil {
				launchers = append(launchers, popupLauncher{path: path, args: chromiumArgs})
			}
		}
	}

	return launchers
}
