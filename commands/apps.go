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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// Apps creates the apps command.
func Apps() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "apps",
			Aliases: []string{"a"},
			Short:   "[Beta] Display commands for working with apps",
			Long:    "[Beta] The subcommands of `doctl app` manage your apps.",
			Hidden:  true,
		},
	}

	create := CmdBuilder(
		cmd,
		RunAppsCreate,
		"create",
		"Create an app",
		`Create an app with the given app spec.`,
		Writer,
		aliasOpt("c"),
		displayerType(&displayers.Apps{}),
	)
	AddStringFlag(create, doctl.ArgAppSpec, "", "", "Path to an app spec in json or yaml format.", requiredOpt())

	CmdBuilder(
		cmd,
		RunAppsGet,
		"get <app id>",
		"Get an app",
		`Get an app with the provided id.

Only basic information is included with the text output format. For complete app details including its spec, use the json format.`,
		Writer,
		aliasOpt("g"),
		displayerType(&displayers.Apps{}),
	)

	CmdBuilder(
		cmd,
		RunAppsList,
		"list",
		"List all apps",
		`List all apps.

Only basic information is included with the text output format. For complete app details including specs, use the json format.`,
		Writer,
		aliasOpt("ls"),
		displayerType(&displayers.Apps{}),
	)

	update := CmdBuilder(
		cmd,
		RunAppsUpdate,
		"update <app id>",
		"Update an app",
		`Update the app with the provided id with the given app spec.`,
		Writer,
		aliasOpt("u"),
		displayerType(&displayers.Apps{}),
	)
	AddStringFlag(update, doctl.ArgAppSpec, "", "", "Path to an app spec in json or yaml format.", requiredOpt())

	CmdBuilder(
		cmd,
		RunAppsDelete,
		"delete <app id>",
		"Deletes an app",
		`Deletes an app with the provided id.

This permanently deletes the app and all its associated deployments.`,
		Writer,
		aliasOpt("d"),
	)

	CmdBuilder(
		cmd,
		RunAppsCreateDeployment,
		"create-deployment <app id>",
		"Create a deployment",
		`Create a deployment for an app.

The deployment will be created using the current app spec.`,
		Writer,
		aliasOpt("cd"),
		displayerType(&displayers.Deployments{}),
	)

	CmdBuilder(
		cmd,
		RunAppsGetDeployment,
		"get-deployment <app id> <deployment id>",
		"Get a deployment",
		`Get a deployment for an app.

Only basic information is included with the text output format. For complete app details including its spec, use the json format.`,
		Writer,
		aliasOpt("gd"),
		displayerType(&displayers.Deployments{}),
	)

	CmdBuilder(
		cmd,
		RunAppsListDeployments,
		"list-deployments <app id>",
		"List all deployments",
		`List all deployments for an app.

Only basic information is included with the text output format. For complete app details including specs, use the json format.`,
		Writer,
		aliasOpt("lsd"),
		displayerType(&displayers.Deployments{}),
	)

	logs := CmdBuilder(
		cmd,
		RunAppsGetLogs,
		"logs <app id> <deployment id> <component name>",
		"Get logs",
		`Get component logs for a deployment of an app.

Three types of logs are supported and can be configured with --`+doctl.ArgAppLogType+`:
- build
- deploy
- run `,
		Writer,
		aliasOpt("l"),
	)
	AddStringFlag(logs, doctl.ArgAppLogType, "", strings.ToLower(string(godo.AppLogTypeRun)), "The type of logs.")

	return cmd
}

// RunAppsCreate creates an app.
func RunAppsCreate(c *CmdConfig) error {
	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAppSpec)
	if err != nil {
		return err
	}

	specFile, err := os.Open(specPath) // guardrails-disable-line
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Failed to open app spec: %s does not exist", specPath)
		}
		return fmt.Errorf("Failed to open app spec: %w", err)
	}
	defer specFile.Close()

	specBytes, err := ioutil.ReadAll(specFile)
	if err != nil {
		return fmt.Errorf("Failed to read app spec: %w", err)
	}

	appSpec, err := parseAppSpec(specBytes)
	if err != nil {
		return err
	}

	app, err := c.Apps().Create(&godo.AppCreateRequest{Spec: appSpec})
	if err != nil {
		return err
	}
	notice("App created")

	return c.Display(displayers.Apps{app})
}

// RunAppsGet gets an app.
func RunAppsGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]

	app, err := c.Apps().Get(id)
	if err != nil {
		return err
	}

	return c.Display(displayers.Apps{app})
}

// RunAppsGet lists all apps.
func RunAppsList(c *CmdConfig) error {
	apps, err := c.Apps().List()
	if err != nil {
		return err
	}

	return c.Display(displayers.Apps(apps))
}

// RunAppsUpdate updates an app.
func RunAppsUpdate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]

	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAppSpec)
	if err != nil {
		return err
	}

	specFile, err := os.Open(specPath) // guardrails-disable-line
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Failed to open app spec: %s does not exist", specPath)
		}
		return fmt.Errorf("Failed to open app spec: %w", err)
	}
	defer specFile.Close()

	specBytes, err := ioutil.ReadAll(specFile)
	if err != nil {
		return fmt.Errorf("Failed to read app spec: %w", err)
	}

	appSpec, err := parseAppSpec(specBytes)
	if err != nil {
		return err
	}

	app, err := c.Apps().Update(id, &godo.AppUpdateRequest{Spec: appSpec})
	if err != nil {
		return err
	}
	notice("App updated")

	return c.Display(displayers.Apps{app})
}

// RunAppsDelete deletes an app.
func RunAppsDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]

	err := c.Apps().Delete(id)
	if err != nil {
		return err
	}
	notice("App deleted")

	return nil
}

// RunAppsCreateDeployment creates a deployment for an app.
func RunAppsCreateDeployment(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]

	deployment, err := c.Apps().CreateDeployment(appID)
	if err != nil {
		return err
	}
	notice("Deployment created")

	return c.Display(displayers.Deployments{deployment})
}

// RunAppsGetDeployment gets a deployment for an app.
func RunAppsGetDeployment(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]
	deploymentID := c.Args[1]

	deployment, err := c.Apps().GetDeployment(appID, deploymentID)
	if err != nil {
		return err
	}

	return c.Display(displayers.Deployments{deployment})
}

// RunAppsListDeployments lists deployments for an app.
func RunAppsListDeployments(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]

	deployments, err := c.Apps().ListDeployments(appID)
	if err != nil {
		return err
	}

	return c.Display(displayers.Deployments(deployments))
}

// RunAppsGetLogs gets app logs for a given component.
func RunAppsGetLogs(c *CmdConfig) error {
	if len(c.Args) < 3 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	appID := c.Args[0]
	deploymentID := c.Args[1]
	component := c.Args[2]

	logTypeStr, err := c.Doit.GetString(c.NS, doctl.ArgAppLogType)
	if err != nil {
		return err
	}
	var logType godo.AppLogType
	switch logTypeStr {
	case strings.ToLower(string(godo.AppLogTypeBuild)):
		logType = godo.AppLogTypeBuild
	case strings.ToLower(string(godo.AppLogTypeDeploy)):
		logType = godo.AppLogTypeDeploy
	case strings.ToLower(string(godo.AppLogTypeRun)):
		logType = godo.AppLogTypeRun
	default:
		return fmt.Errorf("Invalid log type %s", logTypeStr)
	}

	logs, err := c.Apps().GetLogs(appID, deploymentID, component, logType)
	if err != nil {
		return err
	}

	if logs.LiveURL != "" {
		resp, err := http.Get(logs.LiveURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		io.Copy(c.Out, resp.Body)
	} else if len(logs.HistoricURLs) > 0 {
		resp, err := http.Get(logs.HistoricURLs[0])
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		io.Copy(c.Out, resp.Body)
	} else {
		warn("No logs found for app component")
	}

	return nil
}

func parseAppSpec(spec []byte) (*godo.AppSpec, error) {
	var appSpec godo.AppSpec
	err := json.Unmarshal(spec, &appSpec)
	if err == nil {
		return &appSpec, nil
	}

	err = yaml.Unmarshal(spec, &appSpec)
	if err == nil {
		return &appSpec, nil
	}

	return nil, errors.New("Failed to parse app spec: not in json or yaml format")
}
