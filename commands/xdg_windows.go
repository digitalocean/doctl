// Copyright 2016 The Doctl Authors All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build windows

package commands

import (
	"os"
	"path/filepath"
)

func configHome() string {
	// is this even a thing on windows?
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		userName := os.Getenv("USERNAME")
		configHome = filepath.Join("C:/", "Users", userName, "AppData", "Local", "doctl", "config")
	}

	return configHome
}

func aliasHome() string {
	aliasHome := os.Getenv("XDG_CONFIG_HOME")
	if aliasHome == "" {
		userName := os.Getenv("USERNAME")
		aliasHome = filepath.Join("C:/", "Users", userName, "AppData", "Local", "doctl", "alias")
	}

	return aliasHome
}

// legacyConfigCheck is a no-op on windows since go doesn't have a chmod
// on this platform.
func legacyConfigCheck() {
}
