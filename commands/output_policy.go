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
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/spf13/viper"

	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/internal/ui"
)

// resolvedEnv is the Env for this invocation, nil until installOutputPolicy
// sets it. That is the only write and cobra makes it on one goroutine before
// the command body runs, so reads need no synchronization.
var resolvedEnv *ui.Env

// stdoutSink is the reporter every write to the process's stdout goes through,
// nil until installOutputPolicy sets it. It wraps Writer rather than replacing
// it because detection has to see the real file: ui.isTerminal type-asserts an
// *os.File, and a wrapper would cost color, width and the card layout.
var stdoutSink *reportingWriter

// uiEnv returns the capabilities governing this invocation.
func uiEnv() ui.Env {
	if resolvedEnv != nil {
		return *resolvedEnv
	}

	return detectUIEnv()
}

// resolveUIEnv returns the capabilities governing writes to out. Policy from
// --output and --interactive is shared, but detection is per stream.
func resolveUIEnv(out io.Writer) ui.Env {
	if out == nil || out == io.Writer(Writer) {
		return uiEnv()
	}

	return detectUIEnvFor(out)
}

func detectUIEnv() ui.Env {
	return detectUIEnvFor(Writer)
}

func detectUIEnvFor(out io.Writer) ui.Env {
	return ui.Detect(out, os.Stderr,
		ui.WithMachineOutput(outputFormat() != "text"),
		ui.WithInteractive(Interactive),
		ui.WithShow(Show),
	)
}

// installOutputPolicy resolves the Env once and retargets the process-global
// styling stacks: lipgloss and fatih/color both resolve color from state decided
// before doctl knows anything about the invocation. They are pointed at
// different streams because their call sites write to different streams.
func installOutputPolicy() {
	env := detectUIEnv()
	resolvedEnv = &env

	lipgloss.SetColorProfile(env.DataProfile())
	color.NoColor = !env.ErrStyle
	stdoutSink = &reportingWriter{out: env.Writer()}
	template.Output, template.ErrOutput = stdoutSink, env.ErrWriter()
}

// outputFormat reports the requested format, which viper knows only once
// initConfig has folded in config.yaml and the bound --output flag.
func outputFormat() string {
	if v := viper.GetString("output"); v != "" {
		return v
	}

	return Output
}
