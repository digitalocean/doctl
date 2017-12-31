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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigHome(t *testing.T) {
	os.Setenv("USERNAME", "testuser")
	defer os.Unsetenv("USERNAME")

	ch := configHome()
	expected := `C:\Users\testuser\AppData\Local\doctl\config`
	assert.Equal(t, expected, ch)
}

func TestAliasHome(t *testing.T) {
	os.Setenv("USERNAME", "testuser")
	defer os.Unsetenv("USERNAME")

	ch := aliasHome()
	expected := `C:\Users\testuser\AppData\Local\doctl\alias`
	assert.Equal(t, expected, ch)
}
