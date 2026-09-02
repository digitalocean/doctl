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

package ui_test

import (
	"io"
	"strings"
	"testing"

	"github.com/digitalocean/doctl/internal/ui"
	"github.com/muesli/termenv"
)

func TestStyleUsesDesignHexColors(t *testing.T) {
	env := ui.Detect(io.Discard, io.Discard, ui.WithProfile(termenv.TrueColor), ui.WithASCII(false))
	style := ui.NewStyle(env)

	label := style.ErrorLabel()
	// #d74623 => rgb(215, 70, 35)
	if !strings.Contains(label, "38;2;215;70;35") {
		t.Fatalf("error label missing design red #d74623 RGB; got %q", label)
	}
	if !strings.Contains(label, "Error:") {
		t.Fatalf("error label missing text; got %q", label)
	}

	dim := style.Dim("hint")
	// #8090a0 => rgb(128, 144, 160)
	if !strings.Contains(dim, "38;2;128;144;160") {
		t.Fatalf("dim missing design #8090a0 RGB; got %q", dim)
	}

	green := style.Success("ok")
	if !strings.Contains(green, "38;2;0;") || !strings.Contains(green, ";131m") {
		t.Fatalf("green missing design #00c483 RGB; got %q", green)
	}
}
