/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"github.com/digitalocean/doctl"
)

func SandboxExtras(cmd *Command) {

	create := cmdBuilderWithInit(cmd, RunSandboxExtraCreate, "init <path>", "Initialize a local file system directory for the sandbox",
		`The `+"`"+`doctl sandbox init`+"`"+` command specifies a directory in your file system which will hold functions and
supporting artifacts while you're developing them.  When ready, you can upload these to the cloud for
testing.  Later, after the area is committed to a `+"`"+`git`+"`"+` repository, you can create an app from them.
`,
		Writer, false)
	AddStringFlag(create, "language", "l", "js", "Language for the project's initial sample")
	AddBoolFlag(create, "overwrite", "", false, "Clears and reuses an existing directory")

	deploy := cmdBuilderWithInit(cmd, RunSandboxExtraDeploy, "deploy <directories>", "Deploy sandbox local assets to the cloud",
		`At any time you can use `+"`"+`doctl sandbox deploy`+"`"+` to upload the contents of a directory in your file system for
testing in the cloud.  The area must be organized in the fashion expected by an App Platform Functions
component.  The `+"`"+`doctl sandbox init`+"`"+` command will create a properly organized directory for you to work in.`,
		Writer, false)
	AddStringFlag(deploy, "env", "", "", "Path to runtime environment file")
	AddStringFlag(deploy, "build-env", "", "", "Path to build-time environment file")
	AddStringFlag(deploy, "apihost", "", "", "API host to use")
	AddStringFlag(deploy, "auth", "", "", "OpenWhisk auth token to use")
	AddBoolFlag(deploy, "insecure", "", false, "Ignore SSL Certificates")
	AddBoolFlag(deploy, "verbose-build", "", false, "Display build details")
	AddBoolFlag(deploy, "verbose-zip", "", false, "Display start/end of zipping phase for each function")
	AddBoolFlag(deploy, "yarn", "", false, "Use yarn instead of npm for node builds")
	AddStringFlag(deploy, "include", "", "", "Functions or packages to include")
	AddStringFlag(deploy, "exclude", "", "", "Functions or packages to exclude")
	AddBoolFlag(deploy, "remote-build", "", false, "Run builds remotely")
	AddBoolFlag(deploy, "incremental", "", false, "Deploy only changes since last deploy")

	getMetadata := cmdBuilderWithInit(cmd, RunSandboxExtraGetMetadata, "get-metadata <directory>", "Obtain metadata of a sandbox directory",
		`The `+"`"+`doctl sandbox get-metadata`+"`"+` command produces a JSON structure that summarizes the contents of a directory
you have designated for functions development.  This can be useful for feeding into other tools.`,
		Writer, false)
	AddStringFlag(getMetadata, "env", "", "", "Path to environment file")
	AddStringFlag(getMetadata, "include", "", "", "Functions or packages to include")
	AddStringFlag(getMetadata, "exclude", "", "", "Functions or packages to exclude")

	watch := cmdBuilderWithInit(cmd, RunSandboxExtraWatch, "watch <directory>", "Watch a sandbox directory, deploying incrementally on change",
		`Type `+"`"+`doctl sandbox watch <directory>`+"`"+` in a separate terminal window.  It will run until interrupted.
It will watch the directory (which should be one you initialized for sandbox use) and will deploy
the contents to the cloud incrementally as it detects changes.`,
		Writer, false)
	AddStringFlag(watch, "env", "", "", "Path to runtime environment file")
	AddStringFlag(watch, "build-env", "", "", "Path to build-time environment file")
	AddStringFlag(watch, "apihost", "", "", "API host to use")
	AddStringFlag(watch, "auth", "", "", "OpenWhisk auth token to use")
	AddBoolFlag(watch, "insecure", "", false, "Ignore SSL Certificates")
	AddBoolFlag(watch, "verbose-build", "", false, "Display build details")
	AddBoolFlag(watch, "verbose-zip", "", false, "Display start/end of zipping phase for each function")
	AddBoolFlag(watch, "yarn", "", false, "Use yarn instead of npm for node builds")
	AddStringFlag(watch, "include", "", "", "FUnctions and package to include")
	AddStringFlag(watch, "exclude", "", "", "Functions and packages to exclude")
	AddBoolFlag(watch, "remote-build", "", false, "Run builds remotely")
}

func RunSandboxExtraCreate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunSandboxExec("project/create", c, []string{"overwrite"}, []string{"language"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunSandboxExtraDeploy(c *CmdConfig) error {
	output, err := RunSandboxExec("project/deploy", c, []string{"insecure", "verbose-build", "verbose-zip", "yarn", "remote-build", "incremental"}, []string{"env", "build-env", "apihost", "auth", "include", "exclude"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

func RunSandboxExtraGetMetadata(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	output, err := RunSandboxExec("project/get-metadata", c, []string{"json"}, []string{"env", "include", "exclude"})
	if err != nil {
		return err
	}
	PrintSandboxTextOutput(output)
	return nil
}

// This is not the usual boiler-plate because the command is intended to be long-running in a separate window
func RunSandboxExtraWatch(c *CmdConfig) error {
	argCount := len(c.Args)
	if argCount > 1 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	return RunSandboxExecStreaming("project/watch", c, []string{"insecure", "verbose-build", "verbose-zip", "yarn", "remote-build"}, []string{"env", "build-env", "apihost", "auth", "include", "exclude"})
}
