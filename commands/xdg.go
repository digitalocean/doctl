// Copyright 2018 The Doctl Authors All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !windows

package commands

import (
	"fmt"
	"os"
	"path/filepath"
)

func configHome() string {
	if xdgPath := os.Getenv("XDG_CONFIG_HOME"); xdgPath != "" {
		return filepath.Join(xdgPath, "doctl")
	}
	return filepath.Join(homeDir(), ".config", "doctl")
}

func legacyConfigCheck() {
	fi, err := os.Stat(cfgFile)
	expectedPerms := os.FileMode(0600)
	if err == nil && fi.Mode() != expectedPerms {
		msg := fmt.Sprintf("Configuration %q permissions are %#o. Should be set to %#o",
			cfgFile, fi.Mode(), expectedPerms)
		warn(msg)
	}
}
